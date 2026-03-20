package cli

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/DotNaos/moodle-cli/internal/config"
	"github.com/DotNaos/moodle-cli/internal/moodle"
)

var explicitIDSuffix = regexp.MustCompile(`^(.*)\s+\[id:([^\]]+)\]$`)

func resolveCourseID(client *moodle.Client, input string) (string, error) {
	courses, err := client.FetchCourses()
	if err != nil {
		return "", err
	}
	return resolveCourseIDFromCoursesWithCurrent(courses, input, nil)
}

func resolveCourseIDFromCourses(courses []moodle.Course, input string) (string, error) {
	return resolveCourseIDFromCoursesWithCurrent(courses, input, nil)
}

type selectorOptions struct {
	Workspace string
	At        string
}

func resolveCourseIDWithOptions(client *moodle.Client, input string, options selectorOptions) (string, error) {
	courses, err := client.FetchCourses()
	if err != nil {
		return "", err
	}
	var currentCourse *currentLectureCourse
	if isCurrentSelector(input) {
		currentCourse, err = resolveCurrentLectureCourse(client, options)
		if err != nil {
			return "", err
		}
	}
	return resolveCourseIDFromCoursesWithCurrent(courses, input, currentCourse)
}

func resolveCourseIDFromCoursesWithCurrent(courses []moodle.Course, input string, current *currentLectureCourse) (string, error) {
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

	if strings.EqualFold(trimmed, "current") || trimmed == "0" {
		if current == nil || current.ID == 0 {
			return "", fmt.Errorf("no current or upcoming lecture course found for today")
		}
		return fmt.Sprintf("%d", current.ID), nil
	}

	if index, ok := parsePositiveIndex(trimmed); ok {
		if index > len(courses) {
			return "", fmt.Errorf("course index out of range: %s", input)
		}
		return fmt.Sprintf("%d", courses[index-1].ID), nil
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
	return resolveResourceWithCurrent(resources, input, "")
}

func resolveResourceWithOptions(client *moodle.Client, courseID string, resources []moodle.Resource, input string, options selectorOptions) (*moodle.Resource, error) {
	currentMaterialID := ""
	if isCurrentSelector(input) {
		currentResult, err := resolveCurrentLectureResult(client, options)
		if err != nil {
			return nil, err
		}
		if currentResult.Course != nil && fmt.Sprintf("%d", currentResult.Course.ID) == courseID && currentResult.Material != nil {
			currentMaterialID = currentResult.Material.ID
		}
	}
	return resolveResourceWithCurrent(resources, input, currentMaterialID)
}

func resolveResourceWithCurrent(resources []moodle.Resource, input string, currentMaterialID string) (*moodle.Resource, error) {
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

	if strings.EqualFold(trimmed, "current") || trimmed == "0" {
		if currentMaterialID != "" {
			for i := range resources {
				if resources[i].ID == currentMaterialID {
					return &resources[i], nil
				}
			}
		}
		ranked := rankCurrentLectureResources(resources, map[string]string{})
		for _, candidate := range ranked {
			if candidate.Kind != "lecture" {
				continue
			}
			for i := range resources {
				if resources[i].ID == candidate.ID {
					return &resources[i], nil
				}
			}
		}
		if len(ranked) > 0 {
			for i := range resources {
				if resources[i].ID == ranked[0].ID {
					return &resources[i], nil
				}
			}
		}
		return nil, fmt.Errorf("no current material found in course resources")
	}

	if index, ok := parsePositiveIndex(trimmed); ok {
		files := fileResources(resources)
		if index > len(files) {
			return nil, fmt.Errorf("resource index out of range: %s", input)
		}
		return &files[index-1], nil
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

func parsePositiveIndex(value string) (int, bool) {
	index, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || index < 1 {
		return 0, false
	}
	return index, true
}

func isCurrentSelector(value string) bool {
	trimmed := strings.TrimSpace(value)
	return strings.EqualFold(trimmed, "current") || trimmed == "0"
}

func fileResources(resources []moodle.Resource) []moodle.Resource {
	files := make([]moodle.Resource, 0, len(resources))
	for _, resource := range resources {
		if resource.Type != "resource" {
			continue
		}
		files = append(files, resource)
	}
	return files
}

func resolveCurrentLectureResult(client *moodle.Client, options selectorOptions) (currentLectureResult, error) {
	now, err := resolveLectureTimeAt(options.At)
	if err != nil {
		return currentLectureResult{}, err
	}
	cfg, err := config.LoadConfig(opts.ConfigPath)
	if err != nil {
		return currentLectureResult{}, err
	}
	if cfg.CalendarURL == "" {
		return currentLectureResult{}, fmt.Errorf("calendar URL not set. Run: moodle config set --calendar-url <url>")
	}
	return buildCurrentLectureResult(client, cfg.CalendarURL, now, options.Workspace)
}

func resolveCurrentLectureCourse(client *moodle.Client, options selectorOptions) (*currentLectureCourse, error) {
	result, err := resolveCurrentLectureResult(client, options)
	if err != nil {
		return nil, err
	}
	return result.Course, nil
}
