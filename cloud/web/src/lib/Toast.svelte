<script>
    import { toastStore } from './toast.js';
    import { fade, slide } from 'svelte/transition';

    let toast;

    toastStore.subscribe(value => {
        toast = value;
    });
</script>

{#if toast}
    <div class="toast-container" transition:slide>
        <div class="toast {toast.type}">
            {toast.message}
        </div>
    </div>
{/if}

<style>
    .toast-container {
        position: fixed;
        bottom: 24px;
        right: 24px;
        z-index: 9999;
    }
    .toast {
        padding: 14px 24px;
        border-radius: 12px;
        color: white;
        font-weight: 600;
        font-size: 0.95rem;
        box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.2);
        display: flex;
        align-items: center;
        gap: 8px;
    }
    .toast.success {
        background: linear-gradient(135deg, #10b981, #059669);
    }
    .toast.error {
        background: linear-gradient(135deg, #ef4444, #dc2626);
    }
</style>
