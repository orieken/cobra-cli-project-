package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/joho/godotenv"
)

type Config struct {
	ReportsPath   string
	EnvFilePath   string
	ContainerName string
}

type CommandExecutor interface {
	Execute(command []string, output, errorOutput io.Writer) error
}

type DefaultExecutor struct{}

func (e DefaultExecutor) Execute(command []string, output, errorOutput io.Writer) error {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = output
	cmd.Stderr = errorOutput
	return cmd.Run()
}

func main() {
	err := godotenv.Load() // Load .env file
	if err != nil {
		fmt.Println("Error loading .env file")
		os.Exit(1)
	}

	config := Config{
		ReportsPath:   os.Getenv("REPORTS_PATH"),
		EnvFilePath:   os.Getenv("ENV_FILE_PATH"),
		ContainerName: os.Getenv("CONTAINER_NAME"),
	}

	teamFlag := parseFlags()

	command := getCommandForTeam(teamFlag)

	executor := DefaultExecutor{}
	if err := executePodmanCommand(executor, command, config); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute command: %v\n", err)
		os.Exit(1)
	}
}

func parseFlags() string {
	teamFlag := flag.String("team", "", "Specify the team to run tests for")
	flag.Parse()
	if *teamFlag == "" {
		fmt.Println("Please specify a team using the --team flag")
		os.Exit(1)
	}
	return *teamFlag
}

func getCommandForTeam(team string) string {
	commands := map[string]string{
		"abc": "npm run ",
		"def": "npm run ",
		"foo": "npm run ",
	}
	command, exists := commands[team]
	if !exists {
		fmt.Printf("No command found for team: %s\n", team)
		os.Exit(1)
	}
	return command
}

//func checkError(err error, message string) {
//	if err != nil {
//		fmt.Printf("%s: %v\n", message, err)
//		os.Exit(1)
//	}
//}
//
//type CommandExecutor interface {
//	Execute(command []string, output, errorOutput io.Writer) error
//}
//
//type DefaultExecutor struct{}
//
//func (e DefaultExecutor) Execute(command []string, output, errorOutput io.Writer) error {
//	cmd := exec.Command(command[0], command[1:]...)
//	cmd.Stdout = output
//	cmd.Stderr = errorOutput
//	return cmd.Run()
//}

func executePodmanCommand(executor CommandExecutor, command string, config Config) error {
	cmd := []string{"podman", "run", "-it", "--rm", "--network=host",
		"-v", config.ReportsPath + ":/app/src/reports",
		"--env-file", config.EnvFilePath,
		config.ContainerName, command}

	return executor.Execute(cmd, os.Stdout, os.Stderr)
}

//func executePodmanCommand(command string) {
//	podmanCmd := exec.Command("docker", "run", "--rm", "--network=host",
//		"-v", os.Getenv("REPORTS_PATH")+":/app/src/reports",
//		"--env-file", os.Getenv("ENV_FILE_PATH"),
//		os.Getenv("CONTAINER_NAME"), command)
//
//	podmanCmd.Stdout = os.Stdout
//	podmanCmd.Stderr = os.Stderr
//
//	fmt.Printf("Executing: %v\n", podmanCmd.String())
//	err := podmanCmd.Run()
//	checkError(err, "Error running podman command")
//}
