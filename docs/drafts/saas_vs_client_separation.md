# Arquitectura de Separación SaaS vs. Cliente (Tablehub)

Este documento detalla la recomendación y estrategia de diseño para separar de forma definitiva las responsabilidades de la plataforma. La meta es evitar que el panel de administración general del SaaS (utilizado por dueños de la plataforma, DevOps, administradores de sistemas y SREs) se mezcle con la sección de control operativo y de monitoreo de bajo nivel, la cual es responsabilidad del cliente y sus técnicos locales.

---

## 1. Separación de Responsabilidades

### A. Consola del Administrador del SaaS (Super Admin / Control de Negocio)
* **Destinatarios:** Dueños de Tablehub, SREs, DevOps, Soporte SaaS, Facturación.
* **Propósito:** Supervisión a nivel de negocio y provisión de recursos. Controlar *qué* se vende, a *quién*, y *cuántas* licencias se consumen.
* **Métricas Clave (Lo que importa):**
  * Número de clientes (Organizaciones).
  * Planes de facturación y estado de suscripciones (Stripe).
  * Consumo de cuotas: **Total de Hubs registrados** y **Total de dispositivos IoT aprovisionados** por organización vs. los límites del plan contratado.
* **Lo que se debe Ocultar (Irrelevante aquí):**
  * Si un dispositivo IoT específico está "online" o tiene un error de hardware.
  * Alertas de estado operacional de las tablets.
  * Alias amigables de las mesas o configuraciones internas del POS.

### B. Portal del Cliente / Consola de Operaciones del Restaurante (Tenant Console)
* **Destinatarios:** Gerentes de restaurante, técnicos locales de IT de la franquicia, meseros.
* **Propósito:** Operación diaria y mantenimiento local de la infraestructura.
* **Métricas Clave (Lo que importa):**
  * Estado en tiempo real del Hub (conectado al POS).
  * Estado de conexión de las botoneras ESP32 y tablets en mesa (si están online/offline, nivel de batería, etc.).
  * Alertas y errores operativos (ej: "Error de lectura de sensor", "Tablet de Mesa 4 desconectada").
  * Configuración de alias y mapeo de mesas.

---

## 2. Estructura de APIs Recomendada

Para evitar que los controladores del backend mezclen responsabilidades, se proponen dos namespaces separados en las rutas del servidor API:

```
/v1
 ├── /admin                 <-- Namespace exclusivo para SaaS Admin (Super Admin)
 │    ├── /orgs             <-- CRUD de Organizaciones, límites de planes, etc.
 │    │    └── /{id}/stats  <-- Retorna contadores agregados: { total_hubs: X, total_devices: Y }
 │    └── /billing          <-- Planes de Stripe, webhooks y control de pagos
 │
 └── /orgs/{orgID}          <-- Namespace Multi-tenant (Client Portal)
      ├── /branches         <-- Sucursales y configuraciones locales
      ├── /hubs             <-- Hubs locales, estado de conexión (online/offline)
      └── /devices          <-- Terminales IoT en tiempo real (mantenimiento y alertas de error)
```

---

## 3. Mockup Conceptual de la Interfaz SaaS Clean

A continuación se define la estructura visual simplificada de la Consola SaaS Admin para que muestre únicamente lo que le interesa al operador de la plataforma.

### Métricas de Negocio de Alto Nivel
* **Clientes Totales:** 4 Organizaciones
* **Hubs Totales Desplegados:** 4 Hubs
* **Dispositivos IoT Totales Aprovisionados:** 5 Dispositivos

### Tabla de Clientes y Cuotas de Uso

| Cliente / Organización | Plan SaaS | Hubs Activos / Límite | Dispositivos IoT / Límite | Suscripción Stripe | Acción |
| :--- | :--- | :---: | :---: | :--- | :--- |
| **La Pizza Nostra** | `PRO` | 2 / 10 | 2 / 100 | Activa (`cus_Q8k2...`) | [Gestionar Plan] [Ver Detalles] |
| **Tasty Burgers** | `STARTER` | 1 / 3 | 1 / 15 | Activa (`cus_Q8k5...`) | [Gestionar Plan] [Ver Detalles] |
| **Sushi Palace** | `PRO` | 1 / 10 | 1 / 100 | Activa (`cus_Q9a1...`) | [Gestionar Plan] [Ver Detalles] |
| **Café París** | `FREE` | 0 / 1 | 0 / 5 | Sin suscripción activa | [Cambiar Plan] [Ver Detalles] |

---

## 4. Plan de Transición para el Frontend

Para limpiar `cloud/web/src/pages/Dashboard.svelte` de la telemetría operativa de bajo nivel:
1. **Quitar los LEDs y estados individuales de los dispositivos de la vista general del Administrador.**
2. **Reemplazar la pestaña "Hubs & IoT Devices" por una pestaña "Inventario SaaS y Límites"** que se enfoque en cuotas de uso.
3. **Mudar la visualización detallada del estado en línea de los dispositivos al menú del Tenant / Cliente** o a una vista específica del cliente.
