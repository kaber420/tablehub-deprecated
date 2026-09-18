# Plan de Integración y Lecciones Aprendidas: Autenticación con Zitadel

## 0. Lecciones Aprendidas: Configuración Local de Zitadel v4
Tras horas de depuración, descubrimos que Zitadel v4 introdujo cambios arquitectónicos que rompen el desarrollo local simple. Si en el futuro necesitas levantar este entorno desde cero, **ESTOS SON LOS PASOS OBLIGATORIOS** que aprendimos a la mala:

1. **El Problema del Login V2 (Pantalla Negra/Error 404):** Zitadel v4 extrajo la interfaz de login a un contenedor separado (`zitadel-login`). Si levantas el contenedor principal y tratas de loguearte, te toparás con una pantalla negra o un error 404 porque el frontend de login no existe en ese contenedor.
2. **La Solución (Usar Login V1):** Para evitar montar proxies inversos complejos (Traefik) y múltiples contenedores solo para desarrollo local, **DEBES forzar a Zitadel a usar su interfaz de login heredada (V1)**. Esto se logra inyectando la siguiente variable de entorno en el `docker-compose.yml`:
   ```yaml
   ZITADEL_DEFAULTINSTANCE_FEATURES_LOGINV2_REQUIRED=false
   ```
3. **El Problema del TLS (Crasheos al iniciar):** Zitadel v4 asume por defecto que estás usando certificados TLS (`https`). Si usas `--tlsMode external` sin configurar certificados correctamente, el contenedor simplemente crasheará o la UI no cargará. Para desarrollo local HTTP, el comando de inicio en `docker-compose.yml` DEBE incluir `--tlsMode disabled`:
   ```yaml
   command: start-from-init --masterkey "tu-llave-de-32-bytes" --tlsMode disabled --config /zitadel-config.yaml
   ```
4. **Configuración de la Aplicación SPA en Zitadel:**
   - **Application Type:** User Agent (Single Page Application).
   - **Authentication Method:** `None` (PKCE). NUNCA usar `Private Key JWT` para una SPA, ya que el navegador no puede guardar secretos.

## 1. Visión General
El objetivo de este plan es proteger el backend en Go (construido con `go-chi`) para que únicamente responda a peticiones legítimas provenientes del frontend (Svelte) autenticado con Zitadel.

Esto se logra en dos partes: el **frontend** inyecta el token en las peticiones, y el **backend** intercepta y valida criptográficamente ese token antes de procesar la petición.

## 2. Frontend (Svelte): Interceptor HTTP
En lugar de modificar cada petición `fetch` manualmente, crearemos una función centralizada (o un pequeño cliente HTTP) que obtenga el Access Token de memoria y lo inyecte automáticamente en las cabeceras.

**Pasos en Svelte:**
1. Crear `cloud/web/src/lib/api.js`.
2. Implementar una función que llame a `userManager.getUser()` de `oidc-client-ts`.
3. Extraer `user.access_token`.
4. Añadir el encabezado: `Authorization: Bearer <access_token>`.
5. Si el token está expirado, el sistema (a futuro) intentará un *silent renew*, o redirigirá al login.

## 3. Backend (Go): Middleware de Autenticación
Actualmente el proyecto Go utiliza `github.com/go-chi/chi/v5` para el enrutamiento y vemos dependencias de `github.com/golang-jwt/jwt/v5` y `github.com/MicahParks/keyfunc/v3` en el `go.mod`. Esto es ideal, ya que `keyfunc` automatiza la descarga y caché de las llaves públicas de Zitadel (JWKS).

**Pasos en Go:**
1. Crear un archivo `cloud/internal/middleware/auth.go` (o en la carpeta correspondiente de middlewares).
2. Inicializar `keyfunc.NewDefault([]string{"http://localhost:8088/oauth/v2/keys"})` para obtener la firma criptográfica pública de Zitadel.
3. Construir una función `AuthMiddleware(next http.Handler) http.Handler`:
   - Extraer el string de la cabecera `Authorization`.
   - Quitar el prefijo `"Bearer "`.
   - Utilizar `jwt.Parse` junto con la llave de `keyfunc` para descifrar y validar el token.
   - Validar que el token no haya expirado.
   - Opcional pero recomendado: Validar que el `audience` (`aud`) o `client_id` pertenezca a nuestra app (ej. `381303699831521795`).
   - Extraer el ID de usuario (`sub`) e inyectarlo en el contexto de la petición (`r.Context()`).
4. Aplicar el middleware a las rutas protegidas en `chi.Router`.

## 4. Validación 
- Hacer una petición con Postman/cURL sin el encabezado -> Debe retornar `401 Unauthorized`.
- Hacer una petición con un token inventado -> Debe retornar `401 Unauthorized`.
- Hacer una petición usando el cliente `api.js` desde Svelte -> Debe retornar `200 OK` y el servidor debe ser capaz de imprimir el ID del usuario.
