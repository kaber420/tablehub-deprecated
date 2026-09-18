package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommand_Initialization(t *testing.T) {
	assert.Equal(t, "tablehub-cloud", rootCmd.Use)
	assert.Contains(t, rootCmd.Short, "Tablehub Cloud CLI")
	assert.True(t, rootCmd.SilenceUsage)
	assert.True(t, rootCmd.SilenceErrors)
}

func TestRootCommand_PersistentFlags(t *testing.T) {
	f := rootCmd.PersistentFlags()

	v, err := f.GetBool("verbose")
	require.NoError(t, err)
	assert.False(t, v)

	c, err := f.GetString("config")
	require.NoError(t, err)
	assert.Equal(t, "", c)

	nc, err := f.GetBool("no-color")
	require.NoError(t, err)
	assert.False(t, nc)

	q, err := f.GetBool("quiet")
	require.NoError(t, err)
	assert.False(t, q)

	o, err := f.GetString("output")
	require.NoError(t, err)
	assert.Equal(t, "text", o)
}

func TestRootCommand_Subcommands(t *testing.T) {
	expected := []string{"serve", "db", "doctor", "completion", "version", "config"}
	registered := make(map[string]*cobra.Command, len(rootCmd.Commands()))
	for _, c := range rootCmd.Commands() {
		registered[c.Name()] = c
	}
	for _, name := range expected {
		_, ok := registered[name]
		assert.True(t, ok, "expected subcommand %q to be registered under root", name)
	}
}
