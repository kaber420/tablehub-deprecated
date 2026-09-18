# Plan de Implementación: Enlace Hub-Cloud mediante Token de Uso Único y Llaves Locales

Este plan detalla las modificaciones en el código de Tablehub para reemplazar el modelo actual de provisión de llaves desde la nube por un modelo descentralizado y más seguro:
1. El archivo `.thub` contendrá únicamente metadatos y un token de uso único (`bootstrap_token`) firmado por la Nube.
2. El Hub físico generará localmente su par de claves Ed25519 (la llave privada nunca sale del Hub).
3. El Hub enviará su clave pública y el token a un nuevo endpoint de activación en la Nube.
4. La Nube asociará la clave pública al Hub, quemará el token en la base de datos y permitirá la autenticación WebSocket por desafío-respuesta Ed25519.

---

## Cambios Propuestos

### 1. Nube: Base de Datos y Migraciones

#### [NEW] [008_hub_bootstrap_token.sql](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/infrastructure/db/migrations/008_hub_bootstrap_token.sql)
- Migración para relajar el constraint de la clave pública del Hub (para permitir que sea nula al registrarse por primera vez) y agregar la columna `bootstrap_token`.
```sql
BEGIN;
ALTER TABLE hub_registry ALTER COLUMN public_key DROP NOT NULL;
ALTER TABLE hub_registry ADD COLUMN IF NOT EXISTS bootstrap_token TEXT;
COMMIT;
```

---

### 2. Nube: Modelo de Dominio y Repositorio

#### [MODIFY] [hub.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/domain/hub/hub.go)
- Agregar el campo `BootstrapToken string` al struct `Hub`.
- Añadir el método `UpdatePublicKeyAndBurnToken(ctx context.Context, id types.HubID, publicKey string) error` a la interfaz `Repository`.

#### [MODIFY] [hub_repo.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/infrastructure/db/hub_repo.go)
- Modificar el método `Register` para incluir el campo `bootstrap_token` (y aceptar `public_key` vacío).
- Actualizar `GetByID` y queries SELECT para escanear `bootstrap_token`.
- Implementar `UpdatePublicKeyAndBurnToken` para guardar la clave pública enviada por el Hub y resetear (`NULL`) el token de aprovisionamiento en la tabla `hub_registry`.

---

### 3. Nube: Servicios y Controladores REST

#### [MODIFY] [provision.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/application/tunnel/provision.go)
- Eliminar la generación del par de claves Ed25519 en `ProvisionHubUseCase`.
- Generar un token seguro aleatorio (`bootstrap_token`) usando `crypto/rand` en formato hexadecimal.
- Modificar `ProvisionResult` para incluir `bootstrap_token` y remover `public_key` y `private_key`.

#### [MODIFY] [provision_handler.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/interfaces/rest/provision_handler.go)
- Actualizar el formato JSON de descarga de `.thub` para usar el nuevo `ProvisionResult` (con token).
- Implementar `HandleActivate(w http.ResponseWriter, r *http.Request)`:
  - Recibir `hub_id`, `bootstrap_token` y `public_key`.
  - Validar contra la base de datos que el Hub exista y tenga el token correspondiente.
  - Ejecutar `UpdatePublicKeyAndBurnToken` para registrar la clave y anular el token.

#### [MODIFY] [router.go](file:///home/kaber420/Documentos/proyectos/tablehub2/cloud/internal/interfaces/rest/router.go)
- Añadir la ruta pública `/v1/hubs/activate` que apunta a `HandleActivate`.

---

### 4. Hub Local: Procesamiento de Aprovisionamiento y Activación

#### [MODIFY] [settings.go](file:///home/kaber420/Documentos/proyectos/tablehub2/hub/internal/web/settings.go)
- Actualizar el struct `ProvisionPayload` para coincidir con la estructura del `.thub` sin llaves criptográficas.
- Modificar `HandleProvisionUpload`:
  - Llamar a `InitCloudKeys()` para asegurar que el par de llaves Ed25519 del Hub físico se generen localmente (la llave privada se guarda en `data/hub_cloud.key` y la pública en SQLite).
  - Guardar `hub_id`, `restaurant_id` (de `.thub`), `cloud_url` y `bootstrap_token` en SQLite.
  - Realizar una petición POST HTTP a la Nube al endpoint `/v1/hubs/activate` (derivando la URL de `cloud_url`) enviando la clave pública local junto al token.
  - Si la llamada es exitosa:
    - Activar la conexión (`cloud_enabled = true`).
    - Limpiar localmente el `bootstrap_token`.
    - Invocar `RestartCloudConnection()` para iniciar la conexión WebSocket por Ed25519 challenge-response.
  - Si la llamada a la Nube falla, retornar un error 400 con los detalles para la UI.

---

## Plan de Verificación

### Pruebas Unitarias e Integración
- Modificar `cloud/tests/e2e/e2e_test.go` para cargar la migración `008` y simular el nuevo handshake del Hub físico.
- Ejecutar suite de test unitarios: `go test -v ./cloud/internal/infrastructure/db/...`
- Ejecutar pruebas e2e completas: `go test -v -tags=e2e ./cloud/tests/e2e/...`
