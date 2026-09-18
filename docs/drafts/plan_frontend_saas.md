# Plan de Implementación Final: Frontend del SaaS (Tablehub Cloud)

## 1. Visión General
El objetivo es construir la interfaz web del SaaS. Para mantener consistencia con el ecosistema y ahorrar tiempo de desarrollo, **reutilizaremos la arquitectura CSS nativa (Glassmorphism/Neumorphism)** y los componentes Svelte que ya se construyeron exitosamente en el proyecto local (`hub/web`).

## 2. Decisiones de Arquitectura Confirmadas
- **Ubicación:** `cloud/web` (Simetría con el hub).
- **Framework:** Svelte + Vite.
- **Estilos:** CSS Nativo. **No se usará TailwindCSS ni librerías externas.** Se copiará directamente el archivo `app.css` del Hub para heredar el sistema de temas (`theme-glass` y `theme-neumorph`).

## 3. Pasos de Implementación (Fase 1: Setup)

**Paso 1: Inicialización del Proyecto**
- Ejecutar: `pnpm create vite web --template svelte` dentro de la carpeta `cloud/`.
- Entrar a la carpeta generada e instalar dependencias: `pnpm install`.

**Paso 2: Limpieza y Trasplante de Estilos**
- Eliminar el CSS por defecto generado por Vite.
- Copiar el archivo `/hub/web/src/app.css` y pegarlo en `/cloud/web/src/app.css`.
- Modificar `/cloud/web/src/main.js` (o equivalente) para importar `app.css`.
- Modificar `/cloud/web/src/App.svelte` para inyectar la clase del tema base en el documento (ej. `document.body.className = 'theme-glass';`).

**Paso 3: Verificación**
- Ejecutar `pnpm run dev` en `cloud/web`.
- Confirmar visualmente en el navegador que el fondo oscuro translúcido y las tipografías correspondan a la estética premium del Hub.

## 4. Siguientes Fases (Pendientes)
Una vez completado el Setup, las siguientes fases serán:
- Crear vistas de Login y Registro.
- Conectar el Frontend con nuestro servidor Zitadel para autenticación OIDC.
- Construir el Dashboard y la Gestión de Hubs (generación de `.thub`).
