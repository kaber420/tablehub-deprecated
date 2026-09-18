import json
import time
import urllib.request
import urllib.parse
import os
import base64

KEY_PATH = "cloud/machinekey/zitadel-admin.json"
ZITADEL_URL = "http://localhost:8088"

# Check if key exists
if not os.path.exists(KEY_PATH):
    print(f"Error: {KEY_PATH} not found.")
    print("Wait for Zitadel to finish booting and generate the key.")
    exit(1)

with open(KEY_PATH, 'r') as f:
    key_data = json.load(f)

# Zitadel machine key data
client_id = key_data["userId"]
key_id = key_data["keyId"]
private_key = key_data["key"]

print("1. Generando JWT para autenticación de máquina...")

# We use jwt module. We need to install it: pip install PyJWT cryptography
try:
    import jwt
except ImportError:
    print("Por favor instala PyJWT y cryptography: pip install PyJWT cryptography")
    exit(1)

now = int(time.time())
payload = {
    "iss": client_id,
    "sub": client_id,
    "aud": f"{ZITADEL_URL}",
    "iat": now,
    "exp": now + 3600
}

headers = {
    "kid": key_id,
    "alg": "RS256"
}

encoded_jwt = jwt.encode(payload, private_key, algorithm="RS256", headers=headers)

print("2. Obteniendo Access Token...")
token_url = f"{ZITADEL_URL}/oauth/v2/token"
data = urllib.parse.urlencode({
    "grant_type": "urn:ietf:params:oauth:grant-type:jwt-bearer",
    "assertion": encoded_jwt,
    "scope": "openid profile email urn:zitadel:iam:org:project:id:zitadel:aud"
}).encode('utf-8')

req = urllib.request.Request(token_url, data=data)
req.add_header("Content-Type", "application/x-www-form-urlencoded")

try:
    with urllib.request.urlopen(req) as response:
        resp_data = json.loads(response.read().decode())
        access_token = resp_data["access_token"]
        print("Access Token obtenido exitosamente.")
except Exception as e:
    print("Fallo al obtener Access Token:", e)
    if hasattr(e, 'read'):
        print(e.read().decode())
    exit(1)

auth_header = {"Authorization": f"Bearer {access_token}"}

print("3. Creando Proyecto 'tablehub'...")
req_proj = urllib.request.Request(f"{ZITADEL_URL}/management/v1/projects", data=json.dumps({"name": "tablehub"}).encode('utf-8'), headers={"Content-Type": "application/json", **auth_header})
project_id = None
try:
    with urllib.request.urlopen(req_proj) as response:
        resp_data = json.loads(response.read().decode())
        project_id = resp_data.get("id")
        print(f"Proyecto creado. ID: {project_id}")
except Exception as e:
    print("Error creando proyecto (probablemente ya existe):", e)
    if hasattr(e, 'read'):
        print(e.read().decode())
    
    print("Buscando proyecto existente...")
    req_search = urllib.request.Request(f"{ZITADEL_URL}/management/v1/projects/_search", data=json.dumps({}).encode('utf-8'), headers={"Content-Type": "application/json", **auth_header})
    try:
        with urllib.request.urlopen(req_search) as response:
            resp_data = json.loads(response.read().decode())
            for p in resp_data.get("result", []):
                if p["name"] == "tablehub":
                    project_id = p["id"]
                    print(f"Proyecto encontrado. ID: {project_id}")
                    break
    except Exception as e2:
        print("No se pudo buscar el proyecto.", e2)
        exit(1)

if not project_id:
    print("No se obtuvo Project ID.")
    exit(1)

print("4. Creando OIDC App 'tablehub-web'...")
app_data = {
    "name": "tablehub-web",
    "oidcConfig": {
        "redirectUris": ["http://localhost:5173/auth/callback"],
        "postLogoutRedirectUris": ["http://localhost:5173/"],
        "responseTypes": ["OIDC_RESPONSE_TYPE_CODE"],
        "grantTypes": ["OIDC_GRANT_TYPE_AUTHORIZATION_CODE"],
        "appType": "OIDC_APP_TYPE_USER_AGENT",
        "authMethodType": "OIDC_AUTH_METHOD_TYPE_NONE"
    }
}
req_app = urllib.request.Request(f"{ZITADEL_URL}/management/v1/projects/{project_id}/apps/oidc", data=json.dumps(app_data).encode('utf-8'), headers={"Content-Type": "application/json", **auth_header})

client_id_out = None
try:
    with urllib.request.urlopen(req_app) as response:
        resp_data = json.loads(response.read().decode())
        client_id_out = resp_data.get("clientId")
        print(f"Aplicación creada. Client ID: {client_id_out}")
except Exception as e:
    print("Error creando aplicación:", e)
    if hasattr(e, 'read'):
        print(e.read().decode())
    exit(1)

if client_id_out:
    env_path = "cloud/web/.env"
    with open(env_path, 'w') as f:
        f.write(f"VITE_ZITADEL_CLIENT_ID={client_id_out}\n")
    print(f"Client ID guardado exitosamente en {env_path}")
    print("¡CONFIGURACIÓN TERMINADA!")
