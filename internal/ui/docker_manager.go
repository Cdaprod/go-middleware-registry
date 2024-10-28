// File: internal/ui/docker_manager.go
package ui

import (
    "encoding/json"
    "bufio"
    "bytes"
    "context"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "time"
    "archive/tar"
    
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/spinner"
    "github.com/charmbracelet/bubbles/viewport"
    "github.com/docker/docker/api/types"
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/client"
    "github.com/Cdaprod/go-middleware-registry/registry"
)

const (
    MsgTypeError   = "error"
    MsgTypeSuccess = "success"
    MsgTypeInfo    = "info"
    MsgTypeWarning = "warning"
)

// Message types
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

// Stats tracking
type containerStats struct {
    CPUPercentage    float64
    MemoryUsage      float64
    MemoryLimit      float64
    NetworkRx        float64
    NetworkTx        float64
    RunningProcesses int64
}

// Container view representation
type containerView struct {
    ID        string
    Name      string
    Status    string
    Logs      string
    Stats     containerStats
    Selected  bool
    viewport  viewport.Model
}

// DockerManager handles all Docker operations
type DockerManager struct {
    // Core components
    client     *client.Client
    registry   *registry.Registry
    containers *ContainerManager
    
    // State tracking
    activeRepo   string
    containerID  string
    status      map[string]string
    logs        map[string]string
    
    // UI components
    menu       *Menu
    viewports  map[string]viewport.Model
    spinners   map[string]spinner.Model
    operations map[string]string
    
    // Dimensions
    width    int
    height   int
    
    mu      sync.Mutex
}

func NewDockerManager(reg *registry.Registry) (*DockerManager, error) {
    docker, err := client.NewClientWithOpts(client.FromEnv)
    if err != nil {
        return nil, fmt.Errorf("failed to create docker client: %w", err)
    }

    containers, err := NewContainerManager()
    if err != nil {
        return nil, fmt.Errorf("failed to create container manager: %w", err)
    }

    return &DockerManager{
        client:     docker,
        registry:   reg,
        containers: containers,
        status:     make(map[string]string),
        logs:      make(map[string]string),
        viewports: make(map[string]viewport.Model),
        spinners:  make(map[string]spinner.Model),
        operations: make(map[string]string),
    }, nil
}

// Operation initiation
func (dm *DockerManager) startOperation(id, operation string) {
    dm.mu.Lock()
    defer dm.mu.Unlock()

    s := spinner.New()
    s.Spinner = spinner.Dot
    s.Style = spinnerStyle
    dm.spinners[id] = s
    dm.operations[id] = operation
}

// UI Update handling
func (dm *DockerManager) Update(msg tea.Msg) tea.Cmd {
    var cmds []tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        if dm.menu != nil && dm.menu.Visible {
            menu, cmd := dm.menu.Update(msg)
            dm.menu = menu
            if cmd != nil {
                cmds = append(cmds, cmd)
            }
        } else if dm.containers != nil {
            cmd := dm.containers.Update(msg)
            if cmd != nil {
                cmds = append(cmds, cmd)
            }
        }

    case menuMsg:
        switch msg.Type {
        case "select":
            cmd := dm.handleMenuAction(msg.Action)
            if cmd != nil {
                cmds = append(cmds, cmd)
            }
        case "close":
            dm.menu = nil
        }

    case dockerMsg:
        cmd := dm.handleDockerMessage(msg)
        if cmd != nil {
            cmds = append(cmds, cmd)
        }

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
        dm.containerID = msg.containerID
        return dm.showSuccess(fmt.Sprintf("Started container %s", msg.containerID))

    case logsUpdatedMsg:
        if c, exists := dm.containers.containers[msg.containerID]; exists {
            c.Logs = msg.logs
            if vp, ok := dm.viewports[msg.containerID]; ok {
                vp.SetContent(msg.logs)
                dm.viewports[msg.containerID] = vp
            }
        }

    case statsUpdatedMsg:
        if c, exists := dm.containers.containers[msg.containerID]; exists {
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
    if len(dm.containers.containers) > 0 {
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

func calculateCPUPercentage(stats *types.Stats) float64 {
    cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage) - float64(stats.PreCPUStats.CPUUsage.TotalUsage)
    systemDelta := float64(stats.CPUStats.SystemUsage) - float64(stats.PreCPUStats.SystemUsage)

    if systemDelta > 0.0 && cpuDelta > 0.0 {
        return (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
    }
    return 0.0
}

func (dm *DockerManager) showSuccess(message string) tea.Cmd {
    return func() tea.Msg {
        return statusMsg{
            Type:    "success",
            Message: message,
        }
    }
}

func (dm *DockerManager) showError(err error) tea.Cmd {
    return func() tea.Msg {
        return statusMsg{
            Type:    "error",
            Message: err.Error(),
        }
    }
}
func (dm *DockerManager) ShowOperationsMenu(repoName string) tea.Cmd {
    dm.activeRepo = repoName
    dm.menu = DockerOperationsMenu(repoName)
    return nil
}

func (dm *DockerManager) handleMenuAction(action string) tea.Cmd {
    switch action {
    case "run":
        return dm.runContainer
    case "build":
        return dm.buildImage
    case "logs":
        return dm.viewLogs
    case "stop":
        return dm.stopContainer
    case "remove":
        return dm.removeContainer
    }
    return nil
}

// In docker_manager.go, add these implementations:

// Docker operations implementation
func (dm *DockerManager) runContainer() tea.Msg {
    ctx := context.Background()
    
    if dm.activeRepo == "" {
        return dockerMsg{
            Type:    MsgTypeError,
            Message: "No repository selected",
        }
    }

    // Create container configuration
    config := &container.Config{
        Image: dm.activeRepo + ":latest",
        Tty:   true,
    }

    // Create container
    resp, err := dm.client.ContainerCreate(ctx, config, nil, nil, nil, "")
    if err != nil {
        return dockerMsg{
            Type:    MsgTypeError,
            Message: fmt.Sprintf("Failed to create container: %v", err),
        }
    }

    // Start container
    if err := dm.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
        return dockerMsg{
            Type:    MsgTypeError,
            Message: fmt.Sprintf("Failed to start container: %v", err),
        }
    }

    // Add to container manager
    dm.containers.AddContainer(&ContainerView{
        id:   resp.ID,
        name: dm.activeRepo,
    })

    return dockerMsg{
        Type:        MsgTypeSuccess,
        Message:     "Container started successfully",
        ContainerID: resp.ID,
    }
}

func (dm *DockerManager) buildImage() tea.Msg {
    ctx := context.Background()

    if dm.activeRepo == "" {
        return dockerMsg{
            Type:    MsgTypeError,
            Message: "No repository selected",
        }
    }

    // Create build context tar
    buildCtx, err := createBuildContext(dm.activeRepo)
    if err != nil {
        return dockerMsg{
            Type:    MsgTypeError,
            Message: fmt.Sprintf("Failed to create build context: %v", err),
        }
    }

    // Build options
    options := types.ImageBuildOptions{
        Tags:       []string{dm.activeRepo + ":latest"},
        Dockerfile: "Dockerfile",
    }

    // Build the image
    response, err := dm.client.ImageBuild(ctx, buildCtx, options)
    if err != nil {
        return dockerMsg{
            Type:    MsgTypeError,
            Message: fmt.Sprintf("Failed to build image: %v", err),
        }
    }
    defer response.Body.Close()

    // Read build output
    var output strings.Builder
    scanner := bufio.NewScanner(response.Body)
    for scanner.Scan() {
        output.WriteString(scanner.Text() + "\n")
    }

    return dockerMsg{
        Type:    MsgTypeSuccess,
        Message: "Image built successfully",
        Data:    output.String(),
    }
}

func (dm *DockerManager) viewLogs() tea.Msg {
    if dm.activeRepo == "" || dm.containerID == "" {
        return dockerMsg{
            Type:    MsgTypeError,
            Message: "No active container",
        }
    }

    logs, err := dm.containers.GetContainerLogs(dm.containerID)
    if err != nil {
        return dockerMsg{
            Type:    MsgTypeError,
            Message: fmt.Sprintf("Failed to get logs: %v", err),
        }
    }

    return dockerMsg{
        Type:        MsgTypeSuccess,
        ContainerID: dm.containerID,
        Data:        logs,
    }
}

func (dm *DockerManager) stopContainer() tea.Msg {
    ctx := context.Background()

    if dm.containerID == "" {
        return dockerMsg{
            Type:    MsgTypeError,
            Message: "No active container",
        }
    }

    timeout := int(10)
    err := dm.client.ContainerStop(ctx, dm.containerID, container.StopOptions{Timeout: &timeout})
    if err != nil {
        return dockerMsg{
            Type:    MsgTypeError,
            Message: fmt.Sprintf("Failed to stop container: %v", err),
        }
    }

    return dockerMsg{
        Type:        MsgTypeSuccess,
        Message:     "Container stopped successfully",
        ContainerID: dm.containerID,
    }
}

func (dm *DockerManager) removeContainer() tea.Msg {
    ctx := context.Background()

    if dm.containerID == "" {
        return dockerMsg{
            Type:    MsgTypeError,
            Message: "No active container",
        }
    }

    err := dm.client.ContainerRemove(ctx, dm.containerID, types.ContainerRemoveOptions{
        Force: true,
    })
    if err != nil {
        return dockerMsg{
            Type:    MsgTypeError,
            Message: fmt.Sprintf("Failed to remove container: %v", err),
        }
    }

    // Remove from container manager
    dm.containers.RemoveContainer(dm.containerID)

    return dockerMsg{
        Type:    MsgTypeSuccess,
        Message: "Container removed successfully",
    }
}

func (dm *DockerManager) handleDockerMessage(msg dockerMsg) tea.Cmd {
    switch msg.Type {
    case MsgTypeError:
        return func() tea.Msg {
            return statusMsg{
                Type:    "error",
                Message: msg.Message,
            }
        }
    case MsgTypeSuccess:
        if msg.ContainerID != "" {
            dm.containerID = msg.ContainerID
        }
        return func() tea.Msg {
            return statusMsg{
                Type:    "success",
                Message: msg.Message,
            }
        }
    }
    return nil
}

// Helper function to create build context
func createBuildContext(repoPath string) (io.Reader, error) {
    var buf bytes.Buffer
    tw := tar.NewWriter(&buf)

    // Walk through the repository directory
    err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        // Create tar header
        header, err := tar.FileInfoHeader(info, info.Name())
        if err != nil {
            return err
        }

        // Update header name to be relative to repo path
        relPath, err := filepath.Rel(repoPath, path)
        if err != nil {
            return err
        }
        header.Name = relPath

        // Write header
        if err := tw.WriteHeader(header); err != nil {
            return err
        }

        // If not a directory, write file content
        if !info.IsDir() {
            file, err := os.Open(path)
            if err != nil {
                return err
            }
            defer file.Close()

            if _, err := io.Copy(tw, file); err != nil {
                return err
            }
        }

        return nil
    })

    if err != nil {
        return nil, err
    }

    // Close tar writer
    if err := tw.Close(); err != nil {
        return nil, err
    }

    return &buf, nil
}

func (dm *DockerManager) SelectContainer(containerID string) {
    dm.mu.Lock()
    defer dm.mu.Unlock()
    
    // Deselect all containers first
    for _, c := range dm.containers.containers {
        c.Selected = false
    }
    
    // Select the specified container
    if container, exists := dm.containers.containers[containerID]; exists {
        container.Selected = true
        // Create viewport if doesn't exist
        if _, ok := dm.viewports[containerID]; !ok {
            vp := viewport.New(dm.width-4, 10) // Adjust height as needed
            vp.Style = viewportStyle
            dm.viewports[containerID] = vp
        }
    }
}

// Add this function
func (dm *DockerManager) monitorContainer(containerID string) {
    // Start stats monitoring
    go dm.monitorStats(containerID)
    
    // Start logs monitoring
    go func() {
        ctx := context.Background()
        options := types.ContainerLogsOptions{
            ShowStdout: true,
            ShowStderr: true,
            Follow:     true,
            Timestamps: true,
        }

        logs, err := dm.client.ContainerLogs(ctx, containerID, options)
        if err != nil {
            return
        }
        defer logs.Close()

        scanner := bufio.NewScanner(logs)
        for scanner.Scan() {
            dm.mu.Lock()
            if container, exists := dm.containers.containers[containerID]; exists {
                container.Logs += scanner.Text() + "\n"
                if vp, ok := dm.viewports[containerID]; ok {
                    vp.SetContent(container.Logs)
                    dm.viewports[containerID] = vp
                }
            }
            dm.mu.Unlock()
        }
    }()
}