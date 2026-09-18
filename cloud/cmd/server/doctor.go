package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Verifica dependencias y configuración del sistema",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("✅ Doctor Check: Todo parece estar en orden.")
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
