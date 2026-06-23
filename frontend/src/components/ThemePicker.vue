<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { THEMES, applyTheme, initTheme } from '../themes.js'

const currentTheme = ref('papyrus')
const showPicker = ref(false)

onMounted(() => {
  currentTheme.value = initTheme()
  document.addEventListener('click', onDocClick)
})
onUnmounted(() => document.removeEventListener('click', onDocClick))

function onDocClick(e) {
  if (!e.target.closest('.theme-picker-wrap')) showPicker.value = false
}

function select(key) {
  currentTheme.value = applyTheme(key)
  showPicker.value = false
}
</script>

<template>
  <div class="theme-picker-wrap">
    <button class="btn-theme" @click.stop="showPicker = !showPicker">
      <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16">
        <path d="M12 3C7 3 3 7 3 12s4.03 9 9 9c.83 0 1.5-.67 1.5-1.5 0-.39-.15-.74-.39-1.01-.23-.26-.38-.61-.38-.99 0-.83.67-1.5 1.5-1.5H16c2.76 0 5-2.24 5-5 0-4.42-4.03-8-9-8zm-5.5 9c-.83 0-1.5-.67-1.5-1.5S5.67 9 6.5 9 8 9.67 8 10.5 7.33 12 6.5 12zm3-4C8.67 8 8 7.33 8 6.5S8.67 5 9.5 5s1.5.67 1.5 1.5S10.33 8 9.5 8zm5 0c-.83 0-1.5-.67-1.5-1.5S13.67 5 14.5 5s1.5.67 1.5 1.5S15.33 8 14.5 8zm3 4c-.83 0-1.5-.67-1.5-1.5S16.67 9 17.5 9s1.5.67 1.5 1.5-.67 1.5-1.5 1.5z"/>
      </svg>
    </button>
    <div v-if="showPicker" class="theme-dropdown">
      <button
        v-for="(theme, key) in THEMES"
        :key="key"
        class="theme-option"
        :class="{ active: currentTheme === key }"
        @click.stop="select(key)"
      >
        <span class="theme-swatch" :style="{ background: theme.vars['--color-primary'] }"></span>
        <span class="theme-swatch" :style="{ background: theme.vars['--color-secondary'] }"></span>
        <span class="theme-swatch" :style="{ background: theme.vars['--color-cell'] }"></span>
        {{ theme.label }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.theme-picker-wrap { position: relative; }
.btn-theme {
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  padding: 5px 7px;
  cursor: pointer;
  color: var(--color-text-dim);
  display: flex;
  align-items: center;
  transition: background 0.15s;
}
.btn-theme:hover { background: var(--color-cell); }

.theme-dropdown {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  background: var(--color-cell);
  border: 1px solid var(--color-border-soft);
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0,0,0,0.15);
  padding: 6px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  z-index: 500;
  min-width: 140px;
}
.theme-option {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 10px;
  border: none;
  background: transparent;
  border-radius: 5px;
  cursor: pointer;
  font-size: 12px;
  color: var(--color-text);
  text-align: left;
  transition: background 0.1s;
}
.theme-option:hover { background: var(--color-secondary); }
.theme-option.active { background: var(--color-primary); font-weight: 600; }
.theme-swatch {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  border: 1px solid rgba(0,0,0,0.1);
  flex-shrink: 0;
}
</style>
