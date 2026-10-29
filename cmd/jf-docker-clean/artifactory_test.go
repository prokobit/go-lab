package main

import "testing"

func TestBuildAQL(t *testing.T) {
	tests := []struct {
		name       string
		image, tag string
		want       string
	}{
		{"no filters", "", "",
			`items.find({"name":"manifest.json","repo":"docker-local"}).include("path","created","stat.downloads")`},
		{"image only", "team/app", "",
			`items.find({"name":"manifest.json","path":{"$match":"team/app/*"},"repo":"docker-local"}).include("path","created","stat.downloads")`},
		{"tag only", "", "pr-*",
			`items.find({"name":"manifest.json","path":{"$match":"*/pr-*"},"repo":"docker-local"}).include("path","created","stat.downloads")`},
		{"image and tag", "team/*", "v*",
			`items.find({"name":"manifest.json","path":{"$match":"team/*/v*"},"repo":"docker-local"}).include("path","created","stat.downloads")`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildAQL("docker-local", tt.image, tt.tag)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got  %s\nwant %s", got, tt.want)
			}
		})
	}
}
