// Copyright 2015 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package amicache

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/intuit/ami-query/ami"
)

// Supported regions
var supportedRegions = []string{
	"us-east-1",
	"us-west-1",
	"us-west-2",
	"eu-west-1",
	"eu-central-1",
	"ap-northeast-1",
	"ap-southeast-1",
	"ap-southeast-2",
	"sa-east-1",
	"cn-north-1",
}

// Minimum time allowed between cache updates
const minTTL = 1 * time.Minute

// Manager manages the underlying cache of images polled from AWS.
type Manager struct {
	// The underlying cache being used
	c CacheManager

	// AWS API credentials
	awsCreds *credentials.Credentials

	// HTTP client used to communicate with AWS
	client *http.Client

	// The list of supported regions to query
	regions []string

	// Owner IDs used to filter AMI results
	ownerIDs []*string

	// Duration between updates to the cache
	ttl time.Duration

	// Whether or not the cache manager is running
	running bool

	// Quit channel for the CacheManager goroutine
	quitCh chan struct{}

	// Image IDs indexed by region
	imageIDs map[string][]string

	// Read/Write lock to protect imageIDs
	mu sync.RWMutex
}

// NewManager returns a Manager with sensible defaults if none are provided.
func NewManager(cache CacheManager, options ...Option) (*Manager, error) {
	m := Manager{
		c:        cache,
		awsCreds: defaults.DefaultChainCredentials,
		client:   http.DefaultClient,
		regions:  supportedRegions,
		ownerIDs: []*string{},
		ttl:      15 * time.Minute,
		imageIDs: make(map[string][]string),
		quitCh:   make(chan struct{}),
	}
	return &m, m.setOption(options...)
}

// Option is the option interface. It has private methods to prevent its use
// outside of this package.
type Option interface {
	set(*Manager) error
}

// optionFunc is a function adapter that implements the Option interface.
type optionFunc func(*Manager) error

func (fn optionFunc) set(m *Manager) error {
	return fn(m)
}

// AWSCreds sets credentials used to access the AWS APIs.
func AWSCreds(creds *credentials.Credentials) Option {
	return optionFunc(func(m *Manager) error {
		m.awsCreds = creds
		return nil
	})
}

// HTTPClient sets the http.Client used for communicating with the AWS APIs.
func HTTPClient(client *http.Client) Option {
	return optionFunc(func(m *Manager) error {
		if client != nil {
			m.client = client
		}
		return nil
	})
}

// AssumeRole sets the STS role assumed by AMIQuery.
func AssumeRole(roleARN string) Option {
	return optionFunc(func(m *Manager) error {
		if roleARN != "" {
			m.awsCreds = stscreds.NewCredentials(nil, roleARN, time.Minute*5)
		}
		return nil
	})
}

// Regions sets the AWS regions that will be polled for images.
func Regions(regions ...string) Option {
	return optionFunc(func(m *Manager) error {
		if len(regions) == 0 {
			return nil
		}
		m.regions = []string{}
		for _, region := range regions {
			if !supportedRegion(region) {
				return fmt.Errorf("unknown or unsupported region: %s", region)
			}
			m.regions = append(m.regions, region)
		}
		return nil
	})
}

// OwnerIDs sets the owner IDs used to filter images retrieved from AWS.
func OwnerIDs(ids ...string) Option {
	return optionFunc(func(m *Manager) error {
		if len(ids) > 0 {
			for _, id := range ids {
				m.ownerIDs = append(m.ownerIDs, aws.String(id))
			}
		}
		return nil
	})
}

// TTL sets the duration between cache updates. The minimum allowed TTL is 1
// minute.
func TTL(ttl time.Duration) Option {
	return optionFunc(func(m *Manager) error {
		if ttl < minTTL {
			m.ttl = minTTL
		} else {
			m.ttl = ttl
		}
		return nil
	})
}

// CacheManager is the interface used by a Manager to maintain the underlying
// cache.
type CacheManager interface {
	Name() string
	Start() error
	Stop() error
	Set(ami.AMI) error
	Get(string) (ami.AMI, error)
	Delete(string) error
}

// setOptions configures the Manager.
func (m *Manager) setOption(options ...Option) error {
	for _, opt := range options {
		if err := opt.set(m); err != nil {
			return err
		}
	}
	return nil
}

// Start starts the cache manager. It returns a channel that blocks until the
// cache has been filled for the first time.
func (m *Manager) Start() <-chan struct{} {
	filled := make(chan struct{})
	if !m.running {
		var err error
		if filled, err = m.startManager(); err != nil {
			panic(err)
		}
	} else {
		close(filled) // already running so unblock filled
	}
	m.running = true
	return filled
}

// Stop stops the cache manager.
func (m *Manager) Stop() {
	if m.running {
		if err := m.stopManager(); err != nil {
			log.Printf("[%s] failed to cleanly stop: %s", m.c.Name(), err)
			return
		}
	}
	m.running = false
}

// Regions returns the regions being cached.
func (m *Manager) Regions() []string {
	return m.regions
}

// Images returns the cached Images from the provided region.
func (m *Manager) Images(region string) ([]ami.AMI, error) {
	ids, err := m.amiList(region)
	if err != nil {
		return nil, err
	}
	images := make([]ami.AMI, len(ids))
	for i, id := range ids {
		image, err := m.c.Get(id)
		if err != nil {
			return nil, err
		}
		images[i] = image
	}
	return images, nil
}

// FilterImages returns a filtered set of cached images from the provided region.
func (m *Manager) FilterImages(region string, filter *Filter) ([]ami.AMI, error) {
	amis, err := m.Images(region)
	if err != nil {
		return nil, err
	}
	return filter.Apply(amis), nil
}

// startManager runs the cache update loop in a goroutine. It returns a channel
// that will block until the cache is filled for the first time.
func (m *Manager) startManager() (chan struct{}, error) {
	if err := m.c.Start(); err != nil {
		return nil, err
	}

	filled := make(chan struct{})

	go func() {
		m.updateCache()
		close(filled)
		for {
			select {
			case <-time.After(m.ttl):
				m.updateCache()
			case <-m.quitCh:
				m.quitCh <- struct{}{}
				return
			}
		}
	}()

	return filled, nil
}

func (m *Manager) stopManager() error {
	m.quitCh <- struct{}{}
	<-m.quitCh
	return m.c.Stop()
}

// updateCache iterates over AWS regions and caches the images.
func (m *Manager) updateCache() {
	wg := sync.WaitGroup{}
	getImages := func(region string) {
		defer wg.Done()

		conn := ec2.New(&aws.Config{
			HTTPClient:  m.client,
			Credentials: m.awsCreds,
			Region:      aws.String(region),
			Endpoint:    aws.String(EC2Endpoint(region)),
		})

		resp, err := conn.DescribeImages(&ec2.DescribeImagesInput{
			Owners: m.ownerIDs,
		})

		if err != nil {
			log.Printf("[%s] cache update failed for %s: %s", m.c.Name(), region, err)
			return
		}

		ids := make([]string, len(resp.Images))

		for i, image := range resp.Images {
			ami := ami.NewAMI(region, image)
			if err = m.c.Set(ami); err != nil {
				log.Printf("[%s] cache write failed for %s: %s", m.c.Name(), region, err)
				return
			}
			ids[i] = *image.ImageId
		}

		m.mu.Lock()
		m.imageIDs[region] = ids
		m.mu.Unlock()

		log.Printf("[%s] cached %d AMIs from %s", m.c.Name(), len(resp.Images), region)
	}

	wg.Add(len(m.regions))
	for _, region := range m.regions {
		go getImages(region)
	}
	wg.Wait()
}

// amiList returns the list of AMI IDs for the provided region.
func (m *Manager) amiList(region string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !supportedRegion(region) {
		return nil, fmt.Errorf("unknown or unsupported region: %s", region)
	}
	ids, ok := m.imageIDs[region]
	if !ok {
		return []string{}, nil
	}
	return ids, nil
}

// supportedRegion returns true of the provided region is in supportedRegions.
func supportedRegion(region string) bool {
	for _, r := range supportedRegions {
		if r == region {
			return true
		}
	}
	return false
}

// EC2Endpoint returns the AWS EC2 endpoint for the provided region. By default
// it returns an empty string and allows the `aws-sdk-go` library to set it
// appropriately. It's stubbed out here for testing mock endpoints.
var EC2Endpoint = func(region string) string {
	return ""
}
