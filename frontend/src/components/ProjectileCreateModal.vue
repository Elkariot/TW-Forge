<script setup>
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { CreateProjectile } from '../../wailsjs/go/main/App'

const { t } = useI18n()

const props = defineProps(['projectiles'])
const emit = defineEmits(['close', 'created'])

const newName = ref('')
const templateSearch = ref('')
const templateName = ref(props.projectiles[0]?.Name ?? '')
const loading = ref(false)
const error = ref(null)

const filtered = computed(() => {
  const q = templateSearch.value.trim().toLowerCase()
  if (!q) return props.projectiles
  return props.projectiles.filter(p => p.Name.toLowerCase().includes(q))
})

watch(filtered, list => {
  if (list.length > 0 && !list.find(p => p.Name === templateName.value)) {
    templateName.value = list[0].Name
  }
})

async function create() {
  const name = newName.value.trim()
  if (!name) { error.value = t('projectile.error_enter_name'); return }
  if (!templateName.value) { error.value = t('projectile.error_select_template'); return }
  loading.value = true
  error.value = null
  try {
    await CreateProjectile(templateName.value, name)
    emit('created', name)
  } catch (e) {
    error.value = String(e)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="overlay" @click.self="emit('close')">
    <div class="modal">

      <div class="modal-header">
        <span class="modal-title">{{ $t('projectile.new_projectile_title') }}</span>
        <button class="modal-close" @click="emit('close')">✕</button>
      </div>

      <div class="modal-body">
        <label class="field">
          <span class="field-label">{{ $t('projectile.new_projectile_name') }}</span>
          <input
            v-model="newName"
            class="field-input"
            :placeholder="$t('projectile.new_projectile_placeholder')"
            autofocus
            @keydown.enter="create"
          />
        </label>

        <label class="field">
          <span class="field-label">{{ $t('projectile.new_projectile_template_hint') }}</span>
          <input
            v-model="templateSearch"
            class="field-input search-input"
            :placeholder="$t('common.search_placeholder')"
          />
          <select v-model="templateName" class="field-input template-select" size="8">
            <option v-for="p in filtered" :key="p.Name" :value="p.Name">{{ p.Name }}</option>
          </select>
          <span class="field-count">{{ filtered.length }} / {{ projectiles.length }}</span>
        </label>

        <div v-if="error" class="modal-error">{{ error }}</div>
        <div class="modal-actions">
          <button class="btn-create" :disabled="loading || projectiles.length === 0" @click="create">
            {{ loading ? $t('projectile.creating') : $t('projectile.create') }}
          </button>
          <button class="btn-cancel" @click="emit('close')">{{ $t('common.cancel') }}</button>
        </div>
      </div>

    </div>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.65);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
}

.modal {
  background: var(--color-cell);
  border: 1px solid var(--color-border-soft);
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  width: 420px;
  max-height: 90vh;
  overflow: hidden;
  box-shadow: 0 24px 60px rgba(0,0,0,0.5);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.modal-title { font-size: 14px; font-weight: 600; color: var(--color-text); }
.modal-close {
  background: none;
  border: none;
  color: var(--color-text-muted);
  font-size: 16px;
  cursor: pointer;
  padding: 2px 4px;
  line-height: 1;
  border-radius: 3px;
}
.modal-close:hover { color: var(--color-accent); background: rgba(149,232,225,0.2); }

.modal-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 20px 18px;
}
.field { display: flex; flex-direction: column; gap: 5px; }
.field-label { font-size: 11px; color: var(--color-text-dim); }
.field-input {
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text);
  font-size: 13px;
  padding: 6px 8px;
  width: 100%;
  box-sizing: border-box;
}
.field-input:focus { outline: none; border-color: var(--color-accent); }
.field-input option { background: var(--color-cell); }
.search-input { margin-bottom: 4px; }
.template-select { height: 160px; font-family: monospace; }
.field-count { font-size: 10px; color: var(--color-text-muted); align-self: flex-end; margin-top: 2px; }

.modal-error { font-size: 12px; color: var(--color-error); }

.modal-actions { display: flex; gap: 8px; margin-top: 4px; }
.btn-create {
  background: var(--color-primary);
  color: var(--color-text);
  border: none;
  border-radius: 5px;
  padding: 8px 20px;
  font-size: 13px;
  cursor: pointer;
}
.btn-create:disabled { opacity: 0.5; cursor: default; }
.btn-create:not(:disabled):hover { background: var(--color-primary-hover); }
.btn-cancel {
  background: transparent;
  border: 1px solid var(--color-border-soft);
  color: var(--color-text-dim);
  border-radius: 5px;
  padding: 8px 16px;
  font-size: 13px;
  cursor: pointer;
}
.btn-cancel:hover { border-color: var(--color-text-dim); color: var(--color-text-dim); }
</style>
