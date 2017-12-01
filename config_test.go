// Copyright 2017 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package main

import (
	"os"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	env := map[string]string{
		"AMIQUERY_ROLE_NAME":                     "foo",
		"AMIQUERY_OWNER_IDS":                     "1111,2222",
		"AMIQUERY_LISTEN_ADDRESS":                ":8081",
		"AMIQUERY_REGIONS":                       "us-west-1,us-west-2",
		"AMIQUERY_CACHE_MAX_CONCURRENT_REQUESTS": "1",
		"AMIQUERY_CACHE_MAX_REQUEST_RETRIES":     "1",
		"AMIQUERY_APP_LOGFILE":                   "/tmp/app.log",
		"AMIQUERY_HTTP_LOGFILE":                  "/tmp/http.log",
		"SSL_CERTIFICATE_FILE":                   "/tmp/test.crt",
		"SSL_KEY_FILE":                           "/tmp/test.key",
	}

	for k, v := range env {
		if err := os.Setenv(k, v); err != nil {
			t.Fatal(err)
		}
	}

	c, err := NewConfig()
	if err != nil {
		t.Fatal(err)
	}

	if want, got := "foo", c.RoleName; want != got {
		t.Errorf("AMIQUERY_ROLE_NAME - want: %s, got: %s", want, got)
	}

	if want, got := 2, len(c.OwnerIDs); want != got {
		t.Errorf("AMIQUERY_OWNER_IDS - want: %d owner(s), got: %d owner(s)", want, got)
	}

	if want, got := ":8081", c.ListenAddr; want != got {
		t.Errorf("AMIQUERY_LISTEN_ADDRESS - want: %s, got: %s", want, got)
	}

	if want, got := 2, len(c.Regions); want != got {
		t.Errorf("AMIQUERY_REGIONS - want: %d owner(s), got: %d owner(s)", want, got)
	}

	if want, got := 15*time.Minute, c.CacheTTL; want != got {
		t.Errorf("AMIQUERY_CACHE_TTL - want: %s, got: %s", want, got)
	}

	if want, got := 1, c.CacheMaxConcurrentRequests; want != got {
		t.Errorf("AMIQUERY_CACHE_MAX_CONCURRENT_REQUESTS - want: %d, got: %d", want, got)
	}

	if want, got := 1, c.CacheMaxRequestRetries; want != got {
		t.Errorf("AMIQUERY_CACHE_MAX_REQUEST_RETRIES - want: %d, got: %d", want, got)
	}

	if want, got := "/tmp/app.log", c.AppLog; want != got {
		t.Errorf("AMIQUERY_APP_LOGFILE - want: %s, got: %s", want, got)
	}

	if want, got := "/tmp/http.log", c.HTTPLog; want != got {
		t.Errorf("AMIQUERY_HTTP_LOGFILE - want: %s, got: %s", want, got)
	}

	if want, got := "/tmp/test.crt", c.SSLCert; want != got {
		t.Errorf("SSL_CERTIFICATE_FILE - want: %s, got: %s", want, got)
	}

	if want, got := "/tmp/test.key", c.SSLKey; want != got {
		t.Errorf("SSL_KEY_FILE - want: %s, got: %s", want, got)
	}
}
