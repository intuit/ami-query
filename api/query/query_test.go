// Copyright 2015 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package query

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/intuit/ami-query/amicache"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type mockCache struct {
	filterErr error
}

func (mockCache) Regions() []string   { return []string{"us-west-2"} }
func (m *mockCache) StateTag() string { return amicache.DefaultStateTag }
func (m *mockCache) FilterImages(string, *amicache.Filter) ([]amicache.Image, error) {
	images := []amicache.Image{
		{
			OwnerID: "123456789012",
			Region:  "us-west-2",
			Image: &ec2.Image{
				Name:               aws.String("test-ami-1"),
				Description:        aws.String("Test AMI 1"),
				VirtualizationType: aws.String("hvm"),
				CreationDate:       aws.String("2017-11-29T16:00:00.000Z"),
				ImageId:            aws.String("ami-1a2b3c4d"),
				Tags: []*ec2.Tag{{
					Key:   aws.String(amicache.DefaultStateTag),
					Value: aws.String("available"),
				}},
			},
		},
	}
	return images, m.filterErr
}

func TestHandler(t *testing.T) {
	var tests = []struct {
		name       string
		query      string
		statusCode int
		filterErr  error
	}{
		{"amis", "/amis", http.StatusOK, nil},
		{"callback", "/amis?callback=foo", http.StatusOK, nil},
		{"pretty", "/amis?pretty", http.StatusOK, nil},
		{"bad_key", "/amis?foo=bar", http.StatusBadRequest, nil},
		{"bad_tag", "/amis?tag=foobar", http.StatusBadRequest, nil},
		{"bad_region", "/amis?region=us-foo-1", http.StatusBadRequest, errors.New("foo")},
	}

	mc := &mockCache{}
	ts := httptest.NewServer(&API{
		cache:   mc,
		regions: []string{"us-west-2"},
	})
	defer ts.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc.filterErr = tt.filterErr
			defer func() { mc.filterErr = nil }()

			rsp, err := http.Get(ts.URL + tt.query)
			if err != nil {
				t.Errorf("want: <nil>, got: %v", err)
				return
			}

			if rsp.StatusCode != tt.statusCode {
				t.Errorf("want: status %d, got: status %d", tt.statusCode, rsp.StatusCode)
			}
		})
	}
}
