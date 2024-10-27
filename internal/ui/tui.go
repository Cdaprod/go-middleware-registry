// File: internal/ui/tui.go
package ui

import (
	"fmt"
	"strings"

	"github.com/Cdaprod/go-middleware-registry/registry"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// model represents the state of the TUI.
type model struct {
	Tabs       []string
	activeTab  int
	registry   *registry.Registry
	lists      []list.Model
	width      int
	height     int
}

// listItem represents an item in the list.
type listItem struct {
	title string
	desc  string
}

func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return i.desc }
func (i listItem) FilterValue() string { return i.title }

// NewModel initializes the TUI model with the given Registry.
func NewModel(reg *registry.Registry) model {
	// Initialize tabs
	tabs := []string{"Registrar Operations", "Repositories", "Configurations"}

	// Create lists for each tab
	registrarItems := []list.Item{
		listItem{title: "Add Repository", desc: "Add a new repository to the registry"},
		listItem{title: "Scan Projects", desc: "Scan for new repositories"},
		listItem{title: "List All", desc: "List all registered repositories"},
		listItem{title: "Toggle Repository", desc: "Enable/disable a repository"},
		listItem{title: "Configure Repository", desc: "Configure a repository with Docker and Pipeline"},
	}

	// Convert registry items to list items
	var repoItems []list.Item
	for _, item := range reg.ListItems() {
		icon := "📁"
		if item.HasDockerfile {
			icon = "🐳"
		} else {
			icon = "󰊤" // GitHub icon
		}
		repoItems = append(repoItems, listItem{
			title: fmt.Sprintf("%s %s", icon, item.Name),
			desc:  item.Path,
		})
	}

	configItems := []list.Item{
		listItem{title: "Add Secret Key", desc: "Add a new secret key"},
		listItem{title: "Remove Secret Key", desc: "Remove an existing secret key"},
		listItem{title: "Add Workflow", desc: "Add a reusable workflow"},
		listItem{title: "Remove Workflow", desc: "Remove a workflow"},
		listItem{title: "Configure Source URL", desc: "Set URL for listening to registry items"},
		listItem{title: "Export Registry", desc: "Export registry data (JSON/Table)"},
	}

	// Create and style the lists
	lists := make([]list.Model, 3)
	lists[0] = createList(registrarItems, "Registrar Operations")
	lists[1] = createList(repoItems, "Repositories")
	lists[2] = createList(configItems, "Configurations")

	return model{
		Tabs:      tabs,
		registry:  reg,
		lists:     lists,
		activeTab: 0,
	}
}

// createList initializes a list.Model with the given items and title.
func createList(items []list.Item, title string) list.Model {
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = title
	l.SetShowHelp(false)
	return l
}

// Init initializes the TUI.
func (m model) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages and updates the TUI state accordingly.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "right", "l", "tab":
			m.activeTab = min(m.activeTab+1, len(m.Tabs)-1)
		case "left", "h", "shift+tab":
			m.activeTab = max(m.activeTab-1, 0)
		case "enter":
			selectedItem, ok := m.lists[m.activeTab].SelectedItem().(listItem)
			if ok {
				handleListSelection(m.registry, m.activeTab, selectedItem)
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		for i := range m.lists {
			m.lists[i].SetSize(msg.Width-4, msg.Height-7)
		}
	}

	// Handle active list updates
	m.lists[m.activeTab], cmd = m.lists[m.activeTab].Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleListSelection performs actions based on the selected item in the TUI.
func handleListSelection(reg *registry.Registry, tab int, item listItem) {
	switch tab {
	case 0: // Registrar Operations
		switch item.title {
		case "Add Repository":
			// Implement Add Repository logic
			repoName := promptInput("Enter repository name:")
			repoPath := promptInput("Enter repository path:")
			reg.RegistryActor.MsgChan <- AddRepo{Name: repoName, Path: repoPath}
			fmt.Println("Add Repository command sent.")
		case "Scan Projects":
			reg.RegistryActor.MsgChan <- ScanDir{Directory: reg.Config.ProjectsPath}
			fmt.Println("Scan initiated.")
		case "Toggle Repository":
			repoName := promptInput("Enter repository name to toggle:")
			reg.RegistryActor.MsgChan <- ToggleRepo{Name: repoName}
			fmt.Println("Toggle command sent.")
		case "Configure Repository":
			repoName := promptInput("Enter repository name to configure:")
			reg.RegistryActor.MsgChan <- ConfigureRepo{Name: repoName}
			fmt.Println("Configure command sent.")
		}
	case 1: // Repositories
		// Implement actions for repository items
		repoName := strings.TrimPrefix(item.title, "📁 ")
		repoName = strings.TrimPrefix(repoName, "🐳 ")
		fmt.Printf("Repository selected: %s\n", repoName)
		// Example: Display detailed info
		displayRepoInfo(reg.RegistryActor.Repos[repoName])
	case 2: // Configurations
		// Implement configuration actions
		fmt.Printf("Configuration selected: %s\n", item.title)
	}
}

// promptInput is a placeholder function for user input.
// In a real application, you'd implement proper input handling.
func promptInput(prompt string) string {
	fmt.Printf("%s ", prompt)
	var input string
	fmt.Scanln(&input)
	return input
}

// View renders the TUI.
func (m model) View() string {
	doc := strings.Builder{}

	var renderedTabs []string

	for i, t := range m.Tabs {
		var style lipgloss.Style
		isActive := i == m.activeTab
		if isActive {
			style = activeTabStyle
		} else {
			style = inactiveTabStyle
		}
		renderedTabs = append(renderedTabs, style.Render(t))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	doc.WriteString(row)
	doc.WriteString("\n")
	doc.WriteString(windowStyle.
		Width(lipgloss.Width(row)-windowStyle.GetHorizontalFrameSize()).
		Render(m.lists[m.activeTab].View()))

	return docStyle.Render(doc.String())
}

// LaunchTUI starts the TUI with the given Registry.
func LaunchTUI(reg *registry.Registry) error {
	p := tea.NewProgram(
		NewModel(reg),
		tea.WithAltScreen(),
	)
	_, err := p.Run()
	return err
}

// Helper functions for tab navigation.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Styling for the TUI.
func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

var (
	inactiveTabBorder = tabBorderWithBottom("┴", "─", "┴")
	activeTabBorder   = tabBorderWithBottom("┘", " ", "└")
	docStyle          = lipgloss.NewStyle().Padding(1, 2, 1, 2)
	highlightColor    = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	inactiveTabStyle  = lipgloss.NewStyle().
			Border(inactiveTabBorder, true).
			BorderForeground(highlightColor).
			Padding(0, 1)
	activeTabStyle = inactiveTabStyle.Copy().
			Border(activeTabBorder, true)
	windowStyle = lipgloss.NewStyle().
			BorderForeground(highlightColor).
			Padding(2, 0).
			Align(lipgloss.Left).
			Border(lipgloss.NormalBorder()).
			UnsetBorderTop()
)