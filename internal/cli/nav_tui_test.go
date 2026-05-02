package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DotNaos/moodle-services/internal/moodle"
	ver "github.com/DotNaos/moodle-services/internal/version"
	tea "github.com/charmbracelet/bubbletea"
)

type fakeNavigator struct {
	root      navNode
	children  map[string][]navNode
	opened    []string
	downloads []string
	previews  map[string]string
	prints    map[string]string
}

func (f *fakeNavigator) Root() navNode { return f.root }
func (f *fakeNavigator) Preview(node navNode) string {
	if text, ok := f.previews[node.Key]; ok {
		return text
	}
	return node.Title
}
func (f *fakeNavigator) Print(node navNode) (string, error) {
	if text, ok := f.prints[node.Key]; ok {
		return text, nil
	}
	return "preview " + node.Title, nil
}
func (f *fakeNavigator) Children(node navNode) ([]navNode, error) {
	return f.children[node.Key], nil
}
func (f *fakeNavigator) Open(node navNode) (string, error) {
	f.opened = append(f.opened, node.Key)
	return node.Key, nil
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
	if !model.expanded["course"] {
		t.Fatalf("expected enter to expand the selected node")
	}
	if cmd == nil {
		t.Fatalf("expected enter to start background work for expanded course")
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
	subDir := filepath.Join(tempDir, "nested")
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
	if model.dialog == nil || !samePath(t, model.dialog.cwd, subDir) {
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
	if len(nav.downloads) != 1 || !samePath(t, nav.downloads[0], subDir) {
		t.Fatalf("expected download to nested directory, got %v", nav.downloads)
	}
}

func samePath(t *testing.T, left string, right string) bool {
	t.Helper()

	leftInfo, err := os.Stat(left)
	if err != nil {
		t.Fatalf("stat %q failed: %v", left, err)
	}
	rightInfo, err := os.Stat(right)
	if err != nil {
		t.Fatalf("stat %q failed: %v", right, err)
	}
	return os.SameFile(leftInfo, rightInfo)
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
	if node.Title != "Deep Learning" {
		t.Fatalf("expected cleaned course title, got %q", node.Title)
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
	}, true, 24, "", 0)
	if countLines(row) != 1 {
		t.Fatalf("expected a single-line row, got %q", row)
	}
}

func TestRenderTreeRowIndentsChildren(t *testing.T) {
	parent := renderTreeRow(treeRow{
		Node: navNode{Title: "Semesters"},
	}, false, 40, "", 0)
	child := renderTreeRow(treeRow{
		Node:    navNode{Title: "FS26"},
		Depth:   1,
		HasKids: true,
	}, false, 40, "", 0)
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

func TestTUIModelEnterLoadsChildrenInBackground(t *testing.T) {
	nav := &fakeNavigator{
		root: navNode{Key: "root", Kind: navNodeHome, Title: "Moodle"},
		children: map[string][]navNode{
			"root": {
				{Key: "timetable", Kind: navNodeTimetable, Title: "Timetable"},
			},
			"timetable": {
				{Key: "week", Kind: navNodeWeek, Title: "This Week"},
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
			"root":      nav.root,
			"timetable": nav.children["root"][0],
		},
		parentByKey: map[string]string{"root": "", "timetable": "root"},
		childCache:  map[string][]navNode{"root": nav.children["root"]},
		childBusy:   map[string]bool{},
		childErrors: map[string]string{},
		selectedKey: "timetable",
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = next.(tuiModel)
	if !model.expanded["timetable"] {
		t.Fatalf("expected enter to expand the selected node immediately")
	}
	if !model.childBusy["timetable"] {
		t.Fatalf("expected enter to mark timetable as loading")
	}
	if _, ok := model.childCache["timetable"]; ok {
		t.Fatalf("expected children to stay unloaded until the background command completes")
	}
	if cmd == nil {
		t.Fatalf("expected enter to return a background load command")
	}

	msg := cmd()
	next, followup := model.Update(msg)
	model = next.(tuiModel)
	if followup != nil {
		t.Fatalf("did not expect a second command after children loaded")
	}
	if model.childBusy["timetable"] {
		t.Fatalf("expected loading flag to clear after children arrive")
	}
	if len(model.childCache["timetable"]) != 1 {
		t.Fatalf("expected loaded children to be cached, got %+v", model.childCache["timetable"])
	}
}

func TestTreeAndDetailsShowLoadingStateForBusyNode(t *testing.T) {
	nav := &fakeNavigator{
		root: navNode{Key: "root", Kind: navNodeHome, Title: "Moodle"},
		children: map[string][]navNode{
			"root": {
				{Key: "timetable", Kind: navNodeTimetable, Title: "Timetable"},
			},
		},
	}
	model := tuiModel{
		nav:          nav,
		root:         nav.root,
		focus:        focusTree,
		previewCache: map[string]string{},
		expanded:     map[string]bool{"root": true, "timetable": true},
		nodeByKey: map[string]navNode{
			"root":      nav.root,
			"timetable": nav.children["root"][0],
		},
		parentByKey:  map[string]string{"root": "", "timetable": "root"},
		childCache:   map[string][]navNode{"root": nav.children["root"]},
		childBusy:    map[string]bool{"timetable": true},
		childErrors:  map[string]string{},
		selectedKey:  "timetable",
		spinnerFrame: 3,
	}

	tree := model.renderTreePane(40, 12)
	if !strings.Contains(tree, spinnerFrames[3]) || !strings.Contains(tree, "Timetable") {
		t.Fatalf("expected tree pane to show a loading spinner for the busy node, got %q", tree)
	}

	details := model.renderRightPane(40, 12)
	if !strings.Contains(details, "Loading items") {
		t.Fatalf("expected details pane to show loading text, got %q", details)
	}
}

func TestPrintActionShowsLoadedPreviewInBottomPane(t *testing.T) {
	nav := &fakeNavigator{
		root: navNode{Key: "root", Kind: navNodeHome, Title: "Moodle"},
		children: map[string][]navNode{
			"root": {
				{Key: "file", Kind: navNodeResource, Title: "Aufgabenblatt 01", Openable: true, Printable: true, Resource: &moodle.Resource{Name: "Aufgabenblatt 01", Type: "resource", FileType: "pdf"}},
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
			"root": nav.root,
			"file": nav.children["root"][0],
		},
		parentByKey:   map[string]string{"root": "", "file": "root"},
		childCache:    map[string][]navNode{"root": nav.children["root"]},
		childBusy:     map[string]bool{},
		childErrors:   map[string]string{},
		selectedKey:   "file",
		rightSelected: 1,
	}

	before := model.renderBottomPane(80, 12)
	if !strings.Contains(before, "Show the file text in the lower panel.") {
		t.Fatalf("expected print action description before loading, got %q", before)
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = next.(tuiModel)
	if cmd == nil {
		t.Fatalf("expected print action to return a preview command")
	}
	during := model.renderBottomPane(80, 12)
	if !strings.Contains(during, "Loading preview") {
		t.Fatalf("expected loading state after triggering print, got %q", during)
	}

	msg := cmd()
	next, _ = model.Update(msg)
	model = next.(tuiModel)
	after := model.renderBottomPane(80, 12)
	if !strings.Contains(after, "preview Aufgabenblatt 01") {
		t.Fatalf("expected loaded preview text in bottom pane, got %q", after)
	}
	if strings.Contains(after, "Show the file text in the lower panel.") {
		t.Fatalf("expected action description to be replaced by preview content, got %q", after)
	}
}

func TestOpenActionRunsAndUpdatesStatus(t *testing.T) {
	nav := &fakeNavigator{
		root: navNode{Key: "root", Kind: navNodeHome, Title: "Moodle"},
		children: map[string][]navNode{
			"root": {
				{Key: "file", Kind: navNodeResource, Title: "Aufgabenblatt 01", Openable: true, Printable: true, Resource: &moodle.Resource{Name: "Aufgabenblatt 01", Type: "resource", FileType: "pdf"}},
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
			"root": nav.root,
			"file": nav.children["root"][0],
		},
		parentByKey:   map[string]string{"root": "", "file": "root"},
		childCache:    map[string][]navNode{"root": nav.children["root"]},
		childBusy:     map[string]bool{},
		childErrors:   map[string]string{},
		selectedKey:   "file",
		rightSelected: 0,
	}

	next, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = next.(tuiModel)
	if cmd == nil {
		t.Fatalf("expected open action to return a command")
	}
	if model.status != "Opening..." {
		t.Fatalf("expected immediate opening status, got %q", model.status)
	}

	msg := cmd()
	next, _ = model.Update(msg)
	model = next.(tuiModel)
	if len(nav.opened) != 1 || nav.opened[0] != "file" {
		t.Fatalf("expected navigator open to run for file, got %v", nav.opened)
	}
	if model.status != "Opened." {
		t.Fatalf("expected final opened status, got %q", model.status)
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

func TestAutoPreviewLoadsCourseReaderPreview(t *testing.T) {
	nav := &fakeNavigator{
		root: navNode{Key: "root", Kind: navNodeHome, Title: "Moodle"},
		children: map[string][]navNode{
			"root": {
				{Key: "course", Kind: navNodeCourse, Title: "Algorithmen des wissenschaftlichen Rechnens"},
			},
		},
		previews: map[string]string{
			"course": "Course: Algorithmen des wissenschaftlichen Rechnens\n\nThema 1: Sparse Grids\n- Folien Teil 1 · PDF",
		},
	}
	model := tuiModel{
		nav:          nav,
		root:         nav.root,
		focus:        focusTree,
		previewCache: map[string]string{},
		expanded:     map[string]bool{"root": true},
		nodeByKey:    map[string]navNode{"root": nav.root, "course": nav.children["root"][0]},
		parentByKey:  map[string]string{"root": "", "course": "root"},
		childCache:   map[string][]navNode{"root": nav.children["root"]},
		selectedKey:  "course",
	}

	cmd := model.autoPreviewCmd()
	if cmd == nil {
		t.Fatalf("expected auto preview command for course")
	}
	msg := cmd()
	next, _ := model.Update(msg)
	model = next.(tuiModel)

	preview := model.renderBottomPane(80, 12)
	if !strings.Contains(preview, "Thema 1: Sparse Grids") {
		t.Fatalf("expected course reader preview in bottom pane, got %q", preview)
	}
}

func TestReaderModeShowsPDFContentDirectly(t *testing.T) {
	resource := navNode{
		Key:       "file",
		Kind:      navNodeResource,
		Title:     "Slides",
		Openable:  true,
		Printable: true,
		Resource:  &moodle.Resource{Name: "Slides", Type: "resource", FileType: "pdf"},
	}
	model := tuiModel{
		root:           navNode{Key: "root", Kind: navNodeHome, Title: "Moodle"},
		focus:          focusPreview,
		topCollapsed:   true,
		previewCache:   map[string]string{"file": "Page 1\nSparse grids are hierarchical."},
		nodeByKey:      map[string]navNode{"file": resource},
		parentByKey:    map[string]string{"file": "root"},
		selectedKey:    "file",
		lastUpperFocus: focusTree,
	}

	_, body := model.nodePreview(resource)
	if !strings.Contains(body, "Sparse grids are hierarchical.") {
		t.Fatalf("expected pdf text in reader mode, got %q", body)
	}
	if strings.Contains(body, "Item:") {
		t.Fatalf("expected reader mode to hide metadata banner, got %q", body)
	}
}

func TestReaderModeShowsCoursePreviewWithoutCourseBanner(t *testing.T) {
	course := navNode{
		Key:      "course",
		Kind:     navNodeCourse,
		Title:    "Algorithmen des wissenschaftlichen Rechnens",
		CourseID: "42",
	}
	model := tuiModel{
		root:           navNode{Key: "root", Kind: navNodeHome, Title: "Moodle"},
		focus:          focusPreview,
		topCollapsed:   true,
		previewCache:   map[string]string{"course": "Thema 1: Sparse Grids\nLernziele\nStudierende kennen Sparse Grids."},
		nodeByKey:      map[string]navNode{"course": course},
		parentByKey:    map[string]string{"course": "root"},
		selectedKey:    "course",
		lastUpperFocus: focusTree,
	}

	_, body := model.nodePreview(course)
	if !strings.Contains(body, "Thema 1: Sparse Grids") {
		t.Fatalf("expected course reader content, got %q", body)
	}
	if strings.Contains(body, "Course:") {
		t.Fatalf("expected reader mode to hide course banner, got %q", body)
	}
}

func TestMarkdownToggleSwitchesPreviewMode(t *testing.T) {
	model := tuiModel{
		previewMode: previewModeReader,
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	model = next.(tuiModel)
	if model.previewMode != previewModeMarkdown {
		t.Fatalf("expected markdown mode, got %q", model.previewMode)
	}

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	model = next.(tuiModel)
	if model.previewMode != previewModeReader {
		t.Fatalf("expected reader mode, got %q", model.previewMode)
	}
}

func TestMarkdownPreviewFormatsCourseAsMarkdown(t *testing.T) {
	model := tuiModel{previewMode: previewModeMarkdown}
	body := model.formatPreviewBody(navNode{Kind: navNodeCourse}, "Thema 1: Sparse Grids\nLernziele\n- Folien Teil 1 · PDF · 2026-03-05T23:27:00+01:00")
	if !strings.Contains(body, "## Thema 1: Sparse Grids") {
		t.Fatalf("expected section heading in markdown, got %q", body)
	}
	if !strings.Contains(body, "### Lernziele") {
		t.Fatalf("expected heading in markdown, got %q", body)
	}
	if !strings.Contains(body, "- Folien Teil 1 · **PDF** · _2026-03-05T23:27:00+01:00_") {
		t.Fatalf("expected bullet metadata formatting in markdown, got %q", body)
	}
}

func TestPrintPreviewDoesNotTruncateLongDocument(t *testing.T) {
	longText := strings.Repeat("Sparse grids are useful.\n", 200)
	nav := &fakeNavigator{
		root: navNode{Key: "root", Kind: navNodeHome, Title: "Moodle"},
		children: map[string][]navNode{
			"root": {
				{Key: "file", Kind: navNodeResource, Title: "Slides", Openable: true, Printable: true, Resource: &moodle.Resource{Name: "Slides", Type: "resource", FileType: "pdf"}},
			},
		},
		prints: map[string]string{
			"file": longText,
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
		t.Fatalf("expected preview command")
	}
	msg := cmd()
	next, _ := model.Update(msg)
	model = next.(tuiModel)

	got := model.previewCache["file"]
	if !strings.Contains(got, "Sparse grids are useful.") {
		t.Fatalf("expected long preview content to be cached")
	}
	if strings.HasSuffix(got, "\n...") {
		t.Fatalf("expected long preview not to be truncated")
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

func TestTUIVersionLabelForStableRelease(t *testing.T) {
	restore := ver.SetBuildInfoForTesting("v1.2.3", "test", "2026-04-07T22:10:00Z")
	t.Cleanup(restore)
	if got := tuiVersionLabel(); got != "v1.2.3" {
		t.Fatalf("expected stable version label, got %q", got)
	}
}

func TestTUIVersionLabelForPreviewBuild(t *testing.T) {
	restore := ver.SetBuildInfoForTesting("v1.2.3-rc1", "test", "2026-04-07T22:10:00Z")
	t.Cleanup(restore)
	got := tuiVersionLabel()
	if !strings.Contains(got, "v1.2.3-rc1") || !strings.Contains(got, "preview") || !strings.Contains(got, "2026-04-07T22:10:00Z") {
		t.Fatalf("expected preview label with build date, got %q", got)
	}
}

func TestCtrlJMovesFocusToPreviewAndCtrlKReturns(t *testing.T) {
	model := tuiModel{
		focus:          focusRight,
		lastUpperFocus: focusRight,
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	model = next.(tuiModel)
	if model.focus != focusPreview {
		t.Fatalf("expected ctrl+j to focus preview, got %q", model.focus)
	}

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	model = next.(tuiModel)
	if model.focus != focusRight {
		t.Fatalf("expected ctrl+k to return to previous upper pane, got %q", model.focus)
	}
}

func TestCtrlWTogglesReaderMode(t *testing.T) {
	model := tuiModel{
		focus:          focusTree,
		lastUpperFocus: focusTree,
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
	model = next.(tuiModel)
	if !model.topCollapsed || model.focus != focusPreview {
		t.Fatalf("expected ctrl+w to enable reader mode, got collapsed=%v focus=%q", model.topCollapsed, model.focus)
	}

	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlW})
	model = next.(tuiModel)
	if model.topCollapsed || model.focus != focusTree {
		t.Fatalf("expected ctrl+w to restore upper panes, got collapsed=%v focus=%q", model.topCollapsed, model.focus)
	}
}

func TestPreviewFocusScrollsLowerPanel(t *testing.T) {
	resource := navNode{
		Key:       "file",
		Kind:      navNodeResource,
		Title:     "Slides",
		Openable:  true,
		Printable: true,
		Resource:  &moodle.Resource{Name: "Slides", Type: "resource", FileType: "pdf"},
	}
	model := tuiModel{
		root:           navNode{Key: "root", Kind: navNodeHome, Title: "Moodle"},
		focus:          focusPreview,
		lastUpperFocus: focusTree,
		previewCache: map[string]string{
			"file": strings.Join([]string{
				"line 1",
				"line 2",
				"line 3",
				"line 4",
				"line 5",
				"line 6",
				"line 7",
				"line 8",
				"line 9",
				"line 10",
				"line 11",
				"line 12",
			}, "\n"),
		},
		nodeByKey:   map[string]navNode{"file": resource},
		parentByKey: map[string]string{"file": "root"},
		selectedKey: "file",
		width:       100,
		height:      20,
	}

	before := model.renderBottomPane(80, 10)
	if !strings.Contains(before, "line 1") {
		t.Fatalf("expected initial preview to show first lines, got %q", before)
	}

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	model = next.(tuiModel)

	if model.previewViewport.YOffset == 0 {
		t.Fatalf("expected preview scroll to advance")
	}
	after := model.renderBottomPane(80, 10)
	if !strings.Contains(after, "line 12") {
		t.Fatalf("expected later content after scrolling, got %q", after)
	}
}
