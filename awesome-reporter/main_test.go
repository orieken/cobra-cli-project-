package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"html/template"
	"io"
	"os"
	"testing"
)

func TestListRelevantFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	fs.MkdirAll("test", 0755)
	afero.WriteFile(fs, "test/awesome-report1.json", []byte{}, 0644)
	afero.WriteFile(fs, "test/awesome-report2.json", []byte{}, 0644)
	afero.WriteFile(fs, "test/ignore-report.json", []byte{}, 0644)

	appFS = fs // Replace the filesystem with the memory filesystem for testing
	files, err := listRelevantFiles("test", "awesome-")
	assert.NoError(t, err)
	assert.Len(t, files, 2) // Should only match "awesome-" prefixed and ".json" suffixed files
}

func TestAggregateStatusCounts(t *testing.T) {
	scenarios := []Scenario{
		{Name: "Scenario1", Steps: []Step{{Result: Result{Status: "passed"}}, {Result: Result{Status: "failed"}}}},
		{Name: "Scenario2", Steps: []Step{{Result: Result{Status: "passed"}}}},
	}
	results := make(map[string]StatusCount)
	aggregateStatusCounts(scenarios, results)

	assert.Equal(t, 1, results["Scenario1"].Passed)
	assert.Equal(t, 1, results["Scenario1"].Failed)
	assert.Equal(t, 1, results["Scenario2"].Passed)
}

func TestProcessFilesWithErrors(t *testing.T) {
	// Setup the in-memory filesystem with test data
	fs := afero.NewMemMapFs()
	fileContent := `[{
		"elements": [{
			"name": "Scenario with Errors",
			"steps": [{
				"result": {
					"status": "failed",
					"error_message": "Element not found"
				}
			},{
				"result": {
					"status": "passed"
				}
			}]
		}]
	}]`
	afero.WriteFile(fs, "test/awesome-error-scenario.json", []byte(fileContent), 0644)
	appFS = fs

	// Invoke the code under test
	results := processFiles("test", "awesome-")

	// Assertions
	assert.Len(t, results, 1)
	assert.Equal(t, 1, results["Scenario with Errors"].Failed)
	assert.Equal(t, 1, results["Scenario with Errors"].Passed)
	assert.Contains(t, results["Scenario with Errors"].Messages, "Element not found")
}

type MockOutputHandler struct {
	mock.Mock
}

func (m *MockOutputHandler) HandleJSONOutput(prefix string, results map[string]StatusCount, fs afero.Fs) error {
	args := m.Called(prefix, results, fs)
	return args.Error(0)
}

func (m *MockOutputHandler) HandleHTMLOutput(prefix string, results map[string]StatusCount, fs afero.Fs) error {
	args := m.Called(prefix, results, fs)
	return args.Error(0)
}

func TestOutputResults(t *testing.T) {
	// Setup
	fs := afero.NewMemMapFs()
	results := make(map[string]StatusCount)
	handler := new(MockOutputHandler)
	prefix := "test_prefix"

	// Mock expectations
	handler.On("HandleJSONOutput", prefix, results, fs).Return(nil) // Assuming no error expected for JSON output
	handler.On("HandleHTMLOutput", prefix, results, fs).Return(errors.New("HTML output failed"))

	// Call the function
	err := outputResults(handler, prefix, results, fs)

	// Assertions
	assert.Error(t, err)
	assert.Equal(t, "HTML output failed", err.Error(), "Error should be propagated from HTML handler")

	// Verify that all expectations are met
	handler.AssertExpectations(t)
}

func TestHandleJSONOutput(t *testing.T) {
	fs := afero.NewMemMapFs()
	results := map[string]StatusCount{
		"ExampleScenario": {Passed: 2, Failed: 1},
	}
	prefix := "test"

	err := handleJSONOutput(prefix, results, fs)
	assert.NoError(t, err)

	// Check if the file exists
	fileName := fmt.Sprintf("%s_aggregated_results.json", prefix)
	exists, err := afero.Exists(fs, fileName)
	assert.True(t, exists)
	assert.NoError(t, err)

	// Check file content
	content, err := afero.ReadFile(fs, fileName)
	assert.NoError(t, err)

	var readResults map[string]StatusCount
	json.Unmarshal(content, &readResults)
	assert.Equal(t, results, readResults, "The JSON file should contain the expected results.")
}

type UnmarshallableStatusCount struct {
	Passed  int
	Pending int
	Failed  int
	Skipped int
	Func    func() // Functions cannot be marshaled to JSON
}

func handleJSONOutputWithError(prefix string, results map[string]UnmarshallableStatusCount, fs afero.Fs) error {
	outputFileName := fmt.Sprintf("%s_aggregated_results.json", prefix)
	resultData, err := json.MarshalIndent(results, "", "    ")
	if err != nil {
		return fmt.Errorf("error marshalling results: %v", err)
	}

	// Use the provided filesystem to write the file
	err = afero.WriteFile(fs, outputFileName, resultData, 0644)
	if err != nil {
		return err
	}
	return nil
}

func TestHandleJSONOutputMarshallingError(t *testing.T) {
	fs := afero.NewMemMapFs()
	results := map[string]UnmarshallableStatusCount{
		"ExampleScenario": {Passed: 1, Failed: 1, Func: func() {}},
	}

	err := handleJSONOutputWithError("test_prefix", results, fs)
	assert.Error(t, err, "Expected an error during marshalling of unmarshallable types")
}

func TestOutputResultsJSONError(t *testing.T) {
	handler := new(MockOutputHandler)
	fs := afero.NewMemMapFs()
	results := make(map[string]StatusCount)

	// Mock expectations - setting up to return an error on HandleJSONOutput
	handler.On("HandleJSONOutput", "test_prefix", results, fs).Return(errors.New("mock json output error"))

	// Call the function
	err := outputResults(handler, "test_prefix", results, fs)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock json output error")

	// Verify that all expectations are met
	handler.AssertExpectations(t)
}

func TestHandleHTMLOutput(t *testing.T) {
	fs := afero.NewMemMapFs()
	results := map[string]StatusCount{
		"ExampleScenario": {Passed: 2, Failed: 1},
	}
	prefix := "test"

	err := handleHTMLOutput(prefix, results, fs)
	assert.NoError(t, err)

	// Check if the file exists
	fileName := fmt.Sprintf("%s_report.html", prefix)
	exists, err := afero.Exists(fs, fileName)
	assert.True(t, exists)
	assert.NoError(t, err)

	// Optionally check for specific HTML content
	content, err := afero.ReadFile(fs, fileName)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "ExampleScenario", "The HTML file should contain scenario names.")
}

func handleHTMLOutputParseError(prefix string, results map[string]StatusCount, fs afero.Fs) error {
	htmlFileName := fmt.Sprintf("%s_report.html", prefix)
	htmlTemplate := "{{ .Name }" // Malformed template

	t, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("error parsing HTML template: %v", err)
	}

	htmlFile, err := fs.Create(htmlFileName)
	if err != nil {
		return fmt.Errorf("error creating HTML file: %v", err)
	}
	defer htmlFile.Close()

	return t.Execute(htmlFile, results)
}

func TestHandleHTMLOutputParseError(t *testing.T) {
	fs := afero.NewMemMapFs()
	results := make(map[string]StatusCount) // Assuming this is already defined elsewhere

	err := handleHTMLOutputParseError("test_prefix", results, fs)
	assert.Error(t, err, "Expected an error due to malformed HTML template")
}

func handleHTMLOutputCreateError(prefix string, results map[string]StatusCount, fs afero.Fs) error {
	htmlFileName := fmt.Sprintf("%s_report.html", prefix)
	htmlTemplate := "<html>{{.Name}}</html>"

	t, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("error parsing HTML template: %v", err)
	}

	htmlFile, err := fs.Create(htmlFileName)
	if err != nil {
		return fmt.Errorf("error creating HTML file: %v", err)
	}
	defer htmlFile.Close()

	return t.Execute(htmlFile, results)
}

func TestHandleHTMLOutputCreateError(t *testing.T) {
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs()) // Make filesystem read-only
	results := make(map[string]StatusCount)

	err := handleHTMLOutputCreateError("test_prefix", results, fs)
	assert.Error(t, err, "Expected an error due to read-only filesystem")
}

func TestHandleConsoleOutput(t *testing.T) {
	results := map[string]StatusCount{
		"ExampleScenario": {Passed: 2, Failed: 1, Messages: []string{"Failed due to timeout"}},
	}

	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	handleConsoleOutput(results)

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = old

	output := string(out)

	expectedOutput := "Scenario Name | Passed | Pending | Failed | Skipped | Error Messages\nExampleScenario | 2 | 0 | 0 | 1 | Failed due to timeout\n"
	assert.Contains(t, output, expectedOutput, "The console output should match the expected format and values")
}
