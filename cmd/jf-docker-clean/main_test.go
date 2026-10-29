package main

import (
	"regexp"
	"testing"

	"github.com/alecthomas/kong"
)

// defaultProtect returns the --protect regex the CLI uses when the flag is not given.
func defaultProtect(t *testing.T) *regexp.Regexp {
	t.Helper()
	var args cli
	parser, err := kong.New(&args)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parser.Parse([]string{"--url=u", "--token=t", "--image=*", "--tag=*"}); err != nil {
		t.Fatal(err)
	}
	if args.Protect == nil {
		t.Fatal("--protect has no default")
	}
	return args.Protect
}

func TestCheckPattern(t *testing.T) {
	tests := []struct {
		pattern string
		wantErr bool
	}{
		{"*", false},
		{"team/app", false},
		{"v1.*", false},
		{"", true},
		{"v?", false},
	}
	for _, tt := range tests {
		if err := checkPattern("--tag", tt.pattern); (err != nil) != tt.wantErr {
			t.Errorf("checkPattern(%q) error = %v, wantErr %v", tt.pattern, err, tt.wantErr)
		}
	}
}
