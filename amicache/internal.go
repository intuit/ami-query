// Copyright 2015 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package amicache

import (
	"sync"

	"github.com/intuit/ami-query/ami"
)

// InternalManager is a CacheManager that stores AMIs in the process' memory.
type InternalManager struct {
	name  string
	items map[string]ami.AMI
	sync.RWMutex
}

// NewInternalManager creates a new in-process CacheManager.
func NewInternalManager() *InternalManager {
	return &InternalManager{
		name:  "internalcache",
		items: make(map[string]ami.AMI),
	}
}

// Name returns the name of the cache manager.
func (c *InternalManager) Name() string {
	return c.name
}

// Start does nothing.
func (c *InternalManager) Start() error {
	return nil
}

// Stop zeros out its internal memory.
func (c *InternalManager) Stop() error {
	c.Lock()
	c.items = make(map[string]ami.AMI)
	c.Unlock()
	return nil
}

// Set adds or updates an AMI in the cache.
func (c *InternalManager) Set(ami ami.AMI) error {
	c.Lock()
	c.items[*ami.Image.ImageId] = ami
	c.Unlock()
	return nil
}

// Get returns the cached AMI.
func (c *InternalManager) Get(id string) (ami.AMI, error) {
	c.RLock()
	defer c.RUnlock()
	if image, ok := c.items[id]; ok {
		return image, nil
	}
	return ami.AMI{}, nil
}

// Delete removes the AMI from the cache.
func (c *InternalManager) Delete(id string) error {
	c.RLock()
	delete(c.items, id)
	c.RUnlock()
	return nil
}
