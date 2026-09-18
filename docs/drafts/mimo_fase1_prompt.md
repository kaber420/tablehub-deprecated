# Instrucciones para la Fase 1: Maqueta visual con datos mock

El objetivo es refactorizar el enorme `Organizations.svelte` (1000+ líneas) hacia una arquitectura modular con componentes separados, implementando la "Lista de Clientes" (SaaS) con datos mockeados.

## 1. Mover datos mock a fixture
- Extrae las variables `organizations`, `branches`, `hubs`, `saasPlans` de `Organizations.svelte` hacia un nuevo archivo `cloud/web/src/lib/mock/clients.js`.
- Exporta estas variables.

## 2. Refactorizar Organizations.svelte
- Elimina el tab de "Creador de Planes SaaS" (`plans_admin`) y toda su lógica asociada, ya que esto será una página/sección aparte en el futuro (o Fase 4). Quédate solo con la vista de `organizations`.
- Convierte la tabla principal en un componente `<ClientTable />` o mantenla aquí pero simplificada (reemplazando el código de la fila expandida por componentes más pequeños).
- **Añadir Paginación (mock):** Añade variables de estado para `currentPage` y `pageSize` y muestra controles de paginación debajo de la tabla. Calcula los items mostrados usando los mocks extraídos.
- **Añadir Filtros (mock):** La búsqueda de texto ya existe, asegúrate de que funcione combinada con la paginación.

## 3. Extraer BranchTable
- Actualmente, la vista expandida de la tabla muestra un grid complejo con sucursales y hubs (línea 363 a 451 aprox. en `Organizations.svelte`).
- Extrae ese grid a un nuevo componente `cloud/web/src/components/BranchTable.svelte` que reciba el `orgId` y renderice sus sucursales consumiendo del mock.

## 4. Tipos/Interfaces de API
- En `cloud/web/src/lib/mock/clients.js`, comenta la forma (TypeScript/JSDoc) que deberían tener `ClientListResponse` y `ClientDetail` para el endpoint real de la Fase 2.

## Output Esperado
- Edita `Organizations.svelte` en el sitio.
- Crea `lib/mock/clients.js`.
- Crea `components/BranchTable.svelte`.
- (Opcional) Crea `components/ClientTable.svelte` si decides extraer la tabla entera.

Usa tu capacidad de lectura y escritura para implementar estos cambios de una manera elegante y robusta.
