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
	Passed   bool     `json:"passed"`
	Failed   bool     `json:"failed"`
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
		count := StatusCount{}
		errorSet := make(map[string]bool)
		for _, step := range scenario.Steps {
			if step.Result.Status == "failed" {
				count.Failed = true
				if step.Result.ErrorMessage != "" && !errorSet[step.Result.ErrorMessage] {
					count.Messages = append(count.Messages, step.Result.ErrorMessage)
					errorSet[step.Result.ErrorMessage] = true
				}
			}
		}
		if !count.Failed {
			count.Passed = true
		}
		results[scenario.Name] = count
	}
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
<html>
<head><title>Test Report</title></head>
<body>
<h1>Scenarios</h1>
<ul>
{{- range $name, $counts := . }}
    <li>{{$name}} - {{if $counts.Passed}}Passed{{else}}Failed{{end}}: {{join $counts.Messages ", "}}</li>
{{- end }}
</ul>
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

	htmlTemplate := `{{/* your existing template string here */}}`
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
	fmt.Println("Scenario Name | Status | Error Messages")
	for name, count := range results {
		errors := strings.Join(count.Messages, "; ")
		status := "Failed"
		if count.Passed {
			status = "Passed"
		}
		fmt.Printf("%s | %s | %s\n", name, status, errors)
	}
}
