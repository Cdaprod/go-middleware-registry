// File: registry/registry.go
package registry

import (
	"fmt"
//	"io/ioutil"
	"os"
	"path/filepath"
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

// Registry manages a collection of RegistryItems.
type Registry struct {
	Items  map[string]RegistryItem
	Docker *client.Client
	Config *Config
}

// Config holds the configuration settings for the Registry.
type Config struct {
	ProjectsPath string
	DockerHost   string
	LogLevel     string
}

// NewRegistry initializes and returns a new Registry instance.
func NewRegistry() (*Registry, error) {
	config, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	docker, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	reg := &Registry{
		Items:  make(map[string]RegistryItem),
		Docker: docker,
		Config: config,
	}

	// Auto-discover repositories
	if err := reg.discoverRepositories(); err != nil {
		return nil, fmt.Errorf("failed to discover repositories: %w", err)
	}

	return reg, nil
}

// discoverRepositories scans the ProjectsPath for Git repositories with optional Dockerfiles.
func (r *Registry) discoverRepositories() error {
	entries, err := os.ReadDir(r.Config.ProjectsPath)
	if err != nil {
		return fmt.Errorf("failed to read projects directory: %w", err)
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
			item := RegistryItem{
				ID:            entry.Name(),
				Name:          entry.Name(),
				Type:          "repository",
				Status:        "active",
				Path:          projectPath,
				CreatedAt:     info.ModTime(),
				LastUpdated:   info.ModTime(),
				Enabled:       true,
				GitRepo:       repo,
				HasDockerfile: hasDockerfile,
			}
			r.Items[item.ID] = item
		}
	}

	return nil
}

// ListItems returns a slice of all RegistryItems.
func (r *Registry) ListItems() []RegistryItem {
	items := make([]RegistryItem, 0, len(r.Items))
	for _, item := range r.Items {
		items = append(items, item)
	}
	return items
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
