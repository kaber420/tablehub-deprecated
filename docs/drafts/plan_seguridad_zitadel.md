# Plan de Implementación: Seguridad B2B SaaS con Zitadel (OIDC)

Este plan define la arquitectura de seguridad para Tablehub Cloud utilizando **Zitadel** (Open Source) como nuestro Proveedor de Identidad (IAM). Esta es una solución definitiva de grado de producción que maneja toda la complejidad criptográfica, MFA y sesiones, dejando a nuestro backend en Go únicamente la tarea de autorizar peticiones.

## Arquitectura de Solución (Zitadel + Go)

1. **Autoalojamiento:** Desplegaremos Zitadel usando Docker (`docker-compose.yml`) junto a nuestra base de datos PostgreSQL.
2. **Resource Server (Go):** El backend de Tablehub actuará como una API segura (Resource Server). No manejará contraseñas ni login.
3. **Flujo de Trabajo:** 
   - El Frontend (Svelte) redirigirá al usuario a Zitadel para iniciar sesión.
   - Zitadel validará al usuario y le entregará un Token JWT.
   - El Frontend enviará el Token JWT al backend en Go en cada petición REST.
   - El backend en Go validará el token usando las llaves públicas de Zitadel (JWKS) sin hacer consultas a la base de datos de contraseñas.

## ⚠️ Puntos a Considerar: Refactorización a Organización

- **Modelo de Datos B2B:** Tienes toda la razón, llamarlo "restaurant" a nivel de infraestructura SaaS limita el sistema. **Antes de integrar Zitadel**, haremos un refactor (búsqueda y reemplazo) en toda la base de código de Go y base de datos para cambiar `restaurant` y `restaurant_id` a `organization` y `organization_id`.
- **Integración Segura:** Dado que Zitadel guardará a los usuarios, nuestra base de datos en Go solo necesita saber qué `zitadel_user_id` corresponde a qué `organization_id` en Tablehub. No guardaremos emails ni contraseñas en la BD de la aplicación.

## Cambios Propuestos

### Fase 0: Refactorización (Restaurant -> Organization)
- **Base de Datos:** Renombrar la tabla `restaurants` a `organizations` en las migraciones SQL.
- **Código Go:** Reemplazar todas las referencias de `restaurant_id` por `organization_id` en los dominios, interfaces REST y WebSockets.

### 1. Infraestructura Docker

- **Archivo:** `cloud/docker-compose.yml`
- **Acciones:**
  - Agregar el servicio `zitadel` usando la imagen oficial (`ghcr.io/zitadel/zitadel:latest`).
  - Agregar un script de inicialización (`zitadel-setup`) necesario para crear la base de datos de configuración de Zitadel dentro de nuestra instancia de PostgreSQL.

### 2. Base de Datos del Dominio (Tablehub)

- **Archivo:** `cloud/internal/infrastructure/db/migrations/004_users_business.sql`
- **Acciones:**
  - Crear tabla `users` para vincular la identidad con la organización.
  - Campos: `id` (UUID interno), `zitadel_user_id` (String único proveniente de Zitadel), `organization_id` (FK a la tabla `organizations`).

### 3. Middleware y Validación OIDC (Go)

- **Archivos Nuevos:** 
  - `cloud/pkg/auth/jwks.go`: Lógica para conectarse al endpoint de Zitadel (`/oauth/v2/keys`) y descargar/cachear las llaves públicas (JWKS).
  - `cloud/internal/interfaces/rest/middleware.go`: Middleware `RequireAuth` que extrae el token del header, valida la firma, extrae el `zitadel_user_id` y lo inyecta en el contexto.

### 4. Rutas y Aprovisionamiento Seguro

- **Archivos Modificados:**
  - `cloud/internal/interfaces/rest/router.go`: Envolver rutas críticas (`/v1/hubs/provision`) con el middleware `RequireAuth`.
  - `cloud/internal/application/tunnel/provision.go`: Refactorizar para que en lugar de recibir el `organization_id` en el body, lo busque en la base de datos de Tablehub usando el `zitadel_user_id` validado por el middleware.

## Plan de Verificación

1. Levantar la infraestructura modificada con Zitadel: `docker-compose up -d --build`.
2. Acceder a la interfaz web local de Zitadel para crear una Organización, una Aplicación (API) y un Usuario de prueba.
3. Extraer un Token JWT válido usando la consola de Zitadel (o un script curl).
4. Hacer una petición de prueba a `/v1/hubs/provision` sin token y confirmar que retorna `401 Unauthorized`.
5. Hacer la petición con el token válido de Zitadel y confirmar que Go lo procesa, validando la identidad contra el `organization_id` correctamente.
