package main

import (
	"fmt"
	"github.com/Masterminds/semver/v3"
	"io"
	"os"
	"os/exec"
	"strings"
)

type CommandExecutor interface {
	CombinedOutput() ([]byte, error)
}

type RealCommand struct {
	Cmd *exec.Cmd
}

func (rc *RealCommand) CombinedOutput() ([]byte, error) {
	return rc.Cmd.CombinedOutput()
}

// Extracted parsing logic into its own function
func parsePythonVersion(output string) (string, error) {
	if strings.Contains(output, "Python ") {
		versionParts := strings.Split(output, " ")
		if len(versionParts) >= 2 {
			return versionParts[1], nil
		}
	}
	return "", fmt.Errorf("failed to detect Python version")
}

var out io.Writer = os.Stdout // Default to stdout, can be redirected in tests

func isVersionAtLeast310(version string) bool {
	baseVersion, err := semver.NewVersion("3.10")
	if err != nil {
		fmt.Fprintf(out, "Error parsing base version: %v\n", err)
		return false
	}

	currentVersion, err := semver.NewVersion(version)
	if err != nil {
		fmt.Fprintf(out, "Error parsing current version: %v\n", err)
		return false
	}

	return currentVersion.GreaterThan(baseVersion) || currentVersion.Equal(baseVersion)
}

func checkPythonVersion(executor CommandExecutor) error {
	output, err := executor.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error executing Python: %w", err)
	}

	versionOutput := string(output)
	version, err := parsePythonVersion(versionOutput)
	if err != nil {
		return err
	}

	fmt.Println("Installed Python version:", version)
	if isVersionAtLeast310(version) {
		fmt.Println("Python version is 3.10 or higher.")
		return nil
	} else {
		return fmt.Errorf("Python version is below 3.10")
	}
}

type Exiter interface {
	Exit(code int)
}

type RealExiter struct{}

func (re *RealExiter) Exit(code int) {
	os.Exit(code)
}

func runApplication(executor CommandExecutor, exiter Exiter) {
	if err := checkPythonVersion(executor); err != nil {
		fmt.Println(err)
		exiter.Exit(1)
	}
}

func main() {
	command := &RealCommand{Cmd: exec.Command("python", "--version")}
	exiter := &RealExiter{}
	runApplication(command, exiter)
}
