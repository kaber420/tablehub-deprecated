package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBCommand_Registration(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"db"})
	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "db", cmd.Name())
}

func TestDBCommand_Metadata(t *testing.T) {
	assert.Equal(t, "db", dbCmd.Use)
	assert.Contains(t, dbCmd.Short, "base de datos")
	assert.NotNil(t, dbCmd.Run)
}

func TestDBCommand_Usage(t *testing.T) {
	output := captureStdout(t, func() {
		rootCmd.SetArgs([]string{"db"})
		err := rootCmd.Execute()
		require.NoError(t, err)
	})
	assert.Contains(t, output, "migrate")
	assert.Contains(t, output, "seed")
	assert.Contains(t, output, "reset")
}
