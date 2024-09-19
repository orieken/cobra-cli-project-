package awesome_reporter

import (
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
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
		{Name: "Scenario1", Steps: []Step{{Status: "passed"}, {Status: "failed"}}},
		{Name: "Scenario2", Steps: []Step{{Status: "passed"}}},
	}
	results := make(map[string]StatusCount)
	aggregateStatusCounts(scenarios, results)

	assert.Equal(t, 1, results["Scenario1"].Passed)
	assert.Equal(t, 1, results["Scenario1"].Failed)
	assert.Equal(t, 1, results["Scenario2"].Passed)
}

func TestOutputResults(t *testing.T) {
	prefix := "bc"
	fs := afero.NewMemMapFs()
	appFS = fs

	results := map[string]StatusCount{
		"Scenario1": {Passed: 1, Failed: 1},
	}

	resW := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputResults(prefix, results, fs)
	if err != nil {
		t.Errorf("Error writing to in-memory filesystem: %v", err)
	}

	// Check if the file exists in the in-memory filesystem
	exists, err := afero.Exists(fs, "bc_aggregated_results.json")
	if err != nil {
		t.Errorf("Error checking file existence: %v", err)
	}
	if !exists {
		t.Error("File 'bc_aggregated_results.json' not created in in-memory filesystem")
	}

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = resW

	outputString := string(out)
	assert.Contains(t, outputString, "Scenario1", "Output should contain 'Scenario1'")
	assert.Contains(t, outputString, "| 1 | 0 | 1", "Output should contain correct counts")

	content, _ := afero.ReadFile(fs, "bc_aggregated_results.json")
	assert.Contains(t, string(content), "Scenario1", "File should contain 'Scenario1'")
}
