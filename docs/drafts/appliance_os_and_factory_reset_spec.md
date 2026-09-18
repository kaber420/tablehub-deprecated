# Especificación: Estrategia de Reset de Fábrica (Appliance Inmutable vs. Instalación DIY)

## 1. Despliegues Oficiales (Hardware Appliance / Sistema Inmutable)
Para hardware físico oficial entregado por TableHub (ej. Mini PCs / Raspberry Pi dedicadas):
- **Sistema de Archivos de Solo Lectura (OverlayFS / OSTree)**: El sistema operativo Linux y los binarios compilados del Hub residen en una partición inmutable (`/`).
- **Aislamiento de Datos (`/data`)**: Todas las configuraciones locales (`tablehub.db`), claves Ed25519 (`hub_cloud.key`) y almacenamiento JetStream residen exclusivamente en la partición montada `/data`.
- **Reset de Fábrica del Sistema (Hardware Reset)**: Se ejecuta simplemente purgando el contenido de la partición `/data` y reiniciando el servicio. El Hub arrancará limpio en modo Setup sin necesidad de reinstalar el sistema operativo.

## 2. Despliegues Self-Hosted / DIY (Cualquier Distro Linux)
Para usuarios que ejecutan el Hub en su propio servidor Linux o contenedor:
- **Reset de Fábrica a Nivel de Aplicación (Software Reset)**:
  - El botón "Restablecer de Fábrica" en el panel ejecuta el vaciado del directorio de datos configurado (`DATA_DIR`, ej. `./data`), limpiando SQLite y claves privadas.
  - No requiere permisos de superusuario (`sudo`) ni modificaciones al sistema operativo host.
  - Deja el ejecutable listo para ejecutar el `Setup` inicial de nuevo.
