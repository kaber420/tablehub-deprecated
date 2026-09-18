# Plan de Implementación: Eliminación de Solicitudes de Acceso (B2B Solo Invitación)

Este plan detalla los cambios técnicos necesarios para remover por completo la funcionalidad de solicitudes de acceso y auto-registro ("Unirse a Organización") tanto en el frontend como en el backend de Tablehub. La plataforma pasará a un esquema B2B estricto de "Solo Invitación".

---

## Cambios Propuestos

### 1. Nube: Base de Datos y Migraciones

#### [NEW] [009_remove_access_requests.sql](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/infrastructure/db/migrations/009_remove_access_requests.sql)
- Crear una migración SQL para eliminar físicamente la tabla `access_requests` en PostgreSQL:
```sql
BEGIN;
DROP TABLE IF EXISTS access_requests CASCADE;
COMMIT;
```

---

### 2. Nube: Modelo de Dominio y Repositorios

#### [DELETE] [request.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/domain/access/request.go)
- Eliminar el archivo que contiene la definición de la entidad y la interfaz de repositorio de solicitudes de acceso.

#### [DELETE] [access_repo.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/infrastructure/db/access_repo.go)
- Eliminar la implementación en PostgreSQL del repositorio de solicitudes de acceso.

---

### 3. Nube: Servicios y Controladores REST

#### [DELETE] [access_request.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/interfaces/rest/access_request.go)
- Eliminar el handler REST que expone los endpoints de solicitudes de acceso.

#### [MODIFY] [onboarding.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/interfaces/rest/onboarding.go)
- Remover la dependencia de `access.Repository` del struct `OnboardingHandler` y de su constructor `NewOnboardingHandler`.
- Modificar el método `GetMe`: si el usuario no tiene una organización asociada (`u.OrganizationID == uuid.Nil`), retornar inmediatamente `registered: false` y `pending: false`, omitiendo la búsqueda de solicitudes pendientes en la base de datos.

#### [MODIFY] [onboarding_test.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/interfaces/rest/onboarding_test.go)
- Retirar las referencias al mock del repositorio de accesos y ajustar las aserciones de prueba correspondientes.

#### [MODIFY] [router.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/interfaces/rest/router.go)
- Eliminar la inyección de `accessRequestHandler` en el router.
- Remover del enrutador las rutas:
  - `POST /v1/access-requests`
  - `GET /v1/orgs/{orgID}/access-requests`
  - `POST /v1/orgs/{orgID}/access-requests/{id}/resolve`

#### [MODIFY] [main.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/cmd/server/main.go)
- Eliminar la inicialización de `accessRepo` y `accessRequestHandler`.
- Actualizar las inicializaciones de `onboardingHandler` y del `Router` para remover el parámetro descartado.

---

### 4. Nube: Frontend Web (Svelte)

#### [DELETE] [AccessRequests.svelte](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/web/src/pages/AccessRequests.svelte)
- Eliminar por completo el componente de la página de solicitudes de acceso.

#### [MODIFY] [Onboarding.svelte](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/web/src/pages/Onboarding.svelte)
- Modificar la interfaz de bienvenida para remover la opción de "Unirse a Organización".
- Si el usuario no está registrado en ninguna organización, mostrar directamente el formulario para crear una nueva organización (saltándose la pantalla de elección inicial).

#### [MODIFY] [Dock.svelte](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/web/src/lib/Dock.svelte)
- Eliminar la pestaña de solicitudes (`'access-requests'`) del menú lateral.

#### [MODIFY] [App.svelte](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/web/src/App.svelte)
- Eliminar la importación y renderizado dinámico de la página `AccessRequests`.

---

### 5. Nube: ROADMAP.md

#### [MODIFY] [ROADMAP.md](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/ROADMAP.md)
- Modificar el Hito 2 en el roadmap para estipular que el modelo de seguridad del SaaS de Tablehub es exclusivamente de "Solo Invitación", documentando la remoción de auto-uniones públicas.

---

### 6. Verificación de E2E Tests en Cloud

#### [MODIFY] [e2e_test.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/tests/e2e/e2e_test.go)
- Remover la inicialización de `accessRequestHandler` de la configuración de test.
- Agregar la nueva migración `009_remove_access_requests.sql` al cargador de migraciones.

---

## Plan de Verificación

### Pruebas Automatizadas
- Ejecutar tests unitarios de controladores REST:
  ```bash
  go test -v ./cloud/internal/interfaces/rest/...
  ```
- Ejecutar suite de pruebas E2E en la Nube:
  ```bash
  go test -v -tags=e2e ./cloud/tests/e2e/...
  ```

### Pruebas Manuales
- Iniciar sesión en la Nube con un usuario recién registrado (sin organización).
- Comprobar que es redirigido de inmediato al formulario de creación de organización sin opción a unirse a una organización externa.
- Comprobar en el dock/menú lateral de una organización existente que la pestaña de "Solicitudes de Acceso" ha desaparecido por completo.
