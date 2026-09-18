package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionCommand_Registration(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"completion"})
	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "completion", cmd.Name())
}

func TestCompletionCommand_Metadata(t *testing.T) {
	assert.Equal(t, "completion [bash|zsh|fish|powershell]", completionCmd.Use)
	assert.Contains(t, completionCmd.Short, "completado")
	assert.NotNil(t, completionCmd.Run)
	assert.Equal(t, []string{"bash", "zsh", "fish", "powershell"}, completionCmd.ValidArgs)
	assert.NotNil(t, completionCmd.Args)
}

func TestCompletionCommand_InvalidShell_ReturnsError(t *testing.T) {
	rootCmd.SetArgs([]string{"completion", "invalid_shell"})
	err := rootCmd.Execute()
	assert.Error(t, err)
}

func TestCompletionCommand_ValidShells(t *testing.T) {
	shells := []struct {
		name string
		args []string
	}{
		{"bash", []string{"completion", "bash"}},
		{"zsh", []string{"completion", "zsh"}},
		{"fish", []string{"completion", "fish"}},
		{"powershell", []string{"completion", "powershell"}},
	}

	for _, s := range shells {
		t.Run(s.name, func(t *testing.T) {
			output := captureStdout(t, func() {
				rootCmd.SetArgs(s.args)
				err := rootCmd.Execute()
				require.NoError(t, err)
			})
			assert.NotEmpty(t, output)
		})
	}
}
