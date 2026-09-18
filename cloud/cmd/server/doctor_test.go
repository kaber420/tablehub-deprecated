package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoctorCommand_Registration(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"doctor"})
	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "doctor", cmd.Name())
}

func TestDoctorCommand_Metadata(t *testing.T) {
	assert.Equal(t, "doctor", doctorCmd.Use)
	assert.Contains(t, doctorCmd.Short, "dependencias")
	assert.NotNil(t, doctorCmd.Run)
}

func TestDoctorCommand_Run(t *testing.T) {
	output := captureStdout(t, func() {
		rootCmd.SetArgs([]string{"doctor"})
		err := rootCmd.Execute()
		require.NoError(t, err)
	})
	assert.Contains(t, output, "Doctor Check")
	assert.Contains(t, output, "orden")
}
