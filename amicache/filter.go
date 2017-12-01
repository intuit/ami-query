// Copyright 2015 Intuit, Inc.  All rights reserved.
// Use of this source code is governed the MIT license
// that can be found in the LICENSE file.

package amicache

// Filterer is an interface used to apply specified filters on a slice of
// Image objects.
type Filterer interface {
	Filter([]Image) []Image
}

// Filter is used to filter Image objects.
type Filter struct {
	filters []Filterer
}

// NewFilter creates a new Filter.
func NewFilter(filters ...Filterer) *Filter {
	return &Filter{filters: filters}
}

// Apply returns the filtered images.
func (f *Filter) Apply(images []Image) []Image {
	for _, f := range f.filters {
		images = f.Filter(images)
	}
	return images
}

// The FilterFunc type is an adapter to allow the use of ordinary functions as
// filter handlers. If f is a function with the appropriate signature,
// FilterFunc(f) is a Filterer object that calls f.
type FilterFunc func([]Image) []Image

// Filter implements the Filterer interface.
func (f FilterFunc) Filter(images []Image) []Image { return f(images) }

// FilterByImageID returns images with matching AMI ids.
func FilterByImageID(ids ...string) FilterFunc {
	return FilterFunc(func(images []Image) []Image {
		if len(ids) == 0 {
			return images
		}
		newImages := []Image{}
		for i := range images {
			for _, id := range ids {
				if id == *images[i].Image.ImageId {
					newImages = append(newImages, images[i])
				}
			}
		}
		return newImages
	})
}

// FilterByTags returns images with matching tags.
func FilterByTags(tags map[string][]string) FilterFunc {
	return FilterFunc(func(images []Image) []Image {
		if len(tags) == 0 {
			return images
		}
		newImages := []Image{}
		for i := range images {
			tagMatches := 0
			for _, tag := range images[i].Image.Tags {
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
				newImages = append(newImages, images[i])
			}
		}
		return newImages
	})
}

// FilterByAccountID returns images that have the account id in its
// launch permissions.
func FilterByAccountID(id string) FilterFunc {
	return FilterFunc(func(images []Image) []Image {
		if id == "" {
			return images
		}
		newImages := []Image{}
		for _, image := range images {
			for _, iid := range image.launchPerms {
				if id == iid {
					newImages = append(newImages, image)
					break
				}
			}
		}
		return newImages
	})
}
