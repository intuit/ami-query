// Copyright 2015 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package ami

import (
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
)

// Life cycle status weights.
const (
	_                   = iota
	deregistered uint64 = 10000000000 * iota
	development
	unavailable
	exception
	deprecated
	available
)

// statusWeight lookup table.
var statusWeight = map[string]uint64{
	"available":    available,
	"deprecated":   deprecated,
	"exception":    exception,
	"unavailable":  unavailable,
	"development":  development,
	"deregistered": deregistered,
}

// AMI represents an ec2.Image and the region it resides in.
type AMI struct {
	Region string
	Image  *ec2.Image
}

// NewAMI returns a new AMI from the provided ec2.Image and region.
func NewAMI(region string, image *ec2.Image) AMI {
	return AMI{Region: region, Image: image}
}

// Tag returns the value of the provided tag key. An empty string is returned if
// there is no matching key.
func (a *AMI) Tag(key string) string {
	for _, tag := range a.Image.Tags {
		if *tag.Key == key {
			return *tag.Value
		}
	}
	return ""
}

// Tags is a convenience function that returns ec2.Image.Tags as a
// map[string]string
func (a *AMI) Tags() map[string]string {
	tags := make(map[string]string)
	for _, tag := range a.Image.Tags {
		tags[*tag.Key] = *tag.Value
	}
	return tags
}

// Sort is a slice of AMI objects to be sorted
type Sort []AMI

// By sorts the AMI slice according to the function closure "by".
func (s Sort) By(by func(a1, a2 *AMI) bool) {
	sort.Sort(&amiSorter{amis: s, by: by})
}

// amiSorter joins a By function and a slice of AMI objects to be sorted.
type amiSorter struct {
	amis []AMI
	by   func(a1, a2 *AMI) bool
}

// sort.Interface functions
func (s *amiSorter) Len() int           { return len(s.amis) }
func (s *amiSorter) Swap(i, j int)      { s.amis[i], s.amis[j] = s.amis[j], s.amis[i] }
func (s *amiSorter) Less(i, j int) bool { return s.by(&s.amis[i], &s.amis[j]) }

// Status sorts by taking the CreationDate attribute, converting it to UNIX
// epoch, and adds it to the weighted value of the status tag. It sorts from
// newest to oldest AMIs.
func Status(a1, a2 *AMI) bool {
	var dateFmt = "2006-01-02T15:04:05.000Z"
	var icdate, istatus uint64
	var jcdate, jstatus uint64

	// Parse the CreationDate attribute
	if a1.Image.CreationDate != nil {
		if date, err := time.Parse(dateFmt, *a1.Image.CreationDate); err == nil {
			icdate = uint64(date.Unix())
		}
	}

	if a2.Image.CreationDate != nil {
		if date, err := time.Parse(dateFmt, *a2.Image.CreationDate); err == nil {
			jcdate = uint64(date.Unix())
		}
	}

	// Get the status tag
	if status := a1.Tag("status"); status != "" {
		istatus, _ = statusWeight[strings.ToLower(status)]
	}

	if status := a2.Tag("status"); status != "" {
		jstatus, _ = statusWeight[strings.ToLower(status)]
	}

	return icdate+istatus > jcdate+jstatus
}
