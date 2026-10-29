package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
)

type cli struct {
	URL           string         `env:"JFROG_URL" required:"" help:"Artifactory base URL."`
	Token         string         `env:"JFROG_ACCESS_TOKEN" required:"" help:"Access token."`
	User          string         `env:"JFROG_USER" help:"Username (optional)."`
	Repo          string         `default:"docker-local" help:"Docker repository."`
	Image         string         `required:"" help:"Image name pattern, * and ? are wildcards."`
	Tag           string         `required:"" help:"Tag pattern, * and ? are wildcards."`
	Protect       *regexp.Regexp `default:"^v?\\d+\\.\\d+\\.\\d+$" help:"Regex of tags that are never deleted (use '^$' to disable)."`
	KeepLatest    int            `name:"keep-latest" default:"5" help:"Keep N newest matching tags of every image."`
	OlderThanDays int            `name:"older-than-days" default:"7" help:"Delete only tags older than N days."`
	Threads       int            `default:"2" help:"Number of parallel deletions."`
	Delete        bool           `help:"Actually delete (default is dry-run)."`
}

func main() {
	var args cli
	kong.Parse(&args, kong.Description("Clean Docker tags in Artifactory."))

	if err := run(&args); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args *cli) error {
	if err := checkPattern("--image", args.Image); err != nil {
		return err
	}
	if err := checkPattern("--tag", args.Tag); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	c, err := newClient(ctx, args.URL, args.User, args.Token, args.Repo, args.Threads)
	if err != nil {
		return err
	}
	all, err := c.listImages(args.Image, args.Tag)
	if err != nil {
		return err
	}

	olderThan := time.Duration(args.OlderThanDays) * 24 * time.Hour
	selected, kept := Select(all, args.Image, args.Tag, args.Protect, args.KeepLatest, olderThan, time.Now())
	for _, img := range selected {
		printImage("DELETE", args.Repo, img)
	}
	for _, img := range kept {
		printImage("KEEP", args.Repo, img)
	}
	fmt.Printf("%d to delete, %d kept by --keep-latest, %d found\n", len(selected), len(kept), len(all))

	if !args.Delete {
		fmt.Println("dry-run: nothing deleted, use --delete to remove")
		return nil
	}
	return c.deleteImages(selected)
}

// checkPattern rejects empty patterns.
func checkPattern(flag, pattern string) error {
	if pattern == "" {
		return fmt.Errorf("%s must not be empty (use '*' to match everything)", flag)
	}
	return nil
}

func printImage(action, repo string, img Image) {
	fmt.Printf("%s\t%s/%s:%s\t%s\t%d downloads\n", action, repo, img.Name, img.Tag, img.Created.Format(time.RFC3339), img.Downloads)
}
