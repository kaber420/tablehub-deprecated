# Plan Arquitectónico: Seguridad y Autenticación del SaaS (Cloud)

## 1. Problema Actual
Tras analizar la carpeta `cloud`, se comprobó que el backend solo gestiona los túneles WebSocket de forma correcta, pero **no existe un sistema de cuentas de usuario, ni sesiones, ni frontend**. Lo más grave: todos los endpoints REST (incluyendo el de aprovisionamiento que genera las llaves) están abiertos públicamente. Cualquier persona podría conectarse a la API y robar claves.

Este plan aborda la implementación crítica (Fase P0) para solucionar esto antes de crear cualquier panel visual.

## 2. Decisiones a Tomar (Preguntas Abiertas)
1. **Frontend SaaS:** Este plan asegura el motor (backend). ¿Deseas que, inmediatamente después de esto, implementemos el Frontend web de la nube reutilizando Svelte (como en el Hub) o prefieres hacerlo después?
2. **Roles de Usuario:** Propongo crear solo dos roles básicos por ahora: `owner` (el dueño de un restaurante, solo ve su negocio) y `admin_saas` (tú, con acceso global). ¿Te parece suficiente?

## 3. Arquitectura de Solución (Fase P0)

### 3.1 Base de Datos (Usuarios)
- **Nuevo Archivo:** `migrations/004_users.sql`
- **Tablas:** Crear la tabla `users` con `id`, `email`, `password_hash`, `role`, `restaurant_id` (Nulo si es admin), y `created_at`.
- **Acción:** Insertar un usuario admin por defecto para que no nos quedemos fuera del sistema.

### 3.2 Lógica de Autenticación (JWT)
- **Nuevos Archivos:** 
  - `pkg/jwt/jwt.go`: Para crear y verificar los tokens usando un secreto en el `.env`.
  - `application/auth/auth_usecase.go`: Para comparar contraseñas usando `bcrypt` y generar los tokens.
  - `infrastructure/db/user_repo.go`: Para conectar Go con la tabla `users` en PostgreSQL.

### 3.3 Endpoints y Middlewares
- **Endpoint de Login:**
  - Archivo: `interfaces/rest/auth_handler.go`
  - Ruta: `POST /v1/auth/login` (Devolverá el token JWT).
- **El Candado (Middleware):**
  - Archivo: `interfaces/rest/middleware.go`
  - Función: `RequireAuth`. Esta función interceptará toda petición y revisará si trae un token válido en el Header. Si no lo trae, devolverá "401 No Autorizado".
- **Asegurar el Router:**
  - Archivo: `interfaces/rest/router.go`
  - Cambio: Envolver `/v1/hubs/provision` y todo `/v1/admin/*` con este nuevo middleware.

### 3.4 Corrección del Aprovisionamiento Multitenant
- **Archivo:** `application/tunnel/provision.go`
- **Cambio crítico:** Actualmente el endpoint recibe el `restaurant_id` desde el body JSON. Esto es inseguro. Lo cambiaremos para que extraiga el `restaurant_id` de forma inviolable directamente desde el Token JWT del usuario que inició sesión. Así garantizamos que un dueño jamás pueda generar llaves para el restaurante de otro.

## 4. Plan de Verificación
1. Intentaremos acceder al aprovisionamiento mediante consola (`curl`) sin enviar token y confirmaremos que rechaza la entrada.
2. Iniciaremos sesión como el usuario administrador por defecto para obtener nuestro JWT.
3. Enviaremos el JWT y confirmaremos que ahora sí genera la llave `.thub` de forma segura.
