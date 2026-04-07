package cli

import (
	"regexp"
	"strings"

	"github.com/DotNaos/moodle-cli/internal/moodle"
)

func (s *navService) displayCourseName(course moodle.Course) string {
	patterns := []*regexp.Regexp(nil)
	if s.client != nil {
		patterns = s.client.School.CourseNamePatterns
	}
	return moodle.DisplayCourseName(course.Fullname, patterns)
}

func (s *navService) coursePreview(node navNode) string {
	courseText := strings.TrimSpace(s.fetchCourseReaderText(node))
	if courseText != "" {
		return courseText
	}

	return "Browse sections in Moodle order."
}

func (s *navService) fetchCourseReaderText(node navNode) string {
	if node.CourseID == "" || s.client == nil {
		return ""
	}
	if s.coursePagePreview == nil {
		s.coursePagePreview = map[string]string{}
	}
	if text, ok := s.coursePagePreview[node.CourseID]; ok {
		return text
	}
	text, err := s.client.FetchCoursePageReader(node.CourseID)
	if err != nil {
		return err.Error()
	}
	text = strings.TrimSpace(text)
	if text != "" {
		s.coursePagePreview[node.CourseID] = text
	}
	return text
}
