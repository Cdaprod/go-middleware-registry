
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

	DockerItems	    map[string]*DockerItem
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

	wg := &sync.WaitGroup{}

	// Initialize RegistryActor and Coordinator
	registryActor := NewRegistryActor(wg)
	coordinator := NewCoordinatorActor(wg, registryActor)

	reg := &Registry{
		RegistryActor: registryActor,
		Coordinator:   coordinator,
		Docker:        docker,
		Config:        config,
		wg:            wg,
	}

	// Auto-discover repositories
	if err := reg.discoverRepositories(); err != nil {
		return nil, fmt.Errorf("failed to discover repositories: %w", err)
	}

	// Start RegistryActor and Coordinator
	reg.RegistryActor.Start()
	reg.Coordinator.Start()

	return reg, nil
}

// discoverRepositories scans the ProjectsPath for Git repositories and adds them to the registry.
func (r *Registry) discoverRepositories() error {
    entries, err := os.ReadDir(r.Config.ProjectsPath)
    if err != nil {
        return fmt.Errorf("failed to read projects directory: %w", err)
    }

    baseRepoAdded := false
    var wg sync.WaitGroup
    errorsCh := make(chan error, len(entries))

    for _, entry := range entries {
        if !entry.IsDir() {
            continue
        }

        wg.Add(1)
        go func(entry os.DirEntry) {
            defer wg.Done()

            projectPath := filepath.Join(r.Config.ProjectsPath, entry.Name())
            _, err := git.PlainOpen(projectPath)
            isGitRepo := err == nil

            if !isGitRepo {
                return // Not a Git repo, skip
            }

            _, dockerfileErr := os.Stat(filepath.Join(projectPath, "Dockerfile"))
            hasDockerfile := dockerfileErr == nil

            r.RegistryActor.MsgChan <- AddRepo{
                Name: entry.Name(),
                Path: projectPath,
            }

            if hasDockerfile {
                fmt.Printf("Repository '%s' has a Dockerfile.\n", entry.Name())
                // You can also add more logic here if needed
            }

            if !baseRepoAdded {
                fmt.Printf("Setting repository '%s' as base-repo.\n", entry.Name())
                r.Coordinator.AddDependency(entry.Name(), []string{})
                baseRepoAdded = true
            } else {
                fmt.Printf("Repository '%s' discovered and added to the registry with base-repo dependency.\n", entry.Name())
                r.Coordinator.AddDependency(entry.Name(), []string{"base-repo"})
            }
        }(entry)
    }

    wg.Wait()
    close(errorsCh)

    // Check for errors from goroutines
    for err := range errorsCh {
        if err != nil {
            return err
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
