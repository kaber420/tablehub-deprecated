# Plan de Implementación: Login OIDC con Zitadel en Svelte

## 1. Visión General
El objetivo de esta fase es conectar nuestro frontend Svelte (`cloud/web`) con nuestra instancia local de **Zitadel** utilizando el flujo estándar y seguro de la industria: **OIDC Authorization Code Flow con PKCE** (Proof Key for Code Exchange), diseñado específicamente para Single Page Applications (SPAs).

Al usar este flujo, no necesitamos construir pantallas de contraseñas complejas. El usuario hará clic en "Iniciar Sesión", será redirigido a la interfaz segura de Zitadel, y luego devuelto a nuestro SaaS con sus tokens de acceso listos para consumir el backend en Go.

## 2. Decisiones de Arquitectura

1. **Librería OIDC:** Usaremos `oidc-client-ts` (la evolución de `oidc-client-js`). Es la librería más madura y estándar para manejar flujos PKCE, tokens, renovaciones silenciosas y redirecciones en TypeScript/Vanilla JS, siendo perfectamente compatible con Svelte.
2. **Almacenamiento de Tokens:** `oidc-client-ts` gestionará los tokens en memoria o en `sessionStorage` (configurable) para mitigar riesgos de robo por ataques XSS en comparación con `localStorage`.
3. **Flujo de Vistas:**
   - **Vista Pública (Login):** Una pantalla de bienvenida con un gran botón "Iniciar Sesión en Tablehub Cloud" (usando tus estilos premium).
   - **Callback de Redirección (`/callback`):** Una ruta temporal e invisible que captura el token que envía Zitadel en la URL y autentica la sesión.
   - **Vista Protegida (Dashboard):** A donde se redirige al dueño del restaurante una vez autenticado, mostrando el estado de sus Hubs.

## 3. Pasos de Implementación

### Paso 1: Configuración en la consola de Zitadel
Antes de programar, necesitaremos que levantes Zitadel en Docker e ingreses a su consola web para crear una "Aplicación" de tipo "Single Page Application (SPA)".
- **Redirect URI:** `http://localhost:5173/callback` (Ruta de Vite).
- **Post Logout URI:** `http://localhost:5173/`
- Obtendremos el **Client ID** de esta aplicación.

### Paso 2: Instalación de Dependencias
- Ejecutaremos en `cloud/web`: `pnpm install oidc-client-ts` y un router ligero (ej. `svelte-routing` o `tinro` si no queremos complicarnos con vistas manuales) para manejar la ruta del callback.

### Paso 3: Servicio de Autenticación (Svelte)
- Crear `cloud/web/src/lib/auth.js`.
- Configurar el cliente `UserManager` de `oidc-client-ts` con la autoridad de Zitadel (ej. `http://localhost:8080`), el Client ID y las URIs de redirección.
- Exportar un Svelte Store reactivo (ej. `isAuthenticated`, `userProfile`) para que toda la aplicación sepa al instante si el usuario está logueado.

### Paso 4: Creación de Componentes
1. **`Login.svelte`**: Botón que llama a `userManager.signinRedirect()`.
2. **`Callback.svelte`**: Componente de carga que llama a `userManager.signinRedirectCallback()` y luego redirige al Dashboard.
3. **`Dashboard.svelte`**: Protegido. Mostrará los datos del usuario logueado (ej. Email) y tendrá el botón de cerrar sesión.
4. **`App.svelte`**: Actuará como el enrutador principal decidiendo qué componente mostrar según la URL.

### Paso 5: Interceptor de Peticiones (Opcional en esta fase)
- Crear una función para agregar el Access Token al encabezado `Authorization: Bearer <token>` de las llamadas `fetch` que vayan hacia tu backend en Go.

## 4. Próximos Pasos
Si este plan te parece lógico y apruebas el uso de `oidc-client-ts`, dímelo y procederé a instalar las dependencias (Paso 2) y a escribir el servicio `auth.js` (Paso 3). Previamente, asegúrate de tener Zitadel corriendo y configurado con la aplicación SPA (Paso 1).
