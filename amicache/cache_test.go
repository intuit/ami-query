// Copyright 2015 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package amicache

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/go-kit/kit/log"
)

// mockSTSClient mock.
type mockSTSClient struct {
	stsiface.STSAPI

	// Callbacks for API endpoints. Add more as needed.
	// The default implementations return empty values and nil errors.
	assumeRole func(*sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error)
}

// AssumeRole mock.
func (m *mockSTSClient) AssumeRole(input *sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error) {
	if m.assumeRole == nil {
		return &sts.AssumeRoleOutput{}, nil
	}
	return m.assumeRole(input)
}

// mockEC2Client mock.
type mockEC2Client struct {
	ec2iface.EC2API

	// Callbacks for API endpoints. Add more as needed.
	// The default implementations return empty values and nil errors.
	describeImages         func(*ec2.DescribeImagesInput) (*ec2.DescribeImagesOutput, error)
	describeImageAttribute func(*ec2.DescribeImageAttributeInput) (*ec2.DescribeImageAttributeOutput, error)
}

// DescribeImages mock.
func (m *mockEC2Client) DescribeImages(input *ec2.DescribeImagesInput) (*ec2.DescribeImagesOutput, error) {
	if m.describeImages == nil {
		return &ec2.DescribeImagesOutput{}, nil
	}
	return m.describeImages(input)
}

// DescribeImageAttribute mock.
func (m *mockEC2Client) DescribeImageAttribute(input *ec2.DescribeImageAttributeInput) (*ec2.DescribeImageAttributeOutput, error) {
	if m.describeImageAttribute == nil {
		return &ec2.DescribeImageAttributeOutput{}, nil
	}
	return m.describeImageAttribute(input)
}

// Creates a new cache with a single AMI.
func newMockCache(opts ...Option) *Cache {
	svc := &mockSTSClient{
		assumeRole: func(*sts.AssumeRoleInput) (*sts.AssumeRoleOutput, error) {
			return &sts.AssumeRoleOutput{
				Credentials: &sts.Credentials{
					AccessKeyId:     aws.String("foo"),
					SecretAccessKey: aws.String("bar"),
					SessionToken:    aws.String("baz"),
				},
			}, nil
		},
	}

	c := New(svc, "foo", []string{"111122223333"}, opts...)

	c.ec2Svc = func(*session.Session, string, int) ec2iface.EC2API {
		return &mockEC2Client{
			describeImages: func(*ec2.DescribeImagesInput) (*ec2.DescribeImagesOutput, error) {
				return &ec2.DescribeImagesOutput{
					Images: []*ec2.Image{
						{
							Name:               aws.String("test-ami-1"),
							Description:        aws.String("Test AMI 1"),
							VirtualizationType: aws.String("hvm"),
							CreationDate:       aws.String("2017-11-29T16:00:00.000Z"),
							ImageId:            aws.String("ami-1a2b3c4d"),
							Tags: []*ec2.Tag{{
								Key:   aws.String(c.stateTag),
								Value: aws.String("available"),
							}},
						},
					},
				}, nil
			},
			describeImageAttribute: func(*ec2.DescribeImageAttributeInput) (*ec2.DescribeImageAttributeOutput, error) {
				return &ec2.DescribeImageAttributeOutput{
					LaunchPermissions: []*ec2.LaunchPermission{
						{UserId: aws.String("111111111111")},
						{UserId: aws.String("111111111112")},
					},
				}, nil
			},
		}
	}

	return c
}

func TestCacheOptions(t *testing.T) {
	c := New(
		nil,
		"foo",
		[]string{"foo"},
		TagFilter("foo"),
		StateTag("foo"),
		Regions("us-west-1"),
		TTL(15*time.Minute),
		MaxConcurrentRequests(1),
		MaxRequestRetries(1),
		HTTPClient(http.DefaultClient),
		Logger(log.NewNopLogger()),
	)

	if want, got := map[string]struct{}{"us-west-1": struct{}{}}, c.regions; !reflect.DeepEqual(want, got) {
		t.Errorf("Bad Regions Map - want: %v, got: %v", want, got)
	}

	if want, got := "foo", c.tagFilter; want != got {
		t.Errorf("Bad TagFilter - want: %s, got: %s", want, got)
	}

	if want, got := "foo", c.stateTag; want != got {
		t.Errorf("Bad StateTag - want: %s, got: %s", want, got)
	}

	if want, got := []string{"us-west-1"}, c.Regions(); !reflect.DeepEqual(want, got) {
		t.Errorf("Bad Regions Slice - want: %v, got: %v", want, got)
	}

	if want, got := 15*time.Minute, c.ttl; want != got {
		t.Errorf("Bad TTL - want: %s, got: %s", want, got)
	}

	if want, got := 1, c.maxRequests; want != got {
		t.Errorf("Bad MaxConcurrentRequests - want: %d, got: %d", want, got)
	}

	if want, got := 1, c.maxRetries; want != got {
		t.Errorf("Bad MaxRequestRetries - want: %d, got: %d", want, got)
	}

	if want, got := http.DefaultClient, c.httpClient; want != got {
		t.Errorf("Bad HTTPClient - want: %v, got: %v", want, got)
	}

	if want, got := log.NewNopLogger(), c.logger; want != got {
		t.Errorf("Bad Logger - want: %T, got: %T", want, got)
	}
}

func TestMinTTL(t *testing.T) {
	c := New(nil, "foo", []string{"foo"}, TTL(time.Second))
	if want, got := minCacheTTL, c.ttl; want != got {
		t.Errorf("want: %s, got: %s", want, got)
	}
}

func TestStateTagDefault(t *testing.T) {
	c := New(nil, "foo", []string{})
	if want, got := DefaultStateTag, c.StateTag(); want != got {
		t.Errorf("want: %q, got: %q", want, got)
	}
}

func TestCollectLaunchPermissions(t *testing.T) {
	tests := []struct {
		name    string
		collect bool
		want    bool
	}{
		{"collect", true, true},
		{"do_not_collect", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(nil, "foo", []string{}, CollectLaunchPermissions(tt.collect))
			if got := c.CollectLaunchPermissions(); tt.want != got {
				t.Errorf("want: %T, got: %T", tt.want, got)
			}
		})
	}
}

func TestCacheIsRunning(t *testing.T) {
	c := newMockCache()
	warmed := make(chan struct{})

	go func() { c.Run(context.Background(), warmed) }()

	<-warmed

	if want, got := true, c.isRunning(); want != got {
		t.Errorf("want: %t, got: %t", want, got)
	}

	if want, got := errCacheRunning, c.Run(context.Background(), nil); want != got {
		t.Errorf("want: %v, got: %v", want, got)
	}

	c.Stop()

	if want, got := false, c.isRunning(); want != got {
		t.Errorf("want: %t, got: %t", want, got)
	}
}

func TestCacheStopped(t *testing.T) {
	var (
		c      = newMockCache()
		errCh  = make(chan error)
		warmed = make(chan struct{})
	)

	go func() { errCh <- c.Run(context.Background(), warmed) }()

	<-warmed
	c.Stop()

	if want, got := errCacheStopped, <-errCh; want != got {
		t.Errorf("want: %s, got: %s", want, got)
	}
}

func TestCacheContextCanceled(t *testing.T) {
	var (
		c           = newMockCache()
		errCh       = make(chan error)
		warmed      = make(chan struct{})
		ctx, cancel = context.WithCancel(context.Background())
	)

	go func() { errCh <- c.Run(ctx, warmed) }()

	<-warmed
	cancel()

	if want, got := context.Canceled, <-errCh; want != got {
		t.Errorf("want: %s, got: %s", want, got)
	}
}

func TestImages(t *testing.T) {
	tests := []struct {
		name          string
		collectLaunch bool
		region        string
		wantImgLen    int
		wantPermsLen  int
		wantErr       error
	}{
		{"no_errors_with_perms", true, "us-west-1", 1, 2, nil},
		{"no_errors_without_perms", false, "us-west-1", 1, 0, nil},
		{"invalid_region", false, "us-foo-1", 0, 2, errors.New("unknown or unsupported region: us-foo-1")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newMockCache(
				Regions("us-west-1"),
				TagFilter("foo"),
				CollectLaunchPermissions(tt.collectLaunch),
			)
			warmed := make(chan struct{})

			go func() { c.Run(context.Background(), warmed) }()

			defer c.Stop()
			<-warmed

			images, err := c.Images(tt.region)
			if !reflect.DeepEqual(tt.wantErr, err) {
				t.Errorf("want: %v err, got: %v err", tt.wantErr, err)
			}

			if tt.wantErr == nil {
				if got := len(images); tt.wantImgLen != len(images) {
					t.Errorf("want: %d image(s), got: %d image(s)", tt.wantImgLen, got)
				}

				if got := len(images[0].launchPerms); tt.wantPermsLen != got {
					t.Errorf("want: %d perms, got: %d perms", tt.wantPermsLen, got)
				}
			}
		})
	}
}

func TestFilteredImages(t *testing.T) {
	c := newMockCache(Regions("us-west-1"))
	warmed := make(chan struct{})

	go func() { c.Run(context.Background(), warmed) }()

	defer c.Stop()
	<-warmed

	images, err := c.FilterImages("us-west-1", NewFilter(FilterByImageID("ami-1a2b3c4d")))
	if err != nil {
		t.Fatal(err)
	}

	if want, got := 1, len(images); want != got {
		t.Errorf("want: %d image(s), got: %d image(s)", want, got)
	}

	images, err = c.FilterImages("us-west-1", NewFilter(FilterByImageID("foo")))
	if err != nil {
		t.Fatal(err)
	}

	if want, got := 0, len(images); want != got {
		t.Errorf("want: %d image(s), got: %d image(s)", want, got)
	}

	_, err = c.FilterImages("us-foo-1", nil)

	if want, got := "unknown or unsupported region: us-foo-1", err.Error(); want != got {
		t.Errorf("\n\twant err: %q\n\t got err: %q", want, got)
	}
}

func TestPoolSize(t *testing.T) {
	tests := []struct {
		name    string
		max     int
		queue   int
		percent float64
		want    int
	}{
		{"min_workers", 10, 1, 0.05, 1},
		{"2_workers", 10, 42, 0.05, 2},
		{"5_workers", 10, 108, 0.05, 5},
		{"max_workers", 2, 1000, 0.05, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := poolSize(tt.max, tt.queue, tt.percent); tt.want != got {
				t.Errorf("want: %d, got: %d", tt.want, got)
			}
		})
	}
}

type mockAWSErr struct{}

func (mockAWSErr) Error() string   { return "foo" }
func (mockAWSErr) Code() string    { return "42" }
func (mockAWSErr) Message() string { return "foobar" }
func (mockAWSErr) OrigErr() error  { return errors.New("o.g. foo") }

func TestAWSError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"awsError", &mockAWSErr{}, "foobar"},
		{"error", errors.New("foobar"), "foobar"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := awsError(tt.err).Error(); tt.want != got {
				t.Errorf("want: %v, got: %v", tt.want, got)
			}
		})
	}
}
