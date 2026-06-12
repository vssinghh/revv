package sandbox

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// EnsureDocker checks if Docker is available and offers to install it if not.
// Returns nil if Docker is ready to use, error if the user declines or install fails.
func EnsureDocker(ctx context.Context, verbose bool) error {
	// First check if Docker is already available
	if err := CheckAvailable(ctx); err == nil {
		return nil
	}

	// Docker not available — try to install
	installer, err := detectInstaller()
	if err != nil {
		return fmt.Errorf("docker is not installed and cannot be auto-installed: %w\n\nPlease install Docker manually: https://docs.docker.com/get-docker/", err)
	}

	// Ask user for permission
	fmt.Printf("\n╭─────────────────────────────────────────────╮\n")
	fmt.Printf("│  Docker is required but not installed.       │\n")
	fmt.Printf("│                                              │\n")
	fmt.Printf("│  revv will install:                          │\n")
	for _, line := range installer.Description() {
		fmt.Printf("│    %-42s│\n", line)
	}
	fmt.Printf("│                                              │\n")
	fmt.Printf("╰─────────────────────────────────────────────╯\n")
	fmt.Printf("\nProceed with installation? [Y/n] ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "" && response != "y" && response != "yes" {
		return fmt.Errorf("docker installation declined. Install manually:\n\n  %s", installer.ManualInstructions())
	}

	fmt.Println()

	// Run installation
	if err := installer.Install(verbose); err != nil {
		return fmt.Errorf("docker installation failed: %w\n\nTry installing manually:\n  %s", err, installer.ManualInstructions())
	}

	// Start the daemon
	fmt.Println("Starting Docker daemon...")
	if err := installer.StartDaemon(verbose); err != nil {
		return fmt.Errorf("failed to start Docker daemon: %w", err)
	}

	// Wait for daemon to be ready
	fmt.Print("Waiting for Docker to be ready")
	for i := 0; i < 30; i++ {
		if err := CheckAvailable(ctx); err == nil {
			fmt.Println(" ✓")
			return nil
		}
		fmt.Print(".")
		time.Sleep(1 * time.Second)
	}
	fmt.Println(" ✗")

	return fmt.Errorf("docker daemon did not start within 30 seconds. Try running manually:\n  %s", installer.StartCommand())
}

// dockerInstaller defines platform-specific Docker installation logic.
type dockerInstaller interface {
	Description() []string
	ManualInstructions() string
	Install(verbose bool) error
	StartDaemon(verbose bool) error
	StartCommand() string
}

// detectInstaller returns the appropriate installer for the current platform.
func detectInstaller() (dockerInstaller, error) {
	switch runtime.GOOS {
	case "darwin":
		return detectDarwinInstaller()
	case "linux":
		return detectLinuxInstaller()
	default:
		return nil, fmt.Errorf("unsupported OS: %s. Please install Docker manually", runtime.GOOS)
	}
}

// --- macOS (Homebrew + Colima) ---

type brewColimaInstaller struct{}

func detectDarwinInstaller() (dockerInstaller, error) {
	if _, err := exec.LookPath("brew"); err != nil {
		return nil, fmt.Errorf("Homebrew not found. Install it from https://brew.sh then retry")
	}
	return &brewColimaInstaller{}, nil
}

func (b *brewColimaInstaller) Description() []string {
	return []string{
		"• docker (CLI client)",
		"• colima (lightweight runtime)",
		"",
		"Via: Homebrew (free, open source)",
	}
}

func (b *brewColimaInstaller) ManualInstructions() string {
	return "brew install docker colima && colima start"
}

func (b *brewColimaInstaller) StartCommand() string {
	return "colima start"
}

func (b *brewColimaInstaller) Install(verbose bool) error {
	steps := []struct {
		name string
		cmd  string
		args []string
	}{
		{"Installing Docker CLI", "brew", []string{"install", "docker"}},
		{"Installing Colima", "brew", []string{"install", "colima"}},
	}

	for _, step := range steps {
		fmt.Printf("  %s...\n", step.name)
		cmd := exec.Command(step.cmd, step.args...)
		if verbose {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s failed: %w", step.name, err)
		}
	}

	return nil
}

func (b *brewColimaInstaller) StartDaemon(verbose bool) error {
	cmd := exec.Command("colima", "start")
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

// --- Linux (apt/yum) ---

type linuxInstaller struct {
	pkgManager string // "apt" or "yum" or "dnf"
}

func detectLinuxInstaller() (dockerInstaller, error) {
	for _, pm := range []string{"apt-get", "dnf", "yum"} {
		if _, err := exec.LookPath(pm); err == nil {
			return &linuxInstaller{pkgManager: pm}, nil
		}
	}
	return nil, fmt.Errorf("no supported package manager found (apt-get, dnf, yum)")
}

func (l *linuxInstaller) Description() []string {
	return []string{
		"• docker.io (engine + CLI)",
		"",
		fmt.Sprintf("Via: %s", l.pkgManager),
	}
}

func (l *linuxInstaller) ManualInstructions() string {
	switch l.pkgManager {
	case "apt-get":
		return "sudo apt-get update && sudo apt-get install -y docker.io && sudo systemctl start docker"
	case "dnf":
		return "sudo dnf install -y docker && sudo systemctl start docker"
	case "yum":
		return "sudo yum install -y docker && sudo systemctl start docker"
	default:
		return "https://docs.docker.com/engine/install/"
	}
}

func (l *linuxInstaller) StartCommand() string {
	return "sudo systemctl start docker"
}

func (l *linuxInstaller) Install(verbose bool) error {
	var installCmd *exec.Cmd

	switch l.pkgManager {
	case "apt-get":
		// Update first
		fmt.Println("  Updating package list...")
		update := exec.Command("sudo", "apt-get", "update", "-qq")
		if verbose {
			update.Stdout = os.Stdout
			update.Stderr = os.Stderr
		}
		if err := update.Run(); err != nil {
			return fmt.Errorf("apt-get update failed: %w", err)
		}
		installCmd = exec.Command("sudo", "apt-get", "install", "-y", "docker.io")
	case "dnf":
		installCmd = exec.Command("sudo", "dnf", "install", "-y", "docker")
	case "yum":
		installCmd = exec.Command("sudo", "yum", "install", "-y", "docker")
	}

	fmt.Println("  Installing Docker...")
	if verbose {
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
	}
	return installCmd.Run()
}

func (l *linuxInstaller) StartDaemon(verbose bool) error {
	cmd := exec.Command("sudo", "systemctl", "start", "docker")
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}
