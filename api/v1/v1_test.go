// Copyright 2015 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"golang.org/x/net/context"

	"github.com/intuit/ami-query/ami"
	"github.com/intuit/ami-query/amicache"
	"github.com/intuit/ami-query/api"
)

var amiquery *httptest.Server

func TestMain(m *testing.M) {
	// Load test data
	usEast1, err := ioutil.ReadFile("../../testdata/us-east-1-describe-images.xml")
	if err != nil {
		panic(err)
	}
	usWest1, err := ioutil.ReadFile("../../testdata/us-west-1-describe-images.xml")
	if err != nil {
		panic(err)
	}
	usWest2, err := ioutil.ReadFile("../../testdata/us-west-2-describe-images.xml")
	if err != nil {
		panic(err)
	}

	// Mock EC2 service endpoint
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// Set EC2Endpoint to the mocked endpoint and reset it afterwards.
	defer func(previous func(string) string) { amicache.EC2Endpoint = previous }(amicache.EC2Endpoint)
	amicache.EC2Endpoint = func(region string) string {
		return fmt.Sprintf("%s/%s", server.URL, region)
	}

	// Endpoint response handlers
	mux.HandleFunc("/us-east-1/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s", usEast1)
	})
	mux.HandleFunc("/us-west-1/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s", usWest1)
	})
	mux.HandleFunc("/us-west-2/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s", usWest2)
	})

	creds := credentials.NewStaticCredentials("foo", "bar", "baz")

	// Fill cache with test data
	cacheMgr, _ := amicache.NewManager(
		amicache.NewInternalManager(),
		amicache.AWSCreds(creds),
		amicache.Regions("us-east-1", "us-west-1", "us-west-2"),
	)
	<-cacheMgr.Start()

	// Mock amiquery service
	amiquery = httptest.NewServer(&api.ContextAdapter{
		Context: context.WithValue(context.Background(), api.CacheManagerKey, cacheMgr),
		Handler: api.ContextHandlerFunc(Handler),
	})

	// Run tests
	rc := m.Run()

	// Shutdown services
	amiquery.Close()
	server.Close()
	cacheMgr.Stop()

	os.Exit(rc)
}

func TestHandler(t *testing.T) {
	var client = http.DefaultClient
	var tests = []struct {
		uri    string
		status int
		count  int
		body   string
	}{
		{"/amis", http.StatusOK, 12, ""},
		{"/amis?ami=ami-1a2b3c4d", http.StatusOK, 1, ""},
		{"/amis?region=us-west-1", http.StatusOK, 4, ""},
		{"/amis?region=us-west-1&region=us-west-2&region=us-west-2", http.StatusOK, 8, ""},
		{"/amis?status=available", http.StatusOK, 6, ""},
		{"/amis?region=us-east-1&status=available", http.StatusOK, 2, ""},
		{"/amis?tag=osVersion:rhel6.5", http.StatusOK, 6, ""},
		{"/amis?region=us-bogus-1", http.StatusBadRequest, 0, `{"id":"bad_request","message":"unknown or unsupported region: us-bogus-1"}`},
		{"/amis?bogus=1", http.StatusBadRequest, 0, `{"id":"bad_request","message":"unknown query key: bogus"}`},
		{"/amis?tag=name:baseline1:bogus", http.StatusBadRequest, 0, `{"id":"bad_request","message":"invalid query value: name:baseline1:bogus"}`},
	}

	for _, tt := range tests {
		req, err := http.NewRequest("GET", amiquery.URL+tt.uri, nil)
		if err != nil {
			t.Fatal(err)
		}

		var body []byte
		rsp, err := client.Do(req)
		if err == nil {
			body, err = ioutil.ReadAll(rsp.Body)
			rsp.Body.Close()
		}
		if err != nil {
			t.Fatal(err)
		}

		if rsp.StatusCode != tt.status {
			t.Errorf("Query: %s\n\twant: Status %d\n\t got: Status %d", tt.uri, tt.status, rsp.StatusCode)
			continue
		}

		body = bytes.TrimSpace(body)

		if rsp.StatusCode == http.StatusOK {
			var amis []ami.AMI
			err = json.Unmarshal(body, &amis)
			if err != nil {
				t.Fatal(err)
			}
			if tt.count != len(amis) {
				t.Errorf("Query: %s\n\twant: %d AMIs\n\t got: %d AMIs", tt.uri, tt.count, len(amis))
			}
		} else if string(body) != tt.body {
			t.Errorf("Query: %s\n\twant: %s\n\t got: %s", tt.uri, tt.body, string(body))
		}
	}
}
