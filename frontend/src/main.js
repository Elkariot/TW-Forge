import { createApp } from 'vue'
import { createI18n } from 'vue-i18n'
import App from './App.vue'
import './style.css'
import en from './locales/en.json'
import ru from './locales/ru.json'

const savedLang = localStorage.getItem('tw-forge-lang') || 'ru'

const i18n = createI18n({
  legacy: false,
  locale: savedLang,
  fallbackLocale: 'en',
  messages: { en, ru },
})

createApp(App).use(i18n).mount('#app')
