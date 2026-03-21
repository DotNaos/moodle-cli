package cli

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/DotNaos/moodle-cli/internal/config"
	"github.com/DotNaos/moodle-cli/internal/moodle"
)

type navNodeKind string

const (
	navNodeHome              navNodeKind = "home"
	navNodeCurrent           navNodeKind = "current"
	navNodeToday             navNodeKind = "today"
	navNodeTimetable         navNodeKind = "timetable"
	navNodeWeek              navNodeKind = "week"
	navNodeSemesters         navNodeKind = "semesters"
	navNodeSemester          navNodeKind = "semester"
	navNodeCoursesCollection navNodeKind = "courses"
	navNodeCourse            navNodeKind = "course"
	navNodeEvent             navNodeKind = "event"
	navNodeSections          navNodeKind = "sections"
	navNodeSection           navNodeKind = "section"
	navNodeItems             navNodeKind = "items"
	navNodeResource          navNodeKind = "resource"
)

type navNode struct {
	Key            string
	Kind           navNodeKind
	Segment        string
	Title          string
	Subtitle       string
	CourseID       string
	Semester       string
	SectionID      string
	SectionName    string
	UseCurrentSort bool
	Course         *moodle.Course
	Event          *moodle.CalendarEvent
	Resource       *moodle.Resource
	Openable       bool
	Printable      bool
	PreviewText    string
}

type navSummary struct {
	Path      string            `json:"path"`
	Kind      string            `json:"kind"`
	Title     string            `json:"title"`
	Subtitle  string            `json:"subtitle,omitempty"`
	Preview   string            `json:"preview,omitempty"`
	Openable  bool              `json:"openable"`
	Printable bool              `json:"printable"`
	Children  []navChildSummary `json:"children,omitempty"`
}

type navChildSummary struct {
	Index     int    `json:"index"`
	Kind      string `json:"kind"`
	Segment   string `json:"segment"`
	Title     string `json:"title"`
	Subtitle  string `json:"subtitle,omitempty"`
	Openable  bool   `json:"openable"`
	Printable bool   `json:"printable"`
}

type navService struct {
	client          *moodle.Client
	options         selectorOptions
	now             time.Time
	calendarURL     string
	calendarLoaded  bool
	courses         []moodle.Course
	coursesLoaded   bool
	current         currentLectureResult
	currentLoaded   bool
	todayEvents     []moodle.CalendarEvent
	todayLoaded     bool
	courseResources map[string][]moodle.Resource
}

func newNavService(client *moodle.Client, options selectorOptions) (*navService, error) {
	now, err := resolveLectureTimeAt(options.At)
	if err != nil {
		return nil, err
	}
	return &navService{
		client:          client,
		options:         options,
		now:             now,
		courseResources: map[string][]moodle.Resource{},
	}, nil
}

func (s *navService) Root() navNode {
	return navNode{
		Key:     "root",
		Kind:    navNodeHome,
		Segment: "",
		Title:   "Moodle",
	}
}

func (s *navService) ResolvePath(path string) (navNode, error) {
	current := s.Root()
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return current, nil
	}
	segments := strings.Split(trimmed, "/")
	for _, segment := range segments {
		if current.Kind == navNodeSemester && strings.EqualFold(segment, "courses") {
			continue
		}
		if current.Kind == navNodeCourse && strings.EqualFold(segment, "sections") {
			continue
		}
		if current.Kind == navNodeSection && strings.EqualFold(segment, "items") {
			continue
		}
		children, err := s.Children(current)
		if err != nil {
			return navNode{}, err
		}
		if isCurrentSelector(segment) && len(children) > 0 {
			switch current.Kind {
			case navNodeCurrent:
				current = children[0]
				continue
			case navNodeItems:
				if current.UseCurrentSort {
					selected := children[0]
					if result, err := s.currentResult(); err == nil && result.Material != nil {
						for _, child := range children {
							if child.Resource != nil && child.Resource.ID == result.Material.ID {
								selected = child
								break
							}
						}
					}
					current = selected
					continue
				}
			}
		}
		next, err := matchNavChild(children, segment)
		if err != nil {
			return navNode{}, fmt.Errorf("path segment %q: %w", segment, err)
		}
		current = next
	}
	return current, nil
}

func (s *navService) Summary(path string, node navNode) (navSummary, error) {
	children, err := s.Children(node)
	if err != nil {
		return navSummary{}, err
	}
	out := navSummary{
		Path:      path,
		Kind:      string(node.Kind),
		Title:     node.Title,
		Subtitle:  node.Subtitle,
		Preview:   s.Preview(node),
		Openable:  node.Openable,
		Printable: node.Printable,
		Children:  make([]navChildSummary, 0, len(children)),
	}
	for index, child := range children {
		out.Children = append(out.Children, navChildSummary{
			Index:     index + 1,
			Kind:      string(child.Kind),
			Segment:   child.Segment,
			Title:     child.Title,
			Subtitle:  child.Subtitle,
			Openable:  child.Openable,
			Printable: child.Printable,
		})
	}
	return out, nil
}

func (s *navService) Children(node navNode) ([]navNode, error) {
	switch node.Kind {
	case navNodeHome:
		return s.homeChildren()
	case navNodeCurrent:
		return s.currentChildren()
	case navNodeToday:
		return s.todayChildren()
	case navNodeTimetable:
		return s.timetableWeekChildren()
	case navNodeWeek:
		return s.weekChildren(node)
	case navNodeSemesters:
		return s.semesterChildren()
	case navNodeSemester:
		return s.courseListChildren(node.Semester)
	case navNodeCourse:
		return s.sectionChildren(node)
	case navNodeEvent:
		return s.eventChildren(node)
	case navNodeSections:
		return s.sectionChildren(node)
	case navNodeSection:
		return s.itemChildren(navNode{
			Key:         node.Key + ":items",
			Kind:        navNodeItems,
			Segment:     "items",
			Title:       "Items",
			CourseID:    node.CourseID,
			Course:      node.Course,
			SectionID:   node.SectionID,
			SectionName: node.SectionName,
		})
	case navNodeItems:
		return s.itemChildren(node)
	default:
		return nil, nil
	}
}

func (s *navService) Preview(node navNode) string {
	switch node.Kind {
	case navNodeHome:
		return "Current jumps to the active lecture. Today shows today’s timetable. Semesters is full course browsing."
	case navNodeCurrent:
		result, err := s.currentResult()
		if err != nil {
			return err.Error()
		}
		if result.Event == nil {
			return "No current or upcoming lecture found for today."
		}
		lines := []string{
			fmt.Sprintf("Lecture: %s", result.Event.Summary),
			fmt.Sprintf("Time: %s - %s", result.Event.Start.Format("15:04"), result.Event.End.Format("15:04")),
		}
		if result.Event.Location != "" {
			lines = append(lines, fmt.Sprintf("Room: %s", result.Event.Location))
		}
		if result.Course != nil {
			lines = append(lines, fmt.Sprintf("Course: %s", result.Course.Title))
		}
		if result.Material != nil {
			lines = append(lines, fmt.Sprintf("Current material: %s", result.Material.Label))
		}
		return strings.Join(lines, "\n")
	case navNodeToday:
		children, err := s.todayChildren()
		if err != nil {
			return err.Error()
		}
		return previewFromChildren(children)
	case navNodeTimetable:
		return "Browse timetable weeks."
	case navNodeWeek:
		if node.PreviewText != "" {
			return node.PreviewText
		}
		children, err := s.weekChildren(node)
		if err != nil {
			return err.Error()
		}
		return previewFromChildren(children)
	case navNodeSemesters:
		return "Browse Moodle courses grouped by semester."
	case navNodeSemester:
		return fmt.Sprintf("Semester %s", node.Title)
	case navNodeCoursesCollection:
		return fmt.Sprintf("Courses in %s", node.Semester)
	case navNodeCourse:
		lines := []string{}
		if node.Course != nil {
			lines = append(lines, fmt.Sprintf("Course: %s", node.Course.Fullname))
		}
		lines = append(lines, "Browse sections in Moodle order.")
		return strings.Join(lines, "\n")
	case navNodeEvent:
		if node.PreviewText != "" {
			return node.PreviewText
		}
		lines := []string{
			fmt.Sprintf("Lecture: %s", node.Title),
		}
		if node.Event != nil {
			lines = append(lines, fmt.Sprintf("Time: %s - %s", node.Event.Start.Format("15:04"), node.Event.End.Format("15:04")))
			if node.Event.Location != "" {
				lines = append(lines, fmt.Sprintf("Room: %s", node.Event.Location))
			}
		}
		if node.Course != nil {
			lines = append(lines, fmt.Sprintf("Matched course: %s", node.Course.Fullname))
		} else {
			lines = append(lines, "No Moodle course match.")
		}
		return strings.Join(lines, "\n")
	case navNodeSections:
		return "Sections in Moodle order."
	case navNodeSection:
		return fmt.Sprintf("Section: %s", node.Title)
	case navNodeItems:
		if node.UseCurrentSort {
			return "Items ordered for the current lecture: newest relevant file first."
		}
		return "Items in Moodle order."
	case navNodeResource:
		lines := []string{fmt.Sprintf("Item: %s", node.Title)}
		if node.Subtitle != "" {
			lines = append(lines, node.Subtitle)
		}
		if node.Resource != nil {
			if node.Resource.FileType != "" {
				lines = append(lines, fmt.Sprintf("Type: %s", node.Resource.FileType))
			}
			if node.Resource.UploadedAt != "" {
				lines = append(lines, fmt.Sprintf("Uploaded: %s", node.Resource.UploadedAt))
			}
		}
		return strings.Join(lines, "\n")
	default:
		return ""
	}
}

func (s *navService) Open(node navNode) error {
	switch {
	case node.Resource != nil:
		return openResolvedResource(s.client, *node.Resource)
	case node.Course != nil && strings.TrimSpace(node.Course.ViewURL) != "":
		return openURL(node.Course.ViewURL)
	default:
		return fmt.Errorf("node %q cannot be opened", node.Title)
	}
}

func (s *navService) Print(node navNode) (string, error) {
	if node.Resource == nil {
		return "", fmt.Errorf("node %q cannot be printed", node.Title)
	}
	if node.Resource.Type != "resource" {
		return "", fmt.Errorf("resource %q is not a printable file", node.Title)
	}
	return renderDownloadedResource(s.client, node.Resource.URL, node.Resource.FileType, false)
}

func (s *navService) Download(node navNode, outputPath string) (string, error) {
	if node.Resource == nil {
		return "", fmt.Errorf("node %q cannot be downloaded", node.Title)
	}
	if node.Resource.Type != "resource" {
		return "", fmt.Errorf("resource %q is not a file", node.Title)
	}
	path, err := resolveOutputPath(outputPath, *node.Resource)
	if err != nil {
		return "", err
	}
	if err := downloadResourceToFile(s.client, *node.Resource, path); err != nil {
		return "", err
	}
	return path, nil
}

func (s *navService) homeChildren() ([]navNode, error) {
	return []navNode{
		{Key: "current", Kind: navNodeCurrent, Segment: "current", Title: "Current"},
		{Key: "today", Kind: navNodeToday, Segment: "today", Title: "Today"},
		{Key: "semesters", Kind: navNodeSemesters, Segment: "semesters", Title: "Semesters"},
		{Key: "timetable", Kind: navNodeTimetable, Segment: "timetable", Title: "Timetable"},
	}, nil
}

func (s *navService) currentChildren() ([]navNode, error) {
	result, err := s.currentResult()
	if err != nil {
		return nil, err
	}
	if result.Course == nil {
		return nil, nil
	}
	course, err := s.courseByID(fmt.Sprintf("%d", result.Course.ID))
	if err != nil {
		return nil, err
	}
	children := make([]navNode, 0, 3)
	if material := s.currentMaterialNode(course); material != nil {
		children = append(children, *material)
	}
	children = append(children,
		navNode{
			Key:            "current-items",
			Kind:           navNodeItems,
			Segment:        "items",
			Title:          "Items",
			CourseID:       fmt.Sprintf("%d", course.ID),
			Course:         course,
			UseCurrentSort: true,
		},
		navNode{
			Key:      "current-sections",
			Kind:     navNodeSections,
			Segment:  "sections",
			Title:    "Sections",
			CourseID: fmt.Sprintf("%d", course.ID),
			Course:   course,
		},
	)
	return children, nil
}

func (s *navService) currentMaterialNode(course *moodle.Course) *navNode {
	result, err := s.currentResult()
	if err != nil || result.Material == nil {
		return nil
	}
	resource, err := s.resourceByID(fmt.Sprintf("%d", course.ID), result.Material.ID)
	if err != nil {
		resource = &moodle.Resource{
			ID:          result.Material.ID,
			Name:        result.Material.Label,
			URL:         result.Material.URL,
			Type:        "resource",
			CourseID:    fmt.Sprintf("%d", course.ID),
			SectionName: result.Material.SectionTitle,
			FileType:    result.Material.FileType,
			UploadedAt:  result.Material.UploadedAt,
		}
	}
	return &navNode{
		Key:       "current-material:" + resource.ID,
		Kind:      navNodeResource,
		Segment:   "current",
		Title:     resource.Name,
		Subtitle:  buildResourceSubtitle(*resource),
		CourseID:  fmt.Sprintf("%d", course.ID),
		Course:    course,
		Resource:  resource,
		Openable:  true,
		Printable: resource.Type == "resource",
	}
}

func (s *navService) todayChildren() ([]navNode, error) {
	events, err := s.todayLectureEvents()
	if err != nil {
		return nil, err
	}
	return s.groupCalendarEvents(events)
}

func (s *navService) timetableWeekChildren() ([]navNode, error) {
	weeks := make([]navNode, 0, 9)
	now := s.now
	year, week := now.ISOWeek()
	for offset := -1; offset <= 7; offset++ {
		start := isoWeekStart(year, week).AddDate(0, 0, offset*7)
		end := start.AddDate(0, 0, 7)
		label := fmt.Sprintf("%d-W%02d", start.Year(), weekNumber(start))
		subtitle := fmt.Sprintf("%s - %s", start.Format("02 Jan"), end.AddDate(0, 0, -1).Format("02 Jan"))
		weeks = append(weeks, navNode{
			Key:         "week:" + label,
			Kind:        navNodeWeek,
			Segment:     strings.ToLower(label),
			Title:       label,
			Subtitle:    subtitle,
			PreviewText: subtitle,
			Event:       &moodle.CalendarEvent{Start: start, End: end},
		})
	}
	return weeks, nil
}

func (s *navService) weekChildren(node navNode) ([]navNode, error) {
	calendarURL, err := s.fetchCalendarURL()
	if err != nil {
		return nil, err
	}
	start := node.Event.Start
	end := node.Event.End
	events, err := moodle.FetchCalendarEvents(calendarURL, start, end)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(events, func(left, right moodle.CalendarEvent) int {
		if left.Start.Before(right.Start) {
			return -1
		}
		if left.Start.After(right.Start) {
			return 1
		}
		return strings.Compare(left.Summary, right.Summary)
	})
	return s.groupCalendarEvents(events)
}

func (s *navService) groupCalendarEvents(events []moodle.CalendarEvent) ([]navNode, error) {
	courses, err := s.fetchCourses()
	if err != nil {
		return nil, err
	}
	type groupedEvent struct {
		summary   string
		key       string
		course    *moodle.Course
		events    []moodle.CalendarEvent
		locations []string
	}
	grouped := make([]groupedEvent, 0, len(events))
	appendLocation := func(locations []string, value string) []string {
		if strings.TrimSpace(value) == "" {
			return locations
		}
		for _, existing := range locations {
			if existing == value {
				return locations
			}
		}
		return append(locations, value)
	}
	for _, event := range events {
		course, _ := matchCourseForLecture(courses, event.Summary)
		key := event.Summary
		if course != nil {
			key = fmt.Sprintf("%d:%s", course.ID, event.Summary)
		}
		if len(grouped) > 0 && grouped[len(grouped)-1].key == key {
			last := &grouped[len(grouped)-1]
			last.events = append(last.events, event)
			last.locations = appendLocation(last.locations, event.Location)
			continue
		}
		grouped = append(grouped, groupedEvent{
			summary:   event.Summary,
			key:       key,
			course:    course,
			events:    []moodle.CalendarEvent{event},
			locations: appendLocation(nil, event.Location),
		})
	}
	out := make([]navNode, 0, len(grouped))
	for index, group := range grouped {
		timeframes := make([]string, 0, len(group.events))
		for _, event := range group.events {
			timeframes = append(timeframes, fmt.Sprintf("%s-%s", event.Start.Format("15:04"), event.End.Format("15:04")))
		}
		subtitle := strings.Join(timeframes, " · ")
		if len(group.locations) > 0 {
			subtitle += " · " + strings.Join(group.locations, ", ")
		}
		preview := group.summary + "\n" + strings.Join(timeframes, "\n")
		if len(group.locations) > 0 {
			preview += "\n" + strings.Join(group.locations, ", ")
		}
		node := navNode{
			Key:         fmt.Sprintf("event:%d", index),
			Kind:        navNodeEvent,
			Segment:     slugNavSegment(group.summary),
			Title:       group.summary,
			Subtitle:    subtitle,
			Event:       &group.events[0],
			PreviewText: preview,
		}
		if group.course != nil {
			copy := *group.course
			node.Course = &copy
			node.CourseID = fmt.Sprintf("%d", group.course.ID)
			node.PreviewText += "\n" + group.course.Fullname
		}
		out = append(out, node)
	}
	return out, nil
}

func (s *navService) semesterChildren() ([]navNode, error) {
	courses, err := s.fetchCourses()
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	semesters := make([]string, 0)
	for _, course := range courses {
		semester := extractCourseSemester(course)
		if _, ok := seen[semester]; ok {
			continue
		}
		seen[semester] = struct{}{}
		semesters = append(semesters, semester)
	}
	slices.SortFunc(semesters, compareSemesterLabels)
	out := make([]navNode, 0, len(semesters))
	for _, semester := range semesters {
		out = append(out, navNode{
			Key:      "semester:" + semester,
			Kind:     navNodeSemester,
			Segment:  semester,
			Title:    semester,
			Semester: semester,
		})
	}
	return out, nil
}

func (s *navService) courseListChildren(semester string) ([]navNode, error) {
	courses, err := s.fetchCourses()
	if err != nil {
		return nil, err
	}
	out := make([]navNode, 0)
	for _, course := range courses {
		if extractCourseSemester(course) != semester {
			continue
		}
		copy := course
		out = append(out, navNode{
			Key:      fmt.Sprintf("course:%d", course.ID),
			Kind:     navNodeCourse,
			Segment:  slugNavSegment(course.Fullname),
			Title:    course.Fullname,
			Subtitle: course.Shortname,
			CourseID: fmt.Sprintf("%d", course.ID),
			Course:   &copy,
			Openable: true,
		})
	}
	return out, nil
}

func (s *navService) eventChildren(node navNode) ([]navNode, error) {
	if node.Course == nil {
		return nil, nil
	}
	return s.sectionChildren(node)
}

func (s *navService) sectionChildren(node navNode) ([]navNode, error) {
	resources, err := s.fetchCourseResources(node.CourseID)
	if err != nil {
		return nil, err
	}
	type section struct {
		id   string
		name string
	}
	seen := map[string]struct{}{}
	ordered := make([]section, 0)
	for _, resource := range resources {
		key := resource.SectionID + "\x00" + resource.SectionName
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		ordered = append(ordered, section{id: resource.SectionID, name: resource.SectionName})
	}
	out := make([]navNode, 0, len(ordered))
	for index, section := range ordered {
		title := strings.TrimSpace(section.name)
		if title == "" {
			title = fmt.Sprintf("Section %d", index+1)
		}
		out = append(out, navNode{
			Key:         fmt.Sprintf("%s:section:%d", node.Key, index),
			Kind:        navNodeSection,
			Segment:     slugNavSegment(title),
			Title:       title,
			CourseID:    node.CourseID,
			Course:      node.Course,
			SectionID:   section.id,
			SectionName: section.name,
		})
	}
	return out, nil
}

func (s *navService) itemChildren(node navNode) ([]navNode, error) {
	resources, err := s.fetchCourseResources(node.CourseID)
	if err != nil {
		return nil, err
	}
	if node.SectionID != "" || node.SectionName != "" {
		resources = filterResourcesBySection(resources, node.SectionID, node.SectionName)
	}
	if node.UseCurrentSort {
		resources = s.currentSortedResources(node.CourseID, resources)
	}
	out := make([]navNode, 0, len(resources))
	for index, resource := range resources {
		copy := resource
		out = append(out, navNode{
			Key:       fmt.Sprintf("%s:item:%d:%s", node.Key, index, resource.ID),
			Kind:      navNodeResource,
			Segment:   slugNavSegment(resource.Name),
			Title:     resource.Name,
			Subtitle:  buildResourceSubtitle(resource),
			CourseID:  node.CourseID,
			Course:    node.Course,
			Resource:  &copy,
			Openable:  true,
			Printable: resource.Type == "resource",
		})
	}
	return out, nil
}

func (s *navService) currentSortedResources(courseID string, resources []moodle.Resource) []moodle.Resource {
	result, err := s.currentResult()
	if err != nil || result.Course == nil || fmt.Sprintf("%d", result.Course.ID) != courseID {
		return resources
	}
	orderedIDs := resourceIDs(result.Resources)
	orderedFiles := orderedFileResources(resources, orderedIDs)
	seen := make(map[string]struct{}, len(orderedFiles))
	out := make([]moodle.Resource, 0, len(resources))
	for _, resource := range orderedFiles {
		out = append(out, resource)
		seen[resource.ID] = struct{}{}
	}
	for _, resource := range resources {
		if _, ok := seen[resource.ID]; ok {
			continue
		}
		out = append(out, resource)
	}
	return out
}

func (s *navService) currentResult() (currentLectureResult, error) {
	if s.currentLoaded {
		return s.current, nil
	}
	calendarURL, err := s.fetchCalendarURL()
	if err != nil {
		return currentLectureResult{}, err
	}
	result, err := buildCurrentLectureResult(s.client, calendarURL, s.now, s.options.Workspace)
	if err != nil {
		return currentLectureResult{}, err
	}
	s.current = result
	s.currentLoaded = true
	return result, nil
}

func (s *navService) todayLectureEvents() ([]moodle.CalendarEvent, error) {
	if s.todayLoaded {
		return slices.Clone(s.todayEvents), nil
	}
	calendarURL, err := s.fetchCalendarURL()
	if err != nil {
		return nil, err
	}
	startOfDay := time.Date(s.now.Year(), s.now.Month(), s.now.Day(), 0, 0, 0, 0, s.now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	events, err := moodle.FetchCalendarEvents(calendarURL, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(events, func(left, right moodle.CalendarEvent) int {
		if left.Start.Before(right.Start) {
			return -1
		}
		if left.Start.After(right.Start) {
			return 1
		}
		return strings.Compare(left.Summary, right.Summary)
	})
	s.todayEvents = events
	s.todayLoaded = true
	return slices.Clone(events), nil
}

func (s *navService) fetchCourses() ([]moodle.Course, error) {
	if s.coursesLoaded {
		return slices.Clone(s.courses), nil
	}
	courses, err := s.client.FetchCourses()
	if err != nil {
		return nil, err
	}
	s.courses = courses
	s.coursesLoaded = true
	return slices.Clone(courses), nil
}

func (s *navService) fetchCourseResources(courseID string) ([]moodle.Resource, error) {
	if resources, ok := s.courseResources[courseID]; ok {
		return slices.Clone(resources), nil
	}
	resources, _, err := s.client.FetchCourseResources(courseID)
	if err != nil {
		return nil, err
	}
	s.courseResources[courseID] = resources
	return slices.Clone(resources), nil
}

func (s *navService) fetchCalendarURL() (string, error) {
	if s.calendarLoaded {
		if s.calendarURL == "" {
			return "", fmt.Errorf("calendar URL not set. Run: moodle config set --calendar-url <url>")
		}
		return s.calendarURL, nil
	}
	cfg, err := config.LoadConfig(opts.ConfigPath)
	if err != nil {
		return "", err
	}
	s.calendarURL = cfg.CalendarURL
	s.calendarLoaded = true
	if s.calendarURL == "" {
		return "", fmt.Errorf("calendar URL not set. Run: moodle config set --calendar-url <url>")
	}
	return s.calendarURL, nil
}

func (s *navService) courseByID(courseID string) (*moodle.Course, error) {
	courses, err := s.fetchCourses()
	if err != nil {
		return nil, err
	}
	return findCourseByID(courses, courseID)
}

func (s *navService) resourceByID(courseID string, resourceID string) (*moodle.Resource, error) {
	resources, err := s.fetchCourseResources(courseID)
	if err != nil {
		return nil, err
	}
	for i := range resources {
		if resources[i].ID == resourceID {
			return &resources[i], nil
		}
	}
	return nil, fmt.Errorf("resource not found: %s", resourceID)
}

func matchNavChild(children []navNode, segment string) (navNode, error) {
	if len(children) == 0 {
		return navNode{}, fmt.Errorf("no further items")
	}
	if index, ok := parsePositiveIndex(segment); ok {
		if index > len(children) {
			return navNode{}, fmt.Errorf("index out of range")
		}
		return children[index-1], nil
	}
	for _, child := range children {
		if strings.EqualFold(child.Segment, segment) || strings.EqualFold(child.Title, segment) {
			return child, nil
		}
	}
	target := slugNavSegment(segment)
	for _, child := range children {
		if slugNavSegment(child.Segment) == target || slugNavSegment(child.Title) == target {
			return child, nil
		}
	}
	return navNode{}, fmt.Errorf("no child matches")
}

func filterResourcesBySection(resources []moodle.Resource, sectionID string, sectionName string) []moodle.Resource {
	out := make([]moodle.Resource, 0, len(resources))
	for _, resource := range resources {
		if sectionID != "" && resource.SectionID == sectionID {
			out = append(out, resource)
			continue
		}
		if sectionName != "" && resource.SectionName == sectionName {
			out = append(out, resource)
		}
	}
	return out
}

func buildResourceSubtitle(resource moodle.Resource) string {
	parts := make([]string, 0, 3)
	if resource.SectionName != "" {
		parts = append(parts, resource.SectionName)
	}
	if resource.FileType != "" {
		parts = append(parts, strings.ToUpper(resource.FileType))
	}
	if resource.UploadedAt != "" {
		parts = append(parts, resource.UploadedAt)
	}
	return strings.Join(parts, " · ")
}

var semesterTokenPattern = regexp.MustCompile(`\b([FH]S\d{2})\b`)

func extractCourseSemester(course moodle.Course) string {
	candidates := []string{course.Fullname, course.Shortname, course.Category}
	for _, candidate := range candidates {
		match := semesterTokenPattern.FindStringSubmatch(strings.ToUpper(candidate))
		if len(match) == 2 {
			return match[1]
		}
	}
	return "OTHER"
}

func compareSemesterLabels(left, right string) int {
	if left == right {
		return 0
	}
	lp := parseSemesterLabel(left)
	rp := parseSemesterLabel(right)
	if lp.valid && rp.valid {
		if lp.year > rp.year {
			return -1
		}
		if lp.year < rp.year {
			return 1
		}
		if lp.term > rp.term {
			return -1
		}
		if lp.term < rp.term {
			return 1
		}
	}
	return strings.Compare(left, right)
}

func weekNumber(value time.Time) int {
	_, week := value.ISOWeek()
	return week
}

func previewFromChildren(children []navNode) string {
	if len(children) == 0 {
		return ""
	}
	if children[0].PreviewText != "" {
		return children[0].PreviewText
	}
	if children[0].Subtitle != "" {
		return children[0].Title + "\n" + children[0].Subtitle
	}
	return children[0].Title
}

type semesterParts struct {
	valid bool
	term  int
	year  int
}

func parseSemesterLabel(value string) semesterParts {
	match := semesterTokenPattern.FindStringSubmatch(strings.ToUpper(value))
	if len(match) != 2 {
		return semesterParts{}
	}
	year, err := strconv.Atoi(match[1][2:])
	if err != nil {
		return semesterParts{}
	}
	term := 0
	if strings.HasPrefix(match[1], "HS") {
		term = 2
	} else {
		term = 1
	}
	return semesterParts{valid: true, term: term, year: year}
}

func slugNavSegment(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return "item"
	}
	replacer := strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss")
	lower = replacer.Replace(lower)
	var out strings.Builder
	lastDash := false
	for _, r := range lower {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				out.WriteByte('-')
				lastDash = true
			}
		}
	}
	result := strings.Trim(out.String(), "-")
	if result == "" {
		return "item"
	}
	return result
}

func renderNavSummary(summary navSummary) string {
	data, _ := json.MarshalIndent(summary, "", "  ")
	return string(data)
}
