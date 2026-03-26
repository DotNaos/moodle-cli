package moodle

import (
	"html"
	"regexp"
	"strings"
	"time"
)

type Resource struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Type        string `json:"type"` // resource|folder
	CourseID    string `json:"courseId"`
	SectionID   string `json:"sectionId,omitempty"`
	SectionName string `json:"sectionName,omitempty"`
	FileType    string `json:"fileType,omitempty"`
	UploadedAt  string `json:"uploadedAt,omitempty"`
}

type sectionInfo struct {
	index int
	id    string
	name  string
}

func ParseResources(htmlContent string, courseID string, baseURL string) []Resource {
	resources := make([]Resource, 0)
	seen := map[string]struct{}{}

	sectionRe := regexp.MustCompile(`<li[^>]*id="section-\d+"[^>]*data-id="(\d+)"[^>]*data-sectionname="([^"]*)"[^>]*>`)
	sectionMatches := sectionRe.FindAllStringSubmatchIndex(htmlContent, -1)

	sections := make([]sectionInfo, 0, len(sectionMatches))
	for _, match := range sectionMatches {
		if len(match) < 6 {
			continue
		}
		sections = append(sections, sectionInfo{
			index: match[0],
			id:    html.UnescapeString(htmlContent[match[2]:match[3]]),
			name:  html.UnescapeString(htmlContent[match[4]:match[5]]),
		})
	}

	if len(sections) == 0 {
		extractResourcesFromHTML(htmlContent, courseID, "", "General", baseURL, &resources, seen)
		return resources
	}

	for i := 0; i < len(sections); i++ {
		start := sections[i].index
		end := len(htmlContent)
		if i < len(sections)-1 {
			end = sections[i+1].index
		}
		sectionHTML := htmlContent[start:end]
		extractResourcesFromHTML(sectionHTML, courseID, sections[i].id, sections[i].name, baseURL, &resources, seen)
	}

	return resources
}

func extractResourcesFromHTML(htmlContent string, courseID string, sectionID string, sectionName string, baseURL string, resources *[]Resource, seen map[string]struct{}) {
	activityItemRe := regexp.MustCompile(`(?s)<li[^>]*class="[^"]*activity[^"]*resource[^"]*modtype_resource[^"]*"[^>]*>(.*?)</li>`)
	idRe := regexp.MustCompile(`/mod/resource/view\.php\?id=(\d+)`)
	activityNameRe := regexp.MustCompile(`data-activityname="([^"]+)"`)
	instanceNameRe := regexp.MustCompile(`<span[^>]*class="[^"]*instancename[^"]*"[^>]*>([^<]+)`)
	iconRe := regexp.MustCompile(`src="[^"]*/f/([a-z0-9]+)`)
	badgeRe := regexp.MustCompile(`class="activitybadge[^"]*"[^>]*>\s*([^<]+)`)
	detailsRe := regexp.MustCompile(`class="resourcelinkdetails">([^<]+)`)

	for _, match := range activityItemRe.FindAllStringSubmatch(htmlContent, -1) {
		liContent := match[1]
		idMatch := idRe.FindStringSubmatch(liContent)
		if len(idMatch) < 2 {
			continue
		}
		id := idMatch[1]
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		name := ""
		if m := activityNameRe.FindStringSubmatch(liContent); len(m) > 1 {
			name = strings.TrimSpace(html.UnescapeString(m[1]))
		}
		if name == "" {
			if m := instanceNameRe.FindStringSubmatch(liContent); len(m) > 1 {
				name = strings.TrimSpace(html.UnescapeString(m[1]))
			}
		}
		if name == "" {
			continue
		}

		fileType := "pdf"
		if m := iconRe.FindStringSubmatch(liContent); len(m) > 1 {
			fileType = inferFileTypeFromIcon(m[1])
		}
		if m := badgeRe.FindStringSubmatch(liContent); len(m) > 1 {
			badge := strings.ToLower(strings.TrimSpace(m[1]))
			switch {
			case badge == "pdf":
				fileType = "pdf"
			case strings.Contains(badge, "word"):
				fileType = "docx"
			case strings.Contains(badge, "excel") || strings.Contains(badge, "spreadsheet"):
				fileType = "xlsx"
			case strings.Contains(badge, "powerpoint") || strings.Contains(badge, "presentation"):
				fileType = "pptx"
			}
		}

		uploadedAt := ""
		if m := detailsRe.FindStringSubmatch(liContent); len(m) > 1 {
			uploadedAt = parseUploadedAt(m[1])
		}

		*resources = append(*resources, Resource{
			ID:          id,
			Name:        name,
			URL:         strings.TrimRight(baseURL, "/") + "/mod/resource/view.php?id=" + id + "&redirect=1",
			Type:        "resource",
			CourseID:    courseID,
			SectionID:   sectionID,
			SectionName: sectionName,
			FileType:    fileType,
			UploadedAt:  uploadedAt,
		})
	}

	folderRe := regexp.MustCompile(`(?s)<li[^>]*class="[^"]*activity[^"]*folder[^"]*modtype_folder[^"]*"[^>]*>(.*?)</li>`)
	folderIDRe := regexp.MustCompile(`/mod/folder/view\.php\?id=(\d+)`)

	for _, match := range folderRe.FindAllStringSubmatch(htmlContent, -1) {
		liContent := match[1]
		idMatch := folderIDRe.FindStringSubmatch(liContent)
		if len(idMatch) < 2 {
			continue
		}
		id := "folder-" + idMatch[1]
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		name := ""
		if m := activityNameRe.FindStringSubmatch(liContent); len(m) > 1 {
			name = strings.TrimSpace(html.UnescapeString(m[1]))
		}
		if name == "" {
			if m := instanceNameRe.FindStringSubmatch(liContent); len(m) > 1 {
				name = strings.TrimSpace(html.UnescapeString(m[1]))
			}
		}
		if name == "" {
			continue
		}

		*resources = append(*resources, Resource{
			ID:          id,
			Name:        name,
			URL:         strings.TrimRight(baseURL, "/") + "/mod/folder/view.php?id=" + idMatch[1],
			Type:        "folder",
			CourseID:    courseID,
			SectionID:   sectionID,
			SectionName: sectionName,
		})
	}
}

func inferFileTypeFromIcon(icon string) string {
	lower := strings.ToLower(icon)
	switch {
	case strings.Contains(lower, "pdf"):
		return "pdf"
	case strings.Contains(lower, "word") || strings.Contains(lower, "document"):
		return "docx"
	case strings.Contains(lower, "excel") || strings.Contains(lower, "spreadsheet"):
		return "xlsx"
	case strings.Contains(lower, "powerpoint") || strings.Contains(lower, "presentation"):
		return "pptx"
	default:
		return "pdf"
	}
}

func parseUploadedAt(details string) string {
	normalized := html.UnescapeString(strings.TrimSpace(details))
	timestampRe := regexp.MustCompile(`(?i)(?:hochgeladen|uploaded)\s+(\d{1,2}\.\d{1,2}\.\d{4}\s+\d{1,2}:\d{2})`)
	match := timestampRe.FindStringSubmatch(normalized)
	if len(match) != 2 {
		return ""
	}
	parsed, err := parseSwissTimestamp(match[1])
	if err != nil {
		return ""
	}
	return parsed.Format(time.RFC3339)
}
