package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

type Feature struct {
	Elements []Scenario `json:"elements"`
}

type Result struct {
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type Scenario struct {
	Name  string `json:"name"`
	Steps []Step `json:"steps"`
}

type Step struct {
	Result Result `json:"result"`
}

type StatusCount struct {
	Passed   int      `json:"passed"`
	Pending  int      `json:"pending"`
	Failed   int      `json:"failed"`
	Skipped  int      `json:"skipped"`
	Messages []string `json:"error_messages"`
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
	var features []Feature
	if err := json.Unmarshal(data, &features); err != nil {
		if verboseMode {
			fmt.Println("Error parsing JSON:", err)
		}
		return nil, err
	}

	// Extract all scenarios from the features
	var scenarios []Scenario
	for _, feature := range features {
		scenarios = append(scenarios, feature.Elements...)
	}

	return scenarios, nil
}

func aggregateStatusCounts(scenarios []Scenario, results map[string]StatusCount) {
	for _, scenario := range scenarios {
		fmt.Printf("Processing scenario: %s\n", scenario.Name)
		count := results[scenario.Name]
		for _, step := range scenario.Steps {
			status := strings.ToLower(strings.TrimSpace(step.Result.Status))
			fmt.Printf("Step status: %s\n", status)
			switch status {
			case "passed":
				count.Passed++
			case "pending":
				count.Pending++
			case "failed":
				count.Failed++
				if step.Result.ErrorMessage != "" {
					count.Messages = append(count.Messages, step.Result.ErrorMessage)
				}
			case "skipped":
				count.Skipped++
			}
		}
		results[scenario.Name] = count
		fmt.Printf("Results for %s: %+v\n", scenario.Name, count)
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

	fmt.Println("Scenario Name | Passed | Pending | Skipped | Failed | Error Messages")
	for name, count := range results {
		errors := strings.Join(count.Messages, "; ")
		fmt.Printf("%s | %d | %d | %d | %d | %s\n", name, count.Passed, count.Pending, count.Skipped, count.Failed, errors)
	}
	return nil
}
