# Integración Cloud POS: Frontend (Panel SaaS)

Para que el dueño del restaurante pueda conectar su Cloud POS (como TastyIgniter o Toast), necesita una interfaz en el **Panel de Administración SaaS de Tablehub** donde pueda obtener los datos necesarios (URL y Token) para configurarlo.

## 1. Experiencia de Usuario (Flujo de UI)

El restaurante iniciará sesión en su panel de Tablehub en la nube y seguirá este flujo:

1. **Navegación:** Ir a `Configuración` > `Integraciones`.
2. **Selección del POS:** Verá una cuadrícula (grid) con los logos de los sistemas soportados (TastyIgniter, Toast, Square, etc.). Hará clic en "Conectar" en el que utilice.
3. **Generación de Credenciales:** El sistema generará automáticamente dos piezas de información cruciales que el usuario debe copiar.
4. **Instrucciones:** La misma pantalla le mostrará un paso a paso de dónde pegar esa información en su POS.

## 2. Pantalla de Configuración de la Integración

Cuando el usuario selecciona "TastyIgniter", por ejemplo, verá una tarjeta con la siguiente información:

### A. Webhook URL (Endpoint)
Esta es la dirección a la que el POS enviará los datos. El frontend solicitará esta URL al backend.
```text
URL de Webhook:
https://api.tablehub.io/v1/webhooks/tastyigniter/tn_9a8b7c6d5e
```
*(Se incluye un botón de "Copiar al portapapeles" al lado de la URL)*.

### B. Secret Token (API Key)
Este es el "password" que el POS usará para verificar que los datos realmente van hacia la cuenta correcta y de forma segura.
> [!WARNING]
> El Secret Token solo se muestra una vez por motivos de seguridad. Si el usuario lo pierde, tendrá que generar uno nuevo (lo que invalidará el anterior).

```text
Secret Token:
th_sk_live_5x9Q2pL8mN4vR1jK0wZ7yX3bC6
```
*(Se incluye un botón de "Copiar al portapapeles")*.

## 3. Guía Paso a Paso para el Usuario (Mostrada en el UI)

En la misma pantalla, se le debe mostrar al usuario qué hacer con esos datos. Ejemplo para TastyIgniter:

> **¿Cómo configurar TastyIgniter?**
> 1. Inicia sesión en el panel de administrador de tu TastyIgniter.
> 2. Ve a **System** > **Webhooks**.
> 3. Haz clic en **New Webhook**.
> 4. En el campo *Payload URL*, pega la **URL de Webhook** que copiaste arriba.
> 5. En el campo *Secret*, pega el **Secret Token** que copiaste arriba.
> 6. En *Events*, selecciona `order.created` o `order.updated`.
> 7. Guarda los cambios. ¡Listo! Tablehub ahora recibirá tus pedidos.

## 4. Estado de la Integración (Feedback Visual)

El frontend debe mostrar si la integración está funcionando.
- **Estado: Esperando primer evento...** (Punto naranja). Muestra este estado apenas se genera el webhook.
- **Estado: Conectado** (Punto verde). Se actualiza automáticamente (vía WebSocket/NATS al frontend del dashboard SaaS) tan pronto el backend recibe el primer webhook exitoso desde el POS.
- **Botón "Desconectar/Revocar":** Permite al usuario destruir el webhook si cambia de POS o si cree que su clave fue comprometida.

## 5. Diseño del Componente (Referencia)

```markdown
+-------------------------------------------------------------+
|  [Logo TastyIgniter]  Configurar Integración                |
+-------------------------------------------------------------+
|                                                             |
|  Paso 1: Copia tu información de conexión                   |
|                                                             |
|  Webhook URL:                                               |
|  [ https://api.tablehub.io/v1/webhooks/.... ] [Copiar]      |
|                                                             |
|  Secret Token:                                              |
|  [ th_sk_live_5x9Q2pL8mN...                 ] [Copiar]      |
|                                                             |
|-------------------------------------------------------------|
|                                                             |
|  Paso 2: Configura tu POS                                   |
|  1. Ve a System > Webhooks en TastyIgniter.                 |
|  2. Pega la URL y el Secret en los campos correspondientes. |
|                                                             |
|  [ Probar Conexión ]           Estado: 🟢 Activo            |
+-------------------------------------------------------------+
```
