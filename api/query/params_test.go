// Copyright 2017 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package query

import (
	"fmt"
	"net/url"
	"reflect"
	"testing"

	"github.com/intuit/ami-query/amicache"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  Params
	}{
		{
			amicache.DefaultStateTag,
			fmt.Sprintf("%s=available&%[1]s=deprecated&%[1]s=available", amicache.DefaultStateTag),
			Params{
				regions: []string{},
				images:  []string{},
				tags: map[string][]string{
					amicache.DefaultStateTag: []string{"available", "deprecated"},
				},
			},
		},
		{
			"tags",
			"tag=foo1:bar&tag=foo2:bar&tag=foo1:baz&tag=foo1:baz&tag=foo2:baz",
			Params{
				regions: []string{},
				images:  []string{},
				tags: map[string][]string{
					"foo1": []string{"bar", "baz"},
					"foo2": []string{"bar", "baz"},
				},
			},
		},
		{
			"tags_with_colon",
			"tag=foo1:bar&tag=foo2:bar:baz:bot&tag=foo2:bar&tag=foo1:baz&tag=foo2:bar:baz",
			Params{
				regions: []string{},
				images:  []string{},
				tags: map[string][]string{
					"foo1": []string{"bar", "baz"},
					"foo2": []string{"bar:baz:bot", "bar", "bar:baz"},
				},
			},
		},
		{
			"ami",
			"ami=ami-1a2b3c4d&ami=ami-2a2b3c4d&ami=ami-3a2b3c4d&ami=ami-2a2b3c4d",
			Params{
				regions: []string{},
				images:  []string{"ami-1a2b3c4d", "ami-2a2b3c4d", "ami-3a2b3c4d"},
				tags:    map[string][]string{},
			},
		},
		{
			"owner_id",
			"owner_id=foo&owner_id=bar&owner_id=foo",
			Params{
				ownerID: "foo",
				regions: []string{},
				images:  []string{},
				tags:    map[string][]string{},
			},
		},
		{
			"launch_permission",
			"launch_permission=foo&launch_permission=bar&launch_permission=foo",
			Params{
				launchPerm: "foo",
				regions:    []string{},
				images:     []string{},
				tags:       map[string][]string{},
			},
		},
		{
			"callback",
			"callback=foo&callback=bar&callback=foo",
			Params{
				callback: "foo",
				regions:  []string{},
				images:   []string{},
				tags:     map[string][]string{},
			},
		},
		{
			"pretty",
			"pretty",
			Params{
				pretty:  true,
				regions: []string{},
				images:  []string{},
				tags:    map[string][]string{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Params{}
			if err := p.Decode(amicache.DefaultStateTag, &url.URL{RawQuery: tt.query}); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tt.want, p) {
				t.Errorf("\n\twant: %#v\n\t got: %#v", tt.want, p)
			}
		})
	}
}

func TestDecodeBadKey(t *testing.T) {
	p := &Params{}
	err := p.Decode(amicache.DefaultStateTag, &url.URL{RawQuery: "foo=bar"})
	if want, got := "unknown query key: foo", err.Error(); want != got {
		t.Errorf("\n\twant err: %q\n\t got err: %q", want, got)
	}
}

func TestDecodeBadTagValue(t *testing.T) {
	p := &Params{}
	err := p.Decode(amicache.DefaultStateTag, &url.URL{RawQuery: "tag=foobar"})
	if want, got := "invalid query tag value: foobar", err.Error(); want != got {
		t.Errorf("\n\twant err: %q\n\t got err: %q", want, got)
	}
}

func TestDecodeParseError(t *testing.T) {
	p := &Params{}
	err := p.Decode(amicache.DefaultStateTag, &url.URL{RawQuery: `foo=%%bar`})
	if want, got := `invalid URL escape "%%b"`, err.Error(); want != got {
		t.Errorf("\n\twant err: %q\n\t got err: %q", want, got)
	}
}

func TestDedup(t *testing.T) {
	got := dedup([]string{"foo", "bar", "baz", "foo"})
	want := []string{"foo", "bar", "baz"}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("want: %v, got: %v", want, got)
	}
}
