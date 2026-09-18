# Plan de Implementación: Zitadel Zero-Touch (Automatización 100%)

## Objetivo
Refactorizar el despliegue de Zitadel en Docker para que inyecte un `setup.yaml` durante la primera inicialización. Esto generará automáticamente un *Machine User* con permisos de administrador y un *Personal Access Token (PAT)* que se imprimirá en los logs de Docker, eliminando la necesidad de entrar a la consola web manualmente.

## 1. Cambios en Infraestructura (Docker)
**Archivo:** `cloud/docker-compose.yml`
- Inyectar un volumen para leer el archivo `setup.yaml` dentro del contenedor `zitadel-setup`.
- Modificar el comando `setup` para que acepte `--machinekey` o `--config` con nuestra configuración.

**Archivo:** `cloud/setup.yaml` (NUEVO)
- Contendrá la directiva `FirstInstance.Org.Machine` para que Zitadel, al crear la base de datos por primera vez, cree un usuario llamado `tablehub-automation` y le genere un PAT que no expire pronto.

## 2. Refactorización del Script de Automatización
**Archivo:** `scripts/setup-zitadel.sh`
En lugar de pedirte el token de forma manual con `read -s PAT`, el script ahora:
1. Buscará el PAT generado automáticamente leyendo los logs de Docker:
   `PAT=$(docker logs tablehub-zitadel-setup 2>&1 | grep -o "Token: .*" | awk '{print $2}')`
2. Una vez capturado el PAT sin intervención humana, ejecutará los mismos pasos del plan de Mimo:
   - Crear el Proyecto "tablehub".
   - Crear la SPA OIDC "tablehub-web".
   - Extraer el `Client ID` usando `jq`.
   - Escribir las variables en `cloud/web/.env`.

## 3. Beneficios
- **Cero intervenciones humanas:** Con un solo comando (`docker-compose up` seguido del script de bash) se levanta toda la identidad B2B.
- **Preparado para CI/CD:** Si alguna vez quieres desplegar esto en un servidor real, el sistema ya no dependerá de que alguien haga clics en la interfaz.

---
**Nota para la ejecución:** Como este es un proceso de inicialización de "Primera Instancia" (`FirstInstance`), necesitaremos borrar el volumen de Postgres actual (`docker-compose down -v`) para que Zitadel corra el setup inicial de nuevo y procese el `setup.yaml`.
