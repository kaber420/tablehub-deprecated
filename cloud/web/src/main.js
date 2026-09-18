import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'

// Aplicar el tema global
document.body.classList.add('theme-glass');

const app = mount(App, {
  target: document.getElementById('app'),
})

export default app
