// File: internal/ui/container_views.go
package ui

import (
    "fmt"
    "os/exec"
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/viewport"
    "github.com/charmbracelet/lipgloss"
    "github.com/docker/docker/client"
)

// View states
type viewState int

const (
    containerListView viewState = iota
    containerShellView
    containerLogsView
)

// Messages
type containerMsg struct {
    id     string
    output string
}

type execFinishedMsg struct {
    err error
}

// Styles
var (
    containerStyle = lipgloss.NewStyle().
        BorderStyle(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("#874BFD")).
        Padding(1, 2)

    activeContainerStyle = containerStyle.Copy().
        BorderForeground(lipgloss.Color("#00FF00"))

    shellStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("#874BFD")).
        Margin(1, 2).
        Padding(1, 2)

    titleStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#874BFD")).
        Bold(true)
)

// ContainerView represents a single container view
type ContainerView struct {
    id       string
    name     string
    viewport viewport.Model
    shell    *exec.Cmd
    logs     string
    active   bool
}

// ContainerManager manages multiple container views
type ContainerManager struct {
    containers []*ContainerView
    active     int
    state      viewState
    docker     *client.Client
    width      int
    height     int
}

func NewContainerManager() (*ContainerManager, error) {
    docker, err := client.NewClientWithOpts(client.FromEnv)
    if err != nil {
        return nil, err
    }

    return &ContainerManager{
        docker: docker,
        state:  containerListView,
    }, nil
}

// OpenShell opens an interactive shell in the container
func (cv *ContainerView) OpenShell() tea.Cmd {
    return func() tea.Msg {
        cmd := exec.Command("docker", "exec", "-it", cv.id, "/bin/sh")
        return tea.ExecProcess(cmd, func(err error) tea.Msg {
            return execFinishedMsg{err}
        })
    }
}

// Update handles container view updates
func (cm *ContainerManager) Update(msg tea.Msg) tea.Cmd {
    var cmds []tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "tab":
            // Cycle through views
            cm.state = (cm.state + 1) % 3
        case "shift+tab":
            // Cycle backwards
            if cm.state == 0 {
                cm.state = 2
            } else {
                cm.state--
            }
        case "j", "down":
            // Next container
            if cm.active < len(cm.containers)-1 {
                cm.active++
            }
        case "k", "up":
            // Previous container
            if cm.active > 0 {
                cm.active--
            }
        case "enter":
            // Open shell for active container
            if cm.state == containerListView && len(cm.containers) > 0 {
                cm.state = containerShellView
                return cm.containers[cm.active].OpenShell()
            }
        case "l":
            // View logs
            if cm.state == containerListView && len(cm.containers) > 0 {
                cm.state = containerLogsView
                return cm.fetchLogs(cm.containers[cm.active].id)
            }
        }

    case execFinishedMsg:
        if msg.err != nil {
            // Handle shell error
            return nil
        }

    case containerMsg:
        // Update container logs
        for _, c := range cm.containers {
            if c.id == msg.id {
                c.logs = msg.output
                break
            }
        }
    }

    return tea.Batch(cmds...)
}

// View renders the appropriate view based on state
func (cm *ContainerManager) View() string {
    switch cm.state {
    case containerListView:
        return cm.listView()
    case containerShellView:
        return cm.shellView()
    case containerLogsView:
        return cm.logsView()
    default:
        return "Unknown view state"
    }
}

func (cm *ContainerManager) listView() string {
    var b strings.Builder

    b.WriteString(titleStyle.Render("Container List"))
    b.WriteString("\n\n")

    for i, container := range cm.containers {
        style := containerStyle
        if i == cm.active {
            style = activeContainerStyle
        }

        info := fmt.Sprintf("%s\n%s", container.name, container.id[:12])
        b.WriteString(style.Render(info) + "\n")
    }

    b.WriteString("\n" + helpStyle.Render("j/k: navigate • enter: shell • l: logs • tab: switch view • q: quit"))
    return b.String()
}

func (cm *ContainerManager) shellView() string {
    if len(cm.containers) == 0 || cm.active >= len(cm.containers) {
        return "No container selected"
    }

    container := cm.containers[cm.active]
    return shellStyle.Render(fmt.Sprintf("Shell: %s\n\n%s", 
        container.name,
        container.viewport.View(),
    ))
}

func (cm *ContainerManager) logsView() string {
    if len(cm.containers) == 0 || cm.active >= len(cm.containers) {
        return "No container selected"
    }

    container := cm.containers[cm.active]
    return shellStyle.Render(fmt.Sprintf("Logs: %s\n\n%s",
        container.name,
        container.logs,
    ))
}

// fetchLogs retrieves container logs
func (cm *ContainerManager) fetchLogs(containerID string) tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background()
        options := types.ContainerLogsOptions{
            ShowStdout: true,
            ShowStderr: true,
            Follow:     false,
            Tail:       "100",
        }

        logs, err := cm.docker.ContainerLogs(ctx, containerID, options)
        if err != nil {
            return containerMsg{id: containerID, output: fmt.Sprintf("Error fetching logs: %v", err)}
        }
        defer logs.Close()

        buf := new(strings.Builder)
        _, err = io.Copy(buf, logs)
        if err != nil {
            return containerMsg{id: containerID, output: fmt.Sprintf("Error reading logs: %v", err)}
        }

        return containerMsg{id: containerID, output: buf.String()}
    }
}

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
    containerManager *ContainerManager

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


func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        if m.dockerPopup != nil && m.dockerPopup.visible {
            // ... existing docker popup handling ...
        } else if msg.String() == "c" {
            // Toggle container manager view
            cmd := m.containerManager.Update(msg)
            if cmd != nil {
                cmds = append(cmds, cmd)
            }
        }
    }

    // ... rest of update logic ...

    return m, tea.Batch(cmds...)
}