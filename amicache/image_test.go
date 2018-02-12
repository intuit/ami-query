// Copyright 2017 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package amicache

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func TestTags(t *testing.T) {
	i := NewImage(
		&ec2.Image{
			Tags: []*ec2.Tag{{
				Key:   aws.String("test"),
				Value: aws.String("foo"),
			}},
		},
		"foo",
		"bar",
		[]string{"foo"},
	)

	if want, got := "", i.Tag("foo"); want != got {
		t.Errorf("want: %s, got: %s", want, got)
	}

	if want, got := "foo", i.Tag("test"); want != got {
		t.Errorf("want: %s, got: %s", want, got)
	}

	if want, got := map[string]string{"test": "foo"}, i.Tags(); !reflect.DeepEqual(want, got) {
		t.Errorf("want: %v, got: %v", want, got)
	}
}

func TestSortByState(t *testing.T) {
	var (
		img1 = Image{
			Image: &ec2.Image{
				CreationDate: aws.String("2017-10-29T16:00:00.000Z"),
				Tags: []*ec2.Tag{{
					Key:   aws.String(StateTag),
					Value: aws.String("available"),
				}},
			},
		}
		img2 = Image{
			Image: &ec2.Image{
				CreationDate: aws.String("2017-05-15T16:00:00.000Z"),
				Tags: []*ec2.Tag{{
					Key:   aws.String(StateTag),
					Value: aws.String("deprecated"),
				}},
			},
		}
		img3 = Image{
			Image: &ec2.Image{
				CreationDate: aws.String("2017-10-25T16:00:00.000Z"),
				Tags: []*ec2.Tag{{
					Key:   aws.String(StateTag),
					Value: aws.String("foo"),
				}},
			},
		}
		img4 = Image{
			Image: &ec2.Image{
				CreationDate: aws.String("2017-10-25T16:00:00.000Z"),
				Tags: []*ec2.Tag{{
					Key:   aws.String(StateTag),
					Value: aws.String("available"),
				}},
			},
		}
	)

	images := []Image{img3, img2, img4, img1}
	sortedImages := []Image{img1, img4, img2, img3}

	SortByState(images)

	if !reflect.DeepEqual(sortedImages, images) {
		t.Errorf("\n\twant: %+v\n\t got: %+v", sortedImages, images)
	}
}
