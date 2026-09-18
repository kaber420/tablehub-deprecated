# Plan de Implementación: Automatización Completa del Setup de Logto y Generación de `.env`

Este plan detalla el diseño de un script de automatización en Go (`cloud/cmd/setup-logto/main.go`) que configura automáticamente tu instancia local de Logto directamente a través de la base de datos PostgreSQL. Este script registrará la aplicación SPA (Frontend) y M2M (Backend) en las tablas internas de Logto, generará las claves de acceso y creará de forma automática los archivos `.env` y `cloud/web/.env` correspondientes.

De esta forma, **no tendrás que realizar ninguna configuración manual en la interfaz de Logto**, pudiendo correr el sistema completo con OIDC real al instante.

---

## Cambios Propuestos

### 1. Nube: Herramienta de Automatización (Go)

#### [NEW] [main.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/cmd/setup-logto/main.go)
- Crear una herramienta CLI en Go que realice los siguientes pasos:
  1. Conectarse a la base de datos de Logto (`postgres://tablehub:tablehub@localhost:5432/logto?sslmode=disable`).
  2. Obtener el `tenant_id` actual de la tabla `tenants`.
  3. Comprobar si ya existen las aplicaciones `"Tablehub Web"` (SPA) y `"Tablehub Backend"` (M2M).
  4. Si no existen, generar IDs únicos de 21 caracteres e insertarlos en la tabla `applications` de Logto con la configuración OIDC necesaria:
     - **Tablehub Web (SPA):** Configurar `redirectUris` a `http://localhost:5173/auth/callback` y `postLogoutRedirectUris` a `http://localhost:5173/`.
     - **Tablehub Backend (M2M):** Registrar una clave/secreto aleatorio en la tabla `application_secrets`.
  5. Escribir automáticamente el archivo `cloud/.env` con la configuración del backend.
  6. Escribir automáticamente el archivo `cloud/web/.env` con la configuración del frontend.

---

### 2. Nube: Frontend Web (Svelte)

#### [MODIFY] [auth.js](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/web/src/lib/auth.js)
- Modificar el archivo para cargar dinámicamente las variables de configuración del frontend escritas en `.env`:
```javascript
import { UserManager, WebStorageStateStore } from "oidc-client-ts";
import { writable } from "svelte/store";

const authProvider = import.meta.env.VITE_AUTH_PROVIDER || "logto";
const authority = import.meta.env.VITE_OIDC_ISSUER || "http://localhost:3002/oidc";
const client_id = import.meta.env.VITE_OIDC_CLIENT_ID || "";

const scope = authProvider === "logto" 
    ? "openid profile email urn:logto:scope:organizations" 
    : "openid profile email";

const oidcConfig = {
    authority,
    client_id,
    redirect_uri: "http://localhost:5173/auth/callback",
    post_logout_redirect_uri: "http://localhost:5173/",
    response_type: "code",
    scope,
    userStore: new WebStorageStateStore({ store: window.sessionStorage }),
};

export const userManager = new UserManager(oidcConfig);
export const user = writable(null);
export const isAuthenticated = writable(false);

export async function login() {
    await userManager.signinRedirect();
}

export async function logout() {
    await userManager.signoutRedirect();
}

export async function initAuth() {
    try {
        const loggedInUser = await userManager.getUser();
        if (loggedInUser && !loggedInUser.expired) {
            user.set(loggedInUser);
            isAuthenticated.set(true);
        }
    } catch (e) {
        console.error("Auth init error:", e);
    }
}
```

---

## Plan de Ejecución y Verificación

### Paso 1: Ejecutar la Automatización
1. Asegúrate de tener los contenedores levantados (`docker compose up -d` en `cloud/`).
2. Ejecutar la herramienta de configuración automática:
   ```bash
   go run ./cloud/cmd/setup-logto/main.go
   ```
3. El script generará los archivos `cloud/.env` y `cloud/web/.env` con las credenciales y configuraciones correctas en base a lo que se inyectó en la base de datos de Logto.

### Paso 2: Iniciar Servidores
1. Levantar el backend de Go:
   ```bash
   # En la carpeta cloud/
   make dev
   ```
2. Levantar el frontend de Svelte:
   ```bash
   # En la carpeta cloud/web/
   pnpm run dev
   ```

### Paso 3: Probar el Inicio de Sesión Real
1. Abrir `http://localhost:5173` en el navegador.
2. Hacer clic en **"Iniciar Sesión con Identity Provider"**.
3. Se redirigirá a la pantalla real de Logto (`localhost:3002`) para iniciar sesión con tus credenciales y acceder al Dashboard de la Nube para registrar tu Hub.
