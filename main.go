package main

import (
	"fmt"
	"os"

	"github.com/fezcode/atlas.burner/internal/elevation"
	"github.com/fezcode/atlas.burner/internal/tui"
)

var Version = "dev"

func main() {
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "-v" || arg == "--version" {
			fmt.Printf("atlas.burner v%s\n", Version)
			return
		}
		if arg == "-h" || arg == "--help" || arg == "help" {
			showHelp()
			return
		}
	}

	elevation.EnsureElevated()

	if err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Println("Atlas Burner - A beautiful TUI image burner for OS images and USB drives.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  atlas.burner              Start the interactive TUI")
	fmt.Println("  atlas.burner -v           Show version")
	fmt.Println("  atlas.burner -h           Show this help")
	fmt.Println()
	fmt.Println("Features:")
	fmt.Println("  - Browse and download OS images (Linux, Windows, BSD)")
	fmt.Println("  - Burn any local ISO/IMG file to a USB drive")
	fmt.Println("  - Auto-detects removable USB devices")
	fmt.Println("  - Configurable block size, partition table, and verification")
	fmt.Println("  - Cross-platform (Windows, Linux, macOS)")
	fmt.Println()
	fmt.Println("Note: Requires administrator/root privileges to write to USB devices.")
}
