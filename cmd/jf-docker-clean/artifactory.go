package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jfrog/jfrog-client-go/artifactory"
	artauth "github.com/jfrog/jfrog-client-go/artifactory/auth"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/jfrog/jfrog-client-go/config"
	"github.com/jfrog/jfrog-client-go/utils/io/content"
)

type client struct {
	mgr  artifactory.ArtifactoryServicesManager
	repo string
}

func newClient(ctx context.Context, url, user, token, repo string, threads int) (*client, error) {
	details := artauth.NewArtifactoryDetails()
	details.SetUrl(strings.TrimSuffix(url, "/") + "/artifactory/")
	if user != "" {
		details.SetUser(user)
	}
	details.SetAccessToken(token)

	cfg, err := config.NewConfigBuilder().SetServiceDetails(details).SetContext(ctx).SetThreads(threads).Build()
	if err != nil {
		return nil, err
	}
	mgr, err := artifactory.New(cfg)
	if err != nil {
		return nil, err
	}
	return &client{mgr: mgr, repo: repo}, nil
}

// listImages returns the tags of the repository, found through their manifest.json.
// The image and tag patterns ('*' and '?' wildcards) are put into the AQL query.
func (c *client) listImages(image, tag string) ([]Image, error) {
	aql, err := buildAQL(c.repo, image, tag)
	if err != nil {
		return nil, err
	}
	r, err := c.mgr.Aql(aql)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Results []utils.ResultItem `json:"results"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	var images []Image
	for _, it := range resp.Results {
		// path is <image>/<tag>, where image may contain slashes.
		i := strings.LastIndex(it.Path, "/")
		if i < 0 {
			continue
		}
		created, err := time.Parse(time.RFC3339, it.Created)
		if err != nil {
			return nil, fmt.Errorf("parse created of %s: %w", it.Path, err)
		}
		var downloads int64
		for _, st := range it.Stats {
			n, err := st.Downloads.Int64()
			if err != nil {
				return nil, fmt.Errorf("parse downloads of %s: %w", it.Path, err)
			}
			downloads += n
		}
		images = append(images, Image{Name: it.Path[:i], Tag: it.Path[i+1:], Created: created, Downloads: downloads})
	}
	return images, nil
}

// deleteImages removes the tag folders; the client deletes them in parallel
// using the number of threads set in newClient.
func (c *client) deleteImages(images []Image) error {
	items := make([]utils.ResultItem, 0, len(images))
	for _, img := range images {
		items = append(items, utils.ResultItem{Repo: c.repo, Path: img.Name, Name: img.Tag, Type: string(utils.Folder)})
	}
	data, err := json.Marshal(map[string]any{"results": items})
	if err != nil {
		return err
	}

	f, err := os.CreateTemp("", "jf-docker-clean-*.json")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	reader := content.NewContentReader(f.Name(), "results")
	defer func() { _ = reader.Close() }()
	_, err = c.mgr.DeleteFiles(reader)
	return err
}

// buildAQL builds the query listing manifests of the repository. The path of a
// manifest is <image>/<tag>, so the image and tag patterns are combined into a
// single $match on the path. The result may contain more than the patterns
// select (an image pattern can match across '/'), so it still has to be
// filtered with Select.
func buildAQL(repo, image, tag string) (string, error) {
	query := map[string]any{"repo": repo, "name": "manifest.json"}

	if image != "" || tag != "" {
		if image == "" {
			image = "*"
		}
		if tag == "" {
			tag = "*"
		}
		query["path"] = map[string]string{"$match": image + "/" + tag}
	}

	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(query); err != nil {
		return "", err
	}
	return fmt.Sprintf(`items.find(%s).include("path","created","stat.downloads")`, strings.TrimSpace(b.String())), nil
}
