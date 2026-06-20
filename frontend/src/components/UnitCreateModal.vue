<script setup>
import { ref, computed, onMounted } from 'vue'
import { CreateUnit, GetAllUnits } from '../../wailsjs/go/main/App'
import UnitDetail from './UnitDetail.vue'

const props = defineProps(['faction'])
const emit = defineEmits(['close', 'created'])

const phase = ref('setup') // 'setup' | 'edit'
const newType = ref('')
const templateType = ref('')
const allUnits = ref([])
const loading = ref(false)
const error = ref(null)
const createdType = ref(null)

// dictionary автоматически из типа: пробелы → _
const derivedDict = computed(() => newType.value.trim().replace(/ +/g, '_'))

onMounted(async () => {
  try {
    allUnits.value = await GetAllUnits()
    if (allUnits.value.length > 0) templateType.value = allUnits.value[0].Type
  } catch (e) {
    error.value = String(e)
  }
})

async function create() {
  const t = newType.value.trim()
  if (!t) { error.value = 'Введите тип юнита'; return }
  if (!templateType.value) { error.value = 'Выберите шаблон'; return }
  loading.value = true
  error.value = null
  try {
    await CreateUnit(templateType.value, t, props.faction ?? '')
    createdType.value = t
    phase.value = 'edit'
  } catch (e) {
    error.value = String(e)
  } finally {
    loading.value = false
  }
}

function done() {
  emit('created', createdType.value)
}
function onSaved() {
  emit('created', createdType.value)
}
function onDeleted() {
  emit('close')
}
</script>

<template>
  <div class="overlay" @click.self="emit('close')">
    <div class="modal" :class="{ 'modal--wide': phase === 'edit' }">

      <div class="modal-header">
        <span class="modal-title">
          {{ phase === 'setup' ? 'Новый юнит' : `Редактирование: ${createdType}` }}
        </span>
        <div class="modal-header-actions">
          <button v-if="phase === 'edit'" class="btn-done" @click="done">Готово</button>
          <button class="modal-close" @click="phase === 'edit' ? done() : emit('close')">✕</button>
        </div>
      </div>

      <!-- ── Шаг 1: тип + шаблон ────────────────────────────────────── -->
      <div v-if="phase === 'setup'" class="modal-body">
        <label class="field">
          <span class="field-label">Тип нового юнита <span class="field-hint">(пробелы разрешены)</span></span>
          <input
            v-model="newType"
            class="field-input"
            placeholder="например: Roman Elite Archers"
            autofocus
            @keydown.enter="create"
          />
        </label>
        <div v-if="newType.trim()" class="dict-preview">
          dictionary → <code>{{ derivedDict }}</code>
        </div>
        <label class="field">
          <span class="field-label">Скопировать характеристики с</span>
          <select v-model="templateType" class="field-input">
            <option v-for="u in allUnits" :key="u.Type" :value="u.Type">
              {{ u.Name ? `${u.Name}  (${u.Type})` : u.Type }}
            </option>
          </select>
        </label>
        <div v-if="error" class="modal-error">{{ error }}</div>
        <div class="modal-actions">
          <button class="btn-create" :disabled="loading || allUnits.length === 0" @click="create">
            {{ loading ? 'Создание...' : 'Создать' }}
          </button>
          <button class="btn-cancel" @click="emit('close')">Отмена</button>
        </div>
      </div>

      <!-- ── Шаг 2: редактирование юнита ────────────────────────────── -->
      <div v-else class="modal-edit">
        <UnitDetail
          :unit-type="createdType"
          :faction="props.faction"
          @saved="onSaved"
          @deleted="onDeleted"
          @reverted="() => {}"
          @restored="() => {}"
        />
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

.modal--wide {
  width: 92vw;
  height: 90vh;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.modal-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
}
.modal-header-actions { display: flex; align-items: center; gap: 8px; }
.btn-done {
  background: var(--color-primary);
  color: var(--color-text);
  border: none;
  border-radius: 4px;
  padding: 5px 14px;
  font-size: 12px;
  cursor: pointer;
}
.btn-done:hover { background: var(--color-primary-hover); }
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

/* ── Setup body ── */
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
}
.field-input:focus { outline: none; border-color: var(--color-accent); }
.field-input option { background: var(--color-cell); }

.modal-error { font-size: 12px; color: var(--color-error); }
.field-hint { font-size: 10px; color: var(--color-text-muted); font-weight: 400; }
.dict-preview {
  font-size: 11px;
  color: var(--color-text-muted);
  background: var(--color-input-bg);
  border-radius: 4px;
  padding: 5px 8px;
  margin-top: -8px;
}
.dict-preview code { color: #3aba80; font-size: 11px; }

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

/* ── Edit body: UnitDetail занимает всё пространство ── */
.modal-edit {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
</style>
