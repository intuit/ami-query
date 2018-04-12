// Copyright 2017 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package query

import (
	"fmt"
	"net/url"
	"strings"
)

// Params defines all the dimensions of a query.
type Params struct {
	regions    []string
	images     []string
	tags       map[string][]string
	ownerID    string
	launchPerm string
	callback   string
	pretty     bool
}

// Decode populates a Params from a URL.
func (p *Params) Decode(stateTag string, u *url.URL) error {
	params, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return err
	}

	p.regions = []string{}
	p.images = []string{}
	p.tags = map[string][]string{}

	for key, values := range params {
		values = dedup(values)
		switch key {
		case "tag":
			for _, value := range values {
				if i := strings.Index(value, ":"); i != -1 {
					p.tags[value[:i]] = append(p.tags[value[:i]], value[i+1:])
				} else {
					return fmt.Errorf("invalid query tag value: %s", value)
				}
			}
		case stateTag, "state", "status": // aliases for the state tag
			p.tags[stateTag] = append(p.tags[stateTag], values...)
		case "ami":
			p.images = values
		case "region":
			p.regions = values
		case "owner_id":
			p.ownerID = values[0]
		case "launch_permission":
			p.launchPerm = values[0]
		case "callback":
			p.callback = values[0]
		case "pretty":
			p.pretty = p.pretty || values[0] != "0"
		default:
			return fmt.Errorf("unknown query key: %s", key)
		}
	}

	return nil
}

// Removes dups from a string slice.
func dedup(items []string) []string {
	newItems := []string{}
	added := map[string]struct{}{}
	for _, item := range items {
		if _, ok := added[item]; !ok {
			newItems = append(newItems, item)
			added[item] = struct{}{}
		}
	}
	return newItems
}
