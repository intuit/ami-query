// Copyright 2015 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package ami

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func TestStatusSort(t *testing.T) {
	sorted := []AMI{
		NewAMI("us-test-1", image1),
		NewAMI("us-test-1", image2),
		NewAMI("us-test-1", image3),
		NewAMI("us-test-1", image8),
		NewAMI("us-test-1", image4),
		NewAMI("us-test-1", image5),
		NewAMI("us-test-1", image6),
		NewAMI("us-test-1", image7),
	}
	unsorted := []AMI{
		NewAMI("us-test-1", image1),
		NewAMI("us-test-1", image3),
		NewAMI("us-test-1", image8),
		NewAMI("us-test-1", image5),
		NewAMI("us-test-1", image7),
		NewAMI("us-test-1", image6),
		NewAMI("us-test-1", image4),
		NewAMI("us-test-1", image2),
	}

	Sort(unsorted).By(Status)

	if !reflect.DeepEqual(sorted, unsorted) {
		t.Errorf("want: %+v\n\t got: %+v", sorted, unsorted)
	}
}

func TestTags(t *testing.T) {
	image := NewAMI("us-test-1", image1)
	tags := image.Tags()

	if tags["status"] != image.Tag("status") {
		t.Errorf("want: %v, got %v", tags["status"], image.Tag("status"))
	}

	if image.Tag("foo") != "" {
		t.Errorf("want: \"\", got %v", image.Tag("foo"))
	}
}

var (
	image1 = &ec2.Image{
		ImageId:      aws.String("ami-1a2b3c4d"),
		Name:         aws.String("ami1"),
		Description:  aws.String("AMI1"),
		OwnerId:      aws.String("111122223333"),
		CreationDate: aws.String("2013-10-25T16:03:13.000Z"),
		Tags:         []*ec2.Tag{{Key: aws.String("status"), Value: aws.String("available")}},
	}

	image2 = &ec2.Image{
		ImageId:      aws.String("ami-2a2b3c4d"),
		Name:         aws.String("ami2"),
		Description:  aws.String("AMI2"),
		OwnerId:      aws.String("111122223333"),
		CreationDate: aws.String("2013-10-08T14:03:13.000Z"),
		Tags:         []*ec2.Tag{{Key: aws.String("status"), Value: aws.String("available")}},
	}

	image3 = &ec2.Image{
		ImageId:      aws.String("ami-3a2b3c4d"),
		Name:         aws.String("ami3"),
		Description:  aws.String("AMI3"),
		OwnerId:      aws.String("111122223333"),
		CreationDate: aws.String("2013-09-03T01:08:00.000Z"),
		Tags:         []*ec2.Tag{{Key: aws.String("status"), Value: aws.String("deprecated")}},
	}

	image4 = &ec2.Image{
		ImageId:      aws.String("ami-4a2b3c4d"),
		Name:         aws.String("ami4"),
		Description:  aws.String("AMI4"),
		OwnerId:      aws.String("111122223333"),
		CreationDate: aws.String("2013-04-24T01:08:00.000Z"),
		Tags:         []*ec2.Tag{{Key: aws.String("status"), Value: aws.String("exception")}},
	}

	image5 = &ec2.Image{
		ImageId:      aws.String("ami-5a2b3c4d"),
		Name:         aws.String("ami5"),
		Description:  aws.String("AMI5"),
		OwnerId:      aws.String("111122223333"),
		CreationDate: aws.String("2013-05-17T01:08:00.000Z"),
		Tags:         []*ec2.Tag{{Key: aws.String("status"), Value: aws.String("unavailable")}},
	}

	image6 = &ec2.Image{
		ImageId:      aws.String("ami-6a2b3c4d"),
		Name:         aws.String("ami6"),
		Description:  aws.String("AMI6"),
		OwnerId:      aws.String("111122223333"),
		CreationDate: aws.String("2013-10-25T16:03:13.000Z"),
		Tags:         []*ec2.Tag{{Key: aws.String("status"), Value: aws.String("development")}},
	}

	image7 = &ec2.Image{
		ImageId:      aws.String("ami-7a2b3c4d"),
		Name:         aws.String("ami7"),
		Description:  aws.String("AMI7"),
		OwnerId:      aws.String("111122223333"),
		CreationDate: aws.String("2012-12-10T00:08:00.000Z"),
		Tags:         []*ec2.Tag{{Key: aws.String("status"), Value: aws.String("deregistered")}},
	}

	image8 = &ec2.Image{
		ImageId:      aws.String("ami-8a2b3c4d"),
		Name:         aws.String("ami8"),
		Description:  aws.String("AMI8"),
		OwnerId:      aws.String("111122223333"),
		CreationDate: aws.String("2013-01-13T17:28:00.000Z"),
		Tags:         []*ec2.Tag{{Key: aws.String("status"), Value: aws.String("deprecated")}},
	}
)
