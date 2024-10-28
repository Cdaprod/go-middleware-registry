// File: registry/docker.go
package registry

import (
    "context"
    "fmt"
    "io"
    "os"
    "path/filepath"

    "github.com/docker/docker/api/types"
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/filters"
)

// DockerInfo holds information about a repository's Docker capabilities
type DockerInfo struct {
    HasDockerfile bool
    ImageID       string
    ImageTags     []string
    Containers    []types.Container
    LastBuild     string
}

// GetDockerInfo retrieves Docker-related information for a repository
func (r *Registry) GetDockerInfo(repoName string) (*DockerInfo, error) {
    item, exists := r.Items[repoName]
    if !exists {
        return nil, fmt.Errorf("repository not found: %s", repoName)
    }

    info := &DockerInfo{
        HasDockerfile: item.HasDockerfile,
    }

    if !item.HasDockerfile {
        return info, nil
    }

    // Get image information
    imageName := fmt.Sprintf("%s:latest", item.Name)
    images, err := r.Docker.ImageList(context.Background(), types.ImageListOptions{
        Filters: filters.NewArgs(filters.Arg("reference", imageName)),
    })
    if err == nil && len(images) > 0 {
        info.ImageID = images[0].ID
        info.ImageTags = images[0].RepoTags
    }

    // Get container information
    containers, err := r.Docker.ContainerList(context.Background(), types.ContainerListOptions{
        All: true,
        Filters: filters.NewArgs(filters.Arg("ancestor", imageName)),
    })
    if err == nil {
        info.Containers = containers
    }

    return info, nil
}

// BuildImage builds a Docker image for a repository
func (r *Registry) BuildImage(repoName string) error {
    item, exists := r.Items[repoName]
    if !exists {
        return fmt.Errorf("repository not found: %s", repoName)
    }

    if !item.HasDockerfile {
        return fmt.Errorf("repository does not have a Dockerfile: %s", repoName)
    }

    ctx := context.Background()
    buildContext := filepath.Join(item.Path)

    // Create build context tar
    tar, err := createBuildContext(buildContext)
    if err != nil {
        return fmt.Errorf("failed to create build context: %w", err)
    }

    // Build the image
    resp, err := r.Docker.ImageBuild(ctx, tar, types.ImageBuildOptions{
        Tags:       []string{fmt.Sprintf("%s:latest", item.Name)},
        Dockerfile: "Dockerfile",
    })
    if err != nil {
        return fmt.Errorf("failed to build image: %w", err)
    }
    defer resp.Body.Close()

    // Read the response
    _, err = io.Copy(os.Stdout, resp.Body)
    if err != nil {
        return fmt.Errorf("failed to read build response: %w", err)
    }

    return nil
}

// RunContainer starts a container from a repository's image
func (r *Registry) RunContainer(repoName string, config *container.Config) (string, error) {
    ctx := context.Background()

    // Create container
    resp, err := r.Docker.ContainerCreate(ctx, config, nil, nil, nil, "")
    if err != nil {
        return "", fmt.Errorf("failed to create container: %w", err)
    }

    // Start container
    if err := r.Docker.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
        return "", fmt.Errorf("failed to start container: %w", err)
    }

    return resp.ID, nil
}

// StopContainer stops a running container
func (r *Registry) StopContainer(containerID string) error {
    ctx := context.Background()
    timeout := int(10)
    return r.Docker.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

// GetContainerLogs retrieves logs from a container
func (r *Registry) GetContainerLogs(containerID string) (string, error) {
    ctx := context.Background()
    
    options := types.ContainerLogsOptions{
        ShowStdout: true,
        ShowStderr: true,
        Follow:     false,
        Tail:       "100",
    }

    logs, err := r.Docker.ContainerLogs(ctx, containerID, options)
    if err != nil {
        return "", err
    }
    defer logs.Close()

    // Read logs
    buf := new(strings.Builder)
    _, err = io.Copy(buf, logs)
    if err != nil {
        return "", err
    }

    return buf.String(), nil
}

// GetContainerStats retrieves container statistics
func (r *Registry) GetContainerStats(containerID string) (*types.Stats, error) {
    ctx := context.Background()
    
    stats, err := r.Docker.ContainerStats(ctx, containerID, false)
    if err != nil {
        return nil, err
    }
    defer stats.Body.Close()

    var containerStats types.Stats
    if err := json.NewDecoder(stats.Body).Decode(&containerStats); err != nil {
        return nil, err
    }

    return &containerStats, nil
}

// Utility functions
func createBuildContext(contextPath string) (io.Reader, error) {
    // Implementation of tar creation
    // This would create a tar of the build context
    return nil, nil // Placeholder
}