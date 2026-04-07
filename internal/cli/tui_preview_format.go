package cli

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

var (
	previewSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#f8fafc"))
	previewHeadingStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7dd3fc"))
	previewLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#94a3b8"))
	previewTextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#cbd5e1"))
	previewBulletStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#64748b"))
	previewFileTypeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7dd3fc"))
	previewTimestampStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#94a3b8"))
	previewSeparatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#475569"))
)

func (m tuiModel) formatPreviewBody(node navNode, body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}

	if m.previewMode == previewModeMarkdown {
		return strings.TrimSpace(markdownPreviewBody(node, body))
	}

	lines := strings.Split(body, "\n")
	formatted := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			formatted = append(formatted, "")
			continue
		}
		formatted = append(formatted, m.formatPreviewLine(node, trimmed))
	}

	return strings.TrimSpace(strings.Join(formatted, "\n"))
}

func markdownPreviewBody(node navNode, body string) string {
	lines := strings.Split(strings.TrimSpace(body), "\n")
	formatted := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			formatted = append(formatted, "")
			continue
		}
		formatted = append(formatted, markdownPreviewLine(node, trimmed))
	}
	return strings.Join(formatted, "\n")
}

func (m tuiModel) formatPreviewLine(node navNode, line string) string {
	if strings.HasPrefix(line, "- ") {
		return formatPreviewBulletLine(line)
	}
	if label, value, ok := strings.Cut(line, ": "); ok && isPreviewMetadataLabel(label) {
		return previewLabelStyle.Render(label+":") + " " + previewTextStyle.Render(value)
	}
	if node.Kind == navNodeCourse {
		switch {
		case isCourseSectionLine(line):
			return previewSectionStyle.Render(line)
		case isCourseHeadingLine(line):
			return previewHeadingStyle.Render(line)
		default:
			return previewTextStyle.Render(line)
		}
	}
	if node.Resource != nil {
		return previewTextStyle.Render(line)
	}
	return previewTextStyle.Render(line)
}

func markdownPreviewLine(node navNode, line string) string {
	if strings.HasPrefix(line, "- ") {
		return markdownBulletLine(line)
	}
	if label, value, ok := strings.Cut(line, ": "); ok && isPreviewMetadataLabel(label) {
		return "**" + strings.TrimSpace(label) + ":** " + strings.TrimSpace(value)
	}
	if node.Kind == navNodeCourse {
		switch {
		case isCourseSectionLine(line):
			return "## " + line
		case isCourseHeadingLine(line):
			return "### " + line
		default:
			return line
		}
	}
	return line
}

func formatPreviewBulletLine(line string) string {
	parts := strings.Split(strings.TrimPrefix(line, "- "), " · ")
	if len(parts) == 0 {
		return previewBulletStyle.Render("- ") + previewTextStyle.Render(strings.TrimSpace(line))
	}

	rendered := previewBulletStyle.Render("- ") + previewTextStyle.Render(parts[0])
	for _, part := range parts[1:] {
		rendered += previewSeparatorStyle.Render(" · ")
		switch {
		case isPreviewTimestamp(part):
			rendered += previewTimestampStyle.Render(part)
		case isPreviewFileType(part):
			rendered += previewFileTypeStyle.Render(part)
		default:
			rendered += previewTextStyle.Render(part)
		}
	}
	return rendered
}

func markdownBulletLine(line string) string {
	parts := strings.Split(strings.TrimPrefix(line, "- "), " · ")
	if len(parts) == 0 {
		return line
	}

	out := "- " + parts[0]
	for _, part := range parts[1:] {
		switch {
		case isPreviewTimestamp(part):
			out += " · _" + part + "_"
		case isPreviewFileType(part):
			out += " · **" + strings.ToUpper(strings.TrimSpace(part)) + "**"
		default:
			out += " · " + part
		}
	}
	return out
}

func isCourseSectionLine(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	switch {
	case lower == "allgemeine informationen":
		return true
	case strings.HasPrefix(lower, "thema "):
		return true
	case strings.HasPrefix(lower, "section "):
		return true
	case strings.HasPrefix(lower, "week "):
		return true
	default:
		return false
	}
}

func isCourseHeadingLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	switch trimmed {
	case "Lernziele", "Präsenz", "Selbststudium", "Selbststudium (Nach- und Vorbereitung)", "General Information":
		return true
	}
	return len(trimmed) <= 48 && !strings.Contains(trimmed, ".") && unicode.IsUpper([]rune(trimmed)[0])
}

func isPreviewMetadataLabel(label string) bool {
	switch strings.TrimSpace(label) {
	case "Course", "Item", "Type", "Uploaded", "Lecture", "Time", "Room", "Matched course":
		return true
	default:
		return false
	}
}

func isPreviewTimestamp(value string) bool {
	value = strings.TrimSpace(value)
	return strings.Contains(value, "T") && (strings.HasSuffix(value, "Z") || strings.Contains(value, "+") || strings.Contains(value, "-"))
}

func isPreviewFileType(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PDF", "DOCX", "XLSX", "PPTX", "ZIP", "PNG", "JPG", "JPEG":
		return true
	default:
		return false
	}
}
