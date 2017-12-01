// Copyright 2015 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package query

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/intuit/ami-query/amicache"

	"github.com/aws/aws-sdk-go/aws"
)

// APIPathQuery is the url path for the query API.
const APIPathQuery = "/amis"

// API serves the query API.
type API struct {
	cache   cacher
	regions []string
}

// Result contains the matching AMIs for a query.
type Result struct {
	ID                 string            `json:"id"`
	Region             string            `json:"region"`
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	VirtualizationType string            `json:"virtualizationtype"`
	CreationDate       string            `json:"creationdate"`
	Tags               map[string]string `json:"tags"`
}

// NewAPI returns a usable query API.
func NewAPI(cache *amicache.Cache) *API {
	return &API{
		cache:   cache,
		regions: cache.Regions(),
	}
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := &Params{}
	if err := p.Decode(r.URL); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}

	// If no regions were provided, search all cached regions.
	if len(p.regions) == 0 {
		p.regions = a.regions
	}

	images, err := a.getImages(p)
	if err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}

	a.EncodeTo(w, p, images)
}

// EncodeTo writes the JSON formatted results to the http.ResponseWriter.
func (a *API) EncodeTo(w http.ResponseWriter, p *Params, images []amicache.Image) {
	results := []Result{}
	for _, image := range images {
		results = append(results, Result{
			Region:             image.Region,
			ID:                 aws.StringValue(image.Image.ImageId),
			Name:               aws.StringValue(image.Image.Name),
			Description:        aws.StringValue(image.Image.Description),
			VirtualizationType: aws.StringValue(image.Image.VirtualizationType),
			CreationDate:       aws.StringValue(image.Image.CreationDate),
			Tags:               image.Tags(),
		})
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	if p.callback != "" {
		w.Header().Set("Content-Type", "application/javascript")
		fmt.Fprintf(w, "%s(", p.callback)
		enc.Encode(results)
		fmt.Fprint(w, ");")
	} else {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if p.pretty {
			enc.SetIndent("", " ")
		}
		enc.Encode(results)
	}
}

// Get the images from the cache based on the query.
func (a *API) getImages(p *Params) ([]amicache.Image, error) {
	images := []amicache.Image{}
	filter := amicache.NewFilter(
		amicache.FilterByImageID(p.images...),
		amicache.FilterByAccountID(p.acctID),
		amicache.FilterByTags(p.tags),
	)
	for _, region := range p.regions {
		matched, err := a.cache.FilterImages(region, filter)
		if err != nil {
			return nil, err
		}
		images = append(images, matched...)
	}
	amicache.SortByState(images)
	return images, nil
}

// Writes a JSON formatted error message to an http.ResponseWriter.
func writeErr(w http.ResponseWriter, err error, status int) {
	var id string
	switch status {
	case http.StatusBadRequest:
		id = "bad_request"
	case http.StatusInternalServerError:
		id = "internal_error"
	default:
		id = "unknown_error"
		status = http.StatusInternalServerError
	}
	http.Error(w, fmt.Sprintf(`{"id":"%s","message":"%s"}`, id, err), status)
}

// cacher is used to represent an amicache.Cache. Used to mock the cache in tests.
type cacher interface {
	Regions() []string
	FilterImages(string, *amicache.Filter) ([]amicache.Image, error)
}
