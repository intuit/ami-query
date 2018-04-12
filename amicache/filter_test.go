// Copyright 2017 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package amicache

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func testImages() []Image {
	return []Image{
		{
			OwnerID: "123456789012",
			Image: &ec2.Image{
				CreationDate: aws.String("2017-10-29T16:00:00.000Z"),
				ImageId:      aws.String("ami-1a2b3c4d"),
				Tags: []*ec2.Tag{{
					Key:   aws.String(DefaultStateTag),
					Value: aws.String("available"),
				}},
			},
			launchPerms: []string{"123456789012", "123456789013"},
		},
		{
			OwnerID: "123456789012",
			Image: &ec2.Image{
				CreationDate: aws.String("2017-05-15T16:00:00.000Z"),
				ImageId:      aws.String("ami-2a2b3c4d"),
				Tags: []*ec2.Tag{{
					Key:   aws.String(DefaultStateTag),
					Value: aws.String("deprecated"),
				}},
			},
			launchPerms: []string{"123456789012"},
		},
		{
			OwnerID: "123456789012",
			Image: &ec2.Image{
				CreationDate: aws.String("2017-10-25T16:00:00.000Z"),
				ImageId:      aws.String("ami-3a2b3c4d"),
				Tags: []*ec2.Tag{{
					Key:   aws.String(DefaultStateTag),
					Value: aws.String("available"),
				}},
			},
			launchPerms: []string{"123456789012"},
		},
		{
			OwnerID: "123456789013",
			Image: &ec2.Image{
				CreationDate: aws.String("2017-10-25T16:00:00.000Z"),
				ImageId:      aws.String("ami-4a2b3c4d"),
				Tags: []*ec2.Tag{{
					Key:   aws.String(DefaultStateTag),
					Value: aws.String("exception"),
				}},
			},
			launchPerms: []string{"123456789013"},
		},
	}
}

func TestFilterByImageID(t *testing.T) {
	tests := []struct {
		name string
		ids  []string
		want int
	}{
		{"1_id", []string{"ami-1a2b3c4d"}, 1},
		{"2_ids", []string{"ami-1a2b3c4d", "ami-2a2b3c4d"}, 2},
		{"no_ids", []string{}, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images := FilterByImageID(tt.ids...).Filter(testImages())
			if got := len(images); tt.want != got {
				t.Errorf("want: %d image(s), got %d image(s)", tt.want, got)
			}
		})
	}
}

func TestFilterByTags(t *testing.T) {
	tests := []struct {
		name string
		tags map[string][]string
		want int
	}{
		{
			"state_available",
			map[string][]string{DefaultStateTag: []string{"available"}},
			2,
		},
		{
			"state_deprecated",
			map[string][]string{DefaultStateTag: []string{"deprecated"}},
			1,
		},
		{
			"state_deprecated_exception",
			map[string][]string{DefaultStateTag: []string{"deprecated", "exception"}},
			2,
		},
		{
			"no_tags",
			map[string][]string{},
			4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images := FilterByTags(tt.tags).Filter(testImages())
			if got := len(images); tt.want != got {
				t.Errorf("want: %d image(s), got %d image(s)", tt.want, got)
			}
		})
	}
}

func TestFilterByOwnerID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want int
	}{
		{"owner_3_img", "123456789012", 3},
		{"owner_1_img", "123456789013", 1},
		{"no_owner", "", 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images := FilterByOwnerID(tt.id).Filter(testImages())
			if got := len(images); tt.want != got {
				t.Errorf("want: %d image(s), got %d image(s)", tt.want, got)
			}
		})
	}
}

func TestFilterByLaunchPermission(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want int
	}{
		{"perm_3_img", "123456789012", 3},
		{"perm_2_img", "123456789013", 2},
		{"no_perm", "", 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images := FilterByLaunchPermission(tt.id).Filter(testImages())
			if got := len(images); tt.want != got {
				t.Errorf("want: %d image(s), got %d image(s)", tt.want, got)
			}
		})
	}
}
