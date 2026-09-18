# Arquitectura Nube-Borde (Edge-to-Cloud) usando NATS Leaf Nodes

Este documento detalla la arquitectura propuesta para la comunicación robusta y segura entre los Maitre Hubs locales (Edge) y el servicio centralizado SaaS (Cloud) utilizando la tecnología NATS Leaf Nodes.

## 1. El Problema Actual y la Solución
Actualmente, el Maitre Hub abre una conexión WebSocket saliente pura hacia la nube. Si la conexión se pierde, los mensajes en tránsito pueden perderse a menos que se implemente una lógica de reintentos manual y acuses de recibo (ACKs) en la capa de la aplicación (Patrón Outbox).

La solución nativa y más robusta es aprovechar que el Maitre Hub ya utiliza un servidor NATS embebido. Al configurar el servidor NATS de la nube y los servidores NATS locales para comunicarse entre sí, formamos una topología de "Leaf Nodes" (Nodos Hoja).

## 2. ¿Qué son los NATS Leaf Nodes?
Un Leaf Node es una extensión remota de un clúster central de NATS. En el contexto de Tablehub:
- **La Nube (SaaS)** es el "Supercluster" central.
- **Cada Maitre Hub en los restaurantes** es un "Leaf Node".

En lugar de programar WebSockets personalizados, el servidor NATS local en el Maitre Hub se conecta directamente al NATS de la nube. Para la aplicación de Go local, es transparente: simplemente publica un mensaje en el NATS local, y este se enruta automáticamente hacia la nube.

## 3. Aislamiento y Multi-Tenant (NATS Accounts)
En un modelo SaaS, es crítico que los datos de un cliente no se crucen con los de otro. NATS resuelve esto nativamente con **Accounts**:
- El clúster SaaS está configurado con múltiples Accounts aisladas (ej. Account_Restaurante_A, Account_Restaurante_B).
- Cuando el Hub del Restaurante A se conecta como Leaf Node a la nube, se vincula exclusivamente a su Account.
- Es físicamente y criptográficamente imposible que el Restaurante A escuche el tráfico del Restaurante B. El enrutamiento garantiza un aislamiento total del tenant.

## 4. Seguridad Descentralizada (JWT y Ed25519)
Conectar miles de restaurantes a la nube de manera segura no se gestiona con contraseñas estáticas, sino con un sistema de autorización descentralizada basado en la misma criptografía (Ed25519) que el proyecto ya maneja:
1. **El Operador SaaS (Tú)** tiene una clave privada maestra (Account Key).
2. Para cada nuevo restaurante, generas un token de usuario (JWT) firmado con esa clave. Este JWT especifica a qué Account pertenece el usuario.
3. El Maitre Hub utiliza este JWT para autenticar su conexión Leaf Node hacia la nube.
4. **Revocación:** Si un cliente deja de pagar el SaaS, se actualiza el servidor para revocar ese JWT o eliminar su acceso. El Leaf Node será expulsado de la nube instantáneamente.

## 5. Resiliencia y Tolerancia a Fallos (Offline First)
La mayor ventaja de esta arquitectura es cómo maneja las caídas de internet:
- Cuando el internet del restaurante se corta, la conexión Leaf Node se pausa.
- El Maitre Hub sigue operando 100% local. Las mesas siguen pidiendo y los eventos se publican en el NATS local.
- **JetStream** (la capa de persistencia de NATS) se encarga de almacenar en disco todos los eventos críticos que no han podido viajar a la nube.
- En el instante en que regresa el internet, el Leaf Node se reconecta automáticamente. JetStream drena todos los mensajes atrasados en orden cronológico hacia la nube y maneja los ACKs binarios. No se pierde ni una orden, y no tienes que programar ni una sola línea de lógica de reintentos.

## 6. Pasos para la Implementación Futura
1. **Infraestructura Cloud:** Desplegar un clúster NATS en la nube con soporte JetStream habilitado y resolución de Accounts activada.
2. **Generación de JWTs:** Crear la utilidad administrativa para emitir credenciales (Users/JWTs) para cada restaurante nuevo.
3. **Actualización del Hub (Go):** Modificar `StartEmbeddedServer` para que reciba las credenciales y configure el servidor local como un Leaf Node apuntando a la URL del clúster de la nube.
4. **Desactivar el WebSocket Saliente actual:** Reemplazar el flujo de `ConnectToCloud` por la conexión nativa Leaf Node.
