package cmd

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"io/fs"
	"os"
	"testing"
)

// MockFileSystem is a mock type for the FileSystem interface.
type MockFileSystem struct {
	mock.Mock
}

func (m *MockFileSystem) ReadDir(dirname string) ([]fs.DirEntry, error) {
	args := m.Called(dirname)
	return args.Get(0).([]fs.DirEntry), args.Error(1)
}

type mockDirEntry struct {
	name  string
	isDir bool
}

func (mde *mockDirEntry) Name() string               { return mde.name }
func (mde *mockDirEntry) IsDir() bool                { return mde.isDir }
func (mde *mockDirEntry) Type() fs.FileMode          { return 0 }
func (mde *mockDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

// captureOutput captures and returns standard output during the call of a provided function.
func captureOutput(f func()) string {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = old
	return string(out)
}

func TestListPlugins(t *testing.T) {
	os.Setenv("HOME", "/Users/testuser")
	fileSystem = new(MockFileSystem)
	defer func() {
		fileSystem = &OSFileSystem{}
		os.Unsetenv("HOME") // Clean up the environment variable after the test
	}()

	entries := []fs.DirEntry{
		&mockDirEntry{name: "plugin1.so", isDir: false},
		&mockDirEntry{name: "plugin2.so", isDir: false},
	}

	fileSystem.(*MockFileSystem).On("ReadDir", "/Users/testuser/.foo/plugins").Return(entries, nil)
	output := captureOutput(listPlugins)
	assert.Contains(t, output, "Available plugins:")
	assert.Contains(t, output, "plugin1.so")
	assert.Contains(t, output, "plugin2.so")
	fileSystem.(*MockFileSystem).AssertExpectations(t)
}

func TestListPluginsNoPluginsFound(t *testing.T) {
	// Set the HOME environment variable to a consistent test value
	os.Setenv("HOME", "/Users/testuser")
	defer os.Unsetenv("HOME") // Clean up after the test

	fileSystem = new(MockFileSystem)
	defer func() { fileSystem = &OSFileSystem{} }() // Reset to original after test

	fileSystem.(*MockFileSystem).On("ReadDir", "/Users/testuser/.foo/plugins").Return([]fs.DirEntry{}, nil)

	output := captureOutput(listPlugins)
	expectedOutput := "No plugins found.\n"
	assert.Equal(t, expectedOutput, output)
	fileSystem.(*MockFileSystem).AssertExpectations(t)
}

func TestListPluginsReadDirError(t *testing.T) {
	os.Setenv("HOME", "/Users/testuser")
	defer os.Unsetenv("HOME")

	fileSystem = new(MockFileSystem)
	defer func() { fileSystem = &OSFileSystem{} }()

	var nilEntries []fs.DirEntry = nil
	fileSystem.(*MockFileSystem).On("ReadDir", "/Users/testuser/.foo/plugins").Return(nilEntries, os.ErrNotExist)

	output := captureOutput(listPlugins)

	expectedOutput := "Failed to read plugin directory: file does not exist\n"
	assert.Contains(t, output, expectedOutput)
	fileSystem.(*MockFileSystem).AssertExpectations(t)
}
