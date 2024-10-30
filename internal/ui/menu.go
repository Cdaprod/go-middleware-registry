// File: internal/ui/menu.go
package ui

import (
    "strings"
    tea "github.com/charmbracelet/bubbletea"
)

// MenuItem represents a menu option
type MenuItem struct {
    Title       string
    Description string
    Icon        string
    Action      string
    Disabled    bool
}

// Menu represents a popup menu component
type Menu struct {
    Title       string
    Items       []MenuItem
    Selected    int
    Width       int
    Height      int
    Visible     bool
    Style       string // "docker", "container", "default"
}

func NewMenu(title string, items []MenuItem, style string) *Menu {
    return &Menu{
        Title:    title,
        Items:    items,
        Style:    style,
        Visible:  true,
        Selected: 0,
    }
}

func (m *Menu) Update(msg tea.Msg) (*Menu, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "up", "k":
            m.Selected = max(0, m.Selected-1)
        case "down", "j":
            m.Selected = min(len(m.Items)-1, m.Selected+1)
        case "enter":
            if !m.Items[m.Selected].Disabled {
                return m, func() tea.Msg {
                    return menuMsg{
                        Type:   "select",
                        Action: m.Items[m.Selected].Action,
                    }
                }
            }
        case "esc":
            m.Visible = false
            return m, func() tea.Msg {
                return menuMsg{Type: "close"}
            }
        }
    }
    return m, nil
}

func (m Menu) View() string {
    if !m.Visible {
        return ""
    }

    var content strings.Builder
    
    // Apply style based on menu type
    style := menuStyle
    if m.Style == "docker" {
        style = dockerMenuStyle
    }

    // Render title
    content.WriteString(menuTitleStyle.Render(m.Title) + "\n\n")

    // Render items
    for i, item := range m.Items {
        itemStyle := menuItemStyle
        if i == m.Selected {
            itemStyle = selectedMenuItemStyle
        }
        if item.Disabled {
            itemStyle = disabledMenuItemStyle
        }

        // Format item with icon if present
        itemText := item.Title
        if item.Icon != "" {
            itemText = item.Icon + " " + itemText
        }

        // Add selection indicator
        if i == m.Selected {
            itemText = "> " + itemText
        } else {
            itemText = "  " + itemText
        }

        content.WriteString(itemStyle.Render(itemText))
        if item.Description != "" {
            content.WriteString("\n" + Subtle(item.Description))
        }
        content.WriteString("\n")
    }

    // Add help text
    content.WriteString("\n" + helpStyle.Render("↑/↓: navigate • enter: select • esc: cancel"))

    // Render with appropriate style and size
    return style.Width(m.Width).Render(content.String())
}

// Predefined menu configurations
func DockerOperationsMenu(repoName string) *Menu {
    items := []MenuItem{
        {Title: "Run Container", Icon: "🚀", Action: "run"},
        {Title: "Build Image", Icon: "📦", Action: "build"},
        {Title: "View Logs", Icon: "📝", Action: "logs"},
        {Title: "Stop Container", Icon: "⏹️", Action: "stop"},
        {Title: "Remove Container", Icon: "🗑️", Action: "remove"},
        {Title: "Cancel", Icon: "❌", Action: "cancel"},
    }
    return NewMenu("Docker Operations: "+repoName, items, "docker")
}

func ContainerActionsMenu() *Menu {
    items := []MenuItem{
        {Title: "View Details", Icon: "🔍", Action: "details"},
        {Title: "Shell Access", Icon: "💻", Action: "shell"},
        {Title: "View Logs", Icon: "📝", Action: "logs"},
        {Title: "Restart", Icon: "🔄", Action: "restart"},
        {Title: "Stop", Icon: "⏹️", Action: "stop"},
        {Title: "Remove", Icon: "🗑️", Action: "remove"},
    }
    return NewMenu("Container Actions", items, "container")
}