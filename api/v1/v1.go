// Copyright 2015 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package v1

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"golang.org/x/net/context"

	"github.com/intuit/ami-query/ami"
	"github.com/intuit/ami-query/amicache"
	"github.com/intuit/ami-query/api"
)

// response is the JSON formatted response output.
type response struct {
	ID                 string            `json:"id"`
	Region             string            `json:"region"`
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	VirtualizationType string            `json:"virtualizationtype"`
	CreationDate       string            `json:"creationdate"`
	Tags               map[string]string `json:"tags"`
}

// values are the parsed values passed on the query string.
type values struct {
	regions  []string
	amis     []string
	tags     map[string][]string
	callback string
	pretty   bool
}

// Handler is version 1 of the REST API.
func Handler(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	cm, ok := ctx.Value(api.CacheManagerKey).(*amicache.Manager)
	if !ok {
		return http.StatusInternalServerError, fmt.Errorf("no cache")
	}

	v, err := parseQuery(r.URL.RawQuery, cm.Regions())
	if err != nil {
		return http.StatusBadRequest, err
	}

	amis, err := getAMIs(v, cm)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	ami.Sort(amis).By(ami.Status)

	// Build JSON response output from AMIs
	resp := make([]response, len(amis))
	for i := range amis {
		resp[i] = response{
			Region:             amis[i].Region,
			ID:                 aws.StringValue(amis[i].Image.ImageId),
			Name:               aws.StringValue(amis[i].Image.Name),
			Description:        aws.StringValue(amis[i].Image.Description),
			VirtualizationType: aws.StringValue(amis[i].Image.VirtualizationType),
			CreationDate:       aws.StringValue(amis[i].Image.CreationDate),
			Tags:               amis[i].Tags(),
		}
	}

	// Display indented JSON output if requested unless it's a JSON-P request
	var output []byte
	if v.pretty && v.callback == "" {
		output, err = json.MarshalIndent(resp, "", " ")
	} else {
		output, err = json.Marshal(resp)
	}

	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Wrap response in call back if requested
	if v.callback != "" {
		w.Header().Set("Content-Type", "application/javascript")
		fmt.Fprintf(w, "%s(%s);", v.callback, output)
	} else {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, string(output))
	}

	return http.StatusOK, nil
}

// parseQuery parses and validates the query string of the request and returns
// its values.
func parseQuery(query string, cachedRegions []string) (*values, error) {
	parameters, err := url.ParseQuery(query)
	if err != nil {
		return nil, err
	}

	v := &values{
		regions: make([]string, 0),
		amis:    make([]string, 0),
		tags:    make(map[string][]string),
	}

	var regions []string

	for key, values := range parameters {
		switch key {
		case "tag":
			{
				for _, value := range values {
					tag := strings.Split(value, ":")
					if len(tag) != 2 {
						return nil, fmt.Errorf("invalid query value: %s", value)
					}
					v.tags[tag[0]] = append(v.tags[tag[0]], tag[1])
				}
			}
		case "ami":
			v.amis = append(v.amis, values...)
		case "region":
			regions = append(regions, values...)
		case "status":
			v.tags["status"] = append(v.tags["status"], values...)
		case "callback":
			v.callback = values[0]
		case "pretty":
			v.pretty = v.pretty || values[0] != "0"
		default:
			return nil, fmt.Errorf("unknown query key: %s", key)
		}
	}

	// Check that the regions requested are being cached and remove any dups
	if len(regions) > 0 {
		added := make(map[string]bool)
		for _, region := range regions {
			found := false
			for _, cached := range cachedRegions {
				if region == cached {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("unknown or unsupported region: %s", region)
			}
			if _, ok := added[region]; !ok {
				v.regions = append(v.regions, region)
				added[region] = true
			}
		}
	} else {
		// Use all cached regions since none were provided
		v.regions = cachedRegions
	}

	return v, nil
}

// getAMIs retrieves images from each requested region.
func getAMIs(v *values, cm *amicache.Manager) ([]ami.AMI, error) {
	var amis []ami.AMI
	for _, region := range v.regions {
		filter := amicache.NewFilter(
			amicache.FilterByID(v.amis...),
			amicache.FilterByTags(v.tags),
		)
		images, err := cm.FilterImages(region, filter)
		if err != nil {
			log.Printf("cache failure for %s: %s", region, err)
			return nil, err
		}
		amis = append(amis, images...)
	}
	return amis, nil
}
