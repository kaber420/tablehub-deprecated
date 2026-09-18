# Plan para Automatización de Organizaciones en Onboarding (SaaS Zero-Touch)

Este plan detalla los cambios para automatizar la creación de la organización en el proveedor de identidad (Logto, Zitadel o Mock) durante el registro del inquilino (onboarding), resolviendo la compensación ante fallos (rollback) y la asignación dinámica de roles por organización.

## User Review Required

> [!IMPORTANT]
> - **Columna de Base de Datos**: Añadiremos la columna `provider_org_id TEXT NULL UNIQUE` en la tabla `organizations` para registrar el ID que el proveedor de identidad asigne a cada organización. Se usará `sql.NullString` para manejar correctamente los valores nulos en el escaneo de registros antiguos.
> - **Actualización del Puerto**:
>   - Ampliaremos la interfaz `user.IdentityProvider` para soportar `CreateOrganization` y `DeleteOrganization` (para rollback en caso de fallos locales).
>   - Modificaremos la firma de `AssignRole` para aceptar la organización de manera dinámica: `AssignRole(ctx, userID, orgID, role)`.

## Open Questions

Ninguna.

## Proposed Changes

---

### Capa de Dominio

#### [MODIFY] [organization.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/domain/organization/organization.go)
- Añadir el campo `ProviderOrgID string json:"provider_org_id"` en el struct `Organization`.

#### [MODIFY] [identity_provider.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/domain/user/identity_provider.go)
- Actualizar y ampliar el puerto `IdentityProvider`:
```go
type IdentityProvider interface {
	CreateAndInviteUser(ctx context.Context, email, role string) (string, error)
	AssignRole(ctx context.Context, userID, orgID, role string) error // Añadido orgID
	DeleteUser(ctx context.Context, userID string) error
	CreateOrganization(ctx context.Context, name string) (string, error)
	DeleteOrganization(ctx context.Context, orgID string) error // Añadido para Rollback/Compensación
}
```

---

### Capa de Adaptadores e Infraestructura

#### [NEW] [007_org_provider_id.sql](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/infrastructure/db/migrations/007_org_provider_id.sql)
- Migración para añadir `provider_org_id` a la tabla `organizations`.
- Incluir la sección de reversión (down migration).

#### [MODIFY] [organization_repo.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/infrastructure/db/organization_repo.go)
- Actualizar las consultas SQL y métodos para mapear `provider_org_id`.
- Utilizar `sql.NullString` en el escaneo para evitar pánicos con registros nulos legados.

#### [MODIFY] [client.go (Logto)](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/infrastructure/logto/client.go)
- Implementar `CreateOrganization`: llama a `POST /api/organizations` enviando `{"name": name}`.
- Implementar `DeleteOrganization`: llama a `DELETE /api/organizations/{orgID}`.
- Modificar `AssignRole` para usar el parámetro dinámico `orgID` en lugar del configurado de forma estática `c.orgID`.

#### [MODIFY] [client.go (Zitadel)](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/infrastructure/zitadel/client.go)
- Adaptar las firmas de los métodos y retornar no-ops para la creación y eliminación de orgs en el adaptador legacy.

#### [MODIFY] [client.go (Mock)](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/infrastructure/mock/client.go)
- Adaptar firmas e implementar no-ops o mocks funcionales.

---

### Interfaces (REST Onboarding y Handlers)

#### [MODIFY] [onboarding.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/interfaces/rest/onboarding.go)
- Cambiar la inicialización para depender de `user.IdentityProvider`.
- En `HandleOnboarding`:
  1. Crear la organización en el IdP: `identityProvider.CreateOrganization`.
  2. Si falla la base de datos local al guardar la organización o el usuario, disparar el rollback correspondiente llamando a `DeleteOrganization` y `DeleteUser` en el IdP.

#### [MODIFY] [users.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/interfaces/rest/users.go)
- Adaptar la llamada a `AssignRole` pasando el ID de la organización en la invitación de usuarios.

---

## Verification Plan

### Automated Tests
- Actualizar y ejecutar los tests unitarios: `go test ./...`
- Añadir cobertura para los escenarios de fallo y rollback en `onboarding_test.go`.

### Manual Verification
- Levantar el entorno local en modo `mock` y verificar que el flujo de Onboarding realiza la simulación completa sin errores.
