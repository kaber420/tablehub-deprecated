# Plan: Sección "Clientes" del SaaS (Panel de Gestión B2B)

> Estado: CONCEPTO / BORRADOR. El objetivo de este documento es fijar el concepto de la
> sección de Clientes del SaaS y desglosarlo en fases incrementales para construirla poco a poco.
> No es una especificación cerrada: se va afinando fase a fase.

---

## 1. Concepto (qué es y qué NO es)

**La Sección "Clientes"** es el área del SaaS donde el *personal de la plataforma*
(`platform_admin`, `support`, `billing`) gestiona a los **clientes de Tablehub**:
las organizaciones (restaurantes/cadenas) que contratan el servicio.

| | Sección "Clientes" (este plan) | Dashboard del Cliente (`ClientDashboard`) |
|---|---|---|
| Audiencia | Staff de la plataforma (SaaS) | Dueño/admin/viewer de una org |
| Pregunta | "¿Qué clientes tengo y cómo van?" | "¿Cómo va MI restaurante?" |
| Alcance | Todas las orgs (global) | Una sola org (`organization_id`) |
| Datos | Agregados, multi-tenant | Aislados por tenant |

**NO es** el dashboard que ve el cliente final (`ClientDashboard.svelte`). Es la vista
B2B de administración de cuentas. Ya existe un esbozo en `cloud/web/src/pages/Organizations.svelte`
(43 KB, con pestañas `organizations` y `plans_admin` y datos hardcodeados) que sirve como
prototipo visual inicial del concepto.

---

## 2. Audiencia y permisos (basado en `plan_roles_y_cliente_dashboard.md`)

- `platform_admin`: control total (crear, editar, suspender, ver facturación). Permiso `clients:view` + `global:manage`.
- `support`: solo lectura de clientes. Permiso `clients:view`.
- `billing`: solo facturación/planes. Permiso `billing:view`.

> Nota: el `rolePermissions` actual en `cloud/internal/interfaces/rest/middleware.go`
> todavía usa `clients:move` (legacy) y `IsSuperadmin()`. Este plan asume que primero se
> aplica `plan_roles_y_cliente_dashboard.md` (renombrado a `clients:view` y roles de staff).

---

## 3. Modelo de la sección (vistas del concepto)

```
/seccion-clientes
├── Lista de Clientes          (tabla maestra de orgs)
│    ├─ Nombre / Slug
│    ├─ Plan (Free / Plus+)
│    ├─ # Hubs / # Sucursales
│    ├─ Estado (activo / suspendido / trial)
│    └─ Creado / Última actividad
└── Detalle de Cliente {orgID}
     ├─ General      (datos de la org, plan, límites)
     ├─ Hubs         (dispositivos de la org)         ← reusa /v1/orgs/{id}/hubs
     ├─ Sucursales   (branches)                       ← reusa /v1/orgs/{id}/branches
     ├─ Equipo       (usuarios de la org)             ← reusa /v1/orgs/{id}/users
     ├─ Facturación  (plan, licencias Plus+ acumulativas)
     └─ Eventos/Logs (webhooks, actividad)            ← reusa /v1/webhooks
```

El detalle puede ser un **drawer** o página, reutilizando los componentes ya existentes
(`Devices`, `Users`, `Branch`) filtrados por `orgID` en vez de por el `profile.organization_id`
del cliente.

---

## 4. Backend necesario (lo que falta hoy)

Hoy `Organizations.svelte` usa un array hardcodeado (`organizations = [...]`). No hay
endpoints de listado/detalle de clientes a nivel SaaS. Se requiere:

| Endpoint | Método | Descripción | Estado |
|---|---|---|---|
| `/v1/admin/organizations` | GET | Listar todas las orgs (staff global) | FALTA |
| `/v1/admin/organizations/{id}` | GET | Detalle de una org + stats agregadas | FALTA |
| `/v1/admin/organizations/{id}/stats` | GET | Hubs activos, dispositivos, sucursales | FALTA |
| `/v1/admin/organizations` | POST | Crear cliente manualmente (onboarding staff) | FALTA |
| `/v1/admin/organizations/{id}/suspend` | POST | Suspender/reactivar | FALTA |

Dominio ya disponible: `internal/domain/organization` (repo con `List`, `GetByID`, `Create`,
`Update`, `Delete`), y `MockOrganizationRepository` en los tests.

---

## 5. Fases incrementales (ir dándole forma)

### Fase 0 — Concepto
Fijar visión, audiencia, modelo de vistas y backend necesario. ✅

### Fase 0.5 — Migrar roles (Prerrequisito)
- Aplicar DB migration `010_roles_redesign.sql`.
- Actualizar `middleware.go` (reemplazar `IsSuperadmin` por permisos granulares como `clients:view`).
- Actualizar frontend `App.svelte` para usar `STAFF_ROLES` en lugar de `is_superadmin`.
- Eliminar fallbacks silenciosos a datos mock en componentes de staff para evitar ocultar errores de permisos.

### Fase 1 — Maqueta visual con datos mock (prototipo robusto)
- Definir tipos/interfaces de API (`ClientListResponse`, `ClientDetail`) aunque se use un mock.
- Refinar `Organizations.svelte` (actualmente +1000 líneas) extrayéndolo en una `ClientTable` y separar el fixture de datos a `lib/mock/clients.js`.
- Incluir paginación, filtros básicos y ordenamiento en la tabla, operando sobre datos mock.
- Extraer `BranchTable.svelte` (actualmente inlined) para poder reusarlo en el detalle.
- Añadir un "Detalle de Cliente" (Drawer) simulando carga asíncrona por tab y estados vacíos/error.
- Criterio: Se puede navegar la lista paginada y ver los detalles mockeados.

### Fase 2 — Endpoint de listado real
- Crear `admin_handler.go`: `ListOrganizations` con soporte real para paginación y filtros.
- Registrar ruta en `router.go` bajo `/v1/admin/...`.
- Conectar `ClientTable` a `GET /v1/admin/organizations` vía `api.js`.
- Criterio: La tabla se llena desde la DB, reemplazando el mock.

### Fase 3 — Detalle de cliente real
- Endpoint `GET /v1/admin/organizations/{id}` + `/stats`.
- Implementar patrón Wrapper (ej. `AdminHubs.svelte` usando `Devices.svelte`) para añadir acciones de admin sin ensuciar la vista de cliente.
- Conectar tabs del Drawer a la API.
- Criterio: Al hacer clic en un cliente se ven sus datos reales aislados por orgID.

### Fase 4 — Planes y licencias Plus+
- Pestaña "Facturación/Plan" en el detalle.
- Modelar `Plan` (Free / Plus+) y mover licencias acumulativas que actualmente estaban inlined.
- Criterio: Se puede ver y cambiar el plan de un cliente.

### Fase 5 — Acciones de gestión
- Crear cliente, suspender/reactivar, reset de credenciales.
- Confirmaciones y estados de carga/error en la UI.
- Criterio: Flujo completo de alta y suspensión funcional.

---

## 6. Criterios de aceptación general
1. Solo staff con `clients:view` ve la sección; un cliente normal recibe `<Unauthorized />`.
2. La lista maneja paginación real para soportar gran volumen de orgs.
3. La navegación es coherente con el `Dock` existente ("Clientes y Orgs").
4. Sin romper el aislamiento multi-tenant: componentes base no tienen condicionales de admin acoplados.

## 7. Archivos probables a tocar
- `cloud/web/src/pages/Organizations.svelte` (refactor masivo a `ClientTable`)
- `cloud/web/src/components/AdminHubs.svelte`, `BranchTable.svelte` (nuevos/extraídos)
- `cloud/web/src/lib/Dock.svelte`
- `cloud/web/src/lib/api.js`, `cloud/web/src/lib/mock/clients.js`
- `cloud/internal/interfaces/rest/admin_handler.go`
- `cloud/internal/interfaces/rest/router.go`
- `cloud/internal/interfaces/rest/middleware.go` (migración de roles)
- `cloud/web/src/App.svelte` (migración de superadmin)
