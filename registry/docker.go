// File: registry/docker.go
package registry

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"

    "github.com/docker/docker/api/types"
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/filters"
)

type DockerItem struct {
    Name       string `json:"name"`
    Image      string `json:"image"`
    ConfigPath string `json:"config_path,omitempty"`
}

// DockerInfo holds information about a repository's Docker capabilities.
type DockerInfo struct {
    HasDockerfile bool
    ImageID       string
    ImageTags     []string
    Containers    []types.Container
}

// GetDockerInfo retrieves Docker-related information for a repository.
func (r *Registry) GetDockerInfo(repoName string) (*DockerInfo, error) {
    repo, exists := r.RegistryActor.Repos[repoName]
    if !exists {
        return nil, fmt.Errorf("repository not found: %s", repoName)
    }

    info := &DockerInfo{
        HasDockerfile: repo.IsDocker,
    }

    if !repo.IsDocker {
        return info, nil
    }

    // Get image information.
    imageName := fmt.Sprintf("%s:latest", repo.Name)
    images, err := r.Docker.ImageList(context.Background(), types.ImageListOptions{
        Filters: filters.NewArgs(filters.Arg("reference", imageName)),
    })
    if err == nil && len(images) > 0 {
        info.ImageID = images[0].ID
        info.ImageTags = images[0].RepoTags
    }

    // Get container information.
    containers, err := r.Docker.ContainerList(context.Background(), types.ContainerListOptions{
        All:     true,
        Filters: filters.NewArgs(filters.Arg("ancestor", imageName)),
    })
    if err == nil {
        info.Containers = containers
    }

    return info, nil
}

// BuildImage builds a Docker image for a repository.
func (r *Registry) BuildImage(repoName string) error {
    repo, exists := r.RegistryActor.Repos[repoName]
    if !exists {
        return fmt.Errorf("repository not found: %s", repoName)
    }

    if !repo.IsDocker {
        return fmt.Errorf("repository does not have a Dockerfile: %s", repoName)
    }

    ctx := context.Background()
    buildContext := filepath.Join(repo.Path)

    // Create build context tar.
    tar, err := createBuildContext(buildContext)
    if err != nil {
        return fmt.Errorf("failed to create build context: %w", err)
    }

    // Build the image.
    resp, err := r.Docker.ImageBuild(ctx, tar, types.ImageBuildOptions{
        Tags:       []string{fmt.Sprintf("%s:latest", repo.Name)},
        Dockerfile: "Dockerfile",
    })
    if err != nil {
        return fmt.Errorf("failed to build image: %w", err)
    }
    defer resp.Body.Close()

    // Read the response.
    _, err = io.Copy(os.Stdout, resp.Body)
    if err != nil {
        return fmt.Errorf("failed to read build response: %w", err)
    }

    return nil
}

// Utility functions.
func createBuildContext(contextPath string) (io.Reader, error) {
    // Implementation of tar creation.
    // This would create a tar of the build context.
    return nil, nil // Placeholder.
}