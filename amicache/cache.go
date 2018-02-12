// Copyright 2015 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package amicache

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// The minimum time allowed between cache updates.
const minCacheTTL = 5 * time.Minute

// Option is the option interface. It has private methods to prevent its use
// from outside of this package.
type Option interface {
	set(*Cache)
}

// optionFunc is a function adapter that implements the Option interface.
type optionFunc func(*Cache)

func (fn optionFunc) set(m *Cache) { fn(m) }

// TagFilter sets the tag-key used to filter the results of ec2:DescribeImages.
// The value is irrelevant, only the existence of the tag is required.
func TagFilter(tag string) Option {
	return optionFunc(func(c *Cache) {
		c.tagFilter = tag
	})
}

// Regions sets the AWS standard regions that will be polled for AMIs.
func Regions(regions ...string) Option {
	return optionFunc(func(c *Cache) {
		if len(regions) > 0 {
			c.regions = map[string]struct{}{}
			for _, region := range regions {
				c.regions[region] = struct{}{}
			}
		}
	})
}

// TTL sets the duration between cache updates.
func TTL(ttl time.Duration) Option {
	return optionFunc(func(c *Cache) {
		if ttl < minCacheTTL {
			level.Info(c.logger).Log(
				"msg", fmt.Sprintf("%s TTL is too low, adjusting to %s", ttl, minCacheTTL),
			)
			c.ttl = minCacheTTL
		} else {
			c.ttl = ttl
		}
	})
}

// MaxConcurrentRequests sets the maximum number of concurrent DescribeImageAttributes
// API requests for a given Owner.
//
// This is used to control RequestLimitExceed errors.
// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/query-api-troubleshooting.html#api-request-rate
func MaxConcurrentRequests(max int) Option {
	return optionFunc(func(c *Cache) {
		if max > 0 {
			c.maxRequests = max
		}
	})
}

// MaxRequestRetries sets the maximum number of retries for the DescribeImageAttributes
// API request for a given AMI.
//
// This is used to control RequestLimitExceed errors.
// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/query-api-troubleshooting.html#api-request-rate
func MaxRequestRetries(max int) Option {
	return optionFunc(func(c *Cache) {
		if max > 0 {
			c.maxRetries = max
		}
	})
}

// HTTPClient sets the http.Client used for communicating with the AWS APIs.
func HTTPClient(client *http.Client) Option {
	return optionFunc(func(c *Cache) {
		if client != nil {
			c.httpClient = client
		}
	})
}

// Logger sets the go-kit logger.
func Logger(logger log.Logger) Option {
	return optionFunc(func(c *Cache) {
		if logger != nil {
			c.logger = logger
		}
	})
}

// Cache manages the images polled from AWS.
type Cache struct {
	svc         stsiface.STSAPI     // The AWS STS service API client
	roleName    string              // The role assumed in targeted accounts
	ownerIDs    []string            // Owner IDs used to filter AMI results
	cache       map[string]Image    // The cache of AMIs
	regionIndex map[string][]string // Image IDs index by region
	mu          sync.RWMutex        // guards cache and regionIndex
	regions     map[string]struct{} // The list of regions polled for AMIs
	tagFilter   string              // The name of a tag used to filter ec2:DescribeImages
	ttl         time.Duration       // Duration between updates to the cache (default: 15m)
	maxRequests int                 // Max number of goroutines used for DescribeImageAttributes API requests.
	maxRetries  int                 // Max number of retries for DescribeImageAttributes API requests.
	httpClient  *http.Client        // HTTP client used to communicate with AWS
	logger      log.Logger          // go-kit logger
	quitCh      chan chan struct{}  // Used to signal stopping the cache
	running     int32               // accessed atomically (non-zero means it's running)

	// Used to mock out creating an ec2 service for testing.
	ec2Svc func(*session.Session, string, int) ec2iface.EC2API
}

// New returns a Cache with sensible defaults if none are provided.
func New(svc stsiface.STSAPI, roleName string, ownerIDs []string, options ...Option) *Cache {
	c := Cache{
		svc:         svc,
		roleName:    roleName,
		ownerIDs:    ownerIDs,
		cache:       map[string]Image{},
		regionIndex: map[string][]string{},
		regions:     awsStdRegions(),
		ttl:         15 * time.Minute,
		maxRequests: 15,
		maxRetries:  5,
		httpClient:  http.DefaultClient,
		logger:      log.NewNopLogger(),
		quitCh:      make(chan chan struct{}),
		ec2Svc: func(sess *session.Session, region string, maxRetries int) ec2iface.EC2API {
			return ec2.New(sess, aws.NewConfig().
				WithRegion(region).
				WithMaxRetries(maxRetries),
			)
		},
	}
	c.setOptions(options)
	return &c
}

var (
	errCacheRunning = errors.New("cache running")
	errCacheStopped = errors.New("cache stopped")
)

// Run starts the cache and keeps it up to date. It closes warmed after the
// first cache update completes.
func (c *Cache) Run(ctx context.Context, warmed chan struct{}) error {
	if c.isRunning() {
		return errCacheRunning
	}

	atomic.AddInt32(&c.running, 1)
	defer atomic.AddInt32(&c.running, -1)

	// Use a separate warmed channel in case the provided one is nil.
	isWarmed := make(chan struct{})

	go func() {
		c.updateCache(ctx)
		close(isWarmed)
		if warmed != nil {
			close(warmed)
		}
	}()

	for {
		select {
		case <-time.After(c.ttl):
			<-isWarmed // wait just in case the initial update is taking awhile
			c.updateCache(ctx)
		case <-ctx.Done():
			return ctx.Err()
		case q := <-c.quitCh:
			close(q)
			return errCacheStopped
		}
	}
}

// Stop stops the cache.
func (c *Cache) Stop() {
	if c.isRunning() {
		quit := make(chan struct{})
		c.quitCh <- quit
		<-quit
	}
}

// Images returns the cached images from the provided region.
func (c *Cache) Images(region string) ([]Image, error) {
	ids, err := c.idsFromRegion(region)
	if err != nil {
		return nil, err
	}
	images := []Image{}
	for _, id := range ids {
		if image, ok := c.getImage(id); ok {
			images = append(images, image)
		}
	}
	return images, nil
}

// FilterImages returns a filtered set of cached images from the provided region.
func (c *Cache) FilterImages(region string, filter *Filter) ([]Image, error) {
	images, err := c.Images(region)
	if err != nil {
		return nil, err
	}
	return filter.Apply(images), nil
}

// Regions returns the list of AWS regions being cached.
func (c *Cache) Regions() []string {
	regions := []string{}
	for region := range c.regions {
		regions = append(regions, region)
	}
	return regions
}

// setOptions configures a Manager.
func (c *Cache) setOptions(options []Option) {
	for _, opt := range options {
		opt.set(c)
	}
}

// Returns whether or not the cache is running.
func (c *Cache) isRunning() bool {
	return atomic.LoadInt32(&c.running) != 0
}

// updateCache iterates over AWS accounts and regions to cache the images.
func (c *Cache) updateCache(ctx context.Context) {
	var (
		newCache = map[string]Image{}
		newIndex = map[string][]string{}
		doneCh   = make(chan struct{})
		mu       = sync.Mutex{}
		wg       = sync.WaitGroup{}
	)

	wg.Add(len(c.ownerIDs))

	for _, owner := range c.ownerIDs {
		go func(owner string) {
			defer wg.Done()
			logger := log.With(c.logger, "owner_id", owner)

			sess, err := c.assumeRole(owner)
			if err != nil {
				level.Warn(logger).Log("cache_update", "failed", "error", awsError(err))
				return
			}

			wg.Add(len(c.regions))

			for region := range c.regions {
				go func(region string) {
					defer wg.Done()
					logger := log.With(logger, "region", region)

					svc := c.ec2Svc(sess, region, c.maxRetries)
					images, index := getImagesFromOwner(svc, logger, owner, region, c.tagFilter, c.maxRequests)

					mu.Lock()
					newIndex[region] = append(newIndex[region], index...)
					for _, image := range images {
						newCache[*image.Image.ImageId] = image
					}
					mu.Unlock()

					level.Info(logger).Log("cache_update", "completed", "count", len(images))
				}(region)
			}
		}(owner)
	}

	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
	case <-ctx.Done():
		return
	}

	c.mu.Lock()
	c.cache = newCache
	c.regionIndex = newIndex
	c.mu.Unlock()
}

// getImage gets returns an image from the cache if it exists.
func (c *Cache) getImage(id string) (Image, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	image, ok := c.cache[id]
	return image, ok
}

// idsFromRegion returns the list of image IDs from the index.
func (c *Cache) idsFromRegion(region string) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if _, ok := c.regions[region]; !ok {
		return nil, fmt.Errorf("unknown or unsupported region: %s", region)
	}

	ids, ok := c.regionIndex[region]
	if !ok {
		return []string{}, nil
	}

	return ids, nil
}

// Create a session in the targeted account using a service role.
func (c *Cache) assumeRole(account string) (*session.Session, error) {
	rsp, err := c.svc.AssumeRole(&sts.AssumeRoleInput{
		RoleArn:         aws.String(fmt.Sprintf("arn:aws:iam::%s:role/%s", account, c.roleName)),
		Policy:          aws.String(policyDoc),
		RoleSessionName: aws.String("ami-query"),
		DurationSeconds: aws.Int64(900),
	})
	if err != nil {
		return nil, err
	}
	return session.NewSession(aws.NewConfig().
		WithHTTPClient(c.httpClient).
		WithCredentials(credentials.NewStaticCredentials(
			*rsp.Credentials.AccessKeyId,
			*rsp.Credentials.SecretAccessKey,
			*rsp.Credentials.SessionToken,
		)),
	)
}

// getImagesFromOwner gets the images and assoicated launch permissions from the
// provided owner. In accounts with a large number of AMIs (~150 or more), this
// may hit RequestLimitExeeded and trigger retries.
func getImagesFromOwner(svc ec2iface.EC2API, logger log.Logger, owner, region, tagFilter string, maxReq int) ([]Image, []string) {
	input := &ec2.DescribeImagesInput{
		Owners: []*string{aws.String(owner)},
	}

	if tagFilter != "" {
		input.Filters = []*ec2.Filter{{
			Name:   aws.String("tag-key"),
			Values: []*string{aws.String(tagFilter)},
		}}
	}

	rsp, err := svc.DescribeImages(input)
	if err != nil {
		level.Warn(logger).Log("cache_update", "failed", "error", awsError(err))
		return []Image{}, []string{}
	}

	var (
		index    = []string{}
		images   = []Image{}
		mu       = sync.Mutex{}
		wg       = sync.WaitGroup{}
		workerCh = make(chan *ec2.Image)
	)

	// Get the Launch Permissions for an AMI.
	worker := func() {
		defer wg.Done()
		for image := range workerCh {
			logger := log.With(logger, "image_id", *image.ImageId)

			rsp, err := svc.DescribeImageAttribute(&ec2.DescribeImageAttributeInput{
				ImageId:   image.ImageId,
				Attribute: aws.String("launchPermission"),
			})
			if err != nil {
				level.Warn(logger).Log("cache_update", "failed", "error", awsError(err))
				continue
			}

			perms := []string{}
			for _, perm := range rsp.LaunchPermissions {
				perms = append(perms, *perm.UserId)
			}

			level.Debug(logger).Log("perm_count", len(perms))

			mu.Lock()
			index = append(index, *image.ImageId)
			images = append(images, Image{
				Image:       image,
				OwnerID:     owner,
				Region:      region,
				launchPerms: perms,
			})
			mu.Unlock()
		}
	}

	// Allow for a percentage of concurrent API requests.
	for i := 0; i < poolSize(maxReq, len(rsp.Images), 0.05); i++ {
		wg.Add(1)
		go worker()
	}

	for _, image := range rsp.Images {
		workerCh <- image
	}

	close(workerCh)
	wg.Wait()

	return images, index
}

// AWS standard regions provided as a map for fast look-ups.
func awsStdRegions() map[string]struct{} {
	regions := map[string]struct{}{}
	for region := range endpoints.AwsPartition().Regions() {
		regions[region] = struct{}{}
	}
	return regions
}

// A helper to return just the error message from an AWS API error.
func awsError(err error) error {
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return errors.New(awsErr.Message())
		}
	}
	return err
}

// A helper function used for getting launch permissions from AMIs.
func poolSize(max, queue int, percent float64) int {
	size := int(float64(queue) * percent)
	if size < 1 {
		return 1
	} else if size > max {
		return max
	}
	return size
}

// The access policy required by ami-query.
const policyDoc = `{
	"Version": "2012-10-17",
	"Statement": [{
		"Effect": "Allow",
		"Action": [
			"ec2:DescribeImageAttribute",
			"ec2:DescribeImages"
		],
		"Resource": "*"
	}]
}`
