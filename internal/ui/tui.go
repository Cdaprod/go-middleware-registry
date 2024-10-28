// File: internal/ui/tui.go
package ui

import (
    "fmt"
    "strings"
    "time"

    "github.com/Cdaprod/go-middleware-registry/registry"
    "github.com/charmbracelet/bubbles/list"
    "github.com/charmbracelet/bubbles/spinner"
    "github.com/charmbracelet/bubbles/viewport"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

// Message types
type (
    dockerMsg struct {
        Type    string // "error", "success", "info"
        Message string
        Data    interface{}
    }

    clearMessageMsg struct{}

    operationCompleteMsg struct {
        success bool
        message string
    }
)

// UI States
type viewState int

const (
    normalState viewState = iota
    dockerMenuState
    containerViewState
)

// model represents the TUI state
type model struct {
    // Core components
    Tabs      []string
    activeTab int
    registry  *registry.Registry
    lists     []list.Model
    state     viewState

    // Docker components
    dockerManager  *DockerManager
    activeRepo     string
    dockerMenu    *MenuModel
    containerView *ContainerViewModel

    // UI components
    spinner  spinner.Model
    viewport viewport.Model
    width    int
    height   int

    // Messages
    errorMsg   string
    successMsg string
    loading    bool
}

// List item implementation
type listItem struct {
    title string
    desc  string
}

func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return i.desc }
func (i listItem) FilterValue() string { return i.title }

// NewModel initializes the TUI
func NewModel(reg *registry.Registry) (model, error) {
    dockerManager, err := NewDockerManager(reg)
    if err != nil {
        return model{}, err
    }

    s := spinner.New()
    s.Spinner = spinner.Dot
    s.Style = spinnerStyle

    m := model{
        Tabs: []string{
            "Registrar Operations",
            "Repositories",
            "Docker",
            "Configurations",
        },
        registry:      reg,
        dockerManager: dockerManager,
        state:        normalState,
        spinner:      s,
    }

    // Initialize lists
    m.lists = make([]list.Model, len(m.Tabs))
    m.initializeLists()

    return m, nil
}

func (m *model) initializeLists() {
    // Registrar Operations
    registrarItems := []list.Item{
        listItem{title: "Add Repository", desc: "Add a new repository to the registry"},
        listItem{title: "Scan Projects", desc: "Scan for new repositories"},
        listItem{title: "List All", desc: "List all registered repositories"},
        listItem{title: "Toggle Repository", desc: "Enable/disable a repository"},
        listItem{title: "Configure Repository", desc: "Configure repository settings"},
    }
    m.lists[0] = createList(registrarItems, "Registrar Operations")

    // Repositories
    var repoItems []list.Item
    for _, item := range m.registry.ListItems() {
        icon := "📁"
        if item.HasDockerfile {
            icon = "🐳"
        } else if item.GitRepo != nil {
            icon = "󰊤"
        }
        repoItems = append(repoItems, listItem{
            title: fmt.Sprintf("%s %s", icon, item.Name),
            desc:  item.Path,
        })
    }
    m.lists[1] = createList(repoItems, "Repositories")

    // Docker Operations (will be populated dynamically)
    m.lists[2] = createList([]list.Item{}, "Docker Operations")

    // Configurations
    configItems := []list.Item{
        listItem{title: "Global Settings", desc: "Configure global registry settings"},
        listItem{title: "Docker Settings", desc: "Configure Docker integration settings"},
        listItem{title: "Export Data", desc: "Export registry data"},
    }
    m.lists[3] = createList(configItems, "Configurations")
}

func createList(items []list.Item, title string) list.Model {
    l := list.New(items, list.NewDefaultDelegate(), 0, 0)
    l.Title = title
    l.SetShowHelp(false)
    return l
}

// Init initializes the program
func (m model) Init() tea.Cmd {
    return tea.Batch(
        m.spinner.Tick,
        checkDockerStatus(m.registry),
    )
}

// Update handles all UI updates
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c", "q":
            return m, tea.Quit
        case "esc":
            if m.state != normalState {
                m.state = normalState
                return m, nil
            }
        default:
            switch m.state {
            case normalState:
                cmds = append(cmds, m.handleNormalState(msg)...)
            case dockerMenuState:
                cmds = append(cmds, m.handleDockerMenu(msg)...)
            case containerViewState:
                cmds = append(cmds, m.handleContainerView(msg)...)
            }
        }

    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.updateComponentSizes()

    case dockerMsg:
        cmds = append(cmds, m.handleDockerMsg(msg)...)

    case operationCompleteMsg:
        m.loading = false
        if msg.success {
            m.successMsg = msg.message
        } else {
            m.errorMsg = msg.message
        }
        cmds = append(cmds, m.clearMessageAfterDelay())

    case spinner.TickMsg:
        var cmd tea.Cmd
        m.spinner, cmd = m.spinner.Update(msg)
        cmds = append(cmds, cmd)

    case clearMessageMsg:
        m.errorMsg = ""
        m.successMsg = ""
    }

    // Update active list
    if m.state == normalState && m.activeTab < len(m.lists) {
        var cmd tea.Cmd
        m.lists[m.activeTab], cmd = m.lists[m.activeTab].Update(msg)
        if cmd != nil {
            cmds = append(cmds, cmd)
        }
    }

    return m, tea.Batch(cmds...)
}

// Handle different states
func (m model) handleNormalState(msg tea.KeyMsg) []tea.Cmd {
    var cmds []tea.Cmd
    
    switch msg.String() {
    case "tab":
        m.activeTab = (m.activeTab + 1) % len(m.Tabs)
    case "shift+tab":
        m.activeTab = (m.activeTab - 1 + len(m.Tabs)) % len(m.Tabs)
    case "enter":
        cmds = append(cmds, m.handleSelection()...)
    }
    
    return cmds
}

func (m model) handleDockerMenu(msg tea.KeyMsg) []tea.Cmd {
    // Docker menu navigation and selection
    return nil
}

func (m model) handleContainerView(msg tea.KeyMsg) []tea.Cmd {
    // Container view navigation and interaction
    return nil
}

// View renders the UI
func (m model) View() string {
    var b strings.Builder

    // Render tabs
    var renderedTabs []string
    for i, t := range m.Tabs {
        style := inactiveTabStyle
        if i == m.activeTab {
            style = activeTabStyle
        }
        renderedTabs = append(renderedTabs, style.Render(t))
    }
    
    b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...))
    b.WriteString("\n")

    // Render main content
    mainContent := ""
    switch m.state {
    case normalState:
        if m.activeTab < len(m.lists) {
            mainContent = m.lists[m.activeTab].View()
        }
    case dockerMenuState:
        mainContent = m.dockerMenu.View()
    case containerViewState:
        mainContent = m.containerView.View()
    }

    if m.loading {
        mainContent = fmt.Sprintf("%s Loading...", m.spinner.View())
    }

    b.WriteString(windowStyle.
        Width(m.width-4).
        Height(m.height-7).
        Render(mainContent))

    // Render messages
    if m.errorMsg != "" {
        b.WriteString("\n" + errorStyle.Render(m.errorMsg))
    }
    if m.successMsg != "" {
        b.WriteString("\n" + successStyle.Render(m.successMsg))
    }

    // Render help
    help := "\n" + helpStyle.Render("tab: switch view • enter: select • esc: back • q: quit")
    b.WriteString(help)

    return docStyle.Render(b.String())
}

// Utility functions
func (m *model) updateComponentSizes() {
    for i := range m.lists {
        m.lists[i].SetSize(m.width-4, m.height-7)
    }
    
    if m.dockerMenu != nil {
        m.dockerMenu.SetSize(m.width-4, m.height-7)
    }
    
    if m.containerView != nil {
        m.containerView.SetSize(m.width-4, m.height-7)
    }
}

func (m *model) handleSelection() []tea.Cmd {
    var cmds []tea.Cmd
    
    selected := m.lists[m.activeTab].SelectedItem()
    if selected == nil {
        return nil
    }

    item := selected.(listItem)
    
    switch m.activeTab {
    case 0: // Registrar Operations
        cmds = append(cmds, m.handleRegistrarOperation(item.title))
    case 1: // Repositories
        cmds = append(cmds, m.handleRepositorySelection(item))
    case 2: // Docker Operations
        cmds = append(cmds, m.handleDockerOperation(item))
    case 3: // Configurations
        cmds = append(cmds, m.handleConfigOperation(item))
    }
    
    return cmds
}

func (m model) handleDockerMsg(msg dockerMsg) []tea.Cmd {
    var cmds []tea.Cmd
    
    switch msg.Type {
    case "error":
        m.errorMsg = msg.Message
        cmds = append(cmds, m.clearMessageAfterDelay())
    case "success":
        m.successMsg = msg.Message
        cmds = append(cmds, m.clearMessageAfterDelay())
    case "container-started":
        m.state = containerViewState
        m.containerView = NewContainerViewModel(msg.Data.(string))
    }
    
    return cmds
}

func (m model) clearMessageAfterDelay() tea.Cmd {
    return tea.Tick(time.Second*3, func(time.Time) tea.Msg {
        return clearMessageMsg{}
    })
}

// Operation handlers
func (m *model) handleRegistrarOperation(operation string) tea.Cmd {
    m.loading = true
    return func() tea.Msg {
        var success bool
        var message string
        
        switch operation {
        case "Add Repository":
            // Implementation
            success = true
            message = "Repository added successfully"
        case "Scan Projects":
            err := m.registry.ScanRepositories()
            success = err == nil
            message = "Scan completed"
            if !success {
                message = fmt.Sprintf("Scan failed: %v", err)
            }
        }
        
        return operationCompleteMsg{success: success, message: message}
    }
}

func (m *model) handleRepositorySelection(item listItem) tea.Cmd {
    if strings.Contains(item.title, "🐳") {
        m.activeRepo = strings.TrimPrefix(item.title, "🐳 ")
        m.state = dockerMenuState
        m.dockerMenu = NewDockerMenu(m.activeRepo)
        return nil
    }
    return nil
}

func (m *model) handleDockerOperation(item listItem) tea.Cmd {
    return nil
}

func (m *model) handleConfigOperation(item listItem) tea.Cmd {
    return nil
}

// LaunchTUI starts the TUI
func LaunchTUI(reg *registry.Registry) error {
    m, err := NewModel(reg)
    if err != nil {
        return err
    }

    p := tea.NewProgram(m, tea.WithAltScreen())
    _, err = p.Run()
    return err
}

// Helper function to check Docker status
func checkDockerStatus(reg *registry.Registry) tea.Cmd {
    return func() tea.Msg {
        // Implementation
        return nil
    }
}