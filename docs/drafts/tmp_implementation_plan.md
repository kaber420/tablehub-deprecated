# Plan: Arquitectura de Separación de Roles (SaaS vs. Clientes) y Paneles de Control

Este plan detalla la arquitectura de roles del sistema, separando claramente los **Roles de Plataforma SaaS (SaaS Staff / Admins)** de los **Roles de Clientes (Owners / Admins / Viewers en Organizaciones)** en la base de datos, en Logto y en la interfaz de usuario, incorporando consideraciones avanzadas de seguridad y aislamiento (multi-tenancy).

---

## Roles y Niveles de Acceso Propuestos

### 1. Nivel SaaS (Gestión Global del Negocio)
Administran toda la plataforma Tablehub, las licencias de clientes y el estado global. No pertenecen a una única organización cliente.
- **SaaS Admin (Superadmin)**: Control total sobre clientes, facturación, planes y hubs.
- **SaaS Staff**: Soporte y monitoreo, visualización de clientes/organizaciones sin permisos destructivos de facturación o borrado (Solo Lectura / Soporte).

### 2. Nivel Cliente (Gestión de la Franquicia / Organización)
Operan únicamente dentro del contexto de su organización (`OrganizationID`).
- **Client Owner**: Propietario de la organización (el cliente que paga). Puede registrar hubs, sucursales y staff.
- **Client Admin**: Administrador local (gerente de sucursal). Gestiona dispositivos y solicitudes asignadas a sus sucursales.
- **Client Viewer**: Personal de piso (meseros). Solo visualiza estados locales sin permisos administrativos.

---

## User Review Required

> [!WARNING]
> **Gestión de Permisos Cruzados:** En este diseño, un usuario puede poseer múltiples roles simultáneamente (e.g., ser `SaaS Staff` pero también poseer un `Client Profile` si tiene su propia organización). Revisa si esta flexibilidad es el comportamiento deseado.

> [!IMPORTANT]
> **Mapeo de Scopes de Logto:** Es fundamental definir explícitamente qué scopes de Logto mapean a qué operaciones internas (`LogtoRole -> InternalPermission[]`) en el middleware.

---

## Cambios Propuestos

### A. Base de Datos y Modelo de Go (Backend)

#### [MODIFY] [user.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/domain/user/user.go)
- Definir un modelo unificado `UserRole` que incluya tanto el acceso global (SaaS) como los múltiples roles a nivel de organización.
- Almacenar y mantener sincronizado el estado de los roles en la base de datos en cada login, para no depender exclusivamente de `claims` de tokens de Logto que pueden quedar obsoletos antes de su expiración.

#### [MODIFY] [middleware.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/interfaces/rest/middleware.go)
- **RequireSaaSAdmin / RequireSaaSStaff**: Validar los scopes granulares de Logto para asegurar que SaaS Staff solo tenga permisos de lectura, mientras que SaaS Admin tenga lectura y escritura.
- **RequireOrgAccess**: Validar el acceso al `OrganizationID`. 
- **[NEW] TenantIsolation Middleware**: Añadir un middleware explícito de multi-tenancy para garantizar que todas las consultas a base de datos de nivel cliente estén filtradas por `OrganizationID`.

#### [MODIFY] [onboarding.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/interfaces/rest/onboarding.go)
- En `/v1/me` (GetMe), retornar una respuesta con un objeto que soporte roles múltiples:
  ```json
  {
    "registered": true,
    "roles": {
      "saas": ["staff"],
      "clients": [
        {
          "organization_id": "uuid-1",
          "role": "owner"
        }
      ]
    }
  }
  ```

#### [NEW] [audit.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/domain/audit/audit.go)
- Implementar un registro de auditoría (`Audit Log`) para grabar todas las acciones destructivas o cambios críticos realizados por `SaaS Admin` y `Client Owner`.

#### [NEW] [revocation.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/application/auth/revocation.go)
- Crear endpoints/jobs para invalidar activamente tokens y sesiones cuando un administrador revoca accesos de un usuario (para bajas de usuarios inmediatas sin esperar expiración del JWT).

---

### B. Frontend (Svelte)

#### [MODIFY] [Dock.svelte](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/web/src/lib/Dock.svelte)
- Modificar el renderizado de la barra de navegación basada en el array `roles` obtenido de `/v1/me`. Si el usuario tiene acceso `saas` y perfiles `clients`, mostrar un selector de contexto ("Modo Admin / Modo Cliente") o fusionar navegaciones de forma controlada.

#### [MODIFY] [App.svelte](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/web/src/App.svelte)
- Modificar el renderizado de vistas según los accesos, utilizando `<SaaSDashboard />` y `<ClientDashboard />`.
- **[NEW] `<Unauthorized />` Component**: Implementar una vista de fallback para accesos denegados, junto con redirección automática si el usuario entra a una URL donde no tiene el rol necesario.

---

## Plan de Verificación

### Pruebas de Acceso de Roles (Seguridad) y Multi-tenancy
1. **Acceso Cruzado (Tenant Isolation)**: Comprobar que un `Client Owner` de la Org A reciba un `403 Forbidden` o `404 Not Found` si intenta acceder vía API a recursos (mesas, hubs) de la Org B.
2. **SaaS Staff Restringido**: Iniciar sesión con un usuario SaaS Staff y verificar que **no** puede realizar peticiones POST/PUT/DELETE que afecten la facturación o la estructura de un cliente (esperar `403 Forbidden`), solo peticiones GET.
3. **Escenario Híbrido**: Acceder con una cuenta que tiene el rol `SaaS Staff` y también posee un `Client Profile`. Comprobar que el usuario puede alternar contextos sin filtraciones de datos.
4. **Expiración y Revocación**: Eliminar el rol a un usuario activo desde Logto o la DB, y verificar que el backend rechaza inmediatamente sus peticiones siguientes gracias al sistema de sincronización/revocación, sin esperar a que su token original expire.
5. **Comportamiento Frontend**: Navegar a una ruta protegida (ej: `/saas/billing`) con un usuario cliente normal. Comprobar que el componente `<Unauthorized />` es mostrado y que redirecciona correctamente.
