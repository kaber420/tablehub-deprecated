# Review del Plan "Sección Clientes" — DeepSeek (Senior Dev)

Basado en la lectura de `drafts/plan_seccion_clientes_saas.md` y el análisis del código base actual.

---

## 1. Fallas de diseño / maquetado (Fase 1)

### Estructura general

La propuesta Lista + Detalle (drawer) es correcta para un panel B2B de gestión de cuentas. Sin embargo, el borrador omite aspectos críticos de UX para una tabla maestra:

**Lo que falta en la "Lista de Clientes":**

| Aspecto | Problema | Propuesta |
|---|---|---|
| **Paginación / scroll virtual** | Una tabla SaaS real puede tener cientos o miles de orgs. Sin paginación la Fase 2 (endpoint real) será inservible. | Incluir paginación desde Fase 1 con datos mock (ej. 20 por página, controles de página). El endpoint debe devolver `{ data, total, page, per_page }`. |
| **Filtros** | Solo se menciona búsqueda textual. Un staff necesita filtrar por plan (`Free`/`Plus+`), estado (`active`/`suspendido`/`trial`), rango de fechas, # de hubs. | Diseñar barra de filtros en Fase 1 aunque los controles arranquen deshabilitados. |
| **Ordenamiento** | No se menciona orden por columnas. | Cada columna de la tabla debe ser sorteable (nombre, plan, fecha, estado). |
| **KPIs faltantes** | Las columnas propuestas son correctas pero insuficientes para un panel B2B. | Añadir: `MRR estimado` (si aplica), `Licencias Plus+ usadas / totales`, `Última actividad` (timestamp legible), `Versión del firmware` (si aplica a hubs). |
| **Estados vacío y error** | No se mencionan. | Fase 1 debe incluir estados: `loading`, `empty` (sin clientes), `error` (fallo de red). |

**Drawer vs Página independiente:**

El borrador dice "drawer o página" sin definirlo. Para Fase 1 recomiendo **drawer** por rapidez, pero con una advertencia: el detalle de cliente tiene 6 tabs con contenido potencialmente pesado (Hubs, Sucursales, Equipo). Si cada tab hace una llamada API distinta, un drawer puede sentirse lento y desordenado. Sugerencia: arrancar con drawer en Fase 1 (placeholder), pero planificar migración a página dedicada en Fase 3 si la carga de datos lo justifica.

**Las tabs del detalle:**

- `General` (datos de org, plan, límites) — bien.
- `Hubs` — reusa Devices.svelte filtrado por orgID. Bien.
- `Sucursales` — ojo: no existe `Branch.svelte` como componente independiente. Está inlined en `Organizations.svelte`. Habrá que extraerlo.
- `Equipo` — reusa Users.svelte filtrado por orgID. Bien.
- `Facturación` — ok para Fase 4.
- `Eventos/Logs` — bien, pero ¿qué eventos? Definir si son webhooks, actividad de admin, o ambas.

### Problema con la Fase 1 actual

`Organizations.svelte` tiene **1095 líneas** con datos mock inlined, lógica de expansión de filas, y un tab `plans_admin` que no está en el concepto. Refactorizar esto como "Lista de Clientes" implica:
- Extraer `plans_admin` a otro lado o eliminarlo del MVP.
- Separar datos mock a un fixture.
- Reemplazar la expansión inline de filas por un `ClientTable` + `ClientRow`.

**Riesgo:** Subestimar el esfuerzo de refactor. Organizations.svelte no es un archivo nuevo, es uno existente de 1K+ líneas con lógica acoplada.

---

## 2. Reutilización de componentes

### ¿Es sano reusar Devices.svelte, Users.svelte, Branch por orgID?

**Sí, con matices.**

Ambos componentes (`Devices.svelte`, `Users.svelte`) ya aceptan `export let orgID` y están diseñados para funcionar con un orgID arbitrario. En la práctica ya son agnósticos al tenant. Eso es bueno.

**Riesgo de acoplamiento staff-cliente:**

Si el staff y el cliente final usan exactamente el mismo componente, cualquier cambio necesario para el staff (ej. botón "Forzar reconexión" en Hubs, o "Eliminar usuario" en Equipo) se filtrará a la vista del cliente a menos que se oculte condicionalmente. Eso lleva a componentes llenos de `{#if isStaff}` que son difíciles de mantener.

**Recomendación:** Usar una **capa de adaptador** o **wrapper**:

```
AdminHubs.svelte
  └── <Devices orgID={orgID} />
       └── (slot "actions") → botones de admin (suspender, forzar reconexión)
```

O usar un patrón de **componente base + extensiones**:

- `HubList.svelte` (lógica común: fetch, filtros, búsqueda) — sin acciones administrativas.
- `ClientHubList.svelte` — reusa `HubList`.
- `AdminHubList.svelte` — reusa `HubList` + añade columna de acciones staff.

Esto evita condicionales `isStaff` en los componentes base y mantiene la separación de concerns.

**Caso especial: Branch.svelte**

No existe como componente. La tabla de sucursales está inlined en `Organizations.svelte`. La Fase 1 debería incluir la extracción de `BranchTable.svelte` como prerrequisito, no como después.

**Conclusión:** Reutilizar es correcto siempre que se haga con wraers y que el componente base no contenga lógica de staff. El plan actual no menciona esta estrategia de wrapping, lo cual es una omisión.

---

## 3. Separación SaaS-staff vs cliente

### Modelo de permisos: `clients:view` + scope global

El concepto es sólido. El dominio `user.go` ya tiene `IsStaffRole()` y los roles `platform_admin`, `support`, `billing`. Eso está bien.

**Problemas identificados:**

1. **Middleware.go no migrado.** Sigue usando `RoleSuperadmin` y `IsSuperadmin()`. El plan asume que `plan_roles_y_cliente_dashboard.md` se aplica primero, pero no hay un orden explícito. **Si se empieza la Fase 1 sin migrar el middleware, el gating de la sección seguirá siendo `is_superadmin`** y habrá que rehacerlo después.

2. **El frontend App.svelte usa `profile.is_superadmin`.** El plan de roles propone migrar a `STAFF_ROLES.includes(profile.role)`. Si no se hace antes de Fase 1, el nuevo "Clientes" solo lo verán superadmins legacy, no los nuevos `support` o `billing`.

3. **Fuga de tenant potencial:** Los componentes reusados (`Devices.svelte`, `Users.svelte`) tienen un patrón de _fallback a mock data_. Si el endpoint `/v1/admin/organizations/{id}/hubs` falla (porque el middleware rechaza el scope global), el componente caerá a mock data y podría mostrar datos incorrectos. Este comportamiento silencioso es peligroso. **El fallback a mock solo debería existir en development, o eliminarse para las vistas de staff.**

4. **¿Puede un staff ver su propio "ClientDashboard"?** Un `platform_admin` también es dueño de una org en desarrollo local. El sistema de roles debe asegurar que staff NO vea `ClientDashboard` sino siempre la vista staff, o que pueda alternar. El plan de roles actual no cubre este edge case.

**Recomendación:** Migrar middleware y frontend al nuevo sistema de roles (`clients:view`, `STAFF_ROLES`) como **Fase 0.5** — antes de tocar Organizations.svelte. De lo contrario, Fase 1 construirá sobre una base que será refactorizada en el mismo hito.

---

## 4. Robustez incremental

### ¿El orden de fases es adecuado?

**En general sí:** mock → endpoint → detalle → planes → acciones es el orden lógico. Sin embargo:

### Lo que falta antes de empezar Fase 1

| Prerrequisito | Razón |
|---|---|
| **Migración de roles (middleware + frontend)** | Si no, se construye sobre `is_superadmin` y hay rework. |
| **Definir el contrato API** (tipos/interface) | Aunque los datos sean mock en Fase 1, tener los tipos definidos (`ClientListResponse`, `ClientDetail`) evita cambiar la estructura de la tabla en Fase 2. |
| **Extraer BranchTable.svelte** | Es necesario para la tab "Sucursales". Si se pospone a Fase 3, se descubre que no hay componente que reutilizar. |
| **Especificar el comportamiento offline/error** | ¿Qué pasa si el API falla en Fase 2? ¿Mostrar error, reintentar, mock? Definirlo antes. |

### Riesgos de rework entre fases

- **Fase 1 → Fase 2:** Si la tabla en Fase 1 no usa paginación (porque los datos mock son 3 filas), al conectar el endpoint real en Fase 2 habrá que rehacer la tabla. **Mitigación:** Incluir paginación desde Fase 1 aunque sobre datos mock.
- **Fase 1 → Fase 3:** Si el drawer se implementa como "solo maqueta" sin estado real, en Fase 3 se descubre que la estructura del drawer no soporta carga asíncrona por tab. **Mitigación:** Simular carga asíncrona en Fase 1 (timeout + mock data).
- **Fase 4 (Planes):** Las licencias Plus+ acumulativas ya existen en el mock de Organizations.svelte (`plus_licenses`, `getGatewaysLimit()`, etc.). Si no se trasladan al fixture en Fase 1, hay que extraerlas después.

### Lo que falta en el plan

1. **Estrategia de testing.** No se mencionan tests. Para un panel B2B que gestiona datos sensibles, cada fase debería incluir tests mínimos (ej. "staff con `clients:view` ve la lista", "cliente normal ve 403").
2. **Feature flags.** Considerar si la sección entera debería estar detrás de un feature flag para desplegar en producción sin activarla.
3. **Rendimiento.** La tabla de clientes en Fase 2 podría devolver 10K orgs. Sin paginación server-side, el frontend colapsa. Especificar el contrato de paginación desde Fase 1.

### Orden propuesto (corregido)

```
Fase 0.5 — Migrar roles (plan_roles_y_cliente_dashboard.md)
  ├── Aplicar DB migration 010_roles_redesign.sql
  ├── Actualizar middleware.go (reemplazar IsSuperadmin por clients:view)
  ├── Actualizar App.svelte (is_superadmin → STAFF_ROLES)
  └── Eliminar fallback a mock en vistas de staff

Fase 1 — Maqueta con datos mock (pero con paginación y tipos definidos)
  ├── Definir tipos/interfaces de API (aunque no haya endpoint)
  ├── Extraer Organizations.svelte → ClientTable + fixture
  ├── Extraer BranchTable.svelte (necesario para detalle)
  ├── Implementar paginación, filtros, sorting (mock)
  ├── Drawer de detalle con tabs y carga simulada
  └── Estados: loading, empty, error, unauthorized

Fase 2 — Endpoint de listado real (idem plan original)

Fase 3 — Detalle de cliente real (idem plan original)

Fase 4 — Planes y licencias (idem plan original)

Fase 5 — Acciones de gestión (idem plan original)
```

---

## Resumen de hallazgos críticos

| # | Hallazgo | Prioridad |
|---|---|---|
| 1 | Falta paginación y filtros en la tabla maestra desde Fase 1 | Alta |
| 2 | Organizations.svelte (1095 líneas) requiere refactor mayor, no menor | Alta |
| 3 | Migración de roles debe hacerse ANTES de Fase 1 o habrá rework | Alta |
| 4 | El fallback a mock data en componentes puede ocultar errores de permiso en staff | Alta |
| 5 | Falta definir estrategia de wrapping para reutilizar componentes sin acoplar staff/cliente | Media |
| 6 | Branch.svelte no existe como componente independiente; hay que extraerlo | Media |
| 7 | No se mencionan tests, loading states, ni empty states | Media |
| 8 | El contrato API debe definirse en Fase 1 aunque los datos sean mock | Media |
| 9 | No hay feature flag para desplegar la sección gradualmente | Baja |
| 10 | El drawer vs página debe decidirse antes de Fase 3 | Baja |
