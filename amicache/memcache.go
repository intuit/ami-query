// Copyright 2015 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package amicache

import (
	"bytes"
	"encoding/gob"
	"sync"

	"github.com/bradfitz/gomemcache/memcache"

	"github.com/intuit/ami-query/ami"
)

// MemcachedManager is a memcached CacheManager.
type MemcachedManager struct {
	name   string
	client *memcache.Client
	sync.RWMutex
}

// NewMemcachedManager creates a new memcached CacheManager.
func NewMemcachedManager(servers ...string) *MemcachedManager {
	return &MemcachedManager{
		name:   "memcached",
		client: memcache.New(servers...),
	}
}

// Name returns "memcache".
func (c *MemcachedManager) Name() string {
	return c.name
}

// Start does nothing.
func (c *MemcachedManager) Start() error {
	return nil
}

// Stop does nothing.
func (c *MemcachedManager) Stop() error {
	return nil
}

// Set adds or updates an AMI in the cache.
func (c *MemcachedManager) Set(ami ami.AMI) error {
	c.Lock()
	defer c.Unlock()

	value, err := c.marshal(ami)
	if err != nil {
		return err
	}

	item := memcache.Item{Key: *ami.Image.ImageId, Value: value}
	if err = c.client.Set(&item); err != nil {
		return err
	}

	return nil
}

// Get returns the cached AMI.
func (c *MemcachedManager) Get(id string) (ami.AMI, error) {
	c.RLock()
	defer c.RUnlock()

	item, err := c.client.Get(id)
	if err != nil && err == memcache.ErrCacheMiss {
		return ami.AMI{}, nil
	}
	if err != nil {
		return ami.AMI{}, err
	}

	var ami ami.AMI

	if err = c.unmarshal(item.Value, &ami); err != nil {
		return ami, err
	}

	return ami, nil
}

// Delete removes the AMI from the cache.
func (c *MemcachedManager) Delete(id string) error {
	c.RLock()
	c.client.Delete(id)
	c.RUnlock()
	return nil
}

// marshal will gob encode ami.AMI into []byte for storage in memcached.
func (c *MemcachedManager) marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// unmarshal will gob decode a memcache value into []ec2.Image.
func (c *MemcachedManager) unmarshal(data []byte, v interface{}) error {
	return gob.NewDecoder(bytes.NewBuffer(data)).Decode(v)
}
