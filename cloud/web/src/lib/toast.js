import { writable } from 'svelte/store';

export const toastStore = writable(null);

export function showToast(message, type = 'success') {
    toastStore.set({ message, type });
    setTimeout(() => {
        toastStore.set(null);
    }, 4000);
}
