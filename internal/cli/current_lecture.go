package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/DotNaos/moodle-services/internal/config"
	"github.com/DotNaos/moodle-services/internal/moodle"
	"github.com/spf13/cobra"
)

type currentLectureCourse struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Matched string `json:"matched"`
}

type currentLectureResource struct {
	ID           string  `json:"id"`
	Label        string  `json:"label"`
	URL          string  `json:"url"`
	Kind         string  `json:"kind"`
	SectionTitle string  `json:"sectionTitle,omitempty"`
	FileType     string  `json:"fileType,omitempty"`
	UploadedAt   string  `json:"uploadedAt,omitempty"`
	Score        int     `json:"score"`
	LocalPath    *string `json:"localPath,omitempty"`
}

type currentLectureResult struct {
	Now       string                   `json:"now"`
	State     string                   `json:"state"`
	Event     *moodle.CalendarEvent    `json:"event,omitempty"`
	Course    *currentLectureCourse    `json:"course,omitempty"`
	Material  *currentLectureResource  `json:"material,omitempty"`
	Resources []currentLectureResource `json:"resources"`
	Warning   string                   `json:"warning,omitempty"`
}

type localCourseSnapshot struct {
	CourseID  string
	CourseDir string
}

var currentLectureWorkspace string
var currentLectureAt string

var currentLectureCmd = &cobra.Command{
	Use:   "current-lecture",
	Short: "Resolve the current lecture and its best matching materials",
	Long: "Resolve the current lecture from your timetable using the current local time.\n\n" +
		"The command returns the active lecture, or the next lecture later today if none is active.\n" +
		"It then matches the lecture to a Moodle course and ranks likely lecture materials.",
	Example: "  moodle list current-lecture\n" +
		"  moodle --json list current-lecture\n" +
		"  moodle list current-lecture --workspace /Users/oli/school\n" +
		"  moodle list current-lecture --at 2026-03-20T09:30:00+01:00",
	Args: cobra.NoArgs,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		now, err := resolveCurrentLectureTime()
		if err != nil {
			return err
		}

		cfg, err := config.LoadConfig(opts.ConfigPath)
		if err != nil {
			return err
		}
		if cfg.CalendarURL == "" {
			return fmt.Errorf("calendar URL not set. Run: moodle config set --calendar-url <url>")
		}

		client, err := ensureAuthenticatedClient()
		if err != nil {
			return err
		}

		result, err := buildCurrentLectureResult(client, cfg.CalendarURL, now, currentLectureWorkspace)
		if err != nil {
			return err
		}

		return writeCommandOutput(cmd, result, func(w io.Writer) error {
			return renderCurrentLectureText(w, result)
		})
	},
}

func init() {
	currentLectureCmd.Flags().StringVar(&currentLectureWorkspace, "workspace", "", "Optional workspace root for local file matching")
	currentLectureCmd.Flags().StringVar(&currentLectureAt, "at", "", "Override current time for testing (RFC3339)")
}

func resolveCurrentLectureTime() (time.Time, error) {
	return resolveLectureTimeAt(currentLectureAt)
}

func resolveLectureTimeAt(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Now(), nil
	}
	at, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --at timestamp, expected RFC3339: %w", err)
	}
	return at, nil
}

func buildCurrentLectureResult(client *moodle.Client, calendarURL string, now time.Time, workspace string) (currentLectureResult, error) {
	result := currentLectureResult{
		Now:       now.Format(time.RFC3339),
		State:     "none",
		Resources: []currentLectureResource{},
	}

	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	events, err := moodle.FetchCalendarEvents(calendarURL, startOfDay, endOfDay)
	if err != nil {
		return result, err
	}

	event := selectCurrentLectureEvent(events, now)
	if event == nil {
		return result, nil
	}

	result.Event = event
	if event.Start.After(now) {
		result.State = "next"
	} else {
		result.State = "current"
	}

	courses, err := client.FetchCourses()
	if err != nil {
		return result, err
	}
	course, matched := matchCourseForLecture(courses, event.Summary)
	if course == nil {
		return result, nil
	}
	result.Course = &currentLectureCourse{
		ID:      course.ID,
		Title:   course.Fullname,
		URL:     course.ViewURL,
		Matched: matched,
	}

	resources, _, err := client.FetchCourseResources(strconv.Itoa(course.ID))
	if err != nil {
		result.Warning = fmt.Sprintf("matched course but could not fetch resources: %v", err)
		return result, nil
	}

	localFiles := map[string]string{}
	if strings.TrimSpace(workspace) != "" {
		localFiles = findLocalCourseFiles(workspace, strconv.Itoa(course.ID))
	}

	ranked := rankCurrentLectureResources(resources, localFiles)
	result.Resources = ranked
	result.Material = selectBestCurrentLectureMaterial(ranked)

	return result, nil
}

func selectCurrentLectureEvent(events []moodle.CalendarEvent, now time.Time) *moodle.CalendarEvent {
	if len(events) == 0 {
		return nil
	}
	sorted := slices.Clone(events)
	slices.SortFunc(sorted, func(left, right moodle.CalendarEvent) int {
		if left.Start.Before(right.Start) {
			return -1
		}
		if left.Start.After(right.Start) {
			return 1
		}
		return strings.Compare(left.Summary, right.Summary)
	})
	for i := range sorted {
		if !sorted[i].Start.After(now) && !sorted[i].End.Before(now) {
			return &sorted[i]
		}
	}
	for i := range sorted {
		if sorted[i].Start.After(now) && sameLocalDay(sorted[i].Start, now) {
			return &sorted[i]
		}
	}
	return nil
}

func sameLocalDay(left, right time.Time) bool {
	l := left.In(right.Location())
	return l.Year() == right.Year() && l.Month() == right.Month() && l.Day() == right.Day()
}

func normalizeLectureValue(value string) string {
	replacer := strings.NewReplacer("ä", "a", "ö", "o", "ü", "u", "é", "e", "è", "e", "à", "a")
	normalized := strings.ToLower(replacer.Replace(value))
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\([^)]*\)`),
		regexp.MustCompile(`\b(fs|hs)\d{2}\b`),
		regexp.MustCompile(`\bcds[-\s]*\d+\b`),
		regexp.MustCompile(`\bkurs:\b`),
		regexp.MustCompile(`\bmoodle\s*@\s*fhgr\b`),
		regexp.MustCompile(`[^a-z0-9]+`),
	}
	for _, pattern := range patterns {
		normalized = pattern.ReplaceAllString(normalized, " ")
	}
	return strings.Join(strings.Fields(normalized), " ")
}

func scoreCourseMatch(summary string, course moodle.Course) int {
	target := normalizeLectureValue(summary)
	if target == "" {
		return 0
	}
	candidates := []string{course.Fullname, course.Shortname}
	best := 0
	for _, candidate := range candidates {
		normalized := normalizeLectureValue(candidate)
		if normalized == "" {
			continue
		}
		score := 0
		switch {
		case normalized == target:
			score = 100
		case strings.Contains(normalized, target) || strings.Contains(target, normalized):
			score = 75
		default:
			targetWords := strings.Fields(target)
			candidateWords := strings.Fields(normalized)
			shared := 0
			for _, word := range candidateWords {
				for _, targetWord := range targetWords {
					if word == targetWord {
						shared += 1
						break
					}
				}
			}
			score = shared * 10
		}
		if score > best {
			best = score
		}
	}
	return best
}

func matchCourseForLecture(courses []moodle.Course, summary string) (*moodle.Course, string) {
	type scored struct {
		course moodle.Course
		score  int
	}
	scoredCourses := make([]scored, 0, len(courses))
	for _, course := range courses {
		score := scoreCourseMatch(summary, course)
		if score > 0 {
			scoredCourses = append(scoredCourses, scored{course: course, score: score})
		}
	}
	if len(scoredCourses) == 0 {
		return nil, ""
	}
	slices.SortFunc(scoredCourses, func(left, right scored) int {
		if left.score > right.score {
			return -1
		}
		if left.score < right.score {
			return 1
		}
		return strings.Compare(left.course.Fullname, right.course.Fullname)
	})
	if len(scoredCourses) > 1 && scoredCourses[0].score == scoredCourses[1].score {
		return nil, ""
	}
	if scoredCourses[0].score < 20 {
		return nil, ""
	}
	return &scoredCourses[0].course, normalizeLectureValue(scoredCourses[0].course.Fullname)
}

func classifyCurrentLectureResource(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "lösung") || strings.Contains(lower, "loesung") || strings.Contains(lower, "solution"):
		return "solution"
	case strings.Contains(lower, "aufgaben") || strings.Contains(lower, "worksheet") || strings.Contains(lower, "exercise") || strings.Contains(lower, "blatt"):
		return "worksheet"
	case strings.Contains(lower, "folien") || strings.Contains(lower, "slides") || strings.Contains(lower, "vorlesung") || strings.Contains(lower, "lecture"):
		return "lecture"
	default:
		return "other"
	}
}

func rankCurrentLectureResources(resources []moodle.Resource, localFiles map[string]string) []currentLectureResource {
	ranked := make([]currentLectureResource, 0, len(resources))
	type sortableResource struct {
		resource      currentLectureResource
		resourceIndex int
		uploadedAt    time.Time
		hasUploadedAt bool
	}
	sortable := make([]sortableResource, 0, len(resources))
	for resourceIndex, resource := range resources {
		if resource.Type != "resource" {
			continue
		}
		localPath := localFiles[normalizeLectureValue(resource.Name)]
		candidate := currentLectureResource{
			ID:           resource.ID,
			Label:        resource.Name,
			URL:          resource.URL,
			Kind:         classifyCurrentLectureResource(resource.Name),
			SectionTitle: resource.SectionName,
			FileType:     resource.FileType,
			UploadedAt:   resource.UploadedAt,
		}
		if localPath != "" {
			candidate.LocalPath = &localPath
		}
		uploadedAt, hasUploadedAt := parseRFC3339Time(resource.UploadedAt)
		sortable = append(sortable, sortableResource{
			resource:      candidate,
			resourceIndex: resourceIndex,
			uploadedAt:    uploadedAt,
			hasUploadedAt: hasUploadedAt,
		})
	}
	slices.SortFunc(sortable, func(left, right sortableResource) int {
		if left.hasUploadedAt != right.hasUploadedAt {
			if left.hasUploadedAt {
				return -1
			}
			return 1
		}
		if left.hasUploadedAt && right.hasUploadedAt {
			if left.uploadedAt.After(right.uploadedAt) {
				return -1
			}
			if left.uploadedAt.Before(right.uploadedAt) {
				return 1
			}
		}
		if left.resourceIndex < right.resourceIndex {
			return -1
		}
		if left.resourceIndex > right.resourceIndex {
			return 1
		}
		return strings.Compare(left.resource.Label, right.resource.Label)
	})
	for index, entry := range sortable {
		entry.resource.Score = len(sortable) - index
		ranked = append(ranked, entry.resource)
	}
	return ranked
}

func selectBestCurrentLectureMaterial(resources []currentLectureResource) *currentLectureResource {
	if len(resources) == 0 {
		return nil
	}
	for _, candidate := range resources {
		if candidate.Kind == "lecture" && strings.EqualFold(candidate.FileType, "pdf") && candidate.UploadedAt != "" {
			copy := candidate
			return &copy
		}
	}
	for _, candidate := range resources {
		if candidate.Kind == "lecture" && strings.EqualFold(candidate.FileType, "pdf") {
			copy := candidate
			return &copy
		}
	}
	for _, candidate := range resources {
		if strings.EqualFold(candidate.FileType, "pdf") && candidate.UploadedAt != "" {
			copy := candidate
			return &copy
		}
	}
	for _, candidate := range resources {
		if strings.EqualFold(candidate.FileType, "pdf") {
			copy := candidate
			return &copy
		}
	}
	copy := resources[0]
	return &copy
}

func parseRFC3339Time(value string) (time.Time, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func findLocalCourseFiles(workspaceRoot string, courseID string) map[string]string {
	out := map[string]string{}
	for _, snapshot := range findLocalCourseSnapshots(workspaceRoot) {
		if snapshot.CourseID != courseID {
			continue
		}
		entries, err := os.ReadDir(snapshot.CourseDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			fullPath := filepath.Join(snapshot.CourseDir, entry.Name())
			if filepath.Ext(entry.Name()) != "" {
				stem := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
				out[normalizeLectureValue(stem)] = fullPath
			}
			out[normalizeLectureValue(entry.Name())] = fullPath
		}
	}
	return out
}

func findLocalCourseSnapshots(workspaceRoot string) []localCourseSnapshot {
	snapshots := []localCourseSnapshot{}
	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		return snapshots
	}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			if depthBelowRoot(root, path) > 4 {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "moodle-course.json" {
			return nil
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var course struct {
			CourseID string `json:"courseId"`
		}
		if err := json.Unmarshal(payload, &course); err != nil || course.CourseID == "" {
			return nil
		}
		snapshots = append(snapshots, localCourseSnapshot{
			CourseID:  course.CourseID,
			CourseDir: filepath.Dir(path),
		})
		return nil
	})
	return snapshots
}

func depthBelowRoot(root string, current string) int {
	relative, err := filepath.Rel(root, current)
	if err != nil || relative == "." {
		return 0
	}
	return len(strings.Split(relative, string(os.PathSeparator)))
}

func renderCurrentLectureText(w io.Writer, result currentLectureResult) error {
	if _, err := fmt.Fprintf(w, "Now: %s\n", result.Now); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "State: %s\n", result.State); err != nil {
		return err
	}
	if result.Event == nil {
		_, err := fmt.Fprintln(w, "No current or upcoming lecture found for today.")
		return err
	}
	if _, err := fmt.Fprintf(w, "Lecture: %s\n", result.Event.Summary); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Time: %s - %s\n", result.Event.Start.Format(time.RFC3339), result.Event.End.Format(time.RFC3339)); err != nil {
		return err
	}
	if result.Event.Location != "" {
		if _, err := fmt.Fprintf(w, "Room: %s\n", result.Event.Location); err != nil {
			return err
		}
	}
	if result.Course != nil {
		if _, err := fmt.Fprintf(w, "Course: %s [%d]\n", result.Course.Title, result.Course.ID); err != nil {
			return err
		}
	}
	if result.Material != nil {
		if _, err := fmt.Fprintf(w, "Best material: %s\n", result.Material.Label); err != nil {
			return err
		}
		if result.Material.LocalPath != nil {
			if _, err := fmt.Fprintf(w, "Local file: %s\n", *result.Material.LocalPath); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(w, "URL: %s\n", result.Material.URL); err != nil {
				return err
			}
		}
	}
	if result.Warning != "" {
		if _, err := fmt.Fprintf(w, "Warning: %s\n", result.Warning); err != nil {
			return err
		}
	}
	if len(result.Resources) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w, "Resources:"); err != nil {
		return err
	}
	for index, resource := range result.Resources {
		line := fmt.Sprintf("%d. %s [%s]", index+1, resource.Label, resource.Kind)
		if resource.LocalPath != nil {
			line += " local"
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

func buildSelectedCourseResult(client *moodle.Client, courseArg string, resourceArg string, workspace string, at string) (currentLectureResult, error) {
	now, err := resolveLectureTimeAt(at)
	if err != nil {
		return currentLectureResult{}, err
	}
	result := currentLectureResult{
		Now:       now.Format(time.RFC3339),
		State:     "none",
		Resources: []currentLectureResource{},
	}

	currentResult, err := resolveCurrentLectureResult(client, selectorOptions{Workspace: workspace, At: at})
	if err != nil {
		return result, err
	}

	courses, err := client.FetchCourses()
	if err != nil {
		return result, err
	}
	courseID, err := resolveCourseIDFromCoursesWithCurrent(courses, courseArg, currentResult.Course)
	if err != nil {
		return result, err
	}
	course, err := findCourseByID(courses, courseID)
	if err != nil {
		return result, err
	}

	matched := ""
	if currentResult.Course != nil && currentResult.Course.ID == course.ID {
		result.State = currentResult.State
		result.Event = currentResult.Event
		result.Warning = currentResult.Warning
		matched = currentResult.Course.Matched
	}
	result.Course = &currentLectureCourse{
		ID:      course.ID,
		Title:   course.Fullname,
		URL:     course.ViewURL,
		Matched: matched,
	}

	resources, _, err := client.FetchCourseResources(courseID)
	if err != nil {
		result.Warning = fmt.Sprintf("matched course but could not fetch resources: %v", err)
		return result, nil
	}

	localFiles := map[string]string{}
	if strings.TrimSpace(workspace) != "" {
		localFiles = findLocalCourseFiles(workspace, courseID)
	}
	if currentResult.Course != nil && currentResult.Course.ID == course.ID {
		result.Resources = rankCurrentLectureResources(resources, localFiles)
	} else {
		result.Resources = orderedCourseResources(resources, localFiles)
	}

	selected, err := resolveResourceWithCurrentOrder(resources, resourceArg, currentMaterialIDForCourse(courseID, currentResult), resourceIDs(result.Resources))
	if err != nil {
		return result, err
	}
	result.Material = resourceToCurrentLectureResource(*selected, result.Resources, localFiles)
	return result, nil
}

func currentMaterialIDForCourse(courseID string, result currentLectureResult) string {
	if result.Course == nil || result.Material == nil {
		return ""
	}
	if fmt.Sprintf("%d", result.Course.ID) != strings.TrimSpace(courseID) {
		return ""
	}
	return result.Material.ID
}

func resourceToCurrentLectureResource(resource moodle.Resource, ranked []currentLectureResource, localFiles map[string]string) *currentLectureResource {
	for _, candidate := range ranked {
		if candidate.ID == resource.ID {
			copy := candidate
			return &copy
		}
	}
	localPath := localFiles[normalizeLectureValue(resource.Name)]
	result := currentLectureResource{
		ID:           resource.ID,
		Label:        resource.Name,
		URL:          resource.URL,
		Kind:         classifyCurrentLectureResource(resource.Name),
		SectionTitle: resource.SectionName,
		FileType:     resource.FileType,
		UploadedAt:   resource.UploadedAt,
	}
	if localPath != "" {
		result.LocalPath = &localPath
	}
	return &result
}

func orderedCourseResources(resources []moodle.Resource, localFiles map[string]string) []currentLectureResource {
	ordered := make([]currentLectureResource, 0, len(resources))
	score := len(fileResources(resources))
	for _, resource := range resources {
		if resource.Type != "resource" {
			continue
		}
		localPath := localFiles[normalizeLectureValue(resource.Name)]
		candidate := currentLectureResource{
			ID:           resource.ID,
			Label:        resource.Name,
			URL:          resource.URL,
			Kind:         classifyCurrentLectureResource(resource.Name),
			SectionTitle: resource.SectionName,
			FileType:     resource.FileType,
			UploadedAt:   resource.UploadedAt,
			Score:        score,
		}
		if localPath != "" {
			candidate.LocalPath = &localPath
		}
		ordered = append(ordered, candidate)
		score -= 1
	}
	return ordered
}
