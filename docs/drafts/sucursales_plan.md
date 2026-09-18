# Plan de Implementación: Módulo de Sucursales

Este documento detalla el plan para consolidar, revisar y extender el módulo de Sucursales (Branches), el cual es el núcleo organizativo de Tablehub (ya que los Hubs y Webhooks dependen de que exista una sucursal física).

## 1. Estado Actual (Auditoría)

Al revisar el código fuente, hemos detectado que la base del módulo **ya se encuentra implementada**:
- **Backend:** Existen los archivos `branch.go` (dominio), `branch_repo.go` (base de datos) y `branch_handler.go` (API REST). La ruta `GET/POST /v1/orgs/{orgID}/branches` ya está registrada y protegida con middlewares de permisos.
- **Frontend:** Existe el componente `Branches.svelte` (interfaz gráfica) que ya está conectado a `api.js` para crear y listar sucursales.

## 2. Lo que falta por construir (El Plan)

Dado que la funcionalidad básica (CRUD) existe, este plan se enfocará en hacer que el módulo sea robusto, seguro y se integre correctamente con el resto del sistema, en especial con el Dashboard.

### A. Validaciones y Seguridad Backend
- **Problema:** Actualmente `CreateBranch` solo valida que el nombre no esté vacío. No valida la longitud máxima, caracteres especiales, ni verifica si ya existe una sucursal con el mismo nombre en la misma organización.
- **Acción:** Implementar validaciones estrictas en `branch_handler.go` y manejar errores de duplicidad (Unique Constraint en base de datos).

### B. Mejoras en Frontend (`Branches.svelte`)
- **Problema:** La interfaz asume que la petición siempre es exitosa a menos que falle la red. Las alertas de error son genéricas (`alert()`).
- **Acción:** Reemplazar los `alert()` con notificaciones tipo "Toast" nativas del diseño UI de Tablehub. Añadir validación de formulario en tiempo real antes de enviar el POST.

### C. Integración con Dashboard (Métricas Reales)
- **Problema:** El dashboard del cliente muestra "0" en "Sucursales" y "Hubs Activos" (antes mostraba datos falsos hardcodeados).
- **Acción:** Crear un nuevo endpoint `GET /v1/orgs/{orgID}/stats` que haga un `COUNT` en las tablas `branches` y `hubs` para la organización actual. Conectar `ClientDashboard.svelte` a este nuevo endpoint para que el número de sucursales se actualice automáticamente al crear una nueva.

## 3. Plan de Ejecución (Paso a Paso)

1. **Paso 1: Endpoint de Estadísticas (Dashboard)**
   - Crear el método `GetOrgStats` en el backend para alimentar el Dashboard.
   - Conectar `ClientDashboard.svelte` al nuevo endpoint.
2. **Paso 2: Robustecer Backend de Sucursales**
   - Modificar `branch_handler.go` para añadir validaciones estrictas.
   - Asegurar que los errores se devuelvan en formato JSON estructurado.
3. **Paso 3: Pulir Frontend de Sucursales**
   - Eliminar `alert()` de Javascript.
   - Mejorar el feedback visual al crear una sucursal.
4. **Paso 4: Pruebas E2E**
   - Ejecutar la suite de pruebas E2E con el tag correspondiente para garantizar que la creación de sucursales no rompe la asignación de Hubs.
