// Copyright 2017 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package amicache

import (
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
)

// StateTag is the tag key value on an ec2.Image that represents its state.
const StateTag = "status"

// Life cycle state weights.
const (
	_                   = iota
	deregistered uint64 = 10000000000 * iota
	development
	prerelease
	unavailable
	exception
	deprecated
	available
)

// State weight lookup table.
var stateWeight = map[string]uint64{
	"available":    available,
	"deprecated":   deprecated,
	"exception":    exception,
	"unavailable":  unavailable,
	"pre-release":  prerelease,
	"development":  development,
	"deregistered": deregistered,
}

// Image represents an Amazon Machine Image.
type Image struct {
	Image       *ec2.Image
	OwnerID     string
	Region      string
	launchPerms []string
}

// NewImage returns a new Image from the provided ec2.Image and region.
func NewImage(image *ec2.Image, ownerID, region string, perms []string) Image {
	return Image{
		Image:       image,
		OwnerID:     ownerID,
		Region:      region,
		launchPerms: perms,
	}
}

// Tag returns the value of the provided tag key. An empty string is returned if
// there is no matching key.
func (i *Image) Tag(key string) string {
	for _, tag := range i.Image.Tags {
		if *tag.Key == key {
			return *tag.Value
		}
	}
	return ""
}

// Tags is a convenience function that returns ec2.Image.Tags as a
// map[string]string
func (i *Image) Tags() map[string]string {
	tags := make(map[string]string)
	for _, tag := range i.Image.Tags {
		tags[*tag.Key] = *tag.Value
	}
	return tags
}

// SortByState sorts by taking the CreationDate attribute, converting it to
// UNIX epoch, and adds it to the weighted value of the status tag. It sorts
// from newest to oldest AMIs.
func SortByState(images []Image) {
	sort.Slice(images, func(i, j int) bool {
		var dateFmt = "2006-01-02T15:04:05.000Z"
		var icdate, istate uint64
		var jcdate, jstate uint64

		// Parse the CreationDate attribute
		if date, err := time.Parse(dateFmt, *images[i].Image.CreationDate); err == nil {
			icdate = uint64(date.Unix())
		}

		if date, err := time.Parse(dateFmt, *images[j].Image.CreationDate); err == nil {
			jcdate = uint64(date.Unix())
		}

		// Get the state tag
		if state := images[i].Tag(StateTag); state != "" {
			istate, _ = stateWeight[strings.ToLower(state)]
		}

		if state := images[j].Tag(StateTag); state != "" {
			jstate, _ = stateWeight[strings.ToLower(state)]
		}

		return icdate+istate > jcdate+jstate
	})
}
