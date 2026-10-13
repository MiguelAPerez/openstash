package store

import (
	"sort"
	"strings"

	"golang.org/x/mod/semver"
)

func maxVersion(versions []string) string {
	best := versions[0]
	for _, v := range versions[1:] {
		if CompareVersions(v, best) > 0 {
			best = v
		}
	}
	return best
}

// CompareVersions orders two version strings: semver order when both parse
// (shorthands like "1.2" count), any semver above any non-semver string, and
// byte order otherwise. Versions semver treats as equal (build metadata, a "v"
// prefix) fall back to byte order, so the order is total and LatestVersion
// always agrees with the last entry of SortVersions.
func CompareVersions(a, b string) int {
	if c := compareVersion(a, b); c != 0 {
		return c
	}
	return strings.Compare(a, b)
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

// SortVersions orders versions oldest to newest with CompareVersions, the same
// order LatestVersion uses: non-semver strings first (byte order), then semver.
func SortVersions(versions []string) {
	sort.Slice(versions, func(i, j int) bool {
		return CompareVersions(versions[i], versions[j]) < 0
	})
}
