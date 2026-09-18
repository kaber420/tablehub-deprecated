# Reglas del Proyecto (Tablehub)

## Comandos de OpenCode CLI
 se muestra cómo utilizar el agente "plan" y los modelos Mimo y DeepSeek Free desde la terminal:

### Usar el Modo Plan
Para iniciar OpenCode con el agente de planificación:
```bash
opencode run --agent plan "Tu solicitud aquí"
```

### Usar los Modelos (Mimo y DeepSeek)
Para seleccionar un modelo específico, utiliza la bandera `-m`:

- **Usar Mimo-2.5-free:**
  ```bash
  opencode run -m opencode/mimo-v2.5-free "Tu solicitud aquí"
  ```

- **Usar DeepSeek-V4-flash-free:**
  ```bash  
  opencode run -m opencode/deepseek-v4-flash-free "Tu solicitud aquí"
  ```

### Combinar Agente y Modelo
Puedes combinar el agente y el modelo que prefieras utilizando ambas banderas simultáneamente (ej. `--agent [agente] -m [modelo]`).


## Especificaciones del Hardware
- **Pantalla / Placa:** Guition / AXS15231B JC3248W535 (ESP32-S3, 3.5 pulgadas, resolución 320x480).
- **Alimentación y Batería:** La placa JC3248W535 se alimenta principalmente por USB-C y no posee chip/pin de monitoreo de batería (ADC) integrado dedicado a una LiPo por defecto.

## Reglas Generales
- **Planificación Requerida:** No editar ni modificar código sin antes hacer un plan y presentarlo, a menos que se solicite explícitamente saltar la planificación.
- **Compilación de Frontend:** No ejecutar `pnpm install` de forma preventiva. Al compilar o ejecutar el frontend en `hub/web`, utilizar únicamente comandos como `pnpm run build`, `pnpm run dev` o `pnpm run preview`, asumiendo que los paquetes ya están instalados.
- **Edición de Documentos y Planes:** AL EDITAR DOCUMENTOS DE DISEÑO O PLANES (ej. en la carpeta docs/drafts), ESTÁ ESTRICTAMENTE PROHIBIDO sobrescribir y destruir párrafos anteriores para meter nuevas ideas. Siempre se debe AÑADIR (append) la nueva información bajo secciones como "Evolución", "Actualizaciones" o similares, para mantener el historial, el contexto y la esencia del plan original intactos.
- **Test e investigaciones exhaustivas:** usar opencode

## Comandos de Firmware (ESP32)

### Compilar y cargar al ESP32
El agente NO puede ejecutar `sudo` directamente. El comando correcto para compilar y cargar es:
```bash
sudo chmod 666 /dev/ttyACM0 && /home/kaber420/.platformio/penv/bin/pio run -d /home/kaber420/Documentos/proyectos/tablehub2/firmware -e esp32 -t upload
```
- `sudo chmod 666 /dev/ttyACM0` es obligatorio antes de cada upload para desbloquear el puerto serie.
- El agente debe **proponer este comando exacto** para que el usuario lo apruebe y ejecute.
- El agente NO debe intentar resolver el permiso del puerto de ninguna otra forma (no `usermod`, no `dialout`, no `udev rules`).