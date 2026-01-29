package cli

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/DotNaos/moodle-cli/internal/moodle"
)

var explicitIDSuffix = regexp.MustCompile(`^(.*)\s+\[id:([^\]]+)\]$`)

func resolveCourseID(client *moodle.Client, input string) (string, error) {
	courses, err := client.FetchCourses()
	if err != nil {
		return "", err
	}
	return resolveCourseIDFromCourses(courses, input)
}

func resolveCourseIDFromCourses(courses []moodle.Course, input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("course not found: %s", input)
	}

	if id, ok := extractExplicitID(trimmed); ok {
		return id, nil
	}

	for _, course := range courses {
		if fmt.Sprintf("%d", course.ID) == trimmed {
			return trimmed, nil
		}
	}

	matches := make([]moodle.Course, 0, 1)
	for _, course := range courses {
		if strings.EqualFold(course.Fullname, trimmed) || (course.Shortname != "" && strings.EqualFold(course.Shortname, trimmed)) {
			matches = append(matches, course)
		}
	}

	switch len(matches) {
	case 1:
		return fmt.Sprintf("%d", matches[0].ID), nil
	case 0:
		return "", fmt.Errorf("course not found: %s", input)
	default:
		return "", fmt.Errorf("course name is ambiguous: %s (use course id)", input)
	}
}

func resolveResource(resources []moodle.Resource, input string) (*moodle.Resource, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, fmt.Errorf("resource not found: %s", input)
	}

	if id, ok := extractExplicitID(trimmed); ok {
		for i := range resources {
			if resources[i].ID == id {
				return &resources[i], nil
			}
		}
		return nil, fmt.Errorf("resource not found: %s", id)
	}

	for i := range resources {
		if resources[i].ID == trimmed {
			return &resources[i], nil
		}
	}

	matches := make([]moodle.Resource, 0, 1)
	for i := range resources {
		if strings.EqualFold(resources[i].Name, trimmed) {
			matches = append(matches, resources[i])
		}
	}

	switch len(matches) {
	case 1:
		return &matches[0], nil
	case 0:
		return nil, fmt.Errorf("resource not found: %s", input)
	default:
		return nil, fmt.Errorf("resource name is ambiguous: %s (use resource id)", input)
	}
}

func extractExplicitID(value string) (string, bool) {
	match := explicitIDSuffix.FindStringSubmatch(value)
	if len(match) != 3 {
		return "", false
	}
	return strings.TrimSpace(match[2]), true
}
