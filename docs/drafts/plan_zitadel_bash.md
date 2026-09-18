# Plan: Script Bash para configurar Zitadel vía API REST

## Objetivo
Crear `scripts/setup-zitadel.sh` que configure Zitadel (Project + App OIDC SPA)
vía Management API REST con `curl`, guardando el Client ID en `cloud/web/.env`.

## Contexto
- Zitadel corre en `localhost:8080` via Docker Compose.
- Management API v1: `/management/v1/`.
- `cloud/web/.env` no existe todavía.

## Endpoints a utilizar

| Paso | Método | URL |
|------|--------|-----|
| Crear Project | `POST` | `/management/v1/projects` |
| Crear App OIDC | `POST` | `/management/v1/projects/{project_id}/apps/oidc` |
| Configurar OIDC | `PUT` | `/management/v1/projects/{project_id}/apps/{app_id}/oidc_config` |

## Estructura del script

1. **Configuración**: `set -euo pipefail`, variables (`ZITADEL_URL=http://localhost:8080`, `ENV_FILE=../cloud/web/.env`).
2. **Pedir PAT**: `read -sp` (sin eco), validar no vacío.
3. **Validar conexión**: `curl` a `/management/v1/users/_search` con el PAT.
4. **Crear Project** `"tablehub"` → extraer `id` con `jq`.
5. **Crear App OIDC** `"tablehub-web"` → extraer `appId` y `clientId`.
6. **Configurar OIDC**: `authMethodType=NONE` (SPA), `redirectUris=["http://localhost:5173/callback"]`, `postLogoutRedirectUris=["http://localhost:5173/"]`, `grantTypes=[AUTHORIZATION_CODE]`, `responseTypes=[CODE]`.
7. **Guardar en `.env`**: `VITE_ZITADEL_CLIENT_ID` y `VITE_ZITADEL_ISSUER`.
8. **Resumen**: imprimir Project ID, App ID, Client ID.

## Dependencias
- `curl`, `jq` (el script verificará que `jq` esté instalado).
