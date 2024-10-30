// File: internal/ui/styles.go
package ui

import (
    "github.com/charmbracelet/lipgloss"
)

var (
    // Color scheme
    primaryColor    = lipgloss.Color("#874BFD")
    secondaryColor  = lipgloss.Color("#7D56F4")
    successColor    = lipgloss.Color("#04B575")
    warningColor    = lipgloss.Color("#FFA629")
    errorColor      = lipgloss.Color("#FF0000")
    textColor       = lipgloss.Color("#FFFFFF")
    dimmedColor     = lipgloss.Color("#666666")
    highlightColor  = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
    backgroundColor = lipgloss.Color("#1A1A1A")

    // Base styles
    docStyle = lipgloss.NewStyle().
        Padding(1, 2, 1, 2).
        Background(backgroundColor)

    // Tab styles
    inactiveTabBorder = tabBorderWithBottom("┴", "─", "┴")
    activeTabBorder   = tabBorderWithBottom("┘", " ", "└")

    tabStyle = lipgloss.NewStyle().
        Border(inactiveTabBorder, true).
        BorderForeground(primaryColor).
        Padding(0, 1).
        Background(backgroundColor)

    activeTabStyle = tabStyle.Copy().
        Border(activeTabBorder, true).
        Bold(true).
        Foreground(textColor)

    // Window and container styles
    windowStyle = lipgloss.NewStyle().
        BorderForeground(primaryColor).
        Padding(2, 0).
        Border(lipgloss.NormalBorder()).
        UnsetBorderTop().
        Background(backgroundColor)

    containerStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(primaryColor).
        Padding(0, 1).
        Background(backgroundColor)

    activeContainerStyle = containerStyle.Copy().
        BorderForeground(successColor).
        Bold(true)

    // Docker-specific styles
    dockerMenuStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(primaryColor).
        Padding(1, 2).
        Background(backgroundColor)

    dockerPopupStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(primaryColor).
        Padding(1, 2).
        Background(backgroundColor)

    // List styles
    listHeaderStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(primaryColor).
        Padding(0, 1)

    listItemStyle = lipgloss.NewStyle().
        Padding(0, 1)

    selectedItemStyle = listItemStyle.Copy().
        Background(primaryColor).
        Foreground(textColor)

    // Message styles
    errorStyle = lipgloss.NewStyle().
        Foreground(errorColor).
        Bold(true).
        Padding(0, 1)

    successStyle = lipgloss.NewStyle().
        Foreground(successColor).
        Bold(true).
        Padding(0, 1)

    warningStyle = lipgloss.NewStyle().
        Foreground(warningColor).
        Bold(true).
        Padding(0, 1)

    infoStyle = lipgloss.NewStyle().
        Foreground(primaryColor).
        Padding(0, 1)

    // Help and status styles
    helpStyle = lipgloss.NewStyle().
        Foreground(dimmedColor).
        Padding(1, 0)

    statusStyle = lipgloss.NewStyle().
        Foreground(textColor).
        Background(primaryColor).
        Padding(0, 1)

    // Spinner style
    spinnerStyle = lipgloss.NewStyle().
        Foreground(primaryColor).
        Bold(true)

    // Container monitoring styles
    monitorHeaderStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(primaryColor).
        Padding(0, 1).
        BorderStyle(lipgloss.RoundedBorder()).
        BorderForeground(primaryColor)

    monitorDataStyle = lipgloss.NewStyle().
        Foreground(textColor).
        Padding(0, 2)

    statsStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(primaryColor).
        Padding(1)

    // Log styles
    logStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(primaryColor).
        Padding(0, 1).
        MaxHeight(10)

    logEntryStyle = lipgloss.NewStyle().
        Foreground(textColor)

    logErrorStyle = logEntryStyle.Copy().
        Foreground(errorColor)

    logSuccessStyle = logEntryStyle.Copy().
        Foreground(successColor)

    // Button styles
    buttonStyle = lipgloss.NewStyle().
        Padding(0, 3).
        Bold(true)

    activeButtonStyle = buttonStyle.Copy().
        Background(primaryColor).
        Foreground(textColor)

    // Dialog styles
    dialogStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(primaryColor).
        Padding(1, 2).
        Background(backgroundColor)

    dialogTitleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(primaryColor)

    // Layout helpers
    dividerStyle = lipgloss.NewStyle().
        Foreground(dimmedColor).
        SetString("─").
        Padding(0, 1)

    indentStyle = lipgloss.NewStyle().
        PaddingLeft(2)
)

// Helper functions
func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
    border := lipgloss.RoundedBorder()
    border.BottomLeft = left
    border.Bottom = middle
    border.BottomRight = right
    return border
}

// Utility functions for common text styling
func Subtle(s string) string {
    return lipgloss.NewStyle().Foreground(dimmedColor).Render(s)
}

func Highlight(s string) string {
    return lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render(s)
}

func Emphasis(s string) string {
    return lipgloss.NewStyle().Bold(true).Render(s)
}

// Layout helper functions
func JoinHorizontal(styles ...string) string {
    return lipgloss.JoinHorizontal(lipgloss.Top, styles...)
}

func JoinVertical(styles ...string) string {
    return lipgloss.JoinVertical(lipgloss.Left, styles...)
}

func Divider() string {
    return dividerStyle.Render(strings.Repeat("─", 50))
}