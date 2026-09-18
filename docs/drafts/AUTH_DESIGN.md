# Arquitectura de Autenticación y Seguridad (Tablehub)

Este documento detalla los principios de diseño y la implementación técnica para asegurar la interfaz web local y las APIs REST del Hub.

## 1. Conceptos Core

La seguridad de Tablehub se basa en dos pilares fundamentales:
1. **Bcrypt**: Un algoritmo de hash diseñado para ser lento (costoso computacionalmente). Las contraseñas **nunca** se guardarán en texto plano.
2. **JSON Web Tokens (JWT)**: Estándar abierto (RFC 7519) para transmitir sesiones seguras mediante Cookies HttpOnly.

## 2. Flujo de Autenticación Inmutable

El sistema de Tablehub está diseñado para no permitir cambios de credenciales desde la web, garantizando que quien controla el acceso físico/terminal al servidor es el verdadero dueño.

1. **Estado Inicial (Setup de Único Uso)**:
   - Al arrancar, el servidor verifica si existe `admin_password_hash` en SQLite.
   - Si no existe, activa en memoria el endpoint `/api/setup` y el cliente muestra el `SetupView.svelte`.
   - Una vez el administrador envía la contraseña al `/api/setup`, el servidor la encripta, la guarda, **bloquea el endpoint en memoria (devuelve 404 a partir de ese momento)** y el sistema pasa a modo normal sin necesidad de reiniciar.
   - El endpoint `/api/setup` nunca más podrá ser accedido mientras exista una contraseña.

2. **Restablecimiento por CLI (Terminal)**:
   - No existe un botón de "Cambiar contraseña" en la web.
   - Si se pierde la contraseña, el administrador debe ejecutar el binario desde la terminal con un flag especial (ej. `./tablehub -reset-admin`).
   - El CLI le generará una contraseña temporal (o le pedirá teclear una nueva), la guardará en la base de datos, y cerrará el proceso. Así se garantiza seguridad física.

3. **Login Normal y Acceso**:
   - El cliente envía un `POST /api/login` con la contraseña.
   - Go compara con el hash Bcrypt.
   - Si coincide, emite el JWT y lo envía mediante `Set-Cookie: token=...; HttpOnly`.
   - Las peticiones protegidas usan la Cookie y son interceptadas por el `AuthMiddleware`.

4. **Cierre de Sesión (Logout)**:
   - Llamando a `POST /api/logout`, el servidor destruye la cookie (`Max-Age=0`).

## 3. Buenas Prácticas a Aplicar

- **Defecto Cero (Fail-Closed):** Cualquier error en validación rechaza la petición con 401.
- **Protección contra Fuerza Bruta Local:** Rate limiting en `/api/login` o penalización de tiempo por cada intento fallido.
