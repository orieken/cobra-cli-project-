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
	Failed   int      `json:"failed"`
	Messages []string `json:"error_messages"`
}

var (
	appFS       = afero.NewOsFs()
	verboseMode bool
	defaultDir  = "./"
)

func main() {
	var prefix string
	flag.StringVar(&prefix, "prefix", "cucumber_report", "Prefix of JSON files to analyze")
	flag.BoolVar(&verboseMode, "verbose", false, "Enable verbose output")
	flag.Parse()

	handler := DefaultOutputHandler{}

	results := processFiles(defaultDir, prefix)
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

	var scenarios []Scenario
	for _, feature := range features {
		scenarios = append(scenarios, feature.Elements...)
	}

	return scenarios, nil
}

func aggregateStatusCounts(scenarios []Scenario, results map[string]StatusCount) {
	for _, scenario := range scenarios {
		count, exists := results[scenario.Name]
		if !exists {
			count = StatusCount{
				Messages: []string{},
			}
		}

		for _, step := range scenario.Steps {
			switch strings.ToLower(strings.TrimSpace(step.Result.Status)) {
			case "passed":
				count.Passed++
			case "failed":
				count.Failed++
				if step.Result.ErrorMessage != "" && !contains(count.Messages, step.Result.ErrorMessage) {
					count.Messages = append(count.Messages, step.Result.ErrorMessage)
				}
			}
		}
		results[scenario.Name] = count
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

type OutputHandler interface {
	HandleJSONOutput(prefix string, results map[string]StatusCount, fs afero.Fs) error
	HandleHTMLOutput(prefix string, results map[string]StatusCount, fs afero.Fs) error
}

type DefaultOutputHandler struct{}

func (d DefaultOutputHandler) HandleJSONOutput(prefix string, results map[string]StatusCount, fs afero.Fs) error {
	return handleJSONOutput(prefix, results, fs)
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

	handleConsoleOutput(results)
	return nil
}

const htmlTemplate = `
<!DOCTYPE html>
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
            <th>Failed</th>
            <th>Error Messages</th>
        </tr>
    </thead>
    <tbody>
        {{range $name, $counts := .}}
        <tr>
            <td>{{$name}}</td>
            <td>{{$counts.Passed}}</td>
            <td>{{$counts.Failed}}</td>
            <td>{{join $counts.Messages ", "}}</td>
        </tr>
        {{end}}
    </tbody>
</table>
</body>
</html>
`

func handleJSONOutput(prefix string, results map[string]StatusCount, fs afero.Fs) error {
	outputFileName := fmt.Sprintf("%s_aggregated_results.json", prefix)
	resultData, err := json.MarshalIndent(results, "", "    ")
	if err != nil {
		return fmt.Errorf("error marshalling results: %v", err)
	}
	return afero.WriteFile(fs, outputFileName, resultData, 0644)
}

func handleHTMLOutput(prefix string, results map[string]StatusCount, fs afero.Fs) error {
	funcMap := template.FuncMap{
		"join": strings.Join, // Providing implementation for the join function used in the template.
	}

	htmlTemplate := htmlTemplate
	t, err := template.New("report").Funcs(funcMap).Parse(htmlTemplate)
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
	fmt.Println("Scenario Summary")
	fmt.Printf("%-20s | %-6s | %-6s | %s\n", "Scenario Name", "Passed", "Failed", "Error Messages")
	fmt.Println(strings.Repeat("-", 60)) // Adjust the length as needed for better formatting

	for name, count := range results {
		errorMessages := strings.Join(count.Messages, "; ")
		fmt.Printf("%-20s | %-6d | %-6d | %s\n", name, count.Passed, count.Failed, errorMessages)
	}
}
