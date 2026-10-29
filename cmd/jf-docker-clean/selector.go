package main

import (
	"regexp"
	"sort"
	"time"
)

// Image is a tagged Docker image stored in Artifactory as <repo>/<name>/<tag>.
type Image struct {
	Name    string
	Tag     string
	Created time.Time
	// Downloads is the number of pulls of the tag (downloads of its manifest).
	Downloads int64
}

// Select returns the images that should be deleted and the ones kept only
// because of keepLatest.
//
// Images whose tag matches protectRe (nil protects nothing) are never returned
// and do not count towards keepLatest. The rest are filtered by the image name
// and tag patterns (see globMatch, empty matches everything). For every image
// name the keepLatest newest matching images are kept, and of the remaining
// ones only those older than olderThan are deleted
// (olderThan <= 0 disables the age check).
func Select(images []Image, image, tag string, protectRe *regexp.Regexp, keepLatest int, olderThan time.Duration, now time.Time) (deleted, kept []Image) {
	byImage := map[string][]Image{}
	for _, t := range images {
		if protectRe != nil && protectRe.MatchString(t.Tag) {
			continue
		}
		if !globMatch(image, t.Name) || !globMatch(tag, t.Tag) {
			continue
		}
		byImage[t.Name] = append(byImage[t.Name], t)
	}

	for _, group := range byImage {
		sort.Slice(group, func(i, j int) bool {
			if !group[i].Created.Equal(group[j].Created) {
				return group[i].Created.After(group[j].Created)
			}
			return group[i].Tag < group[j].Tag
		})
		if keepLatest > 0 {
			n := min(keepLatest, len(group))
			kept = append(kept, group[:n]...)
			group = group[n:]
		}
		for _, t := range group {
			if olderThan > 0 && now.Sub(t.Created) < olderThan {
				continue
			}
			deleted = append(deleted, t)
		}
	}

	sortImages(deleted)
	sortImages(kept)
	return deleted, kept
}

func sortImages(images []Image) {
	sort.Slice(images, func(i, j int) bool {
		if images[i].Name != images[j].Name {
			return images[i].Name < images[j].Name
		}
		return images[i].Tag < images[j].Tag
	})
}

// globMatch reports whether s matches pattern, where '*' matches any sequence
// of characters (including '/') and '?' any single one, like AQL's $match.
// An empty pattern matches everything.
func globMatch(pattern, s string) bool {
	if pattern == "" {
		return true
	}
	p, i := 0, 0
	star, mark := -1, 0
	for i < len(s) {
		switch {
		case p < len(pattern) && pattern[p] == '*':
			star, mark = p, i
			p++
		case p < len(pattern) && (pattern[p] == '?' || pattern[p] == s[i]):
			p++
			i++
		case star >= 0:
			p = star + 1
			mark++
			i = mark
		default:
			return false
		}
	}
	for p < len(pattern) && pattern[p] == '*' {
		p++
	}
	return p == len(pattern)
}
