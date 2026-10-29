package main

import (
	"reflect"
	"regexp"
	"testing"
	"time"
)

func TestSelect(t *testing.T) {
	now := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	day := func(n int) time.Time { return now.AddDate(0, 0, -n) }

	images := []Image{
		{"app", "v1", day(40), 0},
		{"app", "v2", day(30), 0},
		{"app", "v3", day(5), 0},
		{"app", "latest", day(1), 0},
		{"web", "v1", day(50), 0},
		{"web", "v2", day(2), 0},
	}

	tests := []struct {
		name       string
		image, tag string
		protect    string
		keep       int
		olderThan  time.Duration
		want       []string // image:tag
	}{
		{"no filters selects all", "", "", "", 0, 0, []string{"app:latest", "app:v1", "app:v2", "app:v3", "web:v1", "web:v2"}},
		{"image pattern", "web", "", "", 0, 0, []string{"web:v1", "web:v2"}},
		{"tag pattern", "", "v*", "", 0, 0, []string{"app:v1", "app:v2", "app:v3", "web:v1", "web:v2"}},
		{"keep latest counts only matching tags", "", "v*", "", 1, 0, []string{"app:v1", "app:v2", "web:v1"}},
		{"keep latest per image", "", "", "", 2, 0, []string{"app:v1", "app:v2"}},
		{"keep larger than group", "web", "", "", 5, 0, nil},
		{"older than", "", "", "", 0, 28 * 24 * time.Hour, []string{"app:v1", "app:v2", "web:v1"}},
		{"older than with keep latest", "", "v*", "", 1, 35 * 24 * time.Hour, []string{"app:v1", "web:v1"}},
		{"protect semver", "", "", `^v\d+$`, 0, 0, []string{"app:latest"}},
		{"protected tags do not count for keep latest", "", "", `^v[12]$`, 1, 0, []string{"app:v3"}},
		{"no match", "nope", "", "", 0, 0, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var protectRe *regexp.Regexp
			if tt.protect != "" {
				protectRe = regexp.MustCompile(tt.protect)
			}
			var got []string
			deleted, _ := Select(images, tt.image, tt.tag, protectRe, tt.keep, tt.olderThan, now)
			for _, s := range deleted {
				got = append(got, s.Name+":"+s.Tag)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelectEmpty(t *testing.T) {
	deleted, kept := Select(nil, "", "", nil, 3, time.Hour, time.Now())
	if len(deleted) != 0 || len(kept) != 0 {
		t.Errorf("got %v and %v, want empty", deleted, kept)
	}
}

// Uses the real default of the --protect flag, so changing it in main.go is
// covered by this test.
func TestSelectDefaultProtect(t *testing.T) {
	now := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	protect := defaultProtect(t)

	images := []Image{
		{"app", "1.2.3", now.AddDate(0, 0, -10), 0},
		{"app", "v1.2.3", now.AddDate(0, 0, -9), 0},
		{"app", "1.1.1-dev.10", now.AddDate(0, 0, -8), 0},
		{"app", "1.2.3-rc.3", now.AddDate(0, 0, -7), 0},
		{"app", "latest", now.AddDate(0, 0, -6), 0},
	}

	var got []string
	deleted, _ := Select(images, "", "", protect, 0, 0, now)
	for _, s := range deleted {
		got = append(got, s.Tag)
	}
	want := []string{"1.1.1-dev.10", "1.2.3-rc.3", "latest"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSelectImageWithSlash(t *testing.T) {
	now := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	images := []Image{
		{"team/app", "v1", now, 0},
		{"team/app/sub", "v1", now, 0},
		{"team/web", "v1", now, 0},
	}

	tests := []struct {
		image string
		want  []string
	}{
		{"team/app", []string{"team/app"}},
		{"team/app*", []string{"team/app", "team/app/sub"}},
		{"team/*", []string{"team/app", "team/app/sub", "team/web"}},
		{"*/web", []string{"team/web"}},
	}
	for _, tt := range tests {
		var got []string
		deleted, _ := Select(images, tt.image, "", nil, 0, 0, now)
		for _, s := range deleted {
			got = append(got, s.Name)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("image %q: got %v, want %v", tt.image, got, tt.want)
		}
	}
}

func TestGlobMatch(t *testing.T) {
	tests := []struct {
		pattern, s string
		want       bool
	}{
		{"", "anything", true},
		{"abc", "abc", true},
		{"abc", "abcd", false},
		{"a*", "abc", true},
		{"*c", "abc", true},
		{"a*c", "ac", true},
		{"a*c", "abbbc", true},
		{"a*c", "abbbd", false},
		{"a?c", "abc", true},
		{"a?c", "ac", false},
		{"a?c", "abbc", false},
		{"*", "", true},
		{"**a", "ba", true},
		{"v1.*", "v1.2.3", true},
		{"v1.*", "v10", false},
	}
	for _, tt := range tests {
		if got := globMatch(tt.pattern, tt.s); got != tt.want {
			t.Errorf("globMatch(%q, %q) = %v, want %v", tt.pattern, tt.s, got, tt.want)
		}
	}
}

func TestSelectKept(t *testing.T) {
	now := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	day := func(n int) time.Time { return now.AddDate(0, 0, -n) }

	images := []Image{
		{"app", "v1", day(40), 0},
		{"app", "v2", day(30), 0},
		{"app", "v3", day(5), 0},
		{"app", "1.0.0", day(50), 0},
		{"web", "v1", day(50), 0},
	}
	protect := regexp.MustCompile(`^\d+\.\d+\.\d+$`)

	tests := []struct {
		name        string
		keep        int
		olderThan   time.Duration
		wantDeleted []string
		wantKept    []string
	}{
		{"keep two newest", 2, 0, []string{"app:v1"}, []string{"app:v2", "app:v3", "web:v1"}},
		{"keep more than exists", 5, 0, nil, []string{"app:v1", "app:v2", "app:v3", "web:v1"}},
		{"keep disabled", 0, 0, []string{"app:v1", "app:v2", "app:v3", "web:v1"}, nil},
		{"too new images are neither kept nor deleted", 1, 20 * 24 * time.Hour, []string{"app:v1", "app:v2"}, []string{"app:v3", "web:v1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleted, kept := Select(images, "", "", protect, tt.keep, tt.olderThan, now)
			var gotDeleted, gotKept []string
			for _, s := range deleted {
				gotDeleted = append(gotDeleted, s.Name+":"+s.Tag)
			}
			for _, s := range kept {
				gotKept = append(gotKept, s.Name+":"+s.Tag)
			}
			if !reflect.DeepEqual(gotDeleted, tt.wantDeleted) {
				t.Errorf("deleted: got %v, want %v", gotDeleted, tt.wantDeleted)
			}
			if !reflect.DeepEqual(gotKept, tt.wantKept) {
				t.Errorf("kept: got %v, want %v", gotKept, tt.wantKept)
			}
		})
	}
}
