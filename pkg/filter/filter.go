package filter

import (
	"regexp"
	"sort"
	"time"
)

// Filter provides tag filtering capabilities
type Filter struct {
	Include []*regexp.Regexp
	Exclude []*regexp.Regexp
	Latest  int
}

// NewFilter creates a new filter from string patterns
func NewFilter(include, exclude []string, latest int) (*Filter, error) {
	f := &Filter{
		Latest: latest,
	}

	// Compile include patterns
	for _, pattern := range include {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		f.Include = append(f.Include, re)
	}

	// Compile exclude patterns
	for _, pattern := range exclude {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		f.Exclude = append(f.Exclude, re)
	}

	return f, nil
}

// Match checks if a tag matches the filter rules
func (f *Filter) Match(tag string) bool {
	// Check exclude patterns first
	for _, re := range f.Exclude {
		if re.MatchString(tag) {
			return false
		}
	}

	// If no include patterns, match all (after exclusions)
	if len(f.Include) == 0 {
		return true
	}

	// Check include patterns
	for _, re := range f.Include {
		if re.MatchString(tag) {
			return true
		}
	}

	return false
}

// FilterTags filters a list of tags and returns matching tags
func (f *Filter) FilterTags(tags []TagInfo) []string {
	var matched []TagInfo

	for _, tag := range tags {
		if f.Match(tag.Name) {
			matched = append(matched, tag)
		}
	}

	// Sort by updated time (newest first)
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Updated.After(matched[j].Updated)
	})

	// Apply latest limit
	if f.Latest > 0 && len(matched) > f.Latest {
		matched = matched[:f.Latest]
	}

	// Extract tag names
	result := make([]string, len(matched))
	for i, tag := range matched {
		result[i] = tag.Name
	}

	return result
}

// TagInfo contains tag metadata
type TagInfo struct {
	Name    string
	Updated time.Time
}
