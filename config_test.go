// Copyright 2017 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package main

import (
	"errors"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name string
		vars map[string]string
		want *Config
		err  error
	}{
		{

			name: "default_settings",
			vars: map[string]string{
				"AMIQUERY_ROLE_NAME": "foo",
				"AMIQUERY_OWNER_IDS": "123456789012,123456789013",
			},
			want: &Config{
				ListenAddr: ":8080",
				RoleName:   "foo",
				OwnerIDs:   []string{"123456789012", "123456789013"},
				CacheTTL:   15 * time.Minute,
			},
			err: nil,
		},
		{

			name: "all_settings",
			vars: map[string]string{
				"AMIQUERY_LISTEN_ADDRESS":                ":8081",
				"AMIQUERY_ROLE_NAME":                     "foo",
				"AMIQUERY_TAG_FILTER":                    "foo",
				"AMIQUERY_OWNER_IDS":                     "123456789012,123456789013",
				"AMIQUERY_REGIONS":                       "us-west-1,us-west-2",
				"AMIQUERY_CACHE_TTL":                     "20m",
				"AMIQUERY_CACHE_MAX_CONCURRENT_REQUESTS": "1",
				"AMIQUERY_CACHE_MAX_REQUEST_RETRIES":     "1",
				"AMIQUERY_APP_LOGFILE":                   "/tmp/app.log",
				"AMIQUERY_HTTP_LOGFILE":                  "/tmp/http.log",
				"AMIQUERY_CORS_ALLOWED_ORIGINS":          "foo.com, bar.com , baz.com",
				"SSL_CERTIFICATE_FILE":                   "/tmp/test.crt",
				"SSL_KEY_FILE":                           "/tmp/test.key",
			},
			want: &Config{
				ListenAddr:                 ":8081",
				RoleName:                   "foo",
				TagFilter:                  "foo",
				Regions:                    []string{"us-west-1", "us-west-2"},
				OwnerIDs:                   []string{"123456789012", "123456789013"},
				CacheTTL:                   20 * time.Minute,
				CacheMaxConcurrentRequests: 1,
				CacheMaxRequestRetries:     1,
				AppLog:                     "/tmp/app.log",
				HTTPLog:                    "/tmp/http.log",
				CorsAllowedOrigins:         []string{"foo.com", "bar.com", "baz.com"},
				SSLCert:                    "/tmp/test.crt",
				SSLKey:                     "/tmp/test.key",
			},
			err: nil,
		},
		{
			name: "missing_role",
			vars: map[string]string{},
			want: nil,
			err:  errors.New("AMIQUERY_ROLE_NAME is undefined"),
		},
		{
			name: "missing_owner_ids",
			vars: map[string]string{
				"AMIQUERY_ROLE_NAME": "foo",
			},
			want: nil,
			err:  errors.New("AMIQUERY_OWNER_IDS is undefined"),
		},
		{
			name: "bad_cache_ttl_value",
			vars: map[string]string{
				"AMIQUERY_ROLE_NAME": "foo",
				"AMIQUERY_OWNER_IDS": "123456789012,123456789013",
				"AMIQUERY_CACHE_TTL": "foo",
			},
			want: nil,
			err:  errors.New("failed to read AMIQUERY_CACHE_TTL: time: invalid duration foo"),
		},
		{
			name: "bad_cache_max_requests_value",
			vars: map[string]string{
				"AMIQUERY_ROLE_NAME":                     "foo",
				"AMIQUERY_OWNER_IDS":                     "123456789012,123456789013",
				"AMIQUERY_CACHE_MAX_CONCURRENT_REQUESTS": "1foo",
			},
			want: nil,
			err:  errors.New(`failed to read AMIQUERY_CACHE_MAX_CONCURRENT_REQUESTS: strconv.Atoi: parsing "1foo": invalid syntax`),
		},
		{
			name: "bad_cache_max_retries_value",
			vars: map[string]string{
				"AMIQUERY_ROLE_NAME":                 "foo",
				"AMIQUERY_OWNER_IDS":                 "123456789012,123456789013",
				"AMIQUERY_CACHE_MAX_REQUEST_RETRIES": "1foo",
			},
			want: nil,
			err:  errors.New(`failed to read AMIQUERY_CACHE_MAX_REQUEST_RETRIES: strconv.Atoi: parsing "1foo": invalid syntax`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := clearVars(); err != nil {
				t.Fatal(err)
			}
			for k, v := range tt.vars {
				if err := os.Setenv(k, v); err != nil {
					t.Fatal(err)
				}
			}
			got, err := NewConfig()
			if err != nil && err.Error() != tt.err.Error() {
				t.Errorf("want: %v, got: %v", tt.err, err)
				return
			}
			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("\n\twant: %+v\n\t got: %+v", tt.want, got)
			}
		})
	}
}

func clearVars() error {
	vars := []string{
		"AMIQUERY_LISTEN_ADDRESS",
		"AMIQUERY_ROLE_NAME",
		"AMIQUERY_TAG_FILTER",
		"AMIQUERY_OWNER_IDS",
		"AMIQUERY_REGIONS",
		"AMIQUERY_CACHE_TTL",
		"AMIQUERY_CACHE_MAX_CONCURRENT_REQUESTS",
		"AMIQUERY_CACHE_MAX_REQUEST_RETRIES",
		"AMIQUERY_APP_LOGFILE",
		"AMIQUERY_HTTP_LOGFILE",
		"AMIQUERY_CORS_ALLOWED_ORIGINS",
		"SSL_CERTIFICATE_FILE",
		"SSL_KEY_FILE",
	}
	for _, v := range vars {
		if err := os.Unsetenv(v); err != nil {
			return err
		}
	}
	return nil
}
