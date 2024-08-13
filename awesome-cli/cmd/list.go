package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"io/fs"
	"os"
)

// FileSystem defines the interface for file system operations needed by listPlugins.
type FileSystem interface {
	ReadDir(dirname string) ([]fs.DirEntry, error) // Correct return type
}

// OSFileSystem implements FileSystem using the os package.
type OSFileSystem struct{}

func (osfs *OSFileSystem) ReadDir(dirname string) ([]fs.DirEntry, error) {
	return os.ReadDir(dirname) // Correctly returns []fs.DirEntry
}

// fileSystem is the FileSystem to use for reading directories.
var fileSystem FileSystem = &OSFileSystem{}

// listCmd represents the command to list all plugins.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists all the available plugins",
	Long:  `This command lists all the available plugins in the /.foo/plugins directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		listPlugins()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func listPlugins() {
	pluginDir := os.Getenv("HOME") + "/.foo/plugins"
	files, err := fileSystem.ReadDir(pluginDir)
	if err != nil {
		fmt.Println("Failed to read plugin directory:", err)
		return
	}

	displayPlugins(files)
}

// displayPlugins prints the names of the files in the plugins directory if any are found.
func displayPlugins(files []fs.DirEntry) {
	if len(files) == 0 {
		fmt.Println("No plugins found.")
		return
	}

	fmt.Println("Available plugins:")
	for _, entry := range files {
		if !entry.IsDir() { // Check if the entry is a file
			fmt.Println("  -", entry.Name())
		}
	}
}
