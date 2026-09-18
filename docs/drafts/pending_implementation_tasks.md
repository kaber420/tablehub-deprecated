# Tareas Pendientes de Implementación (Backlog)

Este documento enumera las características y flujos que aún **falta por implementar** en el proyecto TableHub, basándose en las discusiones y planes de arquitectura recientes.

## 1. Interfaz de Configuración Avanzada (Hub UI)
Aunque la pestaña de "Estado de la Nube" y la lógica de desconexión ya están implementadas, quedan pendientes las demás pestañas de configuración:

### Tab: Dispositivos de Hardware
- Interfaz para añadir y gestionar impresoras locales.
- Gestión de comanderas y terminales locales.

### Tab: Sistema Local y Red
- Configuración de red y visualización de IPs locales.
- Estado y mantenimiento de la base de datos SQLite local (`tablehub.db`).

### Tab: Autenticación y Administración
- Flujo para cambiar la contraseña de administración del panel local.

## 2. Opciones de Seguridad Avanzada (Ed25519)
- **Rotación Manual de Claves**: Añadir un botón en la sección avanzada de la UI que permita regenerar explícitamente el archivo `data/hub_cloud.key` sin necesidad de desconectar o borrar la base de datos completa.

## 3. Estado en Tiempo Real (Polling vs Eventos NATS)
- *Alineado con el comentario: "solucionar los bugs y despues esto del polling"*
- Reemplazar el chequeo inicial estático por un sistema real-time o polling eficiente para que la UI del Hub refleje caídas de red inmediatamente sin requerir refrescar la página.

## 4. Scripts de Reset de Fábrica (Hardware Appliance)
*(Definido en `appliance_os_and_factory_reset_spec.md`)*
- Crear el script de utilidad a nivel de software (ej. `tablehub-reset.sh`) que ejecute el borrado seguro del directorio `/data` o `./data`.
- (A futuro) Preparación de la imagen del sistema operativo inmutable (OverlayFS) para la distribución oficial del hardware de TableHub.
