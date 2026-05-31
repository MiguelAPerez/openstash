package store

import (
	"strings"

	"golang.org/x/mod/semver"
)

func maxVersion(versions []string) string {
	best := versions[0]
	for _, v := range versions[1:] {
		if compareVersion(v, best) > 0 {
			best = v
		}
	}
	return best
}

func compareVersion(a, b string) int {
	sa, sb := semverTag(a), semverTag(b)
	if sa != "" && sb != "" {
		return semver.Compare(sa, sb)
	}
	if sa != "" {
		return 1
	}
	if sb != "" {
		return -1
	}
	return strings.Compare(a, b)
}

func semverTag(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	candidate := v
	if !strings.HasPrefix(candidate, "v") {
		candidate = "v" + candidate
	}
	if !semver.IsValid(candidate) {
		return ""
	}
	return semver.Canonical(candidate)
}
