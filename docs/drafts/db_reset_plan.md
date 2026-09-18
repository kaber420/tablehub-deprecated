# Plan de Rediseño de Base de Datos: Método Destructivo (Opción A)

## Objetivo
Simplificar el flujo de desarrollo local y pruebas E2E eliminando el sistema de migraciones incrementales.
Adoptaremos un enfoque destructivo donde la base de datos se recrea desde cero usando un único archivo consolidado (`schema.sql`). Esto acelera la iteración en fases tempranas donde los datos de prueba son descartables.

## Tareas a Implementar

### 1. Consolidación del Esquema
- Analizar los 11 archivos `.sql` existentes en `cloud/internal/infrastructure/db/migrations/`.
- Combinar todas las creaciones de tablas, índices, constraints y funciones en un solo archivo: `cloud/internal/infrastructure/db/schema.sql`.
- El esquema final incluirá las modificaciones más recientes (ej. tabla `pending_operations`, columnas `onboarding_state`, etc.).

### 2. Limpieza de Migraciones
- Eliminar la carpeta `cloud/internal/infrastructure/db/migrations/` y sus 11 archivos para evitar confusión.

### 3. Modificación del Comando `db` en la CLI
- Modificar `cloud/cmd/server/db.go` para implementar el subcomando `reset`.
- **Lógica del comando `tablehub-cloud db reset`**:
  1. Conectarse a la base de datos (`tablehub_cloud`).
  2. Ejecutar `DROP SCHEMA public CASCADE; CREATE SCHEMA public;` para borrar absolutamente todo de forma limpia y rápida.
  3. Leer el archivo `schema.sql` desde el disco.
  4. Ejecutar el contenido del archivo con `pool.Exec` para recrear todas las tablas.
  5. Imprimir en consola: "Base de datos reseteada con éxito".

### 4. Actualización de Tests E2E
- Modificar `cloud/tests/e2e/e2e_test.go`.
- Reemplazar la función `runMigrations` (que actualmente itera sobre los 11 archivos) por una nueva función `setupTestDB`.
- `setupTestDB` ejecutará `DROP SCHEMA public CASCADE; CREATE SCHEMA public;` y luego aplicará el único archivo `schema.sql`.

## Consideraciones Finales
- Al ejecutar `tablehub-cloud db reset`, el desarrollador es consciente de que se purgarán todos los datos locales.
- No hay dependencias externas nuevas requeridas (ni Goose, ni librerías de migración). Todo se hace con la librería estándar de Go y el driver nativo de Postgres (`pgx`).

---
*Este plan está en borrador. Si estás de acuerdo con la estrategia, responde indicando que proceda con la implementación.*
