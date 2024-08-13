package main

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = old
	return string(out)
}

func setupFileSystem(fs afero.Fs) {
	fs.MkdirAll("/Applications/PyCharm.app", 0755)
	fs.MkdirAll("/usr/local/bin/PyCharm", 0755)
	fs.MkdirAll("C:\\Program Files\\JetBrains\\PyCharm 2021.1", 0755)
}

type mockFinder struct {
	path string
	err  error
}

func (mf *mockFinder) Find() string {
	if mf.err != nil {
		return ""
	}
	return mf.path
}

func TestFindPyCharmStandardOutput(t *testing.T) {
	fs := afero.NewMemMapFs()
	setupFileSystem(fs)
	appFS = fs

	out := captureOutput(func() {
		main()
	})

	assert.Contains(t, out, "PyCharm Location:", "Should display the PyCharm installation path.")
}

func TestFindPyCharmShortOutputFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	setupFileSystem(fs)
	appFS = fs // assuming appFS is correctly configured in the global scope of your application logic

	out := captureOutput(func() {
		runApplication(true, false) // Call the refactored logic directly with the `short` parameter
	})

	assert.Contains(t, out, "✔️ PyCharm found.", "Should display check mark when PyCharm is found.")
}

func TestFindPyCharmShortOutputNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	appFS = fs

	out := captureOutput(func() {
		runApplication(true, false)
	})

	assert.Contains(t, out, "❌ PyCharm not found.", "Should display red X when PyCharm is not found.")
}

func TestLaunchPyCharm(t *testing.T) {
	fs := afero.NewMemMapFs()
	appFS = fs
	// Setup the file system with the PyCharm path
	pyCharmPath := "/Applications/PyCharm.app"
	fs.MkdirAll(pyCharmPath, 0755)
	envFilePath := ".env-pycharm"
	afero.WriteFile(fs, envFilePath, []byte(pyCharmPath), 0644)

	// Capture the output when launching PyCharm
	out := captureOutput(func() {
		runApplication(false, true) // directly call the logic with the launch flag
	})

	// Check the output for the expected launch message
	expectedLaunchMessage := fmt.Sprintf("Launching PyCharm from %s...\n", pyCharmPath)
	assert.Contains(t, out, expectedLaunchMessage, "Should output launching message.")
}

func TestLoadPyCharmPathFromEnv(t *testing.T) {
	fs := afero.NewMemMapFs()
	appFS = fs
	envFilePath := ".env-pycharm"
	expectedPath := "/Applications/PyCharm.app"

	// Write expected path to the env file
	afero.WriteFile(fs, envFilePath, []byte(expectedPath), 0644)

	// Test loading the path
	result := loadPyCharmPathFromEnv(envFilePath)
	assert.Equal(t, expectedPath, result, "The paths should match.")
	assert.Equal(t, expectedPath, os.Getenv("PYCHARM_PATH"), "Environment variable PYCHARM_PATH should be set.")

	// Test loading non-existing path
	result = loadPyCharmPathFromEnv("nonexistent.env")
	assert.Equal(t, "", result, "Result should be empty for non-existent file.")
}

func TestSavePyCharmPathToEnv(t *testing.T) {
	fs := afero.NewMemMapFs()
	appFS = fs
	envFilePath := ".env-pycharm"
	pathToSave := "/usr/local/bin/PyCharm"

	savePyCharmPathToEnv(envFilePath, pathToSave)

	// Verify the file contents
	data, _ := afero.ReadFile(fs, envFilePath)
	assert.Equal(t, pathToSave, string(data), "File should contain the PyCharm path.")

	// Verify the environment variable
	assert.Equal(t, pathToSave, os.Getenv("PYCHARM_PATH"), "Environment variable PYCHARM_PATH should be set correctly.")
}

func TestGetSearchPaths(t *testing.T) {
	// You might need to mock runtime.GOOS or structure your test to handle each case.
	paths := getSearchPaths()
	// Assuming the test is run on a specific OS, like macOS:
	assert.Contains(t, paths, "/Applications/", "Should include typical macOS PyCharm paths.")
}

func TestLaunchPyCharmNotFound(t *testing.T) {
	os.Unsetenv("PYCHARM_PATH")
	originalFinder := finder
	mockFinder := &mockFinder{}
	finder = mockFinder
	defer func() { finder = originalFinder }() // Restore the original finder after the test

	out := captureOutput(func() {
		launchPyCharm()
	})

	assert.Contains(t, out, "PyCharm not found.", "Should print 'PyCharm not found.' when PyCharm is not located")
	assert.NotContains(t, out, "Launching PyCharm from", "Should not print a launching message when PyCharm is not found")
}

func TestLaunchPyCharmFound(t *testing.T) {
	os.Unsetenv("PYCHARM_PATH")
	expectedPath := "/Applications/PyCharm.app"
	originalFinder := finder
	mockFinder := &mockFinder{path: expectedPath}
	finder = mockFinder
	defer func() { finder = originalFinder }() // Restore the original finder after the test

	out := captureOutput(func() {
		launchPyCharm()
	})

	expectedLaunchMessage := fmt.Sprintf("Launching PyCharm from %s...\n", expectedPath)
	assert.Contains(t, out, expectedLaunchMessage, "Should output the correct launching message.")
}
