package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeCommand_Registration(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"serve"})
	require.NoError(t, err)
	require.NotNil(t, cmd)
	assert.Equal(t, "serve", cmd.Name())
}

func TestServeCommand_Metadata(t *testing.T) {
	assert.Equal(t, "serve", serveCmd.Use)
	assert.Contains(t, serveCmd.Short, "servidor")
	assert.NotNil(t, serveCmd.Run)
}

func TestServeCommand_Flags(t *testing.T) {
	f := serveCmd.Flags()

	port, err := f.GetInt("port")
	require.NoError(t, err)
	assert.Equal(t, 8080, port)

	host, err := f.GetString("host")
	require.NoError(t, err)
	assert.Equal(t, "0.0.0.0", host)

	env, err := f.GetString("env")
	require.NoError(t, err)
	assert.Equal(t, "dev", env)

	logLevel, err := f.GetString("log-level")
	require.NoError(t, err)
	assert.Equal(t, "info", logLevel)

	gracefulTimeout, err := f.GetString("graceful-timeout")
	require.NoError(t, err)
	assert.Equal(t, "30s", gracefulTimeout)
}
