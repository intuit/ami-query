// Copyright 2015 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the configuration for ami-query.
type Config struct {
	ListenAddr                 string
	RoleName                   string
	TagFilter                  string
	OwnerIDs                   []string
	Regions                    []string
	CacheTTL                   time.Duration
	CacheMaxConcurrentRequests int
	CacheMaxRequestRetries     int
	AppLog                     string
	HTTPLog                    string
	SSLCert                    string
	SSLKey                     string
}

// NewConfig returns a Config with settings pulled from the environment. See
// the README.md for more information.
func NewConfig() (*Config, error) {
	var err error
	var cfg = Config{
		ListenAddr: ":8080",
		CacheTTL:   15 * time.Minute,
		RoleName:   os.Getenv("AMIQUERY_ROLE_NAME"),
		TagFilter:  os.Getenv("AMIQUERY_TAG_FILTER"),
		AppLog:     os.Getenv("AMIQUERY_APP_LOGFILE"),
		HTTPLog:    os.Getenv("AMIQUERY_HTTP_LOGFILE"),
		SSLCert:    os.Getenv("SSL_CERTIFICATE_FILE"),
		SSLKey:     os.Getenv("SSL_KEY_FILE"),
	}

	// The address to listen on.
	if laddr := os.Getenv("AMIQUERY_LISTEN_ADDRESS"); laddr != "" {
		cfg.ListenAddr = laddr
	}

	// The role assumed into in targeted accounts.
	if cfg.RoleName == "" {
		return nil, fmt.Errorf("AMIQUERY_ROLE_NAME is undefined")
	}

	// Owner IDs used to filter AMI results.
	if ownerIDs := os.Getenv("AMIQUERY_OWNER_IDS"); ownerIDs != "" {
		cfg.OwnerIDs = strings.Split(ownerIDs, ",")
	} else {
		return nil, fmt.Errorf("AMIQUERY_OWNER_IDS is undefined")
	}

	// AWS regions to scan for AMIs.
	if regions := os.Getenv("AMIQUERY_REGIONS"); regions != "" {
		cfg.Regions = strings.Split(regions, ",")
	}

	// Duration between cache updates.
	if ttl := os.Getenv("AMIQUERY_CACHE_TTL"); ttl != "" {
		if cfg.CacheTTL, err = time.ParseDuration(ttl); err != nil {
			return nil, fmt.Errorf("failed to read AMIQUERY_CACHE_TTL: %v", err)
		}
	}

	// Maximum number of goroutines used for updating the cache.
	if maxRequests := os.Getenv("AMIQUERY_CACHE_MAX_CONCURRENT_REQUESTS"); maxRequests != "" {
		if cfg.CacheMaxConcurrentRequests, err = strconv.Atoi(maxRequests); err != nil {
			return nil, fmt.Errorf("failed to read AMIQUERY_CACHE_MAX_CONCURRENT_REQUESTS: %v", err)
		}
	}

	// Maximum number of API request retries before giving up.
	if maxRetries := os.Getenv("AMIQUERY_CACHE_MAX_REQUEST_RETRIES"); maxRetries != "" {
		if cfg.CacheMaxRequestRetries, err = strconv.Atoi(maxRetries); err != nil {
			return nil, fmt.Errorf("failed to read AMIQUERY_CACHE_MAX_REQUEST_RETRIES: %v", err)
		}
	}

	return &cfg, nil
}
