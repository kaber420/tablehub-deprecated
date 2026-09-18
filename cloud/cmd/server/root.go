package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "tablehub-cloud",
	Short:         "Tablehub Cloud CLI",
	Long:          "Tablehub Cloud es un robusto backend y CLI.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	_ = godotenv.Load() // Fase 1: Autocarga de variables
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Modo verbose con logs detallados")
	rootCmd.PersistentFlags().StringP("config", "c", "", "Ruta a archivo de configuración alternativo")
	rootCmd.PersistentFlags().Bool("no-color", false, "Deshabilitar colores en la salida")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Modo silencioso, solo errores")
	rootCmd.PersistentFlags().StringP("output", "o", "text", "Formato de salida: text, json, table")
}
