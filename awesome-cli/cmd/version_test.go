package cmd

import (
	"bytes"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"testing"
)

func executeCommand(root *cobra.Command, args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs(args)

	err = root.Execute()
	return buf.String(), err
}

func TestVersionCommand(t *testing.T) {
	output, err := executeCommand(rootCmd, "version")
	assert.NoError(t, err)
	assert.Contains(t, output, "CLI")
}
