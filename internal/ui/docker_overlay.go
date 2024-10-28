// File: internal/ui/docker_overlay.go
package ui

import (
    "context"
    "fmt"
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/docker/docker/api/types"
    "github.com/docker/docker/client"
    "github.com/Cdaprod/go-middleware-registry/registry"
)

// Docker-related messages
type dockerMsg struct {
    containerID string
    status      string
    logs        string
}

// DockerPopup represents the Docker action overlay
type DockerPopup struct {
    visible      bool
    options      []string
    selected     int
    width        int
    height       int
    docker       *client.Client
    repository   *registry.RegistryItem
    containerID  string
    containerLog string
    showLogs     bool
}

// Styling
var (
    popupStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(highlightColor). // Use existing highlight color
        Padding(1, 2).
        Background(lipgloss.Color("#1A1A1A"))

    containerWindowStyle = lipgloss.NewStyle().
        Border(lipgloss.DoubleBorder()).
        BorderForeground(highlightColor).
        Padding(1, 2).
        Background(lipgloss.Color("#0A0A0A"))

    menuItemStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#FFFFFF"))

    selectedMenuItemStyle = lipgloss.NewStyle().
        Foreground(highlightColor).
        Bold(true)
)

// Update model to include Docker popup
type model struct {
    Tabs       []string
    activeTab  int
    registry   *registry.Registry
    lists      []list.Model
    width      int
    height     int
    dockerPopup *DockerPopup // Add this field
}

// NewDockerPopup creates a new Docker popup for a repository
func NewDockerPopup(repo *registry.RegistryItem) (*DockerPopup, error) {
    docker, err := client.NewClientWithOpts(client.FromEnv)
    if err != nil {
        return nil, err
    }
    
    return &DockerPopup{
        visible: true,
        options: []string{
            "🚀 Run Container",
            "🔍 Inspect Dockerfile",
            "📦 Build Image",
            "⏹️ Stop Container",
            "❌ Cancel",
        },
        docker:     docker,
        repository: repo,
    }, nil
}

// Modify the model's Update method to handle Docker popup
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    var cmds []tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        if m.dockerPopup != nil && m.dockerPopup.visible {
            switch msg.String() {
            case "esc":
                m.dockerPopup.visible = false
                m.dockerPopup.showLogs = false
                return m, nil
            case "up", "k":
                m.dockerPopup.selected = max(0, m.dockerPopup.selected-1)
                return m, nil
            case "down", "j":
                m.dockerPopup.selected = min(len(m.dockerPopup.options)-1, m.dockerPopup.selected+1)
                return m, nil
            case "enter":
                return m, m.dockerPopup.executeSelected()
            }
        } else {
            switch msg.String() {
            case "enter":
                if m.activeTab == 1 { // Repositories tab
                    if selected, ok := m.lists[m.activeTab].SelectedItem().(listItem); ok {
                        // Extract repository name from the title (remove icon)
                        repoName := strings.TrimPrefix(selected.title, "🐳 ")
                        repoName = strings.TrimPrefix(repoName, "󰊤 ")
                        repoName = strings.TrimPrefix(repoName, "📁 ")
                        
                        if repo, ok := m.registry.RegistryActor.Repos[repoName]; ok && repo.HasDockerfile {
                            popup, err := NewDockerPopup(repo)
                            if err != nil {
                                // Handle error
                                return m, nil
                            }
                            popup.width = m.width
                            popup.height = m.height
                            m.dockerPopup = popup
                            return m, nil
                        }
                    }
                }
            }
        }
    case dockerMsg:
        if m.dockerPopup != nil {
            m.dockerPopup.containerID = msg.containerID
            m.dockerPopup.containerLog = msg.logs
            if msg.status == "error" {
                m.dockerPopup.showLogs = true
            }
        }
    }

    // Handle other updates
    m.lists[m.activeTab], cmd = m.lists[m.activeTab].Update(msg)
    cmds = append(cmds, cmd)

    return m, tea.Batch(cmds...)
}

// Modify the model's View method to include Docker popup
func (m model) View() string {
    if m.dockerPopup != nil && m.dockerPopup.visible {
        // Return the popup view on top of the main view
        mainView := m.mainView() // Extract existing view logic to mainView()
        popupView := m.dockerPopup.View()
        
        return lipgloss.Place(
            m.width,
            m.height,
            lipgloss.Center,
            lipgloss.Center,
            popupView,
            lipgloss.WithWhitespaceChars(""),
            lipgloss.WithWhitespaceForeground(lipgloss.Color("#666666")),
        )
    }
    
    return m.mainView()
}

// Extract the existing view logic to a separate method
func (m model) mainView() string {
    // Your existing View() implementation here
    doc := strings.Builder{}
    // ... rest of your existing view code ...
    return doc.String()
}

// executeSelected handles Docker action execution
func (p *DockerPopup) executeSelected() tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background()
        
        switch p.selected {
        case 0: // Run Container
            // Send message to registry actor to run container
            return dockerMsg{
                status: "running",
                logs:   "Starting container...\n",
            }
            
        case 1: // Inspect Dockerfile
            return dockerMsg{
                status: "inspecting",
                logs:   fmt.Sprintf("Dockerfile path: %s/Dockerfile\n", p.repository.Path),
            }
            
        case 2: // Build Image
            return dockerMsg{
                status: "building",
                logs:   "Building image...\n",
            }
            
        case 3: // Stop Container
            if p.containerID != "" {
                return dockerMsg{
                    status: "stopping",
                    logs:   "Stopping container...\n",
                }
            }
            
        case 4: // Cancel
            p.visible = false
            return nil
        }
        
        return nil
    }
}

// View renders the Docker popup
func (p *DockerPopup) View() string {
    if !p.visible {
        return ""
    }

    var content strings.Builder

    if p.showLogs {
        monitorContent := fmt.Sprintf(`Container: %s
Status: Running
Logs:
%s`, p.containerID, p.containerLog)
        
        return containerWindowStyle.Width(p.width - 4).Height(p.height - 4).Render(monitorContent)
    }

    content.WriteString("Docker Actions\n\n")
    
    for i, option := range p.options {
        if i == p.selected {
            content.WriteString(selectedMenuItemStyle.Render(fmt.Sprintf("> %s\n", option)))
        } else {
            content.WriteString(menuItemStyle.Render(fmt.Sprintf("  %s\n", option)))
        }
    }

    return popupStyle.Width(40).Render(content.String())
}