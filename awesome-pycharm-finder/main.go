package main

import (
	"flag"
	"fmt"
	"github.com/spf13/afero"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

var appFS = afero.NewOsFs()

var findPyCharmFunc = findPyCharm

type PyCharmFinder interface {
	Find() string
}
type defaultFinder struct{}

func (df *defaultFinder) Find() string {
	fmt.Println("Locating PyCharm...")
	envFilePath := ".env-pycharm"
	if data, err := afero.ReadFile(appFS, envFilePath); err == nil {
		return string(data)
	}
	// Add your path finding logic here...
	return ""
}

var finder PyCharmFinder = &defaultFinder{} // Set the default implementation

func loadPyCharmPathFromEnv(envFilePath string) string {
	data, err := afero.ReadFile(appFS, envFilePath)
	if err == nil {
		os.Setenv("PYCHARM_PATH", string(data))
		return string(data)
	}
	return ""
}

func savePyCharmPathToEnv(envFilePath, path string) {
	afero.WriteFile(appFS, envFilePath, []byte(path), 0644)
	os.Setenv("PYCHARM_PATH", path)
}

func searchForPyCharm(paths []string) string {
	for _, path := range paths {
		found, err := afero.Glob(appFS, filepath.Join(path, "PyCharm*"))
		if err == nil && len(found) > 0 {
			return found[0]
		}
	}
	return ""
}

func findPyCharm() string {
	envFilePath := ".env-pycharm"
	if data, err := afero.ReadFile(appFS, envFilePath); err == nil {
		return string(data)
	}

	paths := getSearchPaths()
	if foundPath := searchForPyCharm(paths); foundPath != "" {
		savePyCharmPathToEnv(envFilePath, foundPath)
		return foundPath
	}
	return ""
}

func getSearchPaths() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{
			filepath.Join(os.Getenv("PROGRAMFILES"), "JetBrains"),
			filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "JetBrains"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "JetBrains"),
		}
	case "darwin":
		return []string{"/Applications/", filepath.Join(os.Getenv("HOME"), "Applications")}
	case "linux":
		return []string{"/usr/local/bin/", "/opt/", filepath.Join(os.Getenv("HOME"), ".local/share/JetBrains")}
	}
	return nil
}

func runApplication(short bool, launch bool) {
	if launch {
		launchPyCharm()
		return
	}

	location := findPyCharm()
	if short {
		if location != "" {
			fmt.Println("✔️ PyCharm found.")
		} else {
			fmt.Println("❌ PyCharm not found.")
		}
	} else {
		if location != "" {
			fmt.Println("PyCharm Location:", location)
		} else {
			fmt.Println("PyCharm not found.")
		}
	}
}

func launchPyCharm() {
	pyCharmPath := os.Getenv("PYCHARM_PATH")
	if pyCharmPath == "" {
		pyCharmPath = finder.Find()
	}
	if pyCharmPath == "" {
		fmt.Println("PyCharm not found.")
		return
	}
	fmt.Printf("Launching PyCharm from %s...\n", pyCharmPath)

	switch runtime.GOOS {
	case "windows":
		exec.Command("cmd", "/C", pyCharmPath).Start()
	case "darwin", "linux":
		exec.Command("open", pyCharmPath).Start()
	}
}

func main() {
	short := flag.Bool("short", false, "Output in short format")
	launch := flag.Bool("launch", false, "Launch PyCharm")
	flag.Parse()

	runApplication(*short, *launch)
}
