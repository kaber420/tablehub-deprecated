package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/tablehub/cloud/internal/config"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Comandos de base de datos",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var dbResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Borra toda la base de datos y la recrea desde schema.sql",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("Error cargando config: %v\n", err)
			os.Exit(1)
		}

		pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
		if err != nil {
			fmt.Printf("Error conectando a db: %v\n", err)
			os.Exit(1)
		}
		defer pool.Close()

		fmt.Println("Borrando esquema public...")
		_, err = pool.Exec(context.Background(), "DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
		if err != nil {
			fmt.Printf("Error reseteando esquema: %v\n", err)
			os.Exit(1)
		}

		schemaPath := filepath.Join("internal", "infrastructure", "db", "schema.sql")
		fmt.Printf("Aplicando esquema desde %s...\n", schemaPath)
		content, err := os.ReadFile(schemaPath)
		if err != nil {
			fmt.Printf("Error leyendo schema.sql: %v\n", err)
			os.Exit(1)
		}

		_, err = pool.Exec(context.Background(), string(content))
		if err != nil {
			fmt.Printf("Error aplicando schema.sql: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Base de datos reseteada con éxito!")
	},
}

func init() {
	dbCmd.AddCommand(dbResetCmd)
	rootCmd.AddCommand(dbCmd)
}
