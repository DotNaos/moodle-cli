package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	ver "github.com/DotNaos/moodle-services/internal/version"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type tuiNavigator interface {
	Root() navNode
	Children(navNode) ([]navNode, error)
	Preview(navNode) string
	Open(navNode) (string, error)
	Print(navNode) (string, error)
	Download(navNode, string) (string, error)
}

type paneFocus string

const (
	focusTree    paneFocus = "tree"
	focusRight   paneFocus = "right"
	focusPreview paneFocus = "preview"
)

type previewMode string

const (
	previewModeReader   previewMode = "reader"
	previewModeMarkdown previewMode = "markdown"
)

type treeRow struct {
	Node       navNode
	Depth      int
	HasKids    bool
	Expanded   bool
	Loading    bool
	TopLevel   bool
	QuickBand  bool
	BrowseBand bool
}

type rightEntryKind string

const (
	rightEntryNode   rightEntryKind = "node"
	rightEntryAction rightEntryKind = "action"
)

type rightEntry struct {
	Kind        rightEntryKind
	Label       string
	Node        navNode
	Action      string
	Description string
}

type tuiModel struct {
	nav             tuiNavigator
	root            navNode
	width           int
	height          int
	status          string
	filterMode      bool
	filterInput     string
	leftFilter      string
	rightFilter     string
	previewCache    map[string]string
	previewBusy     string
	previewMode     previewMode
	dialog          *downloadDialog
	focus           paneFocus
	selectedKey     string
	rightSelected   int
	expanded        map[string]bool
	nodeByKey       map[string]navNode
	parentByKey     map[string]string
	childCache      map[string][]navNode
	childBusy       map[string]bool
	childErrors     map[string]string
	spinnerFrame    int
	previewViewport viewport.Model
	topCollapsed    bool
	lastUpperFocus  paneFocus
}

type tuiOpenMsg struct{ Err error }

type tuiPreviewMsg struct {
	Key  string
	Text string
	Err  error
}

type tuiChildrenMsg struct {
	Parent   navNode
	Children []navNode
	Err      error
}

type tuiSpinnerMsg struct{}

type tuiDownloadMsg struct {
	Path string
	Err  error
}

type downloadDialog struct {
	target   navNode
	cwd      string
	entries  []downloadDialogEntry
	selected int
}

type downloadDialogEntry struct {
	Label string
	Path  string
	Save  bool
	IsDir bool
}

var tuiWorkspace string
var tuiAt string
var launchTUI = runTUI

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open the Moodle terminal UI",
	Args:  cobra.NoArgs,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return launchTUI(selectorOptions{Workspace: tuiWorkspace, At: tuiAt})
	},
}

func init() {
	tuiCmd.Flags().StringVar(&tuiWorkspace, "workspace", "", "Optional workspace root for current-course helpers")
	tuiCmd.Flags().StringVar(&tuiAt, "at", "", "Override current time for testing (RFC3339)")
	markInteractiveOnly(tuiCmd)
}

func runTUI(options selectorOptions) error {
	client, err := ensureAuthenticatedClient()
	if err != nil {
		return err
	}
	service, err := newNavService(client, options)
	if err != nil {
		return err
	}
	root := service.Root()
	model := tuiModel{
		nav:            service,
		root:           root,
		focus:          focusTree,
		lastUpperFocus: focusTree,
		previewCache:   map[string]string{},
		expanded:       map[string]bool{root.Key: true},
		nodeByKey:      map[string]navNode{root.Key: root},
		parentByKey:    map[string]string{root.Key: ""},
		childCache:     map[string][]navNode{},
		childBusy:      map[string]bool{},
		childErrors:    map[string]string{},
	}
	model.selectedKey = root.Key
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err = program.Run()
	return err
}

func (m tuiModel) Init() tea.Cmd {
	m.ensureState()
	return tea.Batch(spinnerTickCmd(), m.requestChildrenCmd(m.root))
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.ensureState()
	m.syncPreviewViewportState()
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tuiSpinnerMsg:
		m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerFrames)
		return m, spinnerTickCmd()
	case tuiOpenMsg:
		if msg.Err != nil {
			logDebug("tui.open", "result: error", "error: "+msg.Err.Error())
			m.status = presentUIError("tui.open", msg.Err)
		} else {
			logDebug("tui.open", "result: success")
			m.status = "Opened."
		}
		return m, nil
	case tuiPreviewMsg:
		if msg.Err != nil {
			logDebug("tui.print", "result: error", "error: "+msg.Err.Error())
			m.status = presentUIError("tui.print", msg.Err)
			m.previewBusy = ""
			return m, nil
		}
		m.previewBusy = ""
		m.previewCache[msg.Key] = msg.Text
		logDebug("tui.print", "result: success", "key: "+msg.Key)
		m.status = "Preview loaded."
		return m, nil
	case tuiDownloadMsg:
		if msg.Err != nil {
			logDebug("tui.download", "result: error", "error: "+msg.Err.Error())
			m.status = presentUIError("tui.download", msg.Err)
			return m, nil
		}
		m.dialog = nil
		logDebug("tui.download", "result: success", "path: "+msg.Path)
		m.status = "Saved to " + msg.Path
		return m, nil
	case tuiChildrenMsg:
		delete(m.childBusy, msg.Parent.Key)
		if msg.Err != nil {
			logDebug("tui.children", "result: error", "parent: "+msg.Parent.Key, "error: "+msg.Err.Error())
			displayErr := presentUIError("tui.children", msg.Err, "parent: "+msg.Parent.Key)
			m.childErrors[msg.Parent.Key] = displayErr
			m.status = displayErr
			return m, nil
		}
		logDebug("tui.children", "result: success", "parent: "+msg.Parent.Key, fmt.Sprintf("count: %d", len(msg.Children)))
		delete(m.childErrors, msg.Parent.Key)
		m.childCache[msg.Parent.Key] = msg.Children
		for _, child := range msg.Children {
			m.nodeByKey[child.Key] = child
			if _, ok := m.parentByKey[child.Key]; !ok {
				m.parentByKey[child.Key] = msg.Parent.Key
			}
		}
		if msg.Parent.Key == m.root.Key && m.selectedKey == m.root.Key {
			if len(msg.Children) > 0 {
				m.selectedKey = msg.Children[0].Key
			}
			return m, m.selectionChangedCmd()
		}
		return m, nil
	case tea.KeyMsg:
		if m.dialog != nil {
			return m.updateDialog(msg)
		}
		if m.filterMode {
			return m.updateFilter(msg)
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "ctrl+h":
			m.focus = focusTree
			m.lastUpperFocus = focusTree
			return m, nil
		case "ctrl+l":
			m.focus = focusRight
			m.lastUpperFocus = focusRight
			return m, nil
		case "ctrl+j":
			m.lastUpperFocus = m.currentUpperFocus()
			m.focus = focusPreview
			return m, nil
		case "ctrl+k":
			if m.focus == focusPreview {
				m.focus = m.returnUpperFocus()
				return m, nil
			}
		case "ctrl+w":
			m.topCollapsed = !m.topCollapsed
			if m.topCollapsed {
				m.lastUpperFocus = m.currentUpperFocus()
				m.focus = focusPreview
			} else if m.focus == focusPreview {
				m.focus = m.returnUpperFocus()
			}
			return m, nil
		case "m":
			m.togglePreviewMode()
			return m, nil
		case "/":
			if m.focus == focusPreview {
				return m, nil
			}
			m.filterMode = true
			m.filterInput = m.currentFilter()
			return m, nil
		case "g":
			if m.focus == focusPreview {
				m.previewViewport.GotoTop()
				return m, nil
			}
			m.setSelection(0)
			return m, m.selectionChangedCmd()
		case "G":
			if m.focus == focusPreview {
				m.previewViewport.GotoBottom()
				return m, nil
			}
			m.setSelection(m.itemCount() - 1)
			return m, m.selectionChangedCmd()
		case "j", "down":
			if m.focus == focusPreview {
				m.previewViewport.LineDown(1)
				return m, nil
			}
			m.moveSelection(1)
			return m, m.selectionChangedCmd()
		case "k", "up":
			if m.focus == focusPreview {
				m.previewViewport.LineUp(1)
				return m, nil
			}
			m.moveSelection(-1)
			return m, m.selectionChangedCmd()
		case "ctrl+d":
			if m.focus == focusPreview {
				m.previewViewport.HalfPageDown()
				return m, nil
			}
		case "ctrl+u":
			if m.focus == focusPreview {
				m.previewViewport.HalfPageUp()
				return m, nil
			}
		case "h", "left":
			if m.focus == focusPreview {
				return m, nil
			}
			model, cmd := m.handleLeft()
			next := model.(tuiModel)
			return next, tea.Batch(cmd, next.selectionChangedCmd())
		case "l", "right":
			if m.focus == focusPreview {
				return m, nil
			}
			model, cmd := m.handleRight()
			next := model.(tuiModel)
			return next, tea.Batch(cmd, next.selectionChangedCmd())
		case "enter":
			if m.focus == focusPreview {
				return m, nil
			}
			if m.focus == focusRight {
				if entry, ok := m.selectedRightEntry(); ok && entry.Kind == rightEntryAction {
					return m.handleEnter()
				}
			}
			model, cmd := m.handleEnter()
			next := model.(tuiModel)
			return next, tea.Batch(cmd, next.selectionChangedCmd())
		case "o":
			return m.handleActionShortcut("open")
		case "p":
			return m.handleActionShortcut("print")
		case "d":
			return m.handleActionShortcut("download")
		}
	}
	return m, nil
}

func (m tuiModel) updateDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	dialog := m.dialog
	switch msg.String() {
	case "esc":
		m.dialog = nil
		m.status = ""
		return m, nil
	case "j", "down":
		if dialog.selected < len(dialog.entries)-1 {
			dialog.selected++
		}
		return m, nil
	case "k", "up":
		if dialog.selected > 0 {
			dialog.selected--
		}
		return m, nil
	case "g":
		dialog.selected = 0
		return m, nil
	case "G":
		if len(dialog.entries) > 0 {
			dialog.selected = len(dialog.entries) - 1
		}
		return m, nil
	case "h", "left":
		parent := filepath.Dir(dialog.cwd)
		if parent == dialog.cwd {
			return m, nil
		}
		dialog.cwd = parent
		dialog.selected = 0
		if err := dialog.reload(); err != nil {
			m.status = err.Error()
		}
		return m, nil
	case "l", "right", "enter":
		if len(dialog.entries) == 0 {
			return m, nil
		}
		entry := dialog.entries[dialog.selected]
		if entry.Save {
			return m, m.downloadCmd(dialog.target, dialog.cwd)
		}
		if entry.IsDir {
			dialog.cwd = entry.Path
			dialog.selected = 0
			if err := dialog.reload(); err != nil {
				m.status = err.Error()
			}
		}
	}
	return m, nil
}

func (m tuiModel) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.setCurrentFilter("")
		m.filterMode = false
		m.filterInput = ""
		m.setSelection(0)
		return m, nil
	case "enter":
		m.filterMode = false
		return m, nil
	case "backspace":
		if len(m.filterInput) > 0 {
			m.filterInput = m.filterInput[:len(m.filterInput)-1]
		}
	default:
		if msg.Type == tea.KeyRunes {
			m.filterInput += msg.String()
		}
	}
	m.setCurrentFilter(strings.TrimSpace(m.filterInput))
	m.setSelection(0)
	return m, nil
}

func (m tuiModel) View() string {
	m.ensureState()
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f8fafc")).Render("Moodle TUI")
	header += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#94a3b8")).Render(strings.Join(m.breadcrumb(), " > "))
	footer := m.renderFooter()
	leftWidth, rightWidth, matrixHeight, previewHeight := m.layoutSizes(header, footer)
	preview := m.renderBottomPane(leftWidth+rightWidth+2, previewHeight)
	if m.topCollapsed {
		return header + "\n\n" + preview + "\n\n" + footer
	}
	left := m.renderTreePane(leftWidth, matrixHeight)
	right := m.renderRightPane(rightWidth, matrixHeight)
	return header + "\n\n" + lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right) + "\n\n" + preview + "\n\n" + footer
}

func (m tuiModel) layoutSizes(header string, footer string) (int, int, int, int) {
	leftWidth := max(40, m.width/2-2)
	rightWidth := max(40, m.width-leftWidth-2)
	if m.width == 0 {
		leftWidth = 48
		rightWidth = 48
	}
	matrixHeight := 12
	if m.height > 0 {
		matrixHeight = max(10, (m.height-8)/2)
	}
	previewHeight := 10
	if m.height > 0 {
		if m.topCollapsed {
			previewHeight = max(10, m.height-countLines(header)-countLines(footer)-5)
		} else {
			previewHeight = max(6, m.height-matrixHeight-8)
		}
	}
	return leftWidth, rightWidth, matrixHeight, previewHeight
}

func (m *tuiModel) ensureState() {
	if m.previewCache == nil {
		m.previewCache = map[string]string{}
	}
	if m.expanded == nil {
		m.expanded = map[string]bool{}
	}
	if m.nodeByKey == nil {
		m.nodeByKey = map[string]navNode{}
	}
	if m.parentByKey == nil {
		m.parentByKey = map[string]string{}
	}
	if m.childCache == nil {
		m.childCache = map[string][]navNode{}
	}
	if m.childBusy == nil {
		m.childBusy = map[string]bool{}
	}
	if m.childErrors == nil {
		m.childErrors = map[string]string{}
	}
	if m.lastUpperFocus == "" {
		m.lastUpperFocus = focusTree
	}
	if m.previewMode == "" {
		m.previewMode = previewModeReader
	}
	if m.previewViewport.Width == 0 && m.previewViewport.Height == 0 {
		m.previewViewport = viewport.New(0, 0)
		m.previewViewport.KeyMap = viewport.KeyMap{}
	}
}

func (m *tuiModel) ensureChildren(node navNode) ([]navNode, error) {
	children, ok := m.childCache[node.Key]
	if !ok {
		return nil, fmt.Errorf("children for %q are not loaded", node.Title)
	}
	return children, nil
}

func (m tuiModel) selectedNode() navNode {
	if node, ok := m.nodeByKey[m.selectedKey]; ok {
		return node
	}
	return m.root
}

func (m tuiModel) breadcrumb() []string {
	selected := m.selectedNode()
	if selected.Key == "" {
		return []string{"Moodle"}
	}
	parts := []string{}
	for node := selected; node.Key != ""; {
		if node.Title != "" {
			parts = append(parts, node.Title)
		}
		parentKey := m.parentByKey[node.Key]
		if parentKey == "" {
			break
		}
		node = m.nodeByKey[parentKey]
	}
	slices.Reverse(parts)
	if len(parts) == 0 || parts[0] != "Moodle" {
		return append([]string{"Moodle"}, parts...)
	}
	return parts
}

func (m tuiModel) currentFilter() string {
	if m.focus == focusRight {
		return m.rightFilter
	}
	return m.leftFilter
}

func (m tuiModel) currentUpperFocus() paneFocus {
	if m.focus == focusRight {
		return focusRight
	}
	return focusTree
}

func (m tuiModel) returnUpperFocus() paneFocus {
	if m.lastUpperFocus == focusRight {
		return focusRight
	}
	return focusTree
}

func (m *tuiModel) setCurrentFilter(value string) {
	if m.focus == focusRight {
		m.rightFilter = value
		m.rightSelected = 0
		return
	}
	m.leftFilter = value
}

func (m tuiModel) treeRows() []treeRow {
	rows := []treeRow{}
	children, ok := m.childCache[m.root.Key]
	if !ok {
		return rows
	}
	pathSet := m.selectedPathSet()
	for _, child := range children {
		m.appendTreeRow(&rows, child, 0, pathSet)
	}
	return rows
}

func (m tuiModel) appendTreeRow(rows *[]treeRow, node navNode, depth int, pathSet map[string]bool) {
	children, cached := m.childCache[node.Key]
	hasKids := nodeMayHaveChildren(node)
	if cached {
		hasKids = len(children) > 0
	}
	row := treeRow{
		Node:       node,
		Depth:      depth,
		HasKids:    hasKids,
		Expanded:   m.expanded[node.Key],
		Loading:    m.childBusy[node.Key],
		TopLevel:   depth == 0,
		QuickBand:  depth == 0 && (node.Kind == navNodeCurrent || node.Kind == navNodeToday),
		BrowseBand: depth == 0 && (node.Kind == navNodeSemesters || node.Kind == navNodeTimetable),
	}
	*rows = append(*rows, row)
	if cached && hasKids && m.expanded[node.Key] && pathSet[node.Key] {
		for _, child := range children {
			m.appendTreeRow(rows, child, depth+1, pathSet)
		}
	}
}

func (m tuiModel) selectedPathSet() map[string]bool {
	out := map[string]bool{m.root.Key: true}
	for current := m.selectedKey; current != ""; current = m.parentByKey[current] {
		out[current] = true
		if current == m.root.Key {
			break
		}
	}
	return out
}

func (m tuiModel) visibleTreeRows() []treeRow {
	rows := m.treeRows()
	if m.leftFilter == "" {
		return rows
	}
	needle := strings.ToLower(m.leftFilter)
	filtered := make([]treeRow, 0, len(rows))
	for _, row := range rows {
		if strings.Contains(strings.ToLower(row.Node.Title), needle) || strings.Contains(strings.ToLower(row.Node.Subtitle), needle) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func (m tuiModel) rightEntries() []rightEntry {
	selected := m.selectedNode()
	if selected.Resource != nil {
		entries := []rightEntry{
			{Kind: rightEntryAction, Label: "Open", Action: "open", Description: "Open this file in the default app."},
			{Kind: rightEntryAction, Label: "Print", Action: "print", Description: "Show the file text in the lower panel."},
			{Kind: rightEntryAction, Label: "Download", Action: "download", Description: "Choose a save folder in the lower panel."},
		}
		return filterRightEntries(entries, m.rightFilter)
	}
	children, ok := m.childCache[selected.Key]
	if !ok {
		return nil
	}
	entries := make([]rightEntry, 0, len(children))
	for _, child := range children {
		entries = append(entries, rightEntry{Kind: rightEntryNode, Label: child.Title, Node: child})
	}
	return filterRightEntries(entries, m.rightFilter)
}

func filterRightEntries(entries []rightEntry, filter string) []rightEntry {
	if filter == "" {
		return entries
	}
	needle := strings.ToLower(filter)
	filtered := make([]rightEntry, 0, len(entries))
	for _, entry := range entries {
		text := entry.Label
		if entry.Kind == rightEntryNode {
			text += " " + entry.Node.Subtitle
		}
		if strings.Contains(strings.ToLower(text), needle) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func (m tuiModel) selectedRightEntry() (rightEntry, bool) {
	entries := m.rightEntries()
	if len(entries) == 0 {
		return rightEntry{}, false
	}
	index := m.rightSelected
	if index < 0 || index >= len(entries) {
		index = 0
	}
	return entries[index], true
}

func (m tuiModel) itemCount() int {
	if m.focus == focusRight {
		return len(m.rightEntries())
	}
	return len(m.visibleTreeRows())
}

func (m *tuiModel) moveSelection(delta int) {
	if m.focus == focusRight {
		entries := m.rightEntries()
		if len(entries) == 0 {
			return
		}
		m.rightSelected += delta
		if m.rightSelected < 0 {
			m.rightSelected = 0
		}
		if m.rightSelected >= len(entries) {
			m.rightSelected = len(entries) - 1
		}
		return
	}
	rows := m.visibleTreeRows()
	if len(rows) == 0 {
		return
	}
	index := 0
	for i, row := range rows {
		if row.Node.Key == m.selectedKey {
			index = i
			break
		}
	}
	index += delta
	if index < 0 {
		index = 0
	}
	if index >= len(rows) {
		index = len(rows) - 1
	}
	m.selectedKey = rows[index].Node.Key
	m.rightSelected = 0
	m.resetPreviewPosition()
}

func (m *tuiModel) setSelection(index int) {
	if m.focus == focusRight {
		entries := m.rightEntries()
		if len(entries) == 0 {
			return
		}
		if index < 0 {
			index = 0
		}
		if index >= len(entries) {
			index = len(entries) - 1
		}
		m.rightSelected = index
		return
	}
	rows := m.visibleTreeRows()
	if len(rows) == 0 {
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= len(rows) {
		index = len(rows) - 1
	}
	m.selectedKey = rows[index].Node.Key
	m.rightSelected = 0
	m.resetPreviewPosition()
}

func (m tuiModel) handleLeft() (tea.Model, tea.Cmd) {
	if m.focus == focusRight {
		m.lastUpperFocus = focusTree
		m.focus = focusTree
		return m, nil
	}
	selected := m.selectedNode()
	children := m.childCache[selected.Key]
	if len(children) > 0 && m.expanded[selected.Key] {
		delete(m.expanded, selected.Key)
		return m, nil
	}
	parentKey := m.parentByKey[selected.Key]
	if parentKey != "" && parentKey != m.root.Key {
		m.selectedKey = parentKey
		m.rightSelected = 0
	}
	return m, nil
}

func (m tuiModel) handleRight() (tea.Model, tea.Cmd) {
	cmd := m.requestChildrenCmd(m.selectedNode())
	m.lastUpperFocus = focusRight
	m.focus = focusRight
	if m.rightSelected >= len(m.rightEntries()) {
		m.rightSelected = 0
	}
	return m, cmd
}

func (m tuiModel) handleEnter() (tea.Model, tea.Cmd) {
	if m.focus == focusRight {
		return m.commitRightSelection()
	}
	selected := m.selectedNode()
	if nodeMayHaveChildren(selected) {
		if children, ok := m.childCache[selected.Key]; ok {
			if len(children) > 0 {
				if m.expanded[selected.Key] {
					delete(m.expanded, selected.Key)
				} else {
					m.expanded[selected.Key] = true
				}
				return m, nil
			}
		} else {
			m.expanded[selected.Key] = true
			return m, m.requestChildrenCmd(selected)
		}
	}
	if selected.Resource != nil {
		m.lastUpperFocus = focusRight
		m.focus = focusRight
		m.rightSelected = 0
	}
	return m, nil
}

func (m tuiModel) commitRightSelection() (tea.Model, tea.Cmd) {
	entry, ok := m.selectedRightEntry()
	if !ok {
		return m, nil
	}
	if entry.Kind == rightEntryAction {
		return m.runAction(entry.Action, m.selectedNode())
	}
	node := entry.Node
	m.selectedKey = node.Key
	m.rightSelected = 0
	m.resetPreviewPosition()
	m.expandVisiblePath(node.Key)
	if children, ok := m.childCache[node.Key]; ok {
		if len(children) > 0 {
			m.expanded[node.Key] = true
			m.focus = focusRight
			return m, nil
		}
	} else if nodeMayHaveChildren(node) {
		m.expanded[node.Key] = true
		m.focus = focusRight
		return m, m.requestChildrenCmd(node)
	}
	if node.Resource != nil {
		m.focus = focusRight
		return m, nil
	}
	if node.Openable {
		return m, m.openCmd(node)
	}
	return m, nil
}

func (m *tuiModel) expandVisiblePath(nodeKey string) {
	for current := nodeKey; current != ""; current = m.parentByKey[current] {
		if current == m.root.Key {
			break
		}
		parent := m.parentByKey[current]
		if parent == "" {
			break
		}
		m.expanded[parent] = true
	}
}

func (m tuiModel) handleActionShortcut(action string) (tea.Model, tea.Cmd) {
	target := m.selectedNode()
	if m.focus == focusRight {
		if entry, ok := m.selectedRightEntry(); ok {
			if entry.Kind == rightEntryAction {
				return m.runAction(action, target)
			}
			target = entry.Node
		}
	}
	return m.runAction(action, target)
}

func (m tuiModel) runAction(action string, target navNode) (tea.Model, tea.Cmd) {
	switch action {
	case "open":
		if !target.Openable {
			return m, nil
		}
		logDebug("tui.open", "action: trigger", "key: "+target.Key, "title: "+target.Title)
		m.status = "Opening..."
		return m, m.openCmd(target)
	case "print":
		if !target.Printable {
			m.status = "Selected item cannot be previewed."
			return m, nil
		}
		logDebug("tui.print", "action: trigger", "key: "+target.Key, "title: "+target.Title)
		m.previewBusy = target.Key
		return m, m.printCmd(target)
	case "download":
		if target.Resource == nil {
			return m, nil
		}
		dialog, err := newDownloadDialog(target)
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		m.dialog = dialog
		logDebug("tui.download", "action: trigger", "key: "+target.Key, "title: "+target.Title, "cwd: "+dialog.cwd)
		m.status = "Download dialog"
		return m, nil
	default:
		return m, nil
	}
}

func (m tuiModel) openCmd(node navNode) tea.Cmd {
	return func() tea.Msg {
		_, err := m.nav.Open(node)
		return tuiOpenMsg{Err: err}
	}
}

func (m tuiModel) downloadCmd(node navNode, outputPath string) tea.Cmd {
	return func() tea.Msg {
		path, err := m.nav.Download(node, outputPath)
		return tuiDownloadMsg{Path: path, Err: err}
	}
}

func (m tuiModel) printCmd(node navNode) tea.Cmd {
	return func() tea.Msg {
		text, err := m.nav.Print(node)
		if err != nil {
			return tuiPreviewMsg{Err: err}
		}
		text = strings.TrimSpace(text)
		return tuiPreviewMsg{Key: node.Key, Text: text}
	}
}

func (m tuiModel) previewCmd(node navNode) tea.Cmd {
	return func() tea.Msg {
		text := strings.TrimSpace(m.nav.Preview(node))
		if text == "" {
			text = "No preview available."
		}
		return tuiPreviewMsg{Key: node.Key, Text: text}
	}
}

func (m *tuiModel) autoPreviewCmd() tea.Cmd {
	target, ok := m.previewTargetNode()
	if !ok || !nodeSupportsAsyncPreview(target) {
		return nil
	}
	if _, ok := m.previewCache[target.Key]; ok {
		return nil
	}
	if m.previewBusy == target.Key {
		return nil
	}
	m.previewBusy = target.Key
	if target.Printable {
		return m.printCmd(target)
	}
	return m.previewCmd(target)
}

func (m *tuiModel) selectionChangedCmd() tea.Cmd {
	return tea.Batch(m.requestChildrenCmd(m.selectedNode()), m.autoPreviewCmd())
}

func (m tuiModel) renderTreePane(width int, height int) string {
	rows := m.visibleTreeRows()
	title := "Navigation"
	if len(rows) == 0 {
		body := paneMutedStyle.Render("(loading)")
		if errText, ok := m.childErrors[m.root.Key]; ok {
			body = paneMutedStyle.Render(errText)
		}
		if _, ok := m.childCache[m.root.Key]; ok {
			body = paneMutedStyle.Render("(empty)")
		}
		return paneBox(m.focus == focusTree).Width(width).Height(height).Render(paneTitleStyle.Render(title) + "\n\n" + body)
	}
	lines := []string{}
	selectedLine := 0
	for _, row := range rows {
		if row.TopLevel && row.Node.Kind == navNodeCurrent {
			lines = append(lines, paneSubtitleStyle.Render("Quick Access"))
		}
		if row.TopLevel && row.Node.Kind == navNodeSemesters {
			lines = append(lines, paneSubtitleStyle.Render("Browse"))
		}
		isSelected := m.focus == focusTree && row.Node.Key == m.selectedKey
		if isSelected {
			selectedLine = len(lines)
		}
		lines = append(lines, renderTreeRow(row, isSelected, width-8, m.leftFilter, m.spinnerFrame))
	}
	header := paneTitleStyle.Render(title)
	content := header + "\n\n" + joinBlocksForHeight(lines, selectedLine, paneBodyLines(height, header))
	return paneBox(m.focus == focusTree).Width(width).Height(height).Render(content)
}

func (m tuiModel) renderRightPane(width int, height int) string {
	selected := m.selectedNode()
	if selected.Key == "" {
		return m.renderPreviewViewport("Details", "", width, height)
	}
	entries := m.rightEntries()
	title := selected.Title
	if title == "" {
		title = "Details"
	}
	header := paneTitleStyle.Render(truncateRunes(title, max(8, width-8)))
	if len(entries) == 0 {
		body := "(empty)"
		if m.childBusy[selected.Key] {
			body = m.loadingLabel("Loading items")
		} else if errText, ok := m.childErrors[selected.Key]; ok {
			body = errText
		}
		return paneBox(m.focus == focusRight).Width(width).Height(height).Render(header + "\n\n" + paneMutedStyle.Render(body))
	}
	lines := make([]string, 0, len(entries))
	for index, entry := range entries {
		active := m.focus == focusRight && index == m.rightSelected
		lines = append(lines, renderRightRow(entry, active, width-8, m.rightFilter))
	}
	selectedIndex := 0
	if m.focus == focusRight {
		selectedIndex = m.rightSelected
	}
	content := header + "\n\n" + joinBlocksForHeight(lines, selectedIndex, paneBodyLines(height, header))
	return paneBox(m.focus == focusRight).Width(width).Height(height).Render(content)
}

func (m tuiModel) renderBottomPane(width int, height int) string {
	if m.dialog != nil {
		return m.renderDownloadDialog(width, height)
	}
	title, body := m.previewSubject()
	return m.renderPreviewViewport(title, body, width, height)
}

func (m tuiModel) previewSubject() (string, string) {
	if m.focus == focusRight {
		if entry, ok := m.selectedRightEntry(); ok {
			if entry.Kind == rightEntryAction {
				if entry.Action == "print" && (m.previewBusy == m.selectedNode().Key || m.previewCache[m.selectedNode().Key] != "") {
					return m.nodePreview(m.selectedNode())
				}
				return entry.Label, entry.Description
			}
			return m.nodePreview(entry.Node)
		}
	}
	return m.nodePreview(m.selectedNode())
}

func (m tuiModel) previewTargetNode() (navNode, bool) {
	if m.focus == focusRight {
		if entry, ok := m.selectedRightEntry(); ok && entry.Kind == rightEntryNode {
			return entry.Node, true
		}
	}
	node := m.selectedNode()
	if node.Key == "" {
		return navNode{}, false
	}
	return node, true
}

func (m tuiModel) nodePreview(node navNode) (string, string) {
	title := node.Title
	if title == "" {
		title = "Details"
	}
	if node.Resource != nil {
		return title, m.resourcePreview(node)
	}
	return title, m.nodeBodyPreview(node)
}

func (m tuiModel) nodeBodyPreview(node navNode) string {
	base := strings.TrimSpace(m.staticNodePreview(node))
	if node.Kind == navNodeCourse {
		if text, ok := m.previewCache[node.Key]; ok {
			if m.preferDocumentPreview() {
				return m.formatPreviewBody(node, text)
			}
			return m.formatPreviewBody(node, appendPreviewBlock(base, text))
		}
		if m.previewBusy == node.Key {
			return m.formatPreviewBody(node, appendPreviewBlock(base, m.loadingLabel("Loading course page")))
		}
	}
	if !nodeMayHaveChildren(node) {
		return m.formatPreviewBody(node, base)
	}
	children, ok := m.childCache[node.Key]
	if !ok {
		if errText, hasErr := m.childErrors[node.Key]; hasErr {
			return m.formatPreviewBody(node, appendPreviewBlock(base, errText))
		}
		if m.childBusy[node.Key] {
			return m.formatPreviewBody(node, appendPreviewBlock(base, m.loadingLabel("Loading")))
		}
		return m.formatPreviewBody(node, appendPreviewBlock(base, "Open this item to load its contents."))
	}
	if len(children) == 0 {
		return m.formatPreviewBody(node, appendPreviewBlock(base, "No items."))
	}
	switch node.Kind {
	case navNodeToday, navNodeWeek:
		return m.formatPreviewBody(node, appendPreviewBlock(base, previewFromChildren(children)))
	default:
		return m.formatPreviewBody(node, appendPreviewBlock(base, m.outlinePreview(children, 2)))
	}
}

func (m tuiModel) outlinePreview(children []navNode, depth int) string {
	lines := []string{}
	for _, child := range children {
		lines = append(lines, outlineNode(child, 0))
		if depth > 1 {
			if grandChildren, ok := m.childCache[child.Key]; ok {
				for _, grandChild := range grandChildren {
					lines = append(lines, outlineNode(grandChild, 1))
				}
			}
		}
	}
	return strings.Join(lines, "\n")
}

func (m tuiModel) staticNodePreview(node navNode) string {
	switch node.Kind {
	case navNodeHome:
		return "Current jumps to the active lecture. Today shows today’s timetable. Semesters is full course browsing."
	case navNodeCurrent:
		return "Current lecture and matching course items."
	case navNodeToday:
		return "Today’s lecture entries."
	case navNodeTimetable:
		return "Browse timetable weeks."
	case navNodeWeek:
		if node.PreviewText != "" {
			return node.PreviewText
		}
		return node.Subtitle
	case navNodeSemesters:
		return "Browse Moodle courses grouped by semester."
	case navNodeSemester:
		return fmt.Sprintf("Semester %s", node.Title)
	case navNodeCourse:
		return fmt.Sprintf("Course: %s", node.Title)
	case navNodeEvent:
		if node.PreviewText != "" {
			return node.PreviewText
		}
		return node.Title
	case navNodeSections:
		return "Sections in Moodle order."
	case navNodeSection:
		return fmt.Sprintf("Section: %s", node.Title)
	case navNodeItems:
		if node.UseCurrentSort {
			return "Items ordered for the current lecture: newest relevant file first."
		}
		return "Items in Moodle order."
	default:
		return ""
	}
}

func (m tuiModel) resourcePreview(node navNode) string {
	body := resourcePreviewText(node)
	if text, ok := m.previewCache[node.Key]; ok {
		if m.preferDocumentPreview() {
			return m.formatPreviewBody(node, text)
		}
		return m.formatPreviewBody(node, appendPreviewBlock(body, text))
	}
	if m.previewBusy == node.Key {
		return m.formatPreviewBody(node, appendPreviewBlock(body, m.loadingLabel("Loading preview")))
	}
	return m.formatPreviewBody(node, body)
}

func nodeSupportsAsyncPreview(node navNode) bool {
	return node.Printable || node.Kind == navNodeCourse
}

func (m tuiModel) preferDocumentPreview() bool {
	return m.topCollapsed || m.focus == focusPreview
}

func (m tuiModel) renderFooter() string {
	parts := []string{"h/j/k/l or arrows", "Ctrl+h/j/k/l=panes", "Ctrl+w=reader", "m=markdown", "Enter=toggle/drill", "/=filter", "o=open", "p=preview", "d=download", "q=quit"}
	if m.focus == focusPreview {
		parts = append(parts, fmt.Sprintf("preview %d/%d", m.previewViewport.YOffset+1, m.maxPreviewScroll()+1))
	}
	parts = append(parts, "mode:"+string(m.previewMode))
	if m.filterMode {
		parts = append(parts, "/"+m.filterInput)
	} else if m.dialog != nil {
		parts = append(parts, "dialog active")
	} else if m.status != "" {
		parts = append(parts, m.status)
	}
	parts = append(parts, tuiVersionLabel())
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#94a3b8")).Render(strings.Join(parts, " · "))
}

func (m *tuiModel) togglePreviewMode() {
	switch m.previewMode {
	case previewModeMarkdown:
		m.previewMode = previewModeReader
		m.status = "Reader mode"
	default:
		m.previewMode = previewModeMarkdown
		m.status = "Markdown mode"
	}
	m.resetPreviewPosition()
}

func tuiVersionLabel() string {
	version := strings.TrimSpace(ver.Version())
	if version == "" {
		version = ver.DefaultVersion
	}
	buildDate := strings.TrimSpace(ver.EffectiveBuildDate())
	if buildDate == "" {
		buildDate = ver.DefaultBuildDate
	}
	if ver.IsDev() {
		if buildDate != ver.DefaultBuildDate {
			return "dev build " + buildDate
		}
		return "dev build"
	}
	parsed, err := ver.ParseSemver(version)
	if err == nil && parsed.Prerelease != "" {
		if buildDate != ver.DefaultBuildDate {
			return version + " preview built " + buildDate
		}
		return version + " preview"
	}
	return version
}

func renderTreeRow(row treeRow, selected bool, width int, filter string, spinnerFrame int) string {
	title := row.Node.Title
	if title == "" {
		title = "(untitled)"
	}
	fold := "  "
	if row.HasKids {
		if row.Expanded {
			fold = "▾ "
		} else {
			fold = "▸ "
		}
	}
	indent := treeIndent(row.Depth)
	prefix := indent + fold
	display := truncateRunes(prefix+title, max(1, width-2))
	if filter != "" {
		display = highlightMatch(display, filter)
	}
	if row.Loading {
		display = truncateRunes(display+" "+spinnerFrames[spinnerFrame%len(spinnerFrames)], max(1, width-2))
	}
	if selected {
		return selectedRowStyle.Render("› " + display)
	}
	return normalRowStyle.Render("  " + display)
}

func treeIndent(depth int) string {
	if depth <= 0 {
		return ""
	}
	return strings.Repeat("   ", depth)
}

func renderRightRow(entry rightEntry, selected bool, width int, filter string) string {
	label := entry.Label
	if label == "" {
		label = "(untitled)"
	}
	display := truncateRunes(label, width)
	if filter != "" {
		display = highlightMatch(display, filter)
	}
	if selected {
		return selectedRowStyle.Render("  " + display)
	}
	return normalRowStyle.Render(display)
}

func outlineNode(node navNode, depth int) string {
	prefix := strings.Repeat("  ", depth) + "- "
	text := node.Title
	if node.Subtitle != "" {
		text += " · " + node.Subtitle
	}
	return prefix + strings.TrimSpace(text)
}

func paneBox(focused bool) lipgloss.Style {
	if focused {
		return paneBoxStyle.Copy().BorderForeground(lipgloss.Color("#7dd3fc"))
	}
	return paneBoxStyle
}

func (m *tuiModel) renderPreviewViewport(title string, text string, width int, height int) string {
	box := paneBox(m.focus == focusPreview || m.topCollapsed)
	header := paneTitleStyle.Render(truncateRunes(title, max(8, width-8)))
	availableLines := paneBodyLines(height, header)
	contentWidth := max(1, width-box.GetHorizontalFrameSize())
	m.previewViewport.Width = contentWidth
	m.previewViewport.Height = availableLines
	m.previewViewport.SetContent(strings.TrimSpace(text))
	if m.previewViewport.YOffset > m.maxPreviewScroll() {
		m.previewViewport.GotoBottom()
	}
	body := paneBodyStyle.Render(m.previewViewport.View())
	style := box.Width(width)
	if height > 0 {
		style = style.Height(height)
	}
	return style.Render(header + "\n\n" + body)
}

func (m *tuiModel) syncPreviewViewportState() {
	title, body := m.previewSubject()
	leftWidth, rightWidth, _, previewHeight := m.layoutSizes("header", "footer")
	width := leftWidth + rightWidth + 2
	if width <= 0 {
		width = 80
	}
	box := paneBox(m.focus == focusPreview || m.topCollapsed)
	header := paneTitleStyle.Render(truncateRunes(title, max(8, width-8)))
	contentWidth := max(1, width-box.GetHorizontalFrameSize())
	contentHeight := paneBodyLines(previewHeight, header)
	yOffset := m.previewViewport.YOffset
	m.previewViewport.Width = contentWidth
	m.previewViewport.Height = contentHeight
	m.previewViewport.SetContent(strings.TrimSpace(body))
	maxOffset := max(0, m.previewViewport.TotalLineCount()-m.previewViewport.VisibleLineCount())
	if yOffset > maxOffset {
		yOffset = maxOffset
	}
	if yOffset < 0 {
		yOffset = 0
	}
	m.previewViewport.SetYOffset(yOffset)
}

func (m tuiModel) maxPreviewScroll() int {
	return max(0, m.previewViewport.TotalLineCount()-m.previewViewport.VisibleLineCount())
}

func (m *tuiModel) resetPreviewPosition() {
	m.previewViewport.SetYOffset(0)
}

func (m tuiModel) requestChildrenCmd(node navNode) tea.Cmd {
	if !nodeMayHaveChildren(node) {
		return nil
	}
	if _, ok := m.childCache[node.Key]; ok {
		return nil
	}
	if m.childBusy[node.Key] {
		return nil
	}
	m.childBusy[node.Key] = true
	delete(m.childErrors, node.Key)
	return func() tea.Msg {
		children, err := m.nav.Children(node)
		return tuiChildrenMsg{Parent: node, Children: children, Err: err}
	}
}

func (m tuiModel) loadingLabel(label string) string {
	return spinnerFrames[m.spinnerFrame] + " " + label + "..."
}

func resourcePreviewText(node navNode) string {
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
}

func appendPreviewBlock(base string, extra string) string {
	base = strings.TrimSpace(base)
	extra = strings.TrimSpace(extra)
	switch {
	case base == "":
		return extra
	case extra == "":
		return base
	default:
		return base + "\n\n" + extra
	}
}

func nodeMayHaveChildren(node navNode) bool {
	switch node.Kind {
	case navNodeHome, navNodeCurrent, navNodeToday, navNodeTimetable, navNodeWeek, navNodeSemesters, navNodeSemester, navNodeCourse, navNodeEvent, navNodeSections, navNodeSection, navNodeItems:
		return true
	default:
		return false
	}
}

func spinnerTickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return tuiSpinnerMsg{}
	})
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func joinBlocksForHeight(blocks []string, selected int, maxLines int) string {
	if len(blocks) == 0 || maxLines <= 0 {
		return ""
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= len(blocks) {
		selected = len(blocks) - 1
	}
	start := selected
	used := countLines(blocks[selected])
	end := selected + 1
	for {
		expanded := false
		if start > 0 {
			next := countLines(blocks[start-1])
			if used+next <= maxLines {
				start--
				used += next
				expanded = true
			}
		}
		if end < len(blocks) {
			next := countLines(blocks[end])
			if used+next <= maxLines {
				used += next
				end++
				expanded = true
			}
		}
		if !expanded {
			break
		}
	}
	return strings.Join(blocks[start:end], "\n")
}

func clampTextLines(text string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) <= maxLines {
		return strings.Join(lines, "\n")
	}
	if maxLines == 1 {
		return "..."
	}
	return strings.Join(append(lines[:maxLines-1], "..."), "\n")
}

func countLines(text string) int {
	if text == "" {
		return 0
	}
	return strings.Count(text, "\n") + 1
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return string(runes)
	}
	if limit <= 1 {
		return "…"
	}
	return string(runes[:limit-1]) + "…"
}

func (m tuiModel) renderDownloadDialog(width int, height int) string {
	dialog := m.dialog
	header := paneTitleStyle.Render("Download")
	filename := ""
	if dialog.target.Resource != nil {
		filename = buildResourceFilename(*dialog.target.Resource)
	}
	pathLine := paneSubtitleStyle.Render(truncateRunes(dialog.cwd, max(12, width-8)))
	lines := []string{
		actionRowStyle.Render("Enter on save line to save " + filename),
		pathLine,
		"",
	}
	for index, entry := range dialog.entries {
		label := entry.Label
		if entry.IsDir {
			label += "/"
		}
		display := truncateRunes(label, max(8, width-8))
		if index == dialog.selected {
			lines = append(lines, selectedRowStyle.Render("› "+display))
		} else {
			lines = append(lines, normalRowStyle.Render("  "+display))
		}
	}
	content := header + "\n\n" + clampTextLines(strings.Join(lines, "\n"), paneBodyLines(height, header))
	return paneBoxStyle.Width(width).Height(height).Render(content)
}

func paneBodyLines(height int, header string) int {
	return max(1, height-countLines(header)-5)
}

func newDownloadDialog(target navNode) (*downloadDialog, error) {
	dialog := &downloadDialog{
		target: target,
		cwd:    resolveDefaultOutputDir(""),
	}
	if err := ensureDir(dialog.cwd); err != nil {
		return nil, err
	}
	if err := dialog.reload(); err != nil {
		return nil, err
	}
	return dialog, nil
}

func (d *downloadDialog) reload() error {
	entries, err := os.ReadDir(d.cwd)
	if err != nil {
		return err
	}
	dirEntries := make([]downloadDialogEntry, 0, len(entries)+1)
	dirEntries = append(dirEntries, downloadDialogEntry{
		Label: "Save here",
		Path:  d.cwd,
		Save:  true,
	})
	dirs := make([]downloadDialogEntry, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirs = append(dirs, downloadDialogEntry{
			Label: entry.Name(),
			Path:  filepath.Join(d.cwd, entry.Name()),
			IsDir: true,
		})
	}
	slices.SortFunc(dirs, func(left, right downloadDialogEntry) int {
		return strings.Compare(strings.ToLower(left.Label), strings.ToLower(right.Label))
	})
	d.entries = append(dirEntries, dirs...)
	if d.selected >= len(d.entries) {
		d.selected = len(d.entries) - 1
	}
	if d.selected < 0 {
		d.selected = 0
	}
	return nil
}

func highlightMatch(value string, needle string) string {
	if needle == "" {
		return value
	}
	valueRunes := []rune(value)
	lowerValueRunes := []rune(strings.ToLower(value))
	lowerNeedleRunes := []rune(strings.ToLower(needle))
	index := runeSliceIndex(lowerValueRunes, lowerNeedleRunes)
	if index < 0 {
		return value
	}
	end := index + len(lowerNeedleRunes)
	if end > len(valueRunes) {
		end = len(valueRunes)
	}
	return string(valueRunes[:index]) + matchHighlightStyle.Render(string(valueRunes[index:end])) + string(valueRunes[end:])
}

func runeSliceIndex(haystack []rune, needle []rune) int {
	if len(needle) == 0 {
		return 0
	}
	if len(needle) > len(haystack) {
		return -1
	}
	for index := 0; index <= len(haystack)-len(needle); index++ {
		match := true
		for offset := range needle {
			if haystack[index+offset] != needle[offset] {
				match = false
				break
			}
		}
		if match {
			return index
		}
	}
	return -1
}

var (
	paneBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#334155")).
			Padding(1, 2)
	paneTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#f8fafc"))
	paneSubtitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#94a3b8"))
	paneBodyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e2e8f0"))
	paneMutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#64748b"))
	selectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#0f172a")).
				Background(lipgloss.Color("#7dd3fc"))
	selectedRightRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#e2e8f0")).
				Underline(true)
	normalRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e2e8f0"))
	matchHighlightStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#facc15"))
	actionRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cbd5e1"))
)

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
