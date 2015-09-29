// Copyright 2015 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/intuit/ami-query/amicache"
)

// Config is a config
type Config struct {
	ListenAddr string
	Regions    []string
	OwnerIDs   []string
	Manager    amicache.CacheManager
	CacheTTL   time.Duration
	RoleARN    string
}

// NewConfig returns a Config with settings pulled from the environment. See
// the README.md for more information.
func NewConfig() (*Config, error) {
	var err error
	var cfg = &Config{
		ListenAddr: ":8080",
		CacheTTL:   15 * time.Minute,
	}

	// Set the listen address
	if laddr := os.Getenv("AMIQUERY_LISTEN_ADDRESS"); laddr != "" {
		cfg.ListenAddr = laddr
	}

	// AWS regions to scan for AMIs
	if regions := os.Getenv("AMIQUERY_REGIONS"); regions != "" {
		cfg.Regions = strings.Split(regions, ",")
	}

	// Duration between cache updates
	if ttl := os.Getenv("AMIQUERY_CACHE_TTL"); ttl != "" {
		if cfg.CacheTTL, err = time.ParseDuration(ttl); err != nil {
			return nil, err
		}
	}

	// Owner IDs used to filter AMI results
	if ownerIDs := os.Getenv("AMIQUERY_OWNER_IDS"); ownerIDs != "" {
		cfg.OwnerIDs = strings.Split(ownerIDs, ",")
	} else {
		return nil, fmt.Errorf("AMIQUERY_OWNER_IDS is undefined")
	}

	// ARN used for AssumeRole
	if arn := os.Getenv("AMIQUERY_ROLE_ARN"); arn != "" {
		cfg.RoleARN = arn
	}

	// The type of underlying cache to use
	cacheType := os.Getenv("AMIQUERY_CACHE_MANAGER")
	switch cacheType {
	case "memcached":
		servers := strings.Split(os.Getenv("AMIQUERY_MEMCACHED_SERVERS"), ",")
		if len(servers) == 0 {
			return nil, fmt.Errorf("AMIQUERY_MEMCACHED_SERVERS is undefined")
		}
		cfg.Manager = amicache.NewMemcachedManager(servers...)
	default:
		cfg.Manager = amicache.NewInternalManager()
	}

	return cfg, nil
}
