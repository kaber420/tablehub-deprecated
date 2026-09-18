package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Gestionar configuración: init, show, set, get",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Uso: tablehub-cloud config [init|show|set|get]")
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
