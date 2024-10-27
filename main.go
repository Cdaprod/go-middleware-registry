// File: main.go
package main

import (
	"fmt"
	"os"

	"github.com/Cdaprod/go-middleware-registry/internal/ui"
	"github.com/Cdaprod/go-middleware-registry/registry"
	"github.com/spf13/cobra"
)

// Root command for the CLI application.
var rootCmd = &cobra.Command{
	Use:   "registry",
	Short: "Registry CLI for managing repositories",
	Long:  "A CLI application for managing repositories in /home/cdaprod/Projects with support for Git repositories and Docker containers.",
}

// List command to list all repositories.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all repositories",
	Run: func(cmd *cobra.Command, args []string) {
		reg, err := registry.NewRegistry()
		if err != nil {
			fmt.Printf("Error initializing registry: %v\n", err)
			os.Exit(1)
		}
		displayTable(reg.ListItems())
	},
}

// Scan command to scan the projects directory for repositories.
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan projects directory for repositories",
	Run: func(cmd *cobra.Command, args []string) {
		reg, err := registry.NewRegistry()
		if err != nil {
			fmt.Printf("Error initializing registry: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Discovered %d repositories in %s\n", len(reg.Items), reg.Config.ProjectsPath)
		displayTable(reg.ListItems())
	},
}

// Info command to show detailed information about a repository.
var infoCmd = &cobra.Command{
	Use:   "info [repository]",
	Short: "Show detailed information about a repository",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		reg, err := registry.NewRegistry()
		if err != nil {
			fmt.Printf("Error initializing registry: %v\n", err)
			os.Exit(1)
		}

		item, exists := reg.Items[args[0]]
		if !exists {
			fmt.Printf("Repository '%s' not found\n", args[0])
			os.Exit(1)
		}

		displayRepoInfo(item)
	},
}

// Interactive command to launch the TUI.
var interactiveCmd = &cobra.Command{
	Use:   "interactive",
	Short: "Launch interactive TUI",
	Run: func(cmd *cobra.Command, args []string) {
		reg, err := registry.NewRegistry()
		if err != nil {
			fmt.Printf("Error initializing registry: %v\n", err)
			os.Exit(1)
		}

		if err := ui.LaunchTUI(reg); err != nil {
			fmt.Printf("Error starting TUI: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(interactiveCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// displayTable prints the list of registry items in a table format.
func displayTable(items []registry.RegistryItem) {
	fmt.Println("Displaying items in table format:")
	for _, item := range items {
		fmt.Printf(" - %s: %s\n", item.Name, item.Path)
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
