package main

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockCommandExecutor struct {
	mock.Mock
}

func (m *MockCommandExecutor) CombinedOutput() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1)
}

func TestParsePythonVersion(t *testing.T) {
	tests := []struct {
		output string
		expect string
		err    string
	}{
		{"Python 3.10.1", "3.10.1", ""},
		{"Python 2.7.16", "2.7.16", ""},
		{"Invalid output", "", "failed to detect Python version"},
	}

	for _, test := range tests {
		version, err := parsePythonVersion(test.output)
		if test.err == "" {
			assert.NoError(t, err)
			assert.Equal(t, test.expect, version)
		} else {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), test.err)
		}
	}
}

func TestIsVersionAtLeast310(t *testing.T) {
	assert.True(t, isVersionAtLeast310("3.10"), "3.10 should be considered at least 3.10")
	assert.True(t, isVersionAtLeast310("3.11"), "3.11 should be considered at least 3.10")
	assert.True(t, isVersionAtLeast310("4.0"), "4.0 should be considered at least 3.10")
	assert.False(t, isVersionAtLeast310("3.9"), "3.9 should not be considered at least 3.10")
}

func TestIsVersionAtLeast310_Errors(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		expectErr bool
		errMsg    string
	}{
		{"InvalidBaseVersion", "3.10.0.0", true, "Error parsing current version:"},
		{"InvalidCurrentVersion", "invalid_version", true, "Error parsing current version:"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buf bytes.Buffer
			out = &buf                         // Redirect output to buffer for testing
			defer func() { out = os.Stdout }() // Restore default output after the test

			result := isVersionAtLeast310(test.version)
			output := buf.String()

			if test.expectErr {
				assert.False(t, result, "Expected result to be false due to parsing error")
				assert.Contains(t, output, test.errMsg, "Expected error message in output")
			} else {
				assert.True(t, result, "Expected result to be true")
			}
		})
	}
}

func TestCheckPythonVersion(t *testing.T) {
	tests := []struct {
		name   string
		output string
		err    error
		expect string
		hasErr bool
	}{
		{"ValidVersion", "Python 3.10.1", nil, "Python version is 3.10 or higher.", false},
		{"InvalidVersion", "Python 2.7.16", nil, "Python version is below 3.10", true},
		{"ExecutionError", "", fmt.Errorf("command error"), "error executing Python: command error", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockExecutor := new(MockCommandExecutor)
			mockExecutor.On("CombinedOutput").Return([]byte(test.output), test.err)

			err := checkPythonVersion(mockExecutor)
			if test.hasErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.expect)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
