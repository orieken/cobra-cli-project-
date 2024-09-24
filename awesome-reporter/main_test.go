package main

import (
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
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

func TestOutputResults(t *testing.T) {
	fs := afero.NewMemMapFs()
	appFS = fs // Replace the filesystem with the memory filesystem for testing

	results := map[string]StatusCount{
		"Scenario1": {
			Passed:   1,
			Failed:   1,
			Messages: []string{"Failed due to timeout", "Element not visible"},
		},
	}

	err := outputResults("testprefix", results, fs)
	assert.NoError(t, err)

	// Verify that the output file is created
	exists, err := afero.Exists(fs, "testprefix_aggregated_results.json")
	assert.True(t, exists)
	assert.NoError(t, err)

	// Optionally read back the file and check content
	content, err := afero.ReadFile(fs, "testprefix_aggregated_results.json")
	assert.NoError(t, err)
	assert.Contains(t, string(content), "Failed due to timeout")
	assert.Contains(t, string(content), "Element not visible")
}

//
//func TestOutputResults(t *testing.T) {
//	prefix := "bc"
//	fs := afero.NewMemMapFs()
//	appFS = fs
//
//	results := map[string]StatusCount{
//		"Scenario1": {Passed: 1, Failed: 1},
//	}
//
//	resW := os.Stdout
//	r, w, _ := os.Pipe()
//	os.Stdout = w
//
//	err := outputResults(prefix, results, fs)
//	if err != nil {
//		t.Errorf("Error writing to in-memory filesystem: %v", err)
//	}
//
//	// Check if the file exists in the in-memory filesystem
//	exists, err := afero.Exists(fs, "bc_aggregated_results.json")
//	if err != nil {
//		t.Errorf("Error checking file existence: %v", err)
//	}
//	if !exists {
//		t.Error("File 'bc_aggregated_results.json' not created in in-memory filesystem")
//	}
//
//	w.Close()
//	out, _ := io.ReadAll(r)
//	os.Stdout = resW
//
//	outputString := string(out)
//	assert.Contains(t, outputString, "Scenario1", "Output should contain 'Scenario1'")
//	assert.Contains(t, outputString, "| 1 | 0 | 1", "Output should contain correct counts")
//
//	content, _ := afero.ReadFile(fs, "bc_aggregated_results.json")
//	assert.Contains(t, string(content), "Scenario1", "File should contain 'Scenario1'")
//}
