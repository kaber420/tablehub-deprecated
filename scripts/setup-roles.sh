#!/bin/bash
set -euo pipefail

ZITADEL_URL="http://localhost:8088"

echo "=== Configuración de Roles y Claims en Zitadel ==="
if [ $# -ge 1 ]; then
    ZITADEL_PAT="$1"
else
    echo "Por favor, ingresa tu Personal Access Token (PAT) de Zitadel:"
    read -sp "PAT: " ZITADEL_PAT
    echo ""
fi

if [[ -z "$ZITADEL_PAT" ]]; then
    echo "Error: El PAT no puede estar vacío."
    exit 1
fi

echo "Buscando proyecto existente 'tablehub'..."
PROJECTS_RESP=$(curl -s -H "Authorization: Bearer ${ZITADEL_PAT}" -H "Content-Type: application/json" -d '{"queries": [{"nameQuery": {"name": "tablehub", "method": "TEXT_QUERY_METHOD_EQUALS"}}]}' "${ZITADEL_URL}/management/v1/projects/_search")

PROJECT_ID=$(echo "$PROJECTS_RESP" | jq -r '.result[0].id')

if [[ "$PROJECT_ID" == "null" || -z "$PROJECT_ID" ]]; then
    echo "Error: No se encontró ningún proyecto llamado 'tablehub'."
    echo "Asegúrate de haber ejecutado primero scripts/setup-zitadel.sh"
    exit 1
fi

echo "Proyecto 'tablehub' encontrado con ID: $PROJECT_ID"

# 1. Activar Project Role Assertion (Assert Roles on Authentication)
echo "Activando 'Assert Roles on Authentication'..."
curl -s -X PUT \
  -H "Authorization: Bearer ${ZITADEL_PAT}" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"tablehub\",
    \"projectRoleAssertion\": true,
    \"projectRoleCheck\": true
  }" \
  "${ZITADEL_URL}/management/v1/projects/${PROJECT_ID}" > /dev/null

echo "Aserción de roles activada con éxito."

# 2. Agregar los roles al proyecto
ROLES=("superadmin" "owner" "admin" "viewer")

for ROLE in "${ROLES[@]}"; do
    echo "Creando rol '${ROLE}' en el proyecto..."
    # Si el rol ya existe, la API de Zitadel responderá con un error de conflicto pero continuará sin problemas.
    curl -s -X POST \
      -H "Authorization: Bearer ${ZITADEL_PAT}" \
      -H "Content-Type: application/json" \
      -d "{
        \"roleKey\": \"${ROLE}\",
        \"displayName\": \"${ROLE^}\"
      }" \
      "${ZITADEL_URL}/management/v1/projects/${PROJECT_ID}/roles" > /dev/null || true
done

echo "=== Configuración de Roles y Claims Finalizada con Éxito ==="
