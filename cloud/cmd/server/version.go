package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Muestra la versión de la app",
	Run: func(cmd *cobra.Command, args []string) {
		quiet, _ := rootCmd.PersistentFlags().GetBool("quiet")
		if quiet {
			fmt.Println("1.0.0")
		} else {
			fmt.Println("Tablehub Cloud CLI v1.0.0")
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
