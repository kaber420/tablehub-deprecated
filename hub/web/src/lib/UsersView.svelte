<script>
  import { onMount } from 'svelte';
  import { Plus, Edit2, Trash2, KeyRound } from '@lucide/svelte';
  import { apiFetch } from './api.js';
  import PrimaryButton from './components/PrimaryButton.svelte';
  import Drawer from './components/Drawer.svelte';

  let users = [];
  let loading = true;
  let error = '';

  let showModal = false;
  let modalMode = 'add'; // 'add' or 'edit'
  let currentUser = { id: 0, name: '', role: 'waiter', pin: '', active: true };
  let saving = false;

  onMount(async () => {
    await loadUsers();
  });

  async function loadUsers() {
    loading = true;
    error = '';
    try {
      const res = await apiFetch('/api/users');
      if (res.ok) {
        users = await res.json();
      } else {
        error = 'No se pudieron cargar los usuarios.';
      }
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  function openAddModal() {
    modalMode = 'add';
    currentUser = { id: 0, name: '', role: 'waiter', pin: '', active: true };
    showModal = true;
  }

  function openEditModal(user) {
    modalMode = 'edit';
    currentUser = { ...user, pin: '' }; // pin is not loaded, only updated
    showModal = true;
  }

  async function saveUser() {
    saving = true;
    error = '';
    try {
      if (modalMode === 'add') {
        const res = await apiFetch('/api/users', {
          method: 'POST',
          body: JSON.stringify(currentUser)
        });
        if (!res.ok) throw new Error('Error al crear usuario');
      } else {
        // Edit details
        const updateData = {
          name: currentUser.name,
          role: currentUser.role,
          active: currentUser.active
        };
        if (currentUser.pin) {
          updateData.pin = currentUser.pin;
        }
        
        const res = await apiFetch(`/api/users/${currentUser.id}`, {
          method: 'PUT',
          body: JSON.stringify(updateData)
        });
        if (!res.ok) throw new Error('Error al actualizar usuario');
      }
      showModal = false;
      await loadUsers();
    } catch (e) {
      error = e.message;
    } finally {
      saving = false;
    }
  }

  async function deleteUser(id) {
    if (!confirm('¿Estás seguro de eliminar este usuario?')) return;
    try {
      const res = await apiFetch(`/api/users/${id}`, { method: 'DELETE' });
      if (res.ok) {
        await loadUsers();
      } else {
        alert('Error al eliminar usuario');
      }
    } catch (e) {
      console.error(e);
    }
  }
</script>

<div class="view-header">
  <h1>Gestión de Usuarios</h1>
</div>

{#if error && !showModal}
  <div class="error-msg">{error}</div>
{/if}

<div class="actions-bar">
  <PrimaryButton on:click={openAddModal}>
    <Plus size={18} /> Nuevo Usuario
  </PrimaryButton>
</div>

{#if loading}
  <div class="loading-state">Cargando usuarios...</div>
{:else}
  <div class="users-list">
    {#each users as user}
      <div class="user-card panel">
        <div class="user-info">
          <div class="user-header">
            <h3>{user.name}</h3>
            <span class="role-badge {user.role}">{user.role === 'admin' ? 'Administrador' : 'Mesero'}</span>
          </div>
          <div class="user-meta">
            <span class="status-indicator {user.active ? 'active' : 'inactive'}"></span>
            <span class="status-text">{user.active ? 'Activo' : 'Inactivo'}</span>
            <span class="meta-dot">•</span>
            <span class="date-text">Creado: {user.created_at ? new Date(user.created_at.replace(' ', 'T')).toLocaleDateString() : 'N/A'}</span>
          </div>
        </div>
        
        <div class="actions-cell">
          <button class="action-btn" title="Editar" on:click={() => openEditModal(user)}>
            <Edit2 size={18} />
          </button>
          <button class="action-btn delete" title="Eliminar" on:click={() => deleteUser(user.id)}>
            <Trash2 size={18} />
          </button>
        </div>
      </div>
    {:else}
      <div class="empty-state panel">
        <p>No hay usuarios registrados.</p>
        <PrimaryButton on:click={openAddModal} style="margin-top: 1rem;">
          Crear el primer usuario
        </PrimaryButton>
      </div>
    {/each}
  </div>
{/if}

<Drawer bind:show={showModal} title={modalMode === 'add' ? 'Nuevo Usuario' : 'Editar Usuario'}>
  {#if error}
    <div class="error-msg">{error}</div>
  {/if}

  <div class="form-group">
    <label for="name">Nombre</label>
    <input type="text" id="name" bind:value={currentUser.name} placeholder="Nombre del empleado" />
  </div>

  <div class="form-group">
    <label for="role">Rol</label>
    <select id="role" bind:value={currentUser.role}>
      <option value="waiter">Mesero</option>
      <option value="admin">Administrador</option>
    </select>
  </div>

  <div class="form-group">
    <label for="pin">
      {modalMode === 'add' ? 'PIN de Acceso (4-6 dígitos)' : 'Nuevo PIN (dejar vacío para no cambiar)'}
    </label>
    <div class="pin-input">
      <KeyRound size={18} class="pin-icon" />
      <input type="password" id="pin" bind:value={currentUser.pin} placeholder="Ej. 1234" />
    </div>
  </div>

  {#if modalMode === 'edit'}
    <div class="form-group checkbox-group">
      <input type="checkbox" id="active" bind:checked={currentUser.active} />
      <label for="active">Usuario Activo</label>
    </div>
  {/if}

  <div class="modal-actions">
    <button class="btn-cancel" on:click={() => showModal = false}>Cancelar</button>
    <PrimaryButton on:click={saveUser} disabled={saving}>
      {saving ? 'Guardando...' : 'Guardar'}
    </PrimaryButton>
  </div>
</Drawer>

<style>
  .view-header {
    text-align: left;
    margin-bottom: 2rem;
  }
  
  .view-header h1 {
    font-size: 2.5rem;
    margin-bottom: 0.5rem;
    background: linear-gradient(135deg, #fff 0%, #a5b4fc 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }
  
  .view-header p {
    color: var(--text-muted);
  }

  .actions-bar {
    display: flex;
    justify-content: flex-end;
    margin-bottom: 1.5rem;
  }

  .users-list {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .user-card {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1.5rem;
    border-radius: 16px;
  }

  .user-header {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-bottom: 0.5rem;
  }

  .user-header h3 {
    margin: 0;
    font-size: 1.25rem;
    color: var(--text-color);
  }

  .user-meta {
    display: flex;
    align-items: center;
    font-size: 0.9rem;
    color: var(--text-muted);
  }

  .meta-dot {
    margin: 0 0.5rem;
    color: rgba(255,255,255,0.2);
  }

  .role-badge {
    padding: 0.25rem 0.75rem;
    border-radius: 20px;
    font-size: 0.8rem;
    font-weight: 600;
  }

  .role-badge.admin {
    background: rgba(99, 102, 241, 0.2);
    color: #818cf8;
  }

  .role-badge.waiter {
    background: rgba(16, 185, 129, 0.2);
    color: #34d399;
  }

  .status-indicator {
    display: inline-block;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    margin-right: 0.5rem;
  }

  .status-indicator.active { background: #10b981; }
  .status-indicator.inactive { background: #ef4444; }

  .actions-cell {
    display: flex;
    gap: 0.5rem;
  }

  .action-btn {
    background: rgba(255,255,255,0.05);
    border: none;
    color: var(--text-color);
    width: 32px;
    height: 32px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: all 0.2s;
  }

  .action-btn:hover {
    background: rgba(255,255,255,0.15);
  }

  .action-btn.delete:hover {
    background: rgba(239, 68, 68, 0.2);
    color: #ef4444;
  }

  .empty-state {
    text-align: center;
    padding: 4rem 2rem;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
  }

  .form-group {
    margin-bottom: 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .form-group label {
    font-size: 0.9rem;
    color: var(--text-muted);
  }

  .pin-input {
    position: relative;
    display: flex;
    align-items: center;
  }

  .pin-input input {
    width: 100%;
    padding-left: 2.5rem;
  }

  .pin-icon {
    position: absolute;
    left: 0.75rem;
    color: var(--text-muted);
  }

  .checkbox-group {
    flex-direction: row;
    align-items: center;
    gap: 0.75rem;
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 1rem;
    margin-top: 2rem;
  }

  .btn-cancel {
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--panel-bg);
    border: var(--panel-border);
    box-shadow: var(--panel-shadow);
    color: var(--text-color);
    padding: 0.75rem 1.5rem;
    border-radius: 8px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-cancel:hover {
    background: var(--button-hover-bg, rgba(255,255,255,0.05));
    box-shadow: var(--panel-shadow-hover);
    transform: translateY(-2px);
  }

  .btn-cancel:active {
    box-shadow: var(--button-hover-shadow, inset 2px 2px 5px rgba(0,0,0,0.5));
    transform: translateY(1px);
  }

  .error-msg {
    color: #ef4444;
    padding: 1rem;
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.2);
    border-radius: 8px;
    margin-bottom: 1rem;
  }
</style>
