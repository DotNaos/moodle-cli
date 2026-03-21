package cli

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DotNaos/moodle-cli/internal/moodle"
	tea "github.com/charmbracelet/bubbletea"
)

type fakeNavigator struct {
	root      navNode
	children  map[string][]navNode
	opened    []string
	downloads []string
}

func (f *fakeNavigator) Root() navNode                      { return f.root }
func (f *fakeNavigator) Preview(node navNode) string        { return node.Title }
func (f *fakeNavigator) Print(node navNode) (string, error) { return "preview " + node.Title, nil }
func (f *fakeNavigator) Children(node navNode) ([]navNode, error) {
	return f.children[node.Key], nil
}
func (f *fakeNavigator) Open(node navNode) error {
	f.opened = append(f.opened, node.Key)
	return nil
}
func (f *fakeNavigator) Download(node navNode, outputPath string) (string, error) {
	f.downloads = append(f.downloads, outputPath)
	return outputPath + "/file.pdf", nil
}

func TestTUIModelSupportsVimAndArrowNavigation(t *testing.T) {
	nav := &fakeNavigator{
		root: navNode{Key: "root", Kind: navNodeHome, Title: "Moodle"},
		children: map[string][]navNode{
			"root": {
				{Key: "current", Kind: navNodeCurrent, Title: "Current"},
				{Key: "today", Kind: navNodeToday, Title: "Today"},
			},
			"current": {
				{Key: "item", Kind: navNodeResource, Title: "Slides", Openable: true, Printable: true},
			},
		},
	}
	model := tuiModel{
		nav:         nav,
		root:        nav.root,
		focus:       focusTree,
		expanded:    map[string]bool{"root": true},
		nodeByKey:   map[string]navNode{"root": nav.root},
		parentByKey: map[string]string{"root": ""},
		childCache:  map[string][]navNode{"root": nav.children["root"], "current": nav.children["current"]},
		selectedKey: "current",
	}
	for _, child := range nav.children["root"] {
		model.nodeByKey[child.Key] = child
		model.parentByKey[child.Key] = "root"
	}
	for _, child := range nav.children["current"] {
		model.nodeByKey[child.Key] = child
		model.parentByKey[child.Key] = "current"
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = next.(tuiModel)
	if model.selectedKey != "today" {
		t.Fatalf("expected arrow down to move to today, got %q", model.selectedKey)
	}

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	model = next.(tuiModel)
	if model.selectedKey != "current" {
		t.Fatalf("expected k to move back to current, got %q", model.selectedKey)
	}

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyRight})
	model = next.(tuiModel)
	if model.focus != focusRight {
		t.Fatalf("expected right arrow to focus the right pane")
	}

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = next.(tuiModel)
	if model.selectedKey != "item" {
		t.Fatalf("expected enter on right pane to drill into item, got %q", model.selectedKey)
	}

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = next.(tuiModel)
	if model.focus != focusTree {
		t.Fatalf("expected left arrow to return focus to the tree")
	}
}

func TestTUIModelEnterTogglesTreeNode(t *testing.T) {
	nav := &fakeNavigator{
		root: navNode{Key: "root", Kind: navNodeHome, Title: "Moodle"},
		children: map[string][]navNode{
			"root": {
				{Key: "course", Kind: navNodeCourse, Title: "Course"},
			},
			"course": {
				{Key: "file", Kind: navNodeResource, Title: "Slides", Openable: true, Printable: true},
			},
		},
	}
	model := tuiModel{
		nav:         nav,
		root:        nav.root,
		focus:       focusTree,
		expanded:    map[string]bool{"root": true},
		nodeByKey:   map[string]navNode{"root": nav.root, "course": nav.children["root"][0], "file": nav.children["course"][0]},
		parentByKey: map[string]string{"root": "", "course": "root", "file": "course"},
		childCache:  map[string][]navNode{"root": nav.children["root"], "course": nav.children["course"]},
		selectedKey: "course",
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = next.(tuiModel)
	if cmd != nil {
		t.Fatalf("did not expect enter on a tree node to open immediately")
	}
	if !model.expanded["course"] {
		t.Fatalf("expected enter to expand the selected node")
	}

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = next.(tuiModel)
	if model.expanded["course"] {
		t.Fatalf("expected second enter to collapse the node")
	}
}

func TestTUIFilterUpdatesLiveAndClearsOnEscape(t *testing.T) {
	nav := &fakeNavigator{
		root: navNode{Key: "root", Kind: navNodeHome, Title: "Moodle"},
		children: map[string][]navNode{
			"root": {
				{Key: "a", Kind: navNodeCourse, Title: "Alpha"},
				{Key: "b", Kind: navNodeCourse, Title: "Beta"},
			},
		},
	}
	model := tuiModel{
		nav:         nav,
		root:        nav.root,
		focus:       focusTree,
		expanded:    map[string]bool{"root": true},
		nodeByKey:   map[string]navNode{"root": nav.root, "a": nav.children["root"][0], "b": nav.children["root"][1]},
		parentByKey: map[string]string{"root": "", "a": "root", "b": "root"},
		childCache:  map[string][]navNode{"root": nav.children["root"]},
		selectedKey: "a",
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model = next.(tuiModel)
	if !model.filterMode {
		t.Fatalf("expected filter mode to start")
	}

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	model = next.(tuiModel)
	if model.leftFilter != "b" {
		t.Fatalf("expected live filter to be applied, got %q", model.leftFilter)
	}
	rows := model.visibleTreeRows()
	if len(rows) != 1 || rows[0].Node.Title != "Beta" {
		t.Fatalf("expected live filtering to narrow to Beta, got %+v", rows)
	}

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = next.(tuiModel)
	if model.filterMode {
		t.Fatalf("expected filter mode to end on escape")
	}
	if model.leftFilter != "" {
		t.Fatalf("expected filter to clear on escape, got %q", model.leftFilter)
	}
}

func TestTUIOpensDownloadDialogAndDownloadsToSelectedFolder(t *testing.T) {
	tempDir := t.TempDir()
	subDir := tempDir + "/nested"
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	nav := &fakeNavigator{
		root: navNode{Key: "root", Kind: navNodeHome, Title: "Moodle"},
		children: map[string][]navNode{
			"root": {
				{Key: "file", Kind: navNodeResource, Title: "Slides", Openable: true, Printable: true, Resource: &moodle.Resource{Name: "Slides", Type: "resource", FileType: "pdf"}},
			},
		},
	}
	model := tuiModel{
		nav:         nav,
		root:        nav.root,
		focus:       focusTree,
		expanded:    map[string]bool{"root": true},
		nodeByKey:   map[string]navNode{"root": nav.root, "file": nav.children["root"][0]},
		parentByKey: map[string]string{"root": "", "file": "root"},
		childCache:  map[string][]navNode{"root": nav.children["root"]},
		selectedKey: "file",
	}

	originalOutputDir := opts.ExportDir
	opts.ExportDir = tempDir
	defer func() { opts.ExportDir = originalOutputDir }()

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = next.(tuiModel)
	if model.dialog == nil {
		t.Fatalf("expected download dialog to open")
	}
	if len(model.dialog.entries) < 2 {
		t.Fatalf("expected save row plus directory entries, got %+v", model.dialog.entries)
	}

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = next.(tuiModel)
	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = next.(tuiModel)
	if model.dialog == nil || model.dialog.cwd != subDir {
		t.Fatalf("expected dialog to enter nested directory, got %+v", model.dialog)
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = next.(tuiModel)
	if cmd != nil {
		t.Fatalf("did not expect command on pure navigation")
	}
	next, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = next.(tuiModel)
	if cmd == nil {
		t.Fatalf("expected save command")
	}
	msg := cmd()
	next, _ = model.Update(msg)
	model = next.(tuiModel)
	if model.dialog != nil {
		t.Fatalf("expected dialog to close after download")
	}
	if len(nav.downloads) != 1 || nav.downloads[0] != subDir {
		t.Fatalf("expected download to nested directory, got %v", nav.downloads)
	}
}

func TestNavServiceResolveCurrentItemPath(t *testing.T) {
	service := &navService{
		now:           timeNow(),
		currentLoaded: true,
		current: currentLectureResult{
			Course:   &currentLectureCourse{ID: 42, Title: "Deep Learning"},
			Material: &currentLectureResource{ID: "10", Label: "Slides", URL: "https://example.com/10", FileType: "pdf"},
			Resources: []currentLectureResource{
				{ID: "10", Label: "Slides", URL: "https://example.com/10", FileType: "pdf"},
				{ID: "11", Label: "Notes", URL: "https://example.com/11", FileType: "pdf"},
			},
		},
		coursesLoaded: true,
		courses: []moodle.Course{
			{ID: 42, Fullname: "Deep Learning", ViewURL: "https://example.com/course/42"},
		},
		courseResources: map[string][]moodle.Resource{
			"42": {
				{ID: "10", Name: "Slides", URL: "https://example.com/10", Type: "resource", FileType: "pdf"},
				{ID: "11", Name: "Notes", URL: "https://example.com/11", Type: "resource", FileType: "pdf"},
			},
		},
	}

	node, err := service.ResolvePath("current/items/current")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.Resource == nil || node.Resource.ID != "10" {
		t.Fatalf("expected current item resource 10, got %+v", node.Resource)
	}

	byIndex, err := service.ResolvePath("current/items/2")
	if err != nil {
		t.Fatalf("unexpected error resolving by index: %v", err)
	}
	if byIndex.Resource == nil || byIndex.Resource.ID != "11" {
		t.Fatalf("expected second item resource 11, got %+v", byIndex.Resource)
	}
}

func TestNavServiceSemesterShowsCoursesDirectly(t *testing.T) {
	service := &navService{
		coursesLoaded: true,
		courses: []moodle.Course{
			{ID: 10, Fullname: "Deep Learning (cds-108) FS26", ViewURL: "https://example.com/10"},
			{ID: 11, Fullname: "High Performance Computing FS26", ViewURL: "https://example.com/11"},
			{ID: 12, Fullname: "Something HS25", ViewURL: "https://example.com/12"},
		},
		courseResources: map[string][]moodle.Resource{},
	}

	node, err := service.ResolvePath("semesters/FS26/1")
	if err != nil {
		t.Fatalf("unexpected error resolving semester course path: %v", err)
	}
	if node.Kind != navNodeCourse {
		t.Fatalf("expected course node, got %s", node.Kind)
	}
	if node.Course == nil || node.Course.ID != 10 {
		t.Fatalf("expected first FS26 course, got %+v", node.Course)
	}

	legacy, err := service.ResolvePath("semesters/FS26/courses/1")
	if err != nil {
		t.Fatalf("unexpected error resolving legacy semester course path: %v", err)
	}
	if legacy.Kind != navNodeCourse {
		t.Fatalf("expected course node from legacy path, got %s", legacy.Kind)
	}
}

func TestNavServiceSectionShowsItemsDirectly(t *testing.T) {
	service := &navService{
		coursesLoaded: true,
		courses: []moodle.Course{
			{ID: 10, Fullname: "Deep Learning (cds-108) FS26", ViewURL: "https://example.com/10"},
		},
		courseResources: map[string][]moodle.Resource{
			"10": {
				{ID: "100", Name: "Slides", URL: "https://example.com/100", Type: "resource", FileType: "pdf", SectionID: "1", SectionName: "Allgemeine Informationen"},
				{ID: "101", Name: "Notes", URL: "https://example.com/101", Type: "resource", FileType: "pdf", SectionID: "1", SectionName: "Allgemeine Informationen"},
			},
		},
	}

	node, err := service.ResolvePath("semesters/FS26/1/sections/1/1")
	if err != nil {
		t.Fatalf("unexpected error resolving flattened section path: %v", err)
	}
	if node.Kind != navNodeResource || node.Resource == nil || node.Resource.ID != "100" {
		t.Fatalf("expected first section item, got %+v", node)
	}

	legacy, err := service.ResolvePath("semesters/FS26/1/sections/1/items/2")
	if err != nil {
		t.Fatalf("unexpected error resolving legacy section items path: %v", err)
	}
	if legacy.Kind != navNodeResource || legacy.Resource == nil || legacy.Resource.ID != "101" {
		t.Fatalf("expected second section item from legacy path, got %+v", legacy)
	}
}

func TestGroupCalendarEventsMergesAdjacentMatches(t *testing.T) {
	service := &navService{
		coursesLoaded: true,
		courses: []moodle.Course{
			{ID: 42, Fullname: "Deep Learning (cds-108) FS26", ViewURL: "https://example.com/42"},
		},
	}
	events := []moodle.CalendarEvent{
		{
			Summary:  "Deep Learning",
			Location: "B1.03",
			Start:    time.Date(2026, 3, 20, 15, 15, 0, 0, time.Local),
			End:      time.Date(2026, 3, 20, 16, 45, 0, 0, time.Local),
		},
		{
			Summary:  "Deep Learning",
			Location: "B1.03",
			Start:    time.Date(2026, 3, 20, 17, 0, 0, 0, time.Local),
			End:      time.Date(2026, 3, 20, 18, 30, 0, 0, time.Local),
		},
	}

	grouped, err := service.groupCalendarEvents(events)
	if err != nil {
		t.Fatalf("unexpected error grouping events: %v", err)
	}
	if len(grouped) != 1 {
		t.Fatalf("expected grouped events to merge into one row, got %d", len(grouped))
	}
	if grouped[0].Subtitle != "15:15-16:45 · 17:00-18:30 · B1.03" {
		t.Fatalf("unexpected grouped subtitle: %q", grouped[0].Subtitle)
	}
}

func TestRootCommandLaunchesTUIOnNoArgs(t *testing.T) {
	original := launchTUI
	defer func() { launchTUI = original }()

	called := false
	launchTUI = func(options selectorOptions) error {
		called = true
		return nil
	}

	if err := rootCmd.RunE(rootCmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected root command to launch the TUI")
	}
}

func timeNow() time.Time {
	return time.Date(2026, 3, 20, 15, 0, 0, 0, time.Local)
}

func TestRenderNavRowStaysSingleLine(t *testing.T) {
	row := renderTreeRow(treeRow{
		Node: navNode{
			Title:    "A very long course title that should be truncated in the grid",
			Subtitle: "This subtitle should not create a second row",
		},
		HasKids:  true,
		Expanded: true,
	}, true, 24, "")
	if countLines(row) != 1 {
		t.Fatalf("expected a single-line row, got %q", row)
	}
}

func TestRenderTreeRowIndentsChildren(t *testing.T) {
	parent := renderTreeRow(treeRow{
		Node: navNode{Title: "Semesters"},
	}, false, 40, "")
	child := renderTreeRow(treeRow{
		Node:    navNode{Title: "FS26"},
		Depth:   1,
		HasKids: true,
	}, false, 40, "")
	if !strings.Contains(child, "   ") {
		t.Fatalf("expected child row to contain visible indentation, got %q", child)
	}
	if child == parent {
		t.Fatalf("expected child row to render differently from parent, parent=%q child=%q", parent, child)
	}
}

func TestRightPaneStaysPassiveUntilFocused(t *testing.T) {
	row := renderRightRow(rightEntry{Kind: rightEntryNode, Label: "Slides"}, false, 24, "")
	if strings.Contains(row, "▸") || strings.Contains(row, "›") {
		t.Fatalf("expected passive right row without marker, got %q", row)
	}

	active := renderRightRow(rightEntry{Kind: rightEntryNode, Label: "Slides"}, true, 24, "")
	if !strings.Contains(active, "Slides") {
		t.Fatalf("expected focused right row to contain label, got %q", active)
	}
}

func TestEnterOnRightExpandsTreePath(t *testing.T) {
	nav := &fakeNavigator{
		root: navNode{Key: "root", Kind: navNodeHome, Title: "Moodle"},
		children: map[string][]navNode{
			"root": {
				{Key: "semesters", Kind: navNodeSemesters, Title: "Semesters"},
			},
			"semesters": {
				{Key: "fs26", Kind: navNodeSemester, Title: "FS26"},
			},
			"fs26": {
				{Key: "course", Kind: navNodeCourse, Title: "Deep Learning"},
			},
		},
	}
	model := tuiModel{
		nav:          nav,
		root:         nav.root,
		focus:        focusRight,
		previewCache: map[string]string{},
		expanded:     map[string]bool{"root": true},
		nodeByKey: map[string]navNode{
			"root":      nav.root,
			"semesters": nav.children["root"][0],
			"fs26":      nav.children["semesters"][0],
			"course":    nav.children["fs26"][0],
		},
		parentByKey: map[string]string{
			"root":      "",
			"semesters": "root",
			"fs26":      "semesters",
			"course":    "fs26",
		},
		childCache: map[string][]navNode{
			"root":      nav.children["root"],
			"semesters": nav.children["semesters"],
			"fs26":      nav.children["fs26"],
		},
		selectedKey:   "semesters",
		rightSelected: 0,
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = next.(tuiModel)
	if model.selectedKey != "fs26" {
		t.Fatalf("expected enter on right to drill to fs26, got %q", model.selectedKey)
	}
	if !model.expanded["semesters"] {
		t.Fatalf("expected parent tree node to be expanded after drilling from right")
	}
}

func TestLeftPaneDoesNotKeepActiveHighlightWhenFocusIsRight(t *testing.T) {
	nav := &fakeNavigator{
		root: navNode{Key: "root", Kind: navNodeHome, Title: "Moodle"},
		children: map[string][]navNode{
			"root": {
				{Key: "current", Kind: navNodeCurrent, Title: "Current"},
				{Key: "today", Kind: navNodeToday, Title: "Today"},
			},
		},
	}
	model := tuiModel{
		nav:          nav,
		root:         nav.root,
		focus:        focusRight,
		previewCache: map[string]string{},
		expanded:     map[string]bool{"root": true},
		nodeByKey: map[string]navNode{
			"root":    nav.root,
			"current": nav.children["root"][0],
			"today":   nav.children["root"][1],
		},
		parentByKey: map[string]string{"root": "", "current": "root", "today": "root"},
		childCache:  map[string][]navNode{"root": nav.children["root"]},
		selectedKey: "current",
	}

	pane := model.renderTreePane(40, 12)
	if strings.Contains(pane, "› Current") {
		t.Fatalf("expected left pane not to show active marker while focus is right, got %q", pane)
	}
}

func TestTreePaneShowsOnlyActiveBranchChildren(t *testing.T) {
	nav := &fakeNavigator{
		root: navNode{Key: "root", Kind: navNodeHome, Title: "Moodle"},
		children: map[string][]navNode{
			"root": {
				{Key: "semesters", Kind: navNodeSemesters, Title: "Semesters"},
			},
			"semesters": {
				{Key: "fs26", Kind: navNodeSemester, Title: "FS26"},
				{Key: "hs25", Kind: navNodeSemester, Title: "HS25"},
			},
			"fs26": {
				{Key: "course-a", Kind: navNodeCourse, Title: "Course A"},
			},
			"hs25": {
				{Key: "course-b", Kind: navNodeCourse, Title: "Course B"},
			},
		},
	}
	model := tuiModel{
		nav:          nav,
		root:         nav.root,
		focus:        focusTree,
		previewCache: map[string]string{},
		expanded:     map[string]bool{"root": true, "semesters": true, "fs26": true, "hs25": true},
		nodeByKey: map[string]navNode{
			"root":      nav.root,
			"semesters": nav.children["root"][0],
			"fs26":      nav.children["semesters"][0],
			"hs25":      nav.children["semesters"][1],
			"course-a":  nav.children["fs26"][0],
			"course-b":  nav.children["hs25"][0],
		},
		parentByKey: map[string]string{
			"root":      "",
			"semesters": "root",
			"fs26":      "semesters",
			"hs25":      "semesters",
			"course-a":  "fs26",
			"course-b":  "hs25",
		},
		childCache: map[string][]navNode{
			"root":      nav.children["root"],
			"semesters": nav.children["semesters"],
			"fs26":      nav.children["fs26"],
			"hs25":      nav.children["hs25"],
		},
		selectedKey: "course-a",
	}

	rows := model.visibleTreeRows()
	titles := make([]string, 0, len(rows))
	for _, row := range rows {
		titles = append(titles, row.Node.Title)
	}
	if !strings.Contains(strings.Join(titles, ","), "Course A") {
		t.Fatalf("expected active branch child to be visible, got %v", titles)
	}
	if strings.Contains(strings.Join(titles, ","), "Course B") {
		t.Fatalf("expected inactive branch child to stay hidden, got %v", titles)
	}
}

func TestAutoPreviewLoadsPrintableResource(t *testing.T) {
	nav := &fakeNavigator{
		root: navNode{Key: "root", Kind: navNodeHome, Title: "Moodle"},
		children: map[string][]navNode{
			"root": {
				{Key: "file", Kind: navNodeResource, Title: "Slides", Openable: true, Printable: true},
			},
		},
	}
	model := tuiModel{
		nav:          nav,
		root:         nav.root,
		focus:        focusTree,
		previewCache: map[string]string{},
		expanded:     map[string]bool{"root": true},
		nodeByKey:    map[string]navNode{"root": nav.root, "file": nav.children["root"][0]},
		parentByKey:  map[string]string{"root": "", "file": "root"},
		childCache:   map[string][]navNode{"root": nav.children["root"]},
		selectedKey:  "file",
	}

	cmd := model.autoPreviewCmd()
	if cmd == nil {
		t.Fatalf("expected auto preview command")
	}
	msg := cmd()
	next, _ := model.Update(msg)
	model = next.(tuiModel)
	if got := model.previewCache["file"]; got == "" {
		t.Fatalf("expected preview cache for file to be populated")
	}
}

func TestStructurePreviewShowsChildren(t *testing.T) {
	nav := &fakeNavigator{
		root: navNode{Key: "root", Kind: navNodeHome, Title: "Moodle"},
		children: map[string][]navNode{
			"root": {
				{Key: "course", Kind: navNodeCourse, Title: "Course"},
			},
			"course": {
				{Key: "section", Kind: navNodeSection, Title: "Section A"},
			},
			"section": {
				{Key: "file", Kind: navNodeResource, Title: "Slides", Subtitle: "PDF"},
			},
		},
	}
	model := tuiModel{
		nav:          nav,
		root:         nav.root,
		focus:        focusTree,
		previewCache: map[string]string{},
		expanded:     map[string]bool{"root": true},
		nodeByKey: map[string]navNode{
			"root":    nav.root,
			"course":  nav.children["root"][0],
			"section": nav.children["course"][0],
			"file":    nav.children["section"][0],
		},
		parentByKey: map[string]string{"root": "", "course": "root", "section": "course", "file": "section"},
		childCache: map[string][]navNode{
			"root":    nav.children["root"],
			"course":  nav.children["course"],
			"section": nav.children["section"],
		},
	}

	title, body := model.nodePreview(nav.children["root"][0])
	if title != "Course" {
		t.Fatalf("unexpected title %q", title)
	}
	if !strings.Contains(body, "Section A") || !strings.Contains(body, "Slides") {
		t.Fatalf("expected structure preview to include nested content, got %q", body)
	}
}
