package main

import (
	"os"
	"strings"
	"testing"
)

// TestVersionMatchesReleaseTag is the release gate: the workflow exports
// RELEASE_TAG (the git tag, "v0.3.0") and the build is refused when the
// version compiled into the binary disagrees with it. Without the variable
// — every local and CI run — there is nothing to compare, and the test is
// skipped rather than silently green.
func TestVersionMatchesReleaseTag(t *testing.T) {
	tag, ok := os.LookupEnv("RELEASE_TAG")
	if !ok {
		t.Skip("RELEASE_TAG is not set; the release workflow sets it")
	}
	want := strings.TrimPrefix(tag, "v")
	if want == "" || want == tag {
		t.Fatalf("RELEASE_TAG %q is not a v<version> tag", tag)
	}
	if version != want {
		t.Fatalf("cmd/trilha-spec/main.go says version %q but the release tag is %q; bump version and retag", version, tag)
	}
}

// TestVersionIsSemver keeps the default honest: three numbers, no "v".
func TestVersionIsSemver(t *testing.T) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		t.Fatalf("version %q is not MAJOR.MINOR.PATCH", version)
	}
	for _, p := range parts {
		if p == "" || strings.Trim(p, "0123456789") != "" {
			t.Fatalf("version %q is not MAJOR.MINOR.PATCH", version)
		}
	}
}
