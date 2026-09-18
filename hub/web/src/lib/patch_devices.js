const fs = require('fs');
const file = '/home/kaber420/Documentos/proyectos/tablehub2/hub/web/src/lib/DevicesView.svelte';
let content = fs.readFileSync(file, 'utf8');

// 1. Add import
content = content.replace(
  "import { apiFetch } from './api.js';",
  "import { apiFetch } from './api.js';\n  import { devicesState } from './devices.svelte.js';"
);

// 2. Remove local state for devices and ws
content = content.replace(/let devices = \[\];\n/, '');
content = content.replace(/let ws;\n\n/, '');

// 3. Replace onMount and onDestroy
content = content.replace(
  /onMount\(async \(\) => \{\n    await fetchDevices\(\);\n    setupWebSocket\(\);\n  \}\);\n\n  onDestroy\(\(\) => \{\n    if \(ws\) ws.close\(\);\n  \}\);\n\n  async function fetchDevices\(\) [\s\S]*?\}\n\n  function setupWebSocket\(\) [\s\S]*?\}\n\n  function updateDeviceInList[\s\S]*?\}\n/,
  `onMount(async () => {
    loading = true;
    errorMsg = '';
    try {
      await devicesState.loadDevices();
    } catch (e) {
      errorMsg = 'Error al cargar dispositivos';
    } finally {
      loading = false;
    }
  });

  async function fetchDevices() {
    loading = true;
    errorMsg = '';
    try {
      await devicesState.loadDevices();
    } catch (e) {
      errorMsg = 'Error al cargar dispositivos';
    } finally {
      loading = false;
    }
  }
`
);

// 4. Replace devices references with devicesState.devices
content = content.replace(/\(devices \|\| \[\]\)/g, '(devicesState.devices || [])');
content = content.replace(/devices = devices\.filter/g, 'devicesState.devices = devicesState.devices.filter');

// 5. Check if we missed any devices array references
content = content.replace(/devices\.length/g, 'devicesState.devices.length');
content = content.replace(/devices\.some/g, 'devicesState.devices.some');

fs.writeFileSync(file, content);
console.log('Done');
