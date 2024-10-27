// File: main.go (Graceful Shutdown)
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Cdaprod/go-middleware-registry/internal/ui"
	"github.com/Cdaprod/go-middleware-registry/registry"
	"github.com/spf13/cobra"
)

// Global Registry instance
var globalRegistry *registry.Registry

// Root command for the CLI application.
var rootCmd = &cobra.Command{
	Use:   "registry",
	Short: "Registry CLI for managing repositories",
	Long:  "A CLI application for managing repositories in /home/cdaprod/Projects with support for Git repositories and Docker containers.",
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all repositories",
	Run: func(cmd *cobra.Command, args []string) {
		if globalRegistry == nil {
			fmt.Println("Registry not initialized.")
			os.Exit(1)
		}
		items := globalRegistry.ListItems()
		displayTable(items)
	},
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan projects directory for repositories",
	Run: func(cmd *cobra.Command, args []string) {
		if globalRegistry == nil {
			fmt.Println("Registry not initialized.")
			os.Exit(1)
		}
		globalRegistry.Actor.MsgChan <- ScanDir{Directory: globalRegistry.Config.ProjectsPath}
		fmt.Printf("Scan initiated for directory: %s\n", globalRegistry.Config.ProjectsPath)
	},
}

var infoCmd = &cobra.Command{
	Use:   "info [repository]",
	Short: "Show detailed information about a repository",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if globalRegistry == nil {
			fmt.Println("Registry not initialized.")
			os.Exit(1)
		}

		item, exists := globalRegistry.Actor.Repos[args[0]]
		if !exists {
			fmt.Printf("Repository '%s' not found\n", args[0])
			os.Exit(1)
		}

		displayRepoInfo(item)
	},
}

var interactiveCmd = &cobra.Command{
	Use:   "interactive",
	Short: "Launch interactive TUI",
	Run: func(cmd *cobra.Command, args []string) {
		if globalRegistry == nil {
			fmt.Println("Registry not initialized.")
			os.Exit(1)
		}

		if err := ui.LaunchTUI(globalRegistry); err != nil {
			fmt.Printf("Error starting TUI: %v\n", err)
			os.Exit(1)
		}
	},
}

var toggleCmd = &cobra.Command{
	Use:   "toggle [repository]",
	Short: "Toggle a repository's active state",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if globalRegistry == nil {
			fmt.Println("Registry not initialized.")
			os.Exit(1)
		}

		globalRegistry.Actor.MsgChan <- ToggleRepo{Name: args[0]}
		fmt.Printf("Toggle command sent for repository: %s\n", args[0])
	},
}

var configureCmd = &cobra.Command{
	Use:   "configure [repository]",
	Short: "Configure a repository with Docker and Pipeline",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if globalRegistry == nil {
			fmt.Println("Registry not initialized.")
			os.Exit(1)
		}

		globalRegistry.Actor.MsgChan <- ConfigureRepo{Name: args[0]}
		fmt.Printf("Configure command sent for repository: %s\n", args[0])
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(interactiveCmd)
	rootCmd.AddCommand(toggleCmd)
	rootCmd.AddCommand(configureCmd)
}

func main() {
	var err error
	globalRegistry, err = registry.NewRegistry()
	if err != nil {
		fmt.Printf("Error initializing registry: %v\n", err)
		os.Exit(1)
	}

	// Handle graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		fmt.Println("\nShutting down gracefully...")
		close(globalRegistry.Actor.MsgChan)
		globalRegistry.actorWg.Wait()
		os.Exit(0)
	}()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// displayTable prints the list of registry items in a table format.
func displayTable(items []registry.RegistryItem) {
	fmt.Println("Displaying items in table format:")
	for _, item := range items {
		status := "Disabled"
		if item.Enabled {
			status = "Enabled"
		}
		fmt.Printf(" - %s: %s [%s]\n", item.Name, item.Path, status)
	}
}

// displayRepoInfo prints detailed information about a specific repository.
func displayRepoInfo(item registry.RegistryItem) {
	fmt.Printf("Repository Information:\n")
	fmt.Printf("  Name:          %s\n", item.Name)
	fmt.Printf("  Path:          %s\n", item.Path)
	fmt.Printf("  Type:          %s\n", item.Type)
	fmt.Printf("  Status:        %s\n", item.Status)
	fmt.Printf("  Created:       %s\n", item.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Last Updated:  %s\n", item.LastUpdated.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Has Dockerfile: %v\n", item.HasDockerfile)

	if item.GitRepo != nil {
		head, err := item.GitRepo.Head()
		if err == nil {
			fmt.Printf("  Current Branch: %s\n", head.Name().Short())
		}

		remotes, err := item.GitRepo.Remotes()
		if err == nil {
			fmt.Printf("  Remotes:\n")
			for _, remote := range remotes {
				fmt.Printf("    - %s: %s\n", remote.Config().Name, remote.Config().URLs[0])
			}
		}
	}
}