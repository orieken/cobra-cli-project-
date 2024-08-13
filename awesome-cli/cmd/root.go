package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var appFS afero.Fs = afero.NewOsFs() // Use afero for filesystem abstraction

var rootCmd = &cobra.Command{
	Use:   "awesome-cli",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.PersistentFlags().BoolVarP(&verboseMode, "verbose", "v", false, "Enable verbose output")
	initializePlugins()
}

//func initializePlugins() {
//	defaultPluginDir := filepath.Join(os.Getenv("HOME"), ".foo", "plugins")
//	loadPlugins(defaultPluginDir)
//	loadConditionalPlugins("/path/to/other/plugins", "awesome-")
//	loadPathPlugins("awesome-")
//}

func initializePlugins() {
	// Only initialize the map if it's nil, indicating that it hasn't been set up before
	if cachedPlugins == nil {
		cachedPlugins = make(map[string]string)
	}

	defaultPluginDir := filepath.Join(os.Getenv("HOME"), ".foo", "plugins")
	loadPlugins(defaultPluginDir)
	loadConditionalPlugins("/path/to/other/plugins", "awesome-")
	loadPathPlugins("awesome-")
}

var cachedPlugins map[string]string

//func loadPlugins(pluginDir string) {
//	cachedPlugins = make(map[string]string) // Initialize the map
//	files, err := afero.ReadDir(appFS, pluginDir)
//	if err != nil {
//		if verboseMode {
//			fmt.Println("Failed to read plugin directory:", err)
//		}
//		return
//	}
//	for _, file := range files {
//		if file.IsDir() || !strings.HasPrefix(file.Name(), "awesome-") {
//			continue
//		}
//		registerPluginCommand(pluginDir, file.Name())
//	}
//}

func loadPlugins(pluginDir string) {
	files, err := afero.ReadDir(appFS, pluginDir)
	if err != nil {
		if verboseMode {
			fmt.Println("Failed to read plugin directory:", err)
		}
		return
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasPrefix(file.Name(), "awesome-") {
			continue
		}
		if _, exists := cachedPlugins[file.Name()]; !exists {
			registerPluginCommand(pluginDir, file.Name())
		}
	}
}

func loadConditionalPlugins(pluginDir, prefix string) {
	files, err := afero.ReadDir(appFS, pluginDir)
	if err != nil {
		fmt.Println("Failed to read plugin directory:", err)
		return
	}
	filterAndRegisterPlugins(files, pluginDir, prefix)
}

func loadPathPlugins(prefix string) {
	path := os.Getenv("PATH")
	dirs := strings.Split(path, string(os.PathListSeparator))
	for _, dir := range dirs {
		files, _ := afero.ReadDir(appFS, dir) // Ignore errors, some dirs might be inaccessible
		filterAndRegisterPlugins(files, dir, prefix)
	}
}

func registerPlugins(files []os.FileInfo, pluginDir string) {
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		registerPluginCommand(pluginDir, file.Name())
	}
}

func filterAndRegisterPlugins(files []os.FileInfo, pluginDir, prefix string) {
	for _, file := range files {
		if file.IsDir() || !startsWith(file.Name(), prefix) {
			continue
		}
		registerPluginCommand(pluginDir, file.Name())
	}
}

func registerPluginCommand(pluginDir, fileName string) {
	if _, exists := cachedPlugins[fileName]; exists {
		return // Plugin already registered, skip re-registration
	}

	commandName := strings.TrimPrefix(fileName, "awesome-")
	pluginPath := filepath.Join(pluginDir, fileName)
	pluginCmd := &cobra.Command{
		Use:   commandName,
		Short: "Runs the " + commandName + " plugin",
		Run: func(cmd *cobra.Command, args []string) {
			executePlugin(pluginPath, args)
		},
	}
	rootCmd.AddCommand(pluginCmd)
	cachedPlugins[fileName] = pluginPath // Cache the plugin path
	if verboseMode {
		fmt.Printf("Loaded plugin: %s\n", fileName)
	}
}

//func registerPluginCommand(pluginDir, fileName string) {
//	commandName := strings.TrimPrefix(fileName, "awesome-") // Remove prefix for display
//	pluginPath := filepath.Join(pluginDir, fileName)
//
//	pluginCmd := &cobra.Command{
//		Use:   commandName, // Use modified command name
//		Short: "Runs the " + commandName + " plugin",
//		Run: func(cmd *cobra.Command, args []string) {
//			executePlugin(pluginPath, args)
//		},
//	}
//	rootCmd.AddCommand(pluginCmd)
//	fmt.Printf("Loaded plugin: %s\n", fileName)
//}

var verboseMode bool

func executePlugin(pluginPath string, args []string) {
	if verboseMode {
		fmt.Println("Executing plugin at:", pluginPath)
	}
	cmd := exec.Command(pluginPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil && verboseMode {
		fmt.Fprintf(os.Stderr, "Error executing plugin %s: %v\n", pluginPath, err)
	}
}

func startsWith(name, prefix string) bool {
	return strings.HasPrefix(name, prefix)
}
