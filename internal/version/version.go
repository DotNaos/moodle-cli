package version

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	DefaultVersion   = "dev"
	DefaultCommit    = "unknown"
	DefaultBuildDate = "unknown"
)

var (
	version   = DefaultVersion
	commit    = DefaultCommit
	buildDate = DefaultBuildDate
)

type Semver struct {
	Major int
	Minor int
	Patch int
}

func Version() string {
	if strings.TrimSpace(version) == "" {
		return DefaultVersion
	}
	return version
}

func Commit() string {
	if strings.TrimSpace(commit) == "" {
		return DefaultCommit
	}
	return commit
}

func BuildDate() string {
	if strings.TrimSpace(buildDate) == "" {
		return DefaultBuildDate
	}
	return buildDate
}

func IsDev() bool {
	return strings.EqualFold(Version(), DefaultVersion)
}

func ParseSemver(raw string) (Semver, error) {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "v")
	if trimmed == "" {
		return Semver{}, fmt.Errorf("empty version")
	}
	if idx := strings.Index(trimmed, "-"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	parts := strings.Split(trimmed, ".")
	if len(parts) != 3 {
		return Semver{}, fmt.Errorf("invalid version: %s", raw)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Semver{}, fmt.Errorf("invalid major version: %w", err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Semver{}, fmt.Errorf("invalid minor version: %w", err)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Semver{}, fmt.Errorf("invalid patch version: %w", err)
	}
	return Semver{Major: major, Minor: minor, Patch: patch}, nil
}

func Compare(a string, b string) (int, error) {
	left, err := ParseSemver(a)
	if err != nil {
		return 0, err
	}
	right, err := ParseSemver(b)
	if err != nil {
		return 0, err
	}

	switch {
	case left.Major != right.Major:
		return compareInt(left.Major, right.Major), nil
	case left.Minor != right.Minor:
		return compareInt(left.Minor, right.Minor), nil
	default:
		return compareInt(left.Patch, right.Patch), nil
	}
}

func compareInt(a int, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
