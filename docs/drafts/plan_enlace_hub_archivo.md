# Plan Definitivo de Emparejamiento (Archivo de Aprovisionamiento)

Este documento define la arquitectura exacta para enlazar el Hub (que puede operar 100% offline/local) con TableHub Cloud (SaaS) mediante la carga de un archivo seguro, permitiendo máxima flexibilidad comercial.

## 1. Naturaleza del Hub (Independencia Local)
- **Operación Local por Defecto:** El Hub es un dispositivo poderoso por sí mismo. Tiene su propia interfaz web local y puede operar un POS o TastyIgniter de forma completamente aislada en la red del restaurante. Nadie está obligado a pagar por la Nube.
- **Conexión Opcional y Configurable:** Si el dueño del restaurante desea las ventajas del SaaS (Webhooks, túneles a internet, control remoto), entrará al panel local del Hub para habilitar la conexión. El dominio de la nube (`cloud.tablehub.com`) **no estará harcodeado permanentemente**; será un campo de texto configurable por si el sistema cambia de dominio o el cliente monta su propio servidor.

## 2. El Archivo de Llave (El mecanismo de enlace)
Has dado en el clavo con la idea de los archivos `.pem`. Para conectar el Hub a la nube, utilizaremos un archivo de identidad criptográfica. 

### El Flujo de Generación (En el SaaS)
1. El usuario entra a su cuenta de TableHub Cloud y hace clic en "Crear conexión de Hub".
2. La Nube genera un par de llaves Ed25519 (Pública y Privada) y un `HubID` único.
3. La Nube guarda la Llave Pública en su base de datos.
4. La Nube le descarga al usuario un archivo de seguridad (por ejemplo, con extensión `.thub` o `tablehub.key`).
5. **¿Qué tiene este archivo por dentro?** Es un archivo de texto encriptado o estructurado (tipo PEM o JSON) que contiene:
   - La URL de tu nube (`wss://cloud.tablehub.com/ws`)
   - El ID del Restaurante
   - El `HubID`
   - La Llave Privada (Private Key)

### El Flujo de Carga (En el Hub Local / Android APK)
1. El usuario entra al panel web local de su Hub (o a la futura app de Android).
2. Va a la sección de "Conexión a la Nube".
3. Arrastra y suelta el archivo `tablehub.key` (o selecciona el archivo en Android).
4. El Hub lee el archivo, extrae la URL y su nueva Llave Privada.
5. El Hub guarda esto en su SQLite y automáticamente levanta la conexión WebSocket hacia esa URL.
6. Como el Hub usa la llave privada que la nube le dio, la autenticación es instantánea y a prueba de balas.

## 3. Ventajas de este Diseño
- **Súper fácil para el usuario:** Cargar un archivito es una acción de 2 segundos. Y para la APK de Android, puedes hacer que la app lea el archivo desde las descargas del teléfono o incluso usar un Código QR que contenga este texto si el archivo es ligero.
- **Seguridad Extrema:** La nube es la que dicta las llaves criptográficas. No hay manera de que un Hub "falso" o no autorizado se conecte.
