// Copyright 2015 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package amicache

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
)

var cacheMgr *Manager

func TestMain(m *testing.M) {
	// Read in test data
	usEast1, err := ioutil.ReadFile("../testdata/us-east-1-describe-images.xml")
	if err != nil {
		panic(err)
	}
	usWest1, err := ioutil.ReadFile("../testdata/us-west-1-describe-images.xml")
	if err != nil {
		panic(err)
	}

	// Mock EC2 service endpoint
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// Set API endpoints to the mocked endpoints and reset them afterwards.
	defer func(previous func(string) string) { EC2Endpoint = previous }(EC2Endpoint)
	EC2Endpoint = func(region string) string {
		return server.URL + "/" + region
	}

	// Endpoint response handlers
	mux.HandleFunc("/us-west-1/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s", usEast1)
	})
	mux.HandleFunc("/us-west-2/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s", usWest1)
	})

	creds := credentials.NewStaticCredentials("foo", "bar", "baz")

	// Fill cache with test data
	cacheMgr, _ = NewManager(
		NewInternalManager(),
		AWSCreds(creds),
		Regions("us-west-1", "us-west-2"),
	)
	<-cacheMgr.Start()

	// Run tests
	rc := m.Run()

	// Shutdown services
	server.Close()
	cacheMgr.Stop()

	os.Exit(rc)
}

func TestAWSCredsOption(t *testing.T) {
	creds := credentials.NewStaticCredentials("foo", "bar", "baz")
	m, err := NewManager(nil, AWSCreds(creds))
	if err != nil {
		t.Fatalf("want nil, got %v", err)
	}
	want, err := creds.Get()
	if err != nil {
		t.Fatalf("want nil, got %v", err)
	}
	got, err := m.awsCreds.Get()
	if err != nil {
		t.Fatalf("want nil, got %v", err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("\n\twant %#v\n\t got %v", want, got)
	}
}

func TestHTTPClientOption(t *testing.T) {
	client := http.DefaultClient
	m, err := NewManager(nil, HTTPClient(client))
	if err != nil {
		t.Fatalf("want nil, got %v", err)
	}
	if !reflect.DeepEqual(client, m.client) {
		t.Errorf("\n\twant %#v\n\t got %v", client, m.client)
	}
}

func TestOwnerIDsOption(t *testing.T) {
	ids := []string{"111122223333", "111122224444"}
	m, err := NewManager(nil, OwnerIDs(ids...))
	if err != nil {
		t.Fatalf("want nil, got %v", err)
	}
	for i, id := range m.ownerIDs {
		if ids[i] != *id {
			t.Errorf("want %v, got %v", ids[i], *id)
		}
	}
}

func TestRegionsOption(t *testing.T) {
	regions := []string{"us-west-1", "us-west-2"}
	m, err := NewManager(nil, Regions(regions...))
	if err != nil {
		t.Fatalf("want nil, got %v", err)
	}
	if !reflect.DeepEqual(regions, m.regions) {
		t.Errorf("want %v, got %v", regions, m.regions)
	}
	if _, err := NewManager(nil, Regions("us-bogus-1")); err == nil {
		t.Errorf("want error, got nil")
	}
}

func TestTTLOption(t *testing.T) {
	ttl := 5 * time.Minute
	m, err := NewManager(nil, TTL(ttl))
	if err != nil {
		t.Fatalf("want nil, got %v", err)
	}
	if m.ttl != ttl {
		t.Errorf("want %v, got %v", ttl, m.ttl)
	}
	if err := TTL(1 * time.Second).set(m); err != nil {
		t.Errorf("want nil, got %v", err)
	} else if m.ttl != minTTL {
		t.Errorf("want %v, got %v", minTTL, m.ttl)
	}
}

func TestCache(t *testing.T) {
	tests := []struct {
		region string
		count  int
	}{
		{"us-west-1", 4},
		{"us-west-2", 4},
		{"us-foo-1", 0},
	}
	for _, tt := range tests {
		images, _ := cacheMgr.Images(tt.region)
		if tt.count != len(images) {
			t.Errorf("%s image count: want %d, got %d", tt.region, tt.count, len(images))
		}
	}
}

func TestFilter(t *testing.T) {
	tests := []struct {
		count   int
		filters []Filterer
	}{
		{2, []Filterer{
			FilterByID("ami-1a2b3c4d", "ami-1b2b3c4d"),
		}},
		{3, []Filterer{
			FilterByTags(map[string][]string{"status": {"available", "deprecated"}}),
		}},
		{2, []Filterer{
			FilterByID("ami-1a2b3c4d", "ami-1c2b3c4d"),
			FilterByTags(map[string][]string{"status": {"available", "exception"}}),
		}},
	}
	for i, tt := range tests {
		img1, _ := cacheMgr.Images("us-west-1")
		img2 := NewFilter(tt.filters...).Apply(img1)
		if tt.count != len(img2) {
			t.Errorf("Filter #%d: want %d, got %d", i+1, tt.count, len(img2))
		}
	}
}
