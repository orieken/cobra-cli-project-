package awesome_reporter

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// Scenario represents the structure of each test scenario.
type Scenario struct {
	Name  string `json:"name"`
	Steps []Step `json:"steps"`
}

// Step represents the structure of each step in a scenario.
type Step struct {
	Status string `json:"status"`
}

// StatusCount holds the counts of different statuses.
type StatusCount struct {
	Passed  int `json:"passed"`
	Pending int `json:"pending"`
	Failed  int `json:"failed"`
}

var (
	appFS       = afero.NewOsFs() // Use afero for filesystem abstraction
	verboseMode bool              // Flag for verbose mode
	defaultDir  = "./"            // Default directory to look for JSON files
)

func main() {
	var prefix string
	flag.StringVar(&prefix, "prefix", "cucumber_report", "Prefix of JSON files to analyze")
	flag.BoolVar(&verboseMode, "verbose", false, "Enable verbose output")
	flag.Parse()

	results := processFiles(defaultDir, prefix)

	if err := outputResults(prefix, results, appFS); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to output results: %v\n", err)
		os.Exit(1)
	}
}

func processFiles(dir, prefix string) map[string]StatusCount {
	results := make(map[string]StatusCount)
	files, err := listRelevantFiles(dir, prefix)
	if err != nil {
		if verboseMode {
			fmt.Println("Error listing files:", err)
		}
		return results
	}

	for _, file := range files {
		scenarios, err := readScenariosFromFile(filepath.Join(dir, file.Name()))
		if err != nil {
			continue
		}
		aggregateStatusCounts(scenarios, results)
	}
	return results
}

func listRelevantFiles(dir, prefix string) ([]os.FileInfo, error) {
	files, err := afero.ReadDir(appFS, dir)
	if err != nil {
		return nil, err
	}
	var relevantFiles []os.FileInfo
	for _, file := range files {
		if file.IsDir() || !strings.HasPrefix(file.Name(), prefix) || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		relevantFiles = append(relevantFiles, file)
	}
	return relevantFiles, nil
}

func readScenariosFromFile(filePath string) ([]Scenario, error) {
	data, err := afero.ReadFile(appFS, filePath)
	if err != nil {
		if verboseMode {
			fmt.Println("Error reading file:", err)
		}
		return nil, err
	}
	var scenarios []Scenario
	if err := json.Unmarshal(data, &scenarios); err != nil {
		if verboseMode {
			fmt.Println("Error parsing JSON:", err)
		}
		return nil, err
	}
	return scenarios, nil
}

func aggregateStatusCounts(scenarios []Scenario, results map[string]StatusCount) {
	for _, scenario := range scenarios {
		count := results[scenario.Name]
		for _, step := range scenario.Steps {
			switch step.Status {
			case "passed":
				count.Passed++
			case "pending":
				count.Pending++
			case "failed":
				count.Failed++
			}
		}
		results[scenario.Name] = count
	}
}

func outputResults(prefix string, results map[string]StatusCount, fs afero.Fs) error {
	outputFileName := fmt.Sprintf("%s_aggregated_results.json", prefix)
	resultData, err := json.MarshalIndent(results, "", "    ")
	if err != nil {
		if verboseMode {
			fmt.Println("Error marshalling results:", err)
		}
		return err
	}

	// Use the provided filesystem to write the file
	err = afero.WriteFile(fs, outputFileName, resultData, 0755)
	if err != nil {
		return err
	}

	fmt.Println("Scenario Name | Passed | Pending | Failed")
	for name, count := range results {
		fmt.Printf("%s | %d | %d | %d\n", name, count.Passed, count.Pending, count.Failed)
	}
	return nil
}
