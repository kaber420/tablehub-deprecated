# Arquitectura de Conexión Externa y Funciones Nube (Acceso Remoto)

Este documento define la estrategia de red y arquitectura de software para cuando los usuarios (gerentes, repartidores, meseros en casa) interactúan con la plataforma TableHub **desde fuera del restaurante** (Internet/Datos Móviles).

## 1. La Regla de Oro de Seguridad (Zero Trust Edge)
**Está estrictamente prohibido exponer el puerto local de NATS (`4222`) o cualquier puerto del Hub a internet (No Port Forwarding).**
Exponer puertos de un restaurante a internet abre la puerta a ataques cibernéticos y es logísticamente insostenible (requeriría IPs estáticas en cada local comercial). Todo el tráfico externo debe pasar siempre y exclusivamente por el **Servidor SaaS en la Nube (Cloud)**.

## 2. Enrutamiento Inteligente (Smart Routing) en Apps Móviles
Las aplicaciones cliente (Kotlin/Flutter) implementarán una lógica de "Enrutamiento Inteligente":
- **Si detectan la red Wi-Fi local del restaurante**: Se conectan a `NATS Local` para operar mesas y comandas con latencia cero.
- **Si están en 4G/5G o en casa**: Desactivan la conexión local de NATS y operan exclusivamente mediante **peticiones a la API REST / NATS del Servidor Cloud**.

## 3. Funciones "Cloud-First" (Datos Globales)
No todo tiene que vivir en el Hub local. Existen datos que pertenecen estructuralmente a la Nube (la "Central") y deben ser accedidos mediante APIs de internet (REST o WebSockets hacia el SaaS):

* **Entregas a Domicilio (Delivery):** La logística de repartidores, rutas GPS y asignación de flotas ocurre a nivel de ciudad o cadena, por lo que los repartidores se conectan a la Nube, no al restaurante físico.
* **Recursos Humanos y Horarios:** Un empleado revisando su turno desde su casa consultará la base de datos de RR.HH. en el SaaS Cloud.
* **Compras e Inventario Centralizado:** Los pedidos a proveedores para surtir el restaurante se manejan desde un portal web que ataca directamente a la Nube.
* **Comunicación Interna Corporativa:** Mensajería entre gerentes de diferentes sucursales.
* **Licenciamiento y Login:** Autenticación inicial, validación de suscripciones y descargas de menús globales.

## 4. Frontera de Datos: Operacional vs Administrativo
Por filosofía de diseño y eficiencia técnica, **se descarta la idea de consultar el estado en vivo de las mesas desde internet**.
La arquitectura define una frontera estricta:

1. **Datos Operacionales Vivos (Exclusivo Local):**
   - El estado en vivo de las mesas, qué platillos se están cocinando y el flujo inmediato del restaurante solo interesa a quienes están físicamente allí. Estos datos viven y mueren en la red local.
2. **Datos Administrativos y Consolidación (Exclusivo Cloud):**
   - El Hub sincroniza pasivamente hacia la nube los cortes de caja, totales de ventas, métricas de rendimiento y cierres de mesas.
   - Si un gerente está en el exterior, **solo consulta estos datos administrativos agregados** directamente en la base de datos de la Nube (estadísticas, inventario mermado, ventas diarias), pero nunca se conecta para "ver el mapa de mesas en vivo". 

*Conclusión: Esta separación drástica reduce el tráfico de red, evita saturar el enlace de internet del restaurante y mantiene la Nube enfocada estrictamente en la administración corporativa y la logística externa.*
