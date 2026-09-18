Hola Mimo.

El usuario de Tablehub nos ha pedido que revises el plan de maquetado (mockup/layout) de la
sección de Clientes/Customers del SaaS. Es un plan nuevo de concepto que se irá construyendo
por fases. Acabamos de crearlo en `drafts/plan_seccion_clientes_saas.md`.

Aquí tienes el borrador completo de `drafts/plan_seccion_clientes_saas.md`:

```markdown
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

### Fase 0 — Concepto (este documento)
Fijar visión, audiencia, modelo de vistas y backend necesario. ✅

### Fase 1 — Maqueta visual con datos mock (prototipo)
- Refinar `Organizations.svelte` como la "Lista de Clientes" del concepto.
- Extraer los datos hardcodeados a un store/fixture claro (`lib/mock/clients.js`).
- Definir la grilla de columnas y el componente `ClientRow` / `ClientTable`.
- Añadir placeholder de "Detalle" (drawer vacío con tabs).
- Criterio: se ve y navega la sección aunque los datos sean ficticios.

### Fase 2 — Endpoint de listado real
- Crear `admin_handler.go`: `ListOrganizations` (requiere `clients:view` + scope global).
- Registrar ruta en `router.go` bajo `/v1/admin/...`.
- Conectar `Organizations.svelte` a `GET /v1/admin/organizations` vía `api.js`.
- Criterio: la tabla se llena desde la DB (o mock repo), no de fixtures.

### Fase 3 — Detalle de cliente
- Endpoint `GET /v1/admin/organizations/{id}` + `/stats`.
- Drawer/página de detalle con tabs General / Hubs / Sucursales / Equipo.
- Reusar componentes existentes pasando `orgID` explícito.
- Criterio: al hacer click en un cliente se ven sus datos reales.

### Fase 4 — Planes y licencias Plus+
- Pestaña "Facturación/Plan" en el detalle y la pestaña `plans_admin` ya esbozada.
- Modelar `Plan` (Free / Plus+) y licencias acumulativas por org.
- Criterio: se puede ver y (si aplica) cambiar el plan de un cliente.

### Fase 5 — Acciones de gestión
- Crear cliente, suspender/reactivar, reset de credenciales.
- Confirmaciones y estados de carga/error en la UI.
- Criterio: flujo completo de alta y suspensión de un cliente.

---

## 6. Criterios de aceptación general
1. Solo staff con `clients:view` (scope global) ve la sección; un cliente normal recibe `<Unauthorized />`.
2. La lista y el detalle vienen del backend, no de fixtures (post Fase 2/3).
3. La navegación es coherente con el `Dock` existente ("Clientes y Orgs").
4. Sin romper el aislamiento multi-tenant de las vistas del cliente final.

## 7. Archivos probables a tocar
- `cloud/web/src/pages/Organizations.svelte` (refactor a "Lista de Clientes")
- `cloud/web/src/lib/Dock.svelte` (ya tiene el item "Clientes y Orgs")
- `cloud/web/src/lib/api.js`, `cloud/web/src/lib/mock/clients.js` (nuevo)
- `cloud/internal/interfaces/rest/admin_handler.go` (nuevo)
- `cloud/internal/interfaces/rest/router.go` (registrar rutas)
- `cloud/internal/interfaces/rest/middleware.go` (permisos `clients:view`)
```

Por favor, revisa este borrador y danos tus sugerencias. Queremos identificar:
1. **Fallas de diseño o de maquetado (Fase 1)** en la propuesta de la "Lista de Clientes" y el
   "Detalle de Cliente": ¿la estructura de tabs/drawer es la correcta? ¿Qué columnas/KPIs faltan
   para una vista B2B útil de gestión de clientes?
2. **Reutilización de componentes:** ¿Es sano reusar `Devices.svelte`, `Users.svelte`,
   `Branch` filtrados por `orgID` explícito, o conviene componentes dedicados? ¿Cómo evitar
   acoplar la vista de staff con la del cliente final?
3. **Separación SaaS-staff vs cliente:** ¿El modelo de permisos (`clients:view` + scope global)
   y la distinción por rol son suficientes y no introducen fugas de tenant?
4. **Robustez incremental:** ¿El orden de fases (mock → endpoint → detalle → planes → acciones)
   es el adecuado para ir dándole forma sin rework? ¿Qué falta antes de empezar la Fase 1?
