package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCommand_Registration(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"version"})
	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "version", cmd.Name())
}

func TestVersionCommand_Metadata(t *testing.T) {
	assert.Equal(t, "version", versionCmd.Use)
	assert.Contains(t, versionCmd.Short, "versi")
	assert.NotNil(t, versionCmd.Run)
}

func TestVersionCommand_Run(t *testing.T) {
	output := captureStdout(t, func() {
		rootCmd.SetArgs([]string{"version"})
		err := rootCmd.Execute()
		require.NoError(t, err)
	})
	assert.Equal(t, "Tablehub Cloud CLI v1.0.0", output)
}

func TestVersionCommand_Run_Quiet(t *testing.T) {
	output := captureStdout(t, func() {
		rootCmd.SetArgs([]string{"version", "--quiet"})
		err := rootCmd.Execute()
		require.NoError(t, err)
	})
	assert.Equal(t, "1.0.0", output)
}
