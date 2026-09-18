package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigCommand_Registration(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"config"})
	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "config", cmd.Name())
}

func TestConfigCommand_Metadata(t *testing.T) {
	assert.Equal(t, "config", configCmd.Use)
	assert.Contains(t, configCmd.Short, "configuraci")
	assert.NotNil(t, configCmd.Run)
}

func TestConfigCommand_Usage(t *testing.T) {
	output := captureStdout(t, func() {
		rootCmd.SetArgs([]string{"config"})
		err := rootCmd.Execute()
		require.NoError(t, err)
	})
	assert.Contains(t, output, "init")
	assert.Contains(t, output, "show")
	assert.Contains(t, output, "set")
	assert.Contains(t, output, "get")
}
