// File: internal/ui/docker_manager.go
package ui

import (
    "context"
    "fmt"
    "strings"
    "sync"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/spinner"
    "github.com/charmbracelet/bubbles/viewport"
    "github.com/docker/docker/api/types/container"
    "github.com/Cdaprod/go-middleware-registry/registry"
)

// Message types for Docker operations
type (
    buildCompleteMsg struct {
        repoName string
        success  bool
        error    error
    }

    containerStartedMsg struct {
        containerID string
        error      error
    }

    containerStoppedMsg struct {
        containerID string
        error      error
    }

    logsUpdatedMsg struct {
        containerID string
        logs       string
        error      error
    }

    statsUpdatedMsg struct {
        containerID string
        stats      containerStats
        error      error
    }
)

// containerStats holds simplified container statistics
type containerStats struct {
    CPUPercentage    float64
    MemoryUsage      float64
    MemoryLimit      float64
    NetworkRx        float64
    NetworkTx        float64
    RunningProcesses int64
}

// DockerManager handles Docker operations in the UI
type DockerManager struct {
    registry    *registry.Registry
    activeRepo  string
    containers  map[string]*containerView
    viewports   map[string]viewport.Model
    spinners    map[string]spinner.Model
    operations  map[string]string
    mu         sync.Mutex
    width      int
    height     int
}

// containerView represents a single container's view state
type containerView struct {
    ID        string
    Name      string
    Status    string
    Logs      string
    Stats     containerStats
    Selected  bool
    viewport  viewport.Model
}

func NewDockerManager(reg *registry.Registry) *DockerManager {
    return &DockerManager{
        registry:    reg,
        containers:  make(map[string]*containerView),
        viewports:   make(map[string]viewport.Model),
        spinners:    make(map[string]spinner.Model),
        operations:  make(map[string]string),
    }
}

// startOperation begins a Docker operation with a spinner
func (dm *DockerManager) startOperation(id, operation string) {
    dm.mu.Lock()
    defer dm.mu.Unlock()

    s := spinner.New()
    s.Spinner = spinner.Dot
    s.Style = spinnerStyle
    dm.spinners[id] = s
    dm.operations[id] = operation
}

// buildImage initiates an image build with progress
func (dm *DockerManager) buildImage(repoName string) tea.Cmd {
    return func() tea.Msg {
        err := dm.registry.BuildImage(repoName)
        return buildCompleteMsg{
            repoName: repoName,
            success:  err == nil,
            error:    err,
        }
    }
}

// runContainer starts a container and begins monitoring
func (dm *DockerManager) runContainer(repoName string) tea.Cmd {
    return func() tea.Msg {
        config := &container.Config{
            Image: repoName + ":latest",
            Tty:   true,
        }
        
        containerID, err := dm.registry.RunContainer(repoName, config)
        if err != nil {
            return containerStartedMsg{error: err}
        }

        // Start monitoring if successfully started
        go dm.monitorContainer(containerID)

        return containerStartedMsg{
            containerID: containerID,
        }
    }
}

// monitorContainer handles continuous container monitoring
func (dm *DockerManager) monitorContainer(containerID string) {
    ctx := context.Background()
    logsCh := make(chan string)
    statsCh := make(chan containerStats)
    
    // Monitor logs
    go func() {
        for {
            logs, err := dm.registry.GetContainerLogs(containerID)
            if err != nil {
                continue
            }
            logsCh <- logs
            time.Sleep(time.Second)
        }
    }()

    // Monitor stats
    go func() {
        for {
            stats, err := dm.registry.GetContainerStats(containerID)
            if err != nil {
                continue
            }
            
            // Convert stats to our simplified format
            statsCh <- containerStats{
                CPUPercentage:    calculateCPUPercentage(stats),
                MemoryUsage:      float64(stats.MemoryStats.Usage),
                MemoryLimit:      float64(stats.MemoryStats.Limit),
                NetworkRx:        float64(stats.Networks["eth0"].RxBytes),
                NetworkTx:        float64(stats.Networks["eth0"].TxBytes),
                RunningProcesses: stats.PidsStats.Current,
            }
            time.Sleep(time.Second)
        }
    }()

    // Update UI with monitoring data
    for {
        select {
        case logs := <-logsCh:
            dm.updateLogs(containerID, logs)
        case stats := <-statsCh:
            dm.updateStats(containerID, stats)
        }
    }
}

// Update handles all Docker-related messages
func (dm *DockerManager) Update(msg tea.Msg) tea.Cmd {
    var cmds []tea.Cmd

    switch msg := msg.(type) {
    case buildCompleteMsg:
        delete(dm.spinners, msg.repoName)
        delete(dm.operations, msg.repoName)
        if msg.error != nil {
            return dm.showError(msg.error)
        }
        return dm.showSuccess(fmt.Sprintf("Built image for %s", msg.repoName))

    case containerStartedMsg:
        if msg.error != nil {
            return dm.showError(msg.error)
        }
        dm.containers[msg.containerID] = &containerView{
            ID:     msg.containerID,
            Status: "running",
        }
        return dm.showSuccess(fmt.Sprintf("Started container %s", msg.containerID))

    case logsUpdatedMsg:
        if c, exists := dm.containers[msg.containerID]; exists {
            c.Logs = msg.logs
            if vp, ok := dm.viewports[msg.containerID]; ok {
                vp.SetContent(msg.logs)
                dm.viewports[msg.containerID] = vp
            }
        }

    case statsUpdatedMsg:
        if c, exists := dm.containers[msg.containerID]; exists {
            c.Stats = msg.stats
        }
    }

    // Update spinners
    for id, s := range dm.spinners {
        var cmd tea.Cmd
        s, cmd = s.Update(msg)
        dm.spinners[id] = s
        cmds = append(cmds, cmd)
    }

    return tea.Batch(cmds...)
}

// View renders the Docker manager UI
func (dm *DockerManager) View() string {
    var b strings.Builder

    // Show active operations with spinners
    for id, operation := range dm.operations {
        if spinner, ok := dm.spinners[id]; ok {
            b.WriteString(fmt.Sprintf("%s %s...\n", spinner.View(), operation))
        }
    }

    // Show running containers
    if len(dm.containers) > 0 {
        b.WriteString("\nRunning Containers:\n")
        for _, c := range dm.containers {
            style := containerStyle
            if c.Selected {
                style = activeContainerStyle
            }

            stats := fmt.Sprintf("CPU: %.1f%% | MEM: %.1f/%.1fMB | Procs: %d",
                c.Stats.CPUPercentage,
                c.Stats.MemoryUsage/1024/1024,
                c.Stats.MemoryLimit/1024/1024,
                c.Stats.RunningProcesses,
            )

            content := fmt.Sprintf("%s\n%s\n%s", c.ID[:12], c.Status, stats)
            b.WriteString(style.Render(content) + "\n")

            // Show logs if container is selected
            if c.Selected {
                if vp, ok := dm.viewports[c.ID]; ok {
                    b.WriteString(vp.View() + "\n")
                }
            }
        }
    }

    return b.String()
}

// Helper functions for CPU percentage calculation
func calculateCPUPercentage(stats *types.Stats) float64 {
    // CPU percentage calculation logic
    return 0.0 // Placeholder
}

// Utility functions for showing success/error messages
func (dm *DockerManager) showSuccess(message string) tea.Cmd {
    return nil // Implement message display
}

func (dm *DockerManager) showError(err error) tea.Cmd {
    return nil // Implement error display
}