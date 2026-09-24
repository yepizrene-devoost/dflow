package selfupdate

import (
	"strconv"
	"strings"
)

// CompareVersions orders two release versions and returns -1, 0 or 1 as a is
// older than, equal to, or newer than b. It is the decision behind "is the
// published release newer than the running binary?".
//
// The rule is deliberately small, because dflow only ever publishes clean
// `vMAJOR.MINOR.PATCH` tags through GoReleaser, and pulling in
// `golang.org/x/mod` to compare three integers is not a trade worth making.
//
// The comparison proceeds in two tiers:
//
//  1. Numeric tier — when both inputs parse as `MAJOR.MINOR.PATCH`, each
//     component is compared as a number. The leading `v` is optional, and a
//     missing patch or minor counts as 0, so "1.2" equals "1.2.0".
//  2. Fallback tier — when either input does not parse (a pre-release suffix
//     such as "1.0.0-rc1", an empty string, a non-numeric component, or more
//     than three components), the two normalized strings are compared
//     byte-for-byte with Go's ordinary string ordering. On that path the empty
//     string sorts oldest, because "" is a prefix of every other string.
//
// The fallback exists so the function is total and deterministic rather than
// returning an error to every caller for input the updater can still handle.
// It is not semver: pre-release ordering is explicitly out of scope here, and a
// version like "1.0.0-rc1" will sort above "1.0.0" under the string rule.
func CompareVersions(a, b string) int {
	aNumeric, aOK := parseVersion(a)
	bNumeric, bOK := parseVersion(b)

	if aOK && bOK {
		return compareNumeric(aNumeric, bNumeric)
	}
	return strings.Compare(normalizeVersion(a), normalizeVersion(b))
}

// IsNewer reports whether candidate is a strictly newer version than current.
// It is CompareVersions with the argument order made explicit, because
// `CompareVersions(x, y) > 0` reads ambiguously at a call site while
// `IsNewer(latest, installed)` does not.
func IsNewer(candidate, current string) bool {
	return CompareVersions(candidate, current) > 0
}

// normalizeVersion strips the optional leading `v` and surrounding whitespace,
// leaving the version body both parsers agree on.
func normalizeVersion(raw string) string {
	return strings.TrimPrefix(strings.TrimSpace(raw), "v")
}

// parseVersion splits a normalized version into its three numeric components.
// The boolean is false whenever the input is not a clean `MAJOR.MINOR.PATCH`
// triple, which routes the comparison to the string fallback instead of
// guessing at a missing or non-numeric part.
func parseVersion(raw string) ([3]uint64, bool) {
	var parsed [3]uint64

	normalized := normalizeVersion(raw)
	if normalized == "" {
		return parsed, false
	}

	parts := strings.Split(normalized, ".")
	if len(parts) > 3 {
		return parsed, false
	}

	for index, part := range parts {
		if part == "" {
			return parsed, false
		}
		value, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return parsed, false
		}
		parsed[index] = value
	}
	return parsed, true
}

// compareNumeric compares two parsed triples component by component.
func compareNumeric(a, b [3]uint64) int {
	for index := range a {
		switch {
		case a[index] < b[index]:
			return -1
		case a[index] > b[index]:
			return 1
		}
	}
	return 0
}
