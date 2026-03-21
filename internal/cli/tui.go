package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type tuiNavigator interface {
	Root() navNode
	Children(navNode) ([]navNode, error)
	Preview(navNode) string
	Open(navNode) error
	Print(navNode) (string, error)
	Download(navNode, string) (string, error)
}

type paneFocus string

const (
	focusTree  paneFocus = "tree"
	focusRight paneFocus = "right"
)

type treeRow struct {
	Node       navNode
	Depth      int
	HasKids    bool
	Expanded   bool
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
	nav           tuiNavigator
	root          navNode
	width         int
	height        int
	status        string
	filterMode    bool
	filterInput   string
	leftFilter    string
	rightFilter   string
	previewCache  map[string]string
	previewBusy   string
	dialog        *downloadDialog
	focus         paneFocus
	selectedKey   string
	rightSelected int
	expanded      map[string]bool
	nodeByKey     map[string]navNode
	parentByKey   map[string]string
	childCache    map[string][]navNode
}

type tuiOpenMsg struct{ Err error }

type tuiPreviewMsg struct {
	Key  string
	Text string
	Err  error
}

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
		nav:          service,
		root:         root,
		focus:        focusTree,
		previewCache: map[string]string{},
		expanded:     map[string]bool{root.Key: true},
		nodeByKey:    map[string]navNode{root.Key: root},
		parentByKey:  map[string]string{root.Key: ""},
		childCache:   map[string][]navNode{},
	}
	if _, err := model.ensureChildren(root); err != nil {
		return err
	}
	rows := model.visibleTreeRows()
	if len(rows) > 0 {
		model.selectedKey = rows[0].Node.Key
	} else {
		model.selectedKey = root.Key
	}
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err = program.Run()
	return err
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tuiOpenMsg:
		if msg.Err != nil {
			m.status = msg.Err.Error()
		} else {
			m.status = "Opened."
		}
		return m, nil
	case tuiPreviewMsg:
		if msg.Err != nil {
			m.status = msg.Err.Error()
			m.previewBusy = ""
			return m, nil
		}
		m.previewBusy = ""
		m.previewCache[msg.Key] = msg.Text
		m.status = "Preview loaded."
		return m, nil
	case tuiDownloadMsg:
		if msg.Err != nil {
			m.status = msg.Err.Error()
			return m, nil
		}
		m.dialog = nil
		m.status = "Saved to " + msg.Path
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
		case "/":
			m.filterMode = true
			m.filterInput = m.currentFilter()
			return m, nil
		case "g":
			m.setSelection(0)
			return m, m.autoPreviewCmd()
		case "G":
			m.setSelection(m.itemCount() - 1)
			return m, m.autoPreviewCmd()
		case "j", "down":
			m.moveSelection(1)
			return m, m.autoPreviewCmd()
		case "k", "up":
			m.moveSelection(-1)
			return m, m.autoPreviewCmd()
		case "h", "left":
			model, cmd := m.handleLeft()
			return model, tea.Batch(cmd, m.autoPreviewCmd())
		case "l", "right":
			model, cmd := m.handleRight()
			return model, tea.Batch(cmd, m.autoPreviewCmd())
		case "enter":
			model, cmd := m.handleEnter()
			return model, tea.Batch(cmd, m.autoPreviewCmd())
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
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f8fafc")).Render("Moodle TUI")
	header += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#94a3b8")).Render(strings.Join(m.breadcrumb(), " > "))

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
		previewHeight = max(6, m.height-matrixHeight-8)
	}

	left := m.renderTreePane(leftWidth, matrixHeight)
	right := m.renderRightPane(rightWidth, matrixHeight)
	preview := m.renderBottomPane(leftWidth+rightWidth+2, previewHeight)
	footer := m.renderFooter()

	return header + "\n\n" + lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right) + "\n\n" + preview + "\n\n" + footer
}

func (m *tuiModel) ensureChildren(node navNode) ([]navNode, error) {
	if children, ok := m.childCache[node.Key]; ok {
		return children, nil
	}
	children, err := m.nav.Children(node)
	if err != nil {
		return nil, err
	}
	m.childCache[node.Key] = children
	for _, child := range children {
		m.nodeByKey[child.Key] = child
		if _, ok := m.parentByKey[child.Key]; !ok {
			m.parentByKey[child.Key] = node.Key
		}
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
	children, err := m.ensureChildren(m.root)
	if err != nil {
		return rows
	}
	pathSet := m.selectedPathSet()
	for _, child := range children {
		m.appendTreeRow(&rows, child, 0, pathSet)
	}
	return rows
}

func (m tuiModel) appendTreeRow(rows *[]treeRow, node navNode, depth int, pathSet map[string]bool) {
	children, _ := m.ensureChildren(node)
	hasKids := len(children) > 0
	row := treeRow{
		Node:       node,
		Depth:      depth,
		HasKids:    hasKids,
		Expanded:   m.expanded[node.Key],
		TopLevel:   depth == 0,
		QuickBand:  depth == 0 && (node.Kind == navNodeCurrent || node.Kind == navNodeToday),
		BrowseBand: depth == 0 && (node.Kind == navNodeSemesters || node.Kind == navNodeTimetable),
	}
	*rows = append(*rows, row)
	if hasKids && m.expanded[node.Key] && pathSet[node.Key] {
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
	children, err := m.ensureChildren(selected)
	if err != nil {
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
}

func (m tuiModel) handleLeft() (tea.Model, tea.Cmd) {
	if m.focus == focusRight {
		m.focus = focusTree
		return m, nil
	}
	selected := m.selectedNode()
	children, _ := m.ensureChildren(selected)
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
	if len(m.rightEntries()) == 0 {
		return m, nil
	}
	m.focus = focusRight
	if m.rightSelected >= len(m.rightEntries()) {
		m.rightSelected = 0
	}
	return m, nil
}

func (m tuiModel) handleEnter() (tea.Model, tea.Cmd) {
	if m.focus == focusRight {
		return m.commitRightSelection()
	}
	selected := m.selectedNode()
	children, err := m.ensureChildren(selected)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	if len(children) > 0 {
		if m.expanded[selected.Key] {
			delete(m.expanded, selected.Key)
		} else {
			m.expanded[selected.Key] = true
		}
		return m, nil
	}
	if selected.Resource != nil {
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
	m.expandVisiblePath(node.Key)
	children, err := m.ensureChildren(node)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	if len(children) > 0 {
		m.expanded[node.Key] = true
		m.focus = focusRight
		return m, nil
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
		return m, m.openCmd(target)
	case "print":
		if !target.Printable {
			m.status = "Selected item cannot be previewed."
			return m, nil
		}
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
		m.status = "Download dialog"
		return m, nil
	default:
		return m, nil
	}
}

func (m tuiModel) openCmd(node navNode) tea.Cmd {
	return func() tea.Msg {
		return tuiOpenMsg{Err: m.nav.Open(node)}
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
		if len(text) > 1600 {
			text = text[:1600] + "\n..."
		}
		return tuiPreviewMsg{Key: node.Key, Text: text}
	}
}

func (m *tuiModel) autoPreviewCmd() tea.Cmd {
	target, ok := m.previewTargetNode()
	if !ok || !target.Printable {
		return nil
	}
	if _, ok := m.previewCache[target.Key]; ok {
		return nil
	}
	if m.previewBusy == target.Key {
		return nil
	}
	m.previewBusy = target.Key
	return m.printCmd(target)
}

func (m tuiModel) renderTreePane(width int, height int) string {
	rows := m.visibleTreeRows()
	title := "Navigation"
	if len(rows) == 0 {
		return paneBoxStyle.Width(width).Height(height).Render(paneTitleStyle.Render(title) + "\n\n" + paneMutedStyle.Render("(empty)"))
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
		lines = append(lines, renderTreeRow(row, isSelected, width-8, m.leftFilter))
	}
	header := paneTitleStyle.Render(title)
	content := header + "\n\n" + joinBlocksForHeight(lines, selectedLine, paneBodyLines(height, header))
	return paneBoxStyle.Width(width).Height(height).Render(content)
}

func (m tuiModel) renderRightPane(width int, height int) string {
	selected := m.selectedNode()
	if selected.Key == "" {
		return renderPreviewPane("Details", "", width, height)
	}
	entries := m.rightEntries()
	title := selected.Title
	if title == "" {
		title = "Details"
	}
	header := paneTitleStyle.Render(truncateRunes(title, max(8, width-8)))
	if len(entries) == 0 {
		return paneBoxStyle.Width(width).Height(height).Render(header + "\n\n" + paneMutedStyle.Render("(empty)"))
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
	return paneBoxStyle.Width(width).Height(height).Render(content)
}

func (m tuiModel) renderBottomPane(width int, height int) string {
	if m.dialog != nil {
		return m.renderDownloadDialog(width, height)
	}
	title, body := m.previewSubject()
	return renderPreviewPane(title, body, width, height)
}

func (m tuiModel) previewSubject() (string, string) {
	if m.focus == focusRight {
		if entry, ok := m.selectedRightEntry(); ok {
			if entry.Kind == rightEntryAction {
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
	switch {
	case node.Resource != nil:
		body := m.nav.Preview(node)
		if text, ok := m.previewCache[node.Key]; ok {
			if body != "" {
				body += "\n\n"
			}
			body += text
		} else if m.previewBusy == node.Key {
			if body != "" {
				body += "\n\n"
			}
			body += "Loading preview..."
		}
		return title, body
	case node.Kind == navNodeCourse || node.Kind == navNodeSemester || node.Kind == navNodeSection || node.Kind == navNodeCurrent || node.Kind == navNodeToday || node.Kind == navNodeWeek || node.Kind == navNodeTimetable:
		return title, m.structurePreview(node, 2)
	default:
		return title, m.nav.Preview(node)
	}
}

func (m tuiModel) structurePreview(node navNode, depth int) string {
	body := strings.TrimSpace(m.nav.Preview(node))
	children, err := m.ensureChildren(node)
	if err != nil || len(children) == 0 || depth <= 0 {
		return body
	}
	lines := []string{}
	for _, child := range children {
		lines = append(lines, outlineNode(child, 0))
		if depth > 1 {
			grandChildren, err := m.ensureChildren(child)
			if err == nil {
				for _, grandChild := range grandChildren {
					lines = append(lines, outlineNode(grandChild, 1))
				}
			}
		}
	}
	outline := strings.Join(lines, "\n")
	if body == "" {
		return outline
	}
	return body + "\n\n" + outline
}

func (m tuiModel) renderFooter() string {
	parts := []string{"h/j/k/l or arrows", "Enter=toggle/drill", "/=filter", "o=open", "p=preview", "d=download", "q=quit"}
	if m.filterMode {
		parts = append(parts, "/"+m.filterInput)
	} else if m.dialog != nil {
		parts = append(parts, "dialog active")
	} else if m.status != "" {
		parts = append(parts, m.status)
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#94a3b8")).Render(strings.Join(parts, " · "))
}

func renderTreeRow(row treeRow, selected bool, width int, filter string) string {
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

func renderPreviewPane(title string, text string, width int, height int) string {
	header := paneTitleStyle.Render(truncateRunes(title, max(8, width-8)))
	availableLines := paneBodyLines(height, header)
	body := paneBodyStyle.Render(clampTextLines(text, availableLines))
	style := paneBoxStyle.Width(width)
	if height > 0 {
		style = style.Height(height)
	}
	return style.Render(header + "\n\n" + body)
}

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
