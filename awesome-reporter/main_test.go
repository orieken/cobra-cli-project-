package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		{Name: "Scenario1", Steps: []Step{{Result: Result{Status: "passed"}}, {Result: Result{Status: "failed", ErrorMessage: "Error 1"}}}},
		{Name: "Scenario2", Steps: []Step{{Result: Result{Status: "passed"}}}},
	}
	results := make(map[string]StatusCount)
	aggregateStatusCounts(scenarios, results)

	assert.True(t, results["Scenario1"].Failed)
	assert.NotEmpty(t, results["Scenario1"].Messages)
	assert.True(t, results["Scenario2"].Passed)
	assert.Empty(t, results["Scenario2"].Messages)
}

func TestProcessFilesWithErrors(t *testing.T) {
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

	results := processFiles("test", "awesome-")

	assert.Len(t, results, 1)
	assert.True(t, results["Scenario with Errors"].Failed)
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
	fs := afero.NewMemMapFs()
	results := make(map[string]StatusCount)
	handler := new(MockOutputHandler)
	prefix := "test_prefix"

	handler.On("HandleJSONOutput", prefix, results, fs).Return(nil)
	handler.On("HandleHTMLOutput", prefix, results, fs).Return(errors.New("HTML output failed"))

	err := outputResults(handler, prefix, results, fs)

	assert.Error(t, err)
	assert.Equal(t, "HTML output failed", err.Error())

	handler.AssertExpectations(t)
}

func TestHandleJSONOutput(t *testing.T) {
	fs := afero.NewMemMapFs()
	results := map[string]StatusCount{
		"ExampleScenario": {Passed: true, Messages: []string{"No error"}},
	}
	prefix := "test"

	err := handleJSONOutput(prefix, results, fs)
	assert.NoError(t, err)

	fileName := fmt.Sprintf("%s_aggregated_results.json", prefix)
	exists, err := afero.Exists(fs, fileName)
	assert.True(t, exists)
	assert.NoError(t, err)

	content, err := afero.ReadFile(fs, fileName)
	assert.NoError(t, err)

	var readResults map[string]StatusCount
	json.Unmarshal(content, &readResults)
	assert.Equal(t, results, readResults)
}

func TestHandleHTMLOutput(t *testing.T) {
	fs := afero.NewMemMapFs()
	results := map[string]StatusCount{
		"ExampleScenario": {Passed: true, Messages: []string{"No error"}},
	}
	prefix := "test"

	err := handleHTMLOutput(prefix, results, fs)
	assert.NoError(t, err)

	fileName := fmt.Sprintf("%s_report.html", prefix)
	exists, err := afero.Exists(fs, fileName)
	assert.True(t, exists)
	assert.NoError(t, err)

	content, err := afero.ReadFile(fs, fileName)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "ExampleScenario")
}

func TestHandleConsoleOutput(t *testing.T) {
	results := map[string]StatusCount{
		"ExampleScenario": {Passed: true, Messages: []string{"No error"}},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	handleConsoleOutput(results)

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = old

	output := string(out)
	expectedOutput := "Scenario Name | Status | Error Messages\nExampleScenario | Passed | No error\n"
	assert.Contains(t, output, expectedOutput)
}
