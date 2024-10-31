// File: registry/registry.go
package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/docker/docker/client"
	git "github.com/go-git/go-git/v5"
)

// RegistryItem represents an individual repository in the registry.
type RegistryItem struct {
	ID            string
	Name          string
	Type          string
	Status        string
	Path          string
	CreatedAt     time.Time
	LastUpdated   time.Time
	Enabled       bool
	GitRepo       *git.Repository
	HasDockerfile bool
}

// Registry manages a collection of RepoActors and the RegistryActor.
type Registry struct {
	RegistryActor  *RegistryActor
	Coordinator    *CoordinatorActor
	Docker         *client.Client
	Config         *Config
	wg             *sync.WaitGroup
}

// Config holds the configuration settings for the Registry.
type Config struct {
    ProjectsPath string
    DockerHost   string
    LogLevel     string
}

// OptsFunc defines the function signature for configuration options.
type OptsFunc func(*Config)

// WithProjectsPath sets the ProjectsPath configuration.
func WithProjectsPath(path string) OptsFunc {
    return func(c *Config) {
        c.ProjectsPath = path
    }
}

// WithDockerHost sets the DockerHost configuration.
func WithDockerHost(host string) OptsFunc {
    return func(c *Config) {
        c.DockerHost = host
    }
}

// WithLogLevel sets the LogLevel configuration.
func WithLogLevel(level string) OptsFunc {
    return func(c *Config) {
        c.LogLevel = level
    }
}

// NewRegistry initializes and returns a new Registry instance.
func NewRegistry(opts ...OptsFunc) (*Registry, error) {
    // Set default configuration values.
    config := &Config{
        ProjectsPath: "/home/cdaprod/Projects",
        DockerHost:   "unix:///var/run/docker.sock",
        LogLevel:     "info",
    }

    // Apply options.
    for _, opt := range opts {
        opt(config)
    }

    // Initialize Docker client with specified host.
    docker, err := client.NewClientWithOpts(client.FromEnv, client.WithHost(config.DockerHost))
    if err != nil {
        return nil, fmt.Errorf("failed to create docker client: %w", err)
    }

    wg := &sync.WaitGroup{}

    // Initialize RegistryActor and Coordinator.
    registryActor := NewRegistryActor(wg)
    coordinator := NewCoordinatorActor(wg, registryActor)

    reg := &Registry{
        RegistryActor: registryActor,
        Coordinator:   coordinator,
        Docker:        docker,
        Config:        config,
        wg:            wg,
    }

    // Auto-discover repositories.
    if err := reg.discoverRepositories(); err != nil {
        return nil, fmt.Errorf("failed to discover repositories: %w", err)
    }

    // Start RegistryActor and Coordinator.
    reg.RegistryActor.Start()
    reg.Coordinator.Start()

    return reg, nil
}

// discoverRepositories scans the ProjectsPath for Git repositories and adds them to the registry.
func (r *Registry) discoverRepositories() error {
    entries, err := os.ReadDir(r.Config.ProjectsPath)
    if err != nil {
        return fmt.Errorf("failed to read projects directory '%s': %w", r.Config.ProjectsPath, err)
    }

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Use entry.Info() to retrieve os.FileInfo
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("failed to retrieve file info: %w", err)
		}

		projectPath := filepath.Join(r.Config.ProjectsPath, entry.Name())

		// Check if it's a git repository
		repo, err := git.PlainOpen(projectPath)
		isGitRepo := err == nil

		// Check for Dockerfile
		_, dockerfileErr := os.Stat(filepath.Join(projectPath, "Dockerfile"))
		hasDockerfile := dockerfileErr == nil

		if isGitRepo {
			// Add the repository to the RegistryActor
			r.RegistryActor.MsgChan <- AddRepo{
				Name: entry.Name(),
				Path: projectPath,
			}

			// Optionally add to the Coordinator for dependency management
			// Example: repoName depends on "base-repo"
			if entry.Name() != "base-repo" {
				r.Coordinator.AddDependency(entry.Name(), []string{"base-repo"})
			}

			fmt.Printf("Repository '%s' discovered and added to the registry.\n", entry.Name())
		}
	}

	return nil
}

// ListItems returns a list of all RegistryItems (repositories).
func (r *Registry) ListItems() []RegistryItem {
	return r.RegistryActor.ListItems()
}

// loadConfig loads configuration settings. Replace this with actual config loading logic as needed.
func loadConfig() (*Config, error) {
	// Simulating config loading using hardcoded values for simplicity.
	// You can integrate Viper or another config library as needed.
	return &Config{
		ProjectsPath: "/home/cdaprod/Projects",
		DockerHost:   "unix:///var/run/docker.sock",
		LogLevel:     "info",
	}, nil
}