#!/bin/bash
set -euo pipefail

ZITADEL_URL="http://localhost:8088"
ENV_FILE="cloud/web/.env"

echo "=== Configuración de Zitadel OIDC SPA ==="
echo "Por favor, ingresa tu Personal Access Token (PAT) de Zitadel (Genéralo desde la consola web):"
read -sp "PAT: " ZITADEL_PAT
echo ""

if [[ -z "$ZITADEL_PAT" ]]; then
    echo "Error: El PAT no puede estar vacío."
    exit 1
fi


echo "Verificando conexión y PAT..."
USER_SEARCH=$(curl -s -f -H "Authorization: Bearer ${ZITADEL_PAT}" "${ZITADEL_URL}/management/v1/users/_search" -d '{}' || echo "")

if [[ -z "$USER_SEARCH" ]]; then
    echo "Error: PAT inválido o Zitadel no está disponible."
    exit 1
fi
echo "Conexión exitosa."

echo "Creando proyecto 'tablehub'..."
PROJECT_RESP=$(curl -s -X POST -H "Authorization: Bearer ${ZITADEL_PAT}" -H "Content-Type: application/json" -d '{"name": "tablehub"}' "${ZITADEL_URL}/management/v1/projects")
PROJECT_ID=$(echo "$PROJECT_RESP" | jq -r '.id')
echo "Proyecto creado con ID: $PROJECT_ID"

echo "Creando App OIDC SPA 'tablehub-web'..."
APP_RESP=$(curl -s -X POST -H "Authorization: Bearer ${ZITADEL_PAT}" -H "Content-Type: application/json" -d '{
  "name": "tablehub-web",
  "redirectUris": ["http://localhost:5173/auth/callback"],
  "responseTypes": ["OIDC_RESPONSE_TYPE_CODE"],
  "grantTypes": ["OIDC_GRANT_TYPE_AUTHORIZATION_CODE"],
  "appType": "OIDC_APP_TYPE_USER_AGENT",
  "authMethodType": "OIDC_AUTH_METHOD_TYPE_NONE",
  "postLogoutRedirectUris": ["http://localhost:5173/"]
}' "${ZITADEL_URL}/management/v1/projects/${PROJECT_ID}/apps/oidc")

APP_ID=$(echo "$APP_RESP" | jq -r '.appId')
CLIENT_ID=$(echo "$APP_RESP" | jq -r '.clientId')

if [[ "$CLIENT_ID" == "null" || -z "$CLIENT_ID" ]]; then
    echo "Error al crear la aplicación:"
    echo "$APP_RESP" | jq .
    exit 1
fi

echo "App creada exitosamente."
echo "App ID: $APP_ID"
echo "Client ID: $CLIENT_ID"

echo "Guardando configuración en $ENV_FILE..."
mkdir -p $(dirname "$ENV_FILE")
echo "VITE_ZITADEL_CLIENT_ID=$CLIENT_ID" > "$ENV_FILE"
echo "VITE_ZITADEL_ISSUER=$ZITADEL_URL" >> "$ENV_FILE"

echo "¡Listo! El Client ID se ha configurado para el frontend Svelte."
