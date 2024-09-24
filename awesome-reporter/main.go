package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
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

	// Create an instance of DefaultOutputHandler
	handler := DefaultOutputHandler{}

	// Process files and get results
	results := processFiles(defaultDir, prefix)

	// Output results using the handler
	if err := outputResults(handler, prefix, results, appFS); err != nil {
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

//func outputResults(prefix string, results map[string]StatusCount, fs afero.Fs) error {
//	outputFileName := fmt.Sprintf("%s_aggregated_results.json", prefix)
//	resultData, err := json.MarshalIndent(results, "", "    ")
//	if err != nil {
//		if verboseMode {
//			fmt.Println("Error marshalling results:", err)
//		}
//		return err
//	}
//
//	// Use the provided filesystem to write the file
//	err = afero.WriteFile(fs, outputFileName, resultData, 0755)
//	if err != nil {
//		return err
//	}
//
//	fmt.Println("Scenario Name | Passed | Pending | Skipped | Failed | Error Messages")
//	for name, count := range results {
//		errors := strings.Join(count.Messages, "; ")
//		fmt.Printf("%s | %d | %d | %d | %d | %s\n", name, count.Passed, count.Pending, count.Skipped, count.Failed, errors)
//	}
//	return nil
//}

type OutputHandler interface {
	HandleJSONOutput(prefix string, results map[string]StatusCount, fs afero.Fs) error
	HandleHTMLOutput(prefix string, results map[string]StatusCount, fs afero.Fs) error
}

type DefaultOutputHandler struct{}

func (d DefaultOutputHandler) HandleJSONOutput(prefix string, results map[string]StatusCount, fs afero.Fs) error {
	return handleJSONOutput(prefix, results, fs) // Assuming this is your actual function
}

func (d DefaultOutputHandler) HandleHTMLOutput(prefix string, results map[string]StatusCount, fs afero.Fs) error {
	return handleHTMLOutput(prefix, results, fs)
}

func outputResults(handler OutputHandler, prefix string, results map[string]StatusCount, fs afero.Fs) error {
	if err := handler.HandleJSONOutput(prefix, results, fs); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to output JSON results: %v\n", err)
		return err
	}

	if err := handler.HandleHTMLOutput(prefix, results, fs); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to output HTML results: %v\n", err)
		return err
	}

	handleConsoleOutput(results) // Adjust accordingly if needed
	return nil
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Scenario Status Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        table { width: 100%; border-collapse: collapse; }
        th, td { border: 1px solid #ccc; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
<h1>Scenario Status Report</h1>
<table>
    <thead>
        <tr>
            <th>Scenario Name</th>
            <th>Passed</th>
            <th>Pending</th>
            <th>Failed</th>
            <th>Skipped</th>
            <th>Error Messages</th>
        </tr>
    </thead>
    <tbody>
        {{range $name, $counts := .}}
        <tr>
            <td>{{$name}}</td>
            <td>{{$counts.Passed}}</td>
            <td>{{$counts.Pending}}</td>
            <td>{{$counts.Failed}}</td>
            <td>{{$counts.Skipped}}</td>
            <td>{{range $counts.Messages}}<div>{{.}}</div>{{end}}</td>
        </tr>
        {{end}}
    </tbody>
</table>
</body>
</html>`

func handleJSONOutput(prefix string, results map[string]StatusCount, fs afero.Fs) error {
	outputFileName := fmt.Sprintf("%s_aggregated_results.json", prefix)
	resultData, err := json.MarshalIndent(results, "", "    ")
	if err != nil {
		return fmt.Errorf("error marshalling results: %v", err)
	}
	return afero.WriteFile(fs, outputFileName, resultData, 0644)
}

func handleHTMLOutput(prefix string, results map[string]StatusCount, fs afero.Fs) error {
	t, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("error parsing HTML template: %v", err)
	}

	htmlFileName := fmt.Sprintf("%s_report.html", prefix)
	htmlFile, err := fs.Create(htmlFileName)
	if err != nil {
		return fmt.Errorf("error creating HTML file: %v", err)
	}
	defer htmlFile.Close()

	return t.Execute(htmlFile, results)
}

func handleConsoleOutput(results map[string]StatusCount) {
	fmt.Println("Scenario Name | Passed | Pending | Failed | Skipped | Error Messages")
	for name, count := range results {
		errors := strings.Join(count.Messages, "; ")
		fmt.Printf("%s | %d | %d | %d | %d | %s\n", name, count.Passed, count.Pending, count.Skipped, count.Failed, errors)
	}
}
