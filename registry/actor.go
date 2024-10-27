// File: registry/actors.go
package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Message is the interface for all messages
type Message interface{}

// Commands for RegistryActor
type AddRepo struct {
	Name string
	Path string
}

type RemoveRepo struct {
	Name string
}

type ScanDir struct {
	Directory string
}

type ToggleRepo struct {
	Name string
}

type ConfigureRepo struct {
	Name string
}

type ReportCompletion struct {
	Name string
}

type ConfigureDocker struct{}
type ConfigurePipeline struct{}
type InitRepo struct{}

// RepoActor manages an individual repository
type RepoActor struct {
	Name        string
	Path        string
	Active      bool
	IsDocker    bool
	HasPipeline bool
	MsgChan     chan Message
	wg          *sync.WaitGroup
}

// NewRepoActor initializes a new RepoActor
func NewRepoActor(name, path string, wg *sync.WaitGroup) *RepoActor {
	return &RepoActor{
		Name:    name,
		Path:    path,
		Active:  true,
		MsgChan: make(chan Message),
		wg:      wg,
	}
}

// Start launches the RepoActor's goroutine
func (r *RepoActor) Start() {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for msg := range r.MsgChan {
			switch m := msg.(type) {
			case ToggleRepo:
				r.Active = !r.Active
				fmt.Printf("Repo '%s' toggled to %v\n", r.Name, r.Active)
			case ConfigureDocker:
				if r.Active && !r.IsDocker {
					r.addDockerfile()
					r.IsDocker = true
					fmt.Printf("Docker configured for repo '%s'\n", r.Name)
				}
			case ConfigurePipeline:
				if r.Active && !r.HasPipeline {
					r.setupPipeline()
					r.HasPipeline = true
					fmt.Printf("Pipeline configured for repo '%s'\n", r.Name)
				}
			case InitRepo:
				if r.Active {
					r.initializeRepo()
				}
			case ReportCompletion:
				fmt.Printf("Repo '%s' has completed its task.\n", m.Name)
			default:
				fmt.Printf("Repo '%s' received unknown message: %v\n", r.Name, msg)
			}
		}
	}()
}

// Helper methods for RepoActor
func (r *RepoActor) addDockerfile() {
	dockerfilePath := filepath.Join(r.Path, "Dockerfile")
	content := "FROM alpine:latest\nCMD [\"echo\", \"Hello, Docker!\"]\n"
	err := os.WriteFile(dockerfilePath, []byte(content), 0644)
	if err != nil {
		fmt.Printf("Error adding Dockerfile to '%s': %v\n", r.Name, err)
	}
}

func (r *RepoActor) setupPipeline() {
	pipelinePath := filepath.Join(r.Path, ".github", "workflows", "pipeline.yml")
	content := "name: CI\non: [push]\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v2\n"
	err := os.MkdirAll(filepath.Dir(pipelinePath), os.ModePerm)
	if err != nil {
		fmt.Printf("Error creating pipeline directory for '%s': %v\n", r.Name, err)
		return
	}
	err = os.WriteFile(pipelinePath, []byte(content), 0644)
	if err != nil {
		fmt.Printf("Error setting up pipeline for '%s': %v\n", r.Name, err)
	}
}

func (r *RepoActor) initializeRepo() {
	// Simulate repository initialization (e.g., cloning, setting up)
	fmt.Printf("Initializing repository '%s'...\n", r.Name)
	time.Sleep(1 * time.Second) // Simulate time-consuming task
	fmt.Printf("Repository '%s' initialized.\n", r.Name)
}

// RegistryActor manages all repositories
type RegistryActor struct {
	Repos      map[string]*RepoActor
	MsgChan    chan Message
	wg         *sync.WaitGroup
	mutex      sync.Mutex
}

// NewRegistryActor initializes a new RegistryActor
func NewRegistryActor(wg *sync.WaitGroup) *RegistryActor {
	return &RegistryActor{
		Repos:   make(map[string]*RepoActor),
		MsgChan: make(chan Message),
		wg:      wg,
	}
}

// Start launches the RegistryActor's goroutine
func (r *RegistryActor) Start() {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for msg := range r.MsgChan {
			switch m := msg.(type) {
			case AddRepo:
				r.addRepo(m.Name, m.Path)
			case RemoveRepo:
				r.removeRepo(m.Name)
			case ScanDir:
				r.scanDirectory(m.Directory)
			case ToggleRepo:
				r.toggleRepo(m.Name)
			case ConfigureRepo:
				r.configureRepo(m.Name)
			default:
				fmt.Printf("Registry received unknown message: %v\n", msg)
			}
		}
	}()
}

// Add a new repository
func (r *RegistryActor) addRepo(name, path string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, exists := r.Repos[name]; exists {
		fmt.Printf("Repository '%s' already exists.\n", name)
		return
	}
	repo := NewRepoActor(name, path, r.wg)
	repo.Start()
	r.Repos[name] = repo
	fmt.Printf("Repository '%s' added.\n", name)
	// Initialize the repo
	repo.MsgChan <- InitRepo{}
}

// Remove a repository
func (r *RegistryActor) removeRepo(name string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if repo, exists := r.Repos[name]; exists {
		repo.MsgChan <- ReportCompletion{Name: name}
		close(repo.MsgChan)
		delete(r.Repos, name)
		fmt.Printf("Repository '%s' removed.\n", name)
	} else {
		fmt.Printf("Repository '%s' not found.\n", name)
	}
}

// Toggle a repository's active state
func (r *RegistryActor) toggleRepo(name string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if repo, exists := r.Repos[name]; exists {
		repo.MsgChan <- ToggleRepo{Name: name}
	} else {
		fmt.Printf("Repository '%s' not found for toggling.\n", name)
	}
}

// Configure a repository
func (r *RegistryActor) configureRepo(name string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if repo, exists := r.Repos[name]; exists {
		// Example: Configure Docker and Pipeline
		repo.MsgChan <- ConfigureDocker{}
		repo.MsgChan <- ConfigurePipeline{}
	} else {
		fmt.Printf("Repository '%s' not found for configuration.\n", name)
	}
}

// Scan a directory for repositories
func (r *RegistryActor) scanDirectory(directory string) {
	fmt.Printf("Scanning directory '%s' for repositories...\n", directory)
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == ".git" {
			repoPath := filepath.Dir(path)
			repoName := filepath.Base(repoPath)
			r.MsgChan <- AddRepo{Name: repoName, Path: repoPath}
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error scanning directory: %v\n", err)
	}
}

// ListItems returns a slice of all RegistryItems.
func (r *RegistryActor) ListItems() []RegistryItem {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	items := make([]RegistryItem, 0, len(r.Repos))
	for _, repo := range r.Repos {
		item := RegistryItem{
			ID:            repo.Name,
			Name:          repo.Name,
			Type:          "repository",
			Status:        "active",
			Path:          repo.Path,
			CreatedAt:     time.Now(), // Placeholder
			LastUpdated:   time.Now(), // Placeholder
			Enabled:       repo.Active,
			GitRepo:       nil,         // Placeholder
			HasDockerfile: repo.IsDocker,
		}
		items = append(items, item)
	}
	return items
}

// CoordinatorActor manages dependencies and graph-based progression (Optional)
type CoordinatorActor struct {
	Graph      map[string][]string // Dependencies: key depends on values
	Completed  map[string]bool
	MsgChan    chan RepoCompleted
	wg         *sync.WaitGroup
	registry   *RegistryActor
	mutex      sync.Mutex
}

// RepoCompleted message signifies a repo has completed its task
type RepoCompleted struct {
	Name string
}

// NewCoordinatorActor initializes a new CoordinatorActor
func NewCoordinatorActor(wg *sync.WaitGroup, registry *RegistryActor) *CoordinatorActor {
	return &CoordinatorActor{
		Graph:     make(map[string][]string),
		Completed: make(map[string]bool),
		MsgChan:   make(chan RepoCompleted),
		wg:        wg,
		registry:  registry,
	}
}

// Start launches the CoordinatorActor's goroutine
func (c *CoordinatorActor) Start() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for msg := range c.MsgChan {
			c.handleCompletion(msg)
		}
	}()
}

// AddDependency adds a dependency to the graph
func (c *CoordinatorActor) AddDependency(repo string, dependsOn []string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Graph[repo] = dependsOn
}

// handleCompletion processes the completion of a repository task
func (c *CoordinatorActor) handleCompletion(msg RepoCompleted) {
	c.mutex.Lock()
	c.Completed[msg.Name] = true
	fmt.Printf("Coordinator: Repository '%s' completed.\n", msg.Name)

	// Check which repositories can now proceed
	for repo, deps := range c.Graph {
		if c.Completed[repo] {
			continue // Already completed
		}
		allDepsMet := true
		for _, dep := range deps {
			if !c.Completed[dep] {
				allDepsMet = false
				break
			}
		}
		if allDepsMet {
			fmt.Printf("Coordinator: All dependencies met for '%s'. Proceeding...\n", repo)
			// Send a message to configure the repo
			c.registry.Repos[repo].MsgChan <- ConfigureDocker{}
			c.registry.Repos[repo].MsgChan <- ConfigurePipeline{}
			c.Completed[repo] = true // Mark as processed
		}
	}

	c.mutex.Unlock()
}