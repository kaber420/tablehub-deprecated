# Plan: Refactorización de la Aplicación a CLI Integrada

> Estado: CONCEPTO / BORRADOR. El objetivo de este documento es fijar el concepto para eliminar el `dev-cli` (que no sirvió y fue una pérdida de tiempo) y convertir la aplicación principal de Go en un lanzador (CLI) autónomo y robusto.

---

## 1. Concepto (qué es y qué NO es)

**La CLI Integrada** es la evolución del binario principal de Tablehub Cloud (`tablehub-cloud`). En lugar de ser un simple script que requiere exportar variables a mano o usar Makefiles, la aplicación se convierte en su propio lanzador formal.

**NO es** un script externo, ni un `dev-cli` genérico, ni un Makefile. Es el estándar absoluto de aplicaciones profesionales en Go (como Docker, Kubernetes, etc.).

---

## 2. Arquitectura de la CLI

El binario principal usará `Cobra` (el estándar de Go) para manejar comandos y `godotenv` para auto-cargar variables de entorno desde el archivo `.env` local antes de iniciar.

### 2.1 Comandos Principales

```text
tablehub-cloud (binario único)
├── serve           (Carga .env y levanta el servidor HTTP/WS de la nube)
├── migrate         (Ejecuta migraciones de base de datos)
├── seed            (Siembra datos iniciales de restaurantes)
├── version         (Muestra la versión de la app)
├── doctor          (Verifica dependencias y configuración del sistema)
├── config          (Gestionar configuración: init, show, set, get)
└── completion      (Generar scripts de completado para bash/zsh/fish/powershell)
```

### 2.2 Flags Globales (disponibles en todos los comandos)

| Flag | Alias | Descripción |
|------|-------|-------------|
| `--help` | `-h` | Mostrar ayuda del comando (viene con Cobra) |
| `--verbose` | `-v` | Modo verbose con logs detallados |
| `--config` | `-c` | Ruta a archivo de configuración alternativo |
| `--no-color` | | Deshabilitar colores en la salida |
| `--quiet` | `-q` | Modo silencioso, solo errores |
| `--output` | `-o` | Formato de salida: `text`, `json`, `table` |

### 2.3 Flags del Comando `serve`

| Flag | Alias | Descripción | Default |
|------|-------|-------------|---------|
| `--port` | `-p` | Puerto del servidor HTTP | `8080` |
| `--host` | | Host del servidor | `0.0.0.0` |
| `--env` | `-e` | Entorno: `dev`, `staging`, `prod` | `dev` |
| `--log-level` | | Nivel de log: `debug`, `info`, `warn`, `error` | `info` |
| `--graceful-timeout` | | Timeout para graceful shutdown | `30s` |

### 2.4 Comandos de Base de Datos

```text
tablehub-cloud db
├── migrate         (Ejecutar migraciones pendientes)
├── status          (Estado actual de la DB)
├── seed            (Sembrar datos iniciales)
└── reset           (Resetear DB - solo en desarrollo)
```

### 2.5 Comandos de Utilidad

```text
tablehub-cloud doctor
├── Verifica conexión a PostgreSQL
├── Verifica variables de entorno requeridas
├── Verifica versión de Go compilada
└── Verifica dependencias del sistema

tablehub-cloud config
├── init            (Generar archivo .env de ejemplo)
├── show            (Mostrar configuración actual)
├── set <key> <val> (Establecer valor de configuración)
└── get <key>       (Obtener valor de configuración)
```

---

## 3. Cambios necesarios (lo que falta hoy)

Hoy `cloud/cmd/server/main.go` arranca el servidor de golpe y exige que el usuario o un script inyecte las variables. Se requiere:

- **Autocarga del `.env`**: Modificar el inicio para buscar el `.env` automáticamente.
- **Estructura Cobra**: Implementar comandos formales (`serve`, `version`, `doctor`, etc.).
- **Eliminación del viejo CLI**: Borrar la carpeta `scripts/dev-cli` que solo generó ruido y no resolvió el problema estructural de la aplicación.
- **Manejo de errores consistente**: Usar `cmd.SilenceErrors` y `cmd.SilenceUsage` para evitar mensajes duplicados.
- **Código de salida apropiado**: Diferentes códigos para diferentes tipos de errores (1 = error general, 2 = error de uso, 3 = error de conexión).

---

## 4. Estructura de Directorios Propuesta

```text
cloud/cmd/server/
├── main.go              (Punto de entrada mínimo, solo inicializa Cobra)
├── root.go              (Comando raíz con flags globales)
├── serve.go             (Comando serve)
├── version.go           (Comando version)
├── doctor.go            (Comando doctor)
├── config.go            (Comando config y subcomandos)
├── db.go                (Comando db y subcomandos)
├── completion.go        (Generación de completado)
└── migrate.go           (Comando migrate)
```

---

## 5. Ejemplos de Uso

```bash
# Iniciar servidor
./tablehub-cloud serve
./tablehub-cloud serve --port 9090 --env prod
./tablehub-cloud serve -v  # modo verbose

# Verificar configuración
./tablehub-cloud doctor
./tablehub-cloud config show

# Gestión de base de datos
./tablehub-cloud db migrate
./tablehub-cloud db seed
./tablehub-cloud db status

# Utilidades
./tablehub-cloud version
./tablehub-cloud completion bash > /etc/bash_completion.d/tablehub-cloud
./tablehub-cloud completion zsh > ~/.zsh/completions/_tablehub-cloud

# Flags de salida
./tablehub-cloud config show --output json
./tablehub-cloud db status --output table
./tablehub-cloud version --quiet  # solo el número de versión
```

---

## 6. Fases Incrementales

### Fase 1 — Autocarga de variables
- Importar `github.com/joho/godotenv` en `main.go`.
- Criterio: Ejecutar `./tablehub-cloud` lee el `.env` mágicamente sin poner comandos larguísimos.

### Fase 2 — Implementación del Lanzador Formal (Cobra)
- Reestructurar `cloud/cmd/server/main.go` en múltiples archivos (`root.go`, `serve.go`, etc.).
- Implementar comando `serve` con todos sus flags.
- Implementar comando `version` con información de build.
- Criterio: La aplicación levanta la nube explícitamente con `./tablehub-cloud serve`.

### Fase 3 — Comandos de Utilidad
- Implementar `doctor` para verificar dependencias.
- Implementar `config` con subcomandos `init`, `show`, `set`, `get`.
- Implementar `completion` para generación de scripts de shell.
- Criterio: Comandos de utilidad funcionales y documentados.

### Fase 4 — Comandos de Base de Datos
- Implementar `db migrate`, `db status`, `db seed`.
- Criterio: Gestión de DB completamente integrada en la CLI.

### Fase 5 — Limpieza de basura
- Eliminar la carpeta `scripts/dev-cli` por completo.
- Criterio: Cero rastros de scripts pendejos en el código base.

---

## 7. Criterios de Aceptación General

1. No se requiere volver a usar `export $(cat .env)` ni Makefiles para iniciar la nube.
2. La aplicación es 100% autónoma y se lanza a sí misma de forma profesional.
3. El `dev-cli` desaparece de la faz de la tierra.
4. `--help` funciona en todos los comandos y subcomandos.
5. `./tablehub-cloud doctor` verifica que todo está configurado correctamente.
6. `./tablehub-cloud completion bash` genera el script de completado válido.
7. Los errores muestran mensajes claros y códigos de salida apropiados.
8. La CLI puede usar `--output json` para integración con otros scripts.

---

## 8. Dependencias a Agregar

```go
// go.mod
github.com/spf13/cobra v1.8.0
github.com/joho/godotenv v1.5.1
```

---

## 9. Archivos Probables a Tocar

- `cloud/cmd/server/main.go` (Refactor masivo → mínimo entrypoint)
- `cloud/cmd/server/root.go` (NUEVO - comando raíz)
- `cloud/cmd/server/serve.go` (NUEVO - comando serve)
- `cloud/cmd/server/version.go` (NUEVO - comando version)
- `cloud/cmd/server/doctor.go` (NUEVO - comando doctor)
- `cloud/cmd/server/config.go` (NUEVO - comando config)
- `cloud/cmd/server/db.go` (NUEVO - comando db)
- `cloud/cmd/server/completion.go` (NUEVO - generación de completado)
- `cloud/internal/config/config.go` (Agregar campos de configuración)
- `go.mod` (Agregar dependencias)
- `scripts/dev-cli/*` (BORRAR TODO)
