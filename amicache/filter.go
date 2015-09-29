// Copyright 2015 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package amicache

import (
	"github.com/intuit/ami-query/ami"
)

// Filterer is an interface used to apply specific filters on a slice of
// ami.AMI objects.
type Filterer interface {
	Filter([]ami.AMI) []ami.AMI
}

// Filter is used to filter ami.AMI objects.
type Filter struct {
	filters []Filterer
}

// NewFilter creates a new Filter with the specified Filterer interfaces.
func NewFilter(filters ...Filterer) *Filter {
	return &Filter{filters: filters}
}

// Apply returns the filtered amis.
func (f *Filter) Apply(amis []ami.AMI) []ami.AMI {
	for _, f := range f.filters {
		amis = f.Filter(amis)
	}
	return amis
}

// The FilterFunc type is an adapter to allow the use of ordinary functions as
// filter handlers. If f is a function with the appropriate signature,
// FilterFunc(f) is a Filterer object that calls f.
type FilterFunc func([]ami.AMI) []ami.AMI

// Filter returns f(a).
func (f FilterFunc) Filter(a []ami.AMI) []ami.AMI {
	return f(a)
}

// FilterByID filters on one or more AMI IDs.
func FilterByID(ids ...string) FilterFunc {
	return FilterFunc(func(amis []ami.AMI) []ami.AMI {
		if len(ids) == 0 {
			return amis
		}
		var newAMIs []ami.AMI
		for i := range amis {
			for _, id := range ids {
				if id == *amis[i].Image.ImageId {
					newAMIs = append(newAMIs, amis[i])
				}
			}
		}
		return newAMIs
	})
}

// FilterByTags filters on a set of tags.
func FilterByTags(tags map[string][]string) FilterFunc {
	return FilterFunc(func(amis []ami.AMI) []ami.AMI {
		if len(tags) == 0 {
			return amis
		}
		var newAMIs []ami.AMI
		for i := range amis {
			tagMatches := 0
			for _, tag := range amis[i].Image.Tags {
				if values, ok := tags[*tag.Key]; ok {
					for _, val := range values {
						if val == *tag.Value {
							tagMatches++
							break
						}
					}
				}
			}
			if tagMatches == len(tags) {
				newAMIs = append(newAMIs, amis[i])
			}
		}
		return newAMIs
	})
}
