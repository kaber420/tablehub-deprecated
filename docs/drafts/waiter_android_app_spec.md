# Especificación Técnica: Aplicación Android para Meseros (Borrador)

Este documento define la arquitectura, pila tecnológica y requerimientos de seguridad para la aplicación móvil nativa de meseros (APK) de Tablehub.

---

## 1. Pila Tecnológica Recomendada

Para garantizar el rendimiento en dispositivos móviles y tablets de uso rudo en restaurantes:

* **Lenguaje:** Kotlin (nativo).
* **Interfaz de Usuario (UI):** Jetpack Compose (declarativo y moderno).
* **Cliente HTTP/API:** Retrofit + OkHttp (para peticiones REST).
* **Tiempo Real (WebSockets):** OkHttp WebSocket Listener (mantiene el canal abierto con el Hub o la Nube).
* **Autenticación (OIDC):** `AppAuth-Android` (librería recomendada por Zitadel para flujos OAuth2 seguros).
* **Base de Datos Local (Caché):** Room DB (para guardar salones, mesas, platillos del menú y configuración localmente).

---

## 2. Gestión de Conectividad Híbrida (Online/Offline)

La app debe alternar automáticamente entre la nube y la red local sin interrumpir la operación:

```
                  ┌──> Conexión Nube ───> WebSocket a Tablehub Cloud (Con Internet)
                  │
APK Mesero ───────┼──> Descubrimiento ──> Búsqueda de Hub vía UDP/mDNS en Wi-Fi local
                  │
                  └──> Conexión Local ──> WebSocket directo a la IP del Hub (Sin Internet)
```

### Lógica de Reconexión:
* **Detección Activa:** La APK pinguea a Tablehub Cloud periódicamente. Si detecta pérdida de conexión, inicia la búsqueda del Hub local en la subred de la Wi-Fi privada del restaurante mediante mDNS (Multicast DNS).
* **Sincronización Silenciosa:** Al conectarse directamente al Hub local, cambia la base del endpoint API de `https://cloud.tablehub.com` a `http://192.168.1.XX` de forma transparente.
* **Autenticación del Hub con Llave de la Nube (Anti-Spoofing):** Para evitar ataques de suplantación del Hub (mDNS Spoofing/Evil Twin), la APK descarga previamente la lista de llaves públicas de los Hubs autorizados de la sucursal desde Tablehub Cloud (cuando tiene internet). Al descubrir el Hub local por mDNS en modo offline, la APK exige al Hub local que firme un desafío (challenge-response) o presente un certificado autofirmado que sea verificado criptográficamente contra la llave pública descargada de la nube. Si no coincide, la conexión es rechazada.

---

## 3. Servicio en Segundo Plano y Notificaciones de Piso

Android tiene políticas estrictas de ahorro de batería que cierran procesos en segundo plano. Para evitar que un mesero pierda la alerta de un comensal:

1. **Notificaciones Push (FCM) en Modo Online (Estándar Comercial):**
   - Cuando el local tiene conexión a internet activa, **no se mantiene ningún WebSocket abierto en segundo plano**.
   - Se utiliza Firebase Cloud Messaging (FCM) para despertar el dispositivo mediante notificaciones push de alta prioridad administradas por el sistema operativo Android. Esto reduce el consumo de batería a un 0% para la aplicación de Tablehub cuando está en segundo plano.
2. **Android Foreground Service (Solo Modo Local Offline y Wi-Fi de Sucursal):**
   - El WebSocket local con la IP del Hub se activa únicamente si la sucursal pierde internet y la APK se encuentra conectada físicamente al **Wi-Fi dedicado para el personal (Wi-Fi Staff)** del restaurante.
   - El ciclo de vida de este WebSocket es controlado dinámicamente mediante `ConnectivityManager.NetworkCallback` en Android: tan pronto como el mesero se desconecta del Wi-Fi de la sucursal o sale de su cobertura, el servicio en primer plano apaga el WebSocket local y detiene el proceso para preservar la batería. No se mantendrán conexiones activas "eternas" en segundo plano cuando el mesero está fuera de su área de trabajo.
3. **Integración con WearOS (Smartwatches):**
   - La APK enviará notificaciones estándar del sistema para que se reflejen de inmediato en la vibración del smartwatch del mesero, asegurando la recepción del llamado de mesa en cocinas o salones con alto nivel de ruido.

---

## 4. Requerimientos de Seguridad en el Dispositivo

* **Almacenamiento Encriptado:** Los tokens JWT (tanto el de Zitadel como el del Hub local) y el Refresh Token se almacenarán usando `EncryptedSharedPreferences`, que utiliza cifrado por hardware (Android Keystore System).
* **Validación de SSL Pinning (Modo Online):** Para prevenir ataques de intermediario (Man-in-the-Middle) en internet, la app verificará el certificado SSL del servidor Cloud de Tablehub.
* **Seguridad en Redes Locales:** Para el modo offline local a través de HTTP (sin SSL de dominio en la IP del Hub), la APK solo aceptará peticiones locales si el dispositivo está conectado a la red Wi-Fi Staff preestablecida.
