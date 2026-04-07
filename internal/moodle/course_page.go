package moodle

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var (
	courseSectionPattern      = regexp.MustCompile(`<li[^>]*id="section-\d+"[^>]*data-id="(\d+)"[^>]*data-sectionname="([^"]*)"[^>]*>`)
	courseSummaryPattern      = regexp.MustCompile(`(?is)<div[^>]*class="[^"]*summarytext[^"]*"[^>]*>(.*?)</div>`)
	courseLabelPattern        = regexp.MustCompile(`(?is)<li[^>]*class="[^"]*activity[^"]*label[^"]*modtype_label[^"]*"[^>]*>(.*?)</li>`)
	blockTagPattern           = regexp.MustCompile(`(?i)</?(?:p|div|section|article|ul|ol|li|h[1-6]|br)[^>]*>`)
	stripTagPattern           = regexp.MustCompile(`(?is)<[^>]+>`)
	whitespaceCollapsePattern = regexp.MustCompile(`[ \t\r\f\v]+`)
	blankLinePattern          = regexp.MustCompile(`\n{3,}`)
)

func (c *Client) FetchCoursePageReader(courseID string) (string, error) {
	htmlContent, err := c.FetchPage("/course/view.php?id=" + courseID)
	if err != nil {
		return "", err
	}
	return RenderCoursePageReader(htmlContent, courseID, c.BaseURL), nil
}

func RenderCoursePageReader(htmlContent string, courseID string, baseURL string) string {
	sections := findCourseSections(htmlContent)
	if len(sections) == 0 {
		return fallbackCoursePageReader(htmlContent, courseID, baseURL)
	}

	blocks := make([]string, 0, len(sections))
	for index, section := range sections {
		start := section.index
		end := len(htmlContent)
		if index < len(sections)-1 {
			end = sections[index+1].index
		}
		block := renderCourseSectionReader(htmlContent[start:end], courseID, baseURL, index, section.name)
		if block != "" {
			blocks = append(blocks, block)
		}
	}

	text := strings.TrimSpace(strings.Join(blocks, "\n\n"))
	if text == "" {
		return fallbackCoursePageReader(htmlContent, courseID, baseURL)
	}
	return text
}

func findCourseSections(htmlContent string) []sectionInfo {
	matches := courseSectionPattern.FindAllStringSubmatchIndex(htmlContent, -1)
	sections := make([]sectionInfo, 0, len(matches))
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}
		sections = append(sections, sectionInfo{
			index: match[0],
			id:    html.UnescapeString(htmlContent[match[2]:match[3]]),
			name:  html.UnescapeString(htmlContent[match[4]:match[5]]),
		})
	}
	return sections
}

func renderCourseSectionReader(sectionHTML string, courseID string, baseURL string, index int, sectionName string) string {
	lines := []string{readerSectionTitle(index, sectionName)}

	if summary := extractFirstReaderText(courseSummaryPattern, sectionHTML); summary != "" {
		lines = append(lines, summary)
	}

	seen := map[string]struct{}{}
	for _, label := range extractReaderTexts(courseLabelPattern, sectionHTML) {
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		lines = append(lines, label)
	}

	resources := make([]Resource, 0)
	extractResourcesFromHTML(sectionHTML, courseID, "", sectionName, baseURL, &resources, map[string]struct{}{})
	for _, resource := range resources {
		line := readerResourceLine(resource)
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		lines = append(lines, line)
	}

	if len(lines) == 1 {
		return ""
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func fallbackCoursePageReader(htmlContent string, courseID string, baseURL string) string {
	resources := ParseResources(htmlContent, courseID, baseURL)
	if len(resources) == 0 {
		return "No readable course content found."
	}
	lines := []string{"Course materials"}
	for _, resource := range resources {
		lines = append(lines, readerResourceLine(resource))
	}
	return strings.Join(lines, "\n")
}

func extractFirstReaderText(pattern *regexp.Regexp, htmlContent string) string {
	matches := pattern.FindAllStringSubmatch(htmlContent, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		if text := normalizeReaderHTML(match[1]); text != "" {
			return text
		}
	}
	return ""
}

func extractReaderTexts(pattern *regexp.Regexp, htmlContent string) []string {
	matches := pattern.FindAllStringSubmatch(htmlContent, -1)
	texts := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		if text := normalizeReaderHTML(match[1]); text != "" {
			texts = append(texts, text)
		}
	}
	return texts
}

func normalizeReaderHTML(fragment string) string {
	if strings.TrimSpace(fragment) == "" {
		return ""
	}
	text := strings.ReplaceAll(fragment, "&nbsp;", " ")
	text = blockTagPattern.ReplaceAllString(text, "\n")
	text = stripTagPattern.ReplaceAllString(text, " ")
	text = html.UnescapeString(text)

	rawLines := strings.Split(text, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		cleaned := strings.ReplaceAll(strings.TrimSpace(line), "\u00a0", " ")
		cleaned = whitespaceCollapsePattern.ReplaceAllString(cleaned, " ")
		if cleaned == "" {
			continue
		}
		if isReaderNoiseLine(cleaned) {
			continue
		}
		lines = append(lines, cleaned)
	}

	normalized := strings.Join(lines, "\n")
	normalized = blankLinePattern.ReplaceAllString(normalized, "\n\n")
	return strings.TrimSpace(normalized)
}

func isReaderNoiseLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	if strings.Trim(trimmed, "_") == "" {
		return true
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "aktivität ") && strings.HasSuffix(lower, " auswählen") {
		return true
	}
	if strings.HasPrefix(lower, "activity ") && strings.HasSuffix(lower, " select") {
		return true
	}
	return false
}

func readerSectionTitle(index int, sectionName string) string {
	title := strings.TrimSpace(sectionName)
	if title == "" {
		return fmt.Sprintf("Section %d", index+1)
	}
	return title
}

func readerResourceLine(resource Resource) string {
	parts := []string{resource.Name}
	if resource.FileType != "" {
		parts = append(parts, strings.ToUpper(resource.FileType))
	} else if resource.Type != "" {
		parts = append(parts, strings.ToUpper(resource.Type))
	}
	if resource.UploadedAt != "" {
		parts = append(parts, resource.UploadedAt)
	}
	return "- " + strings.Join(parts, " · ")
}
