<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ListTemplates, GetTemplateState, CreateTemplate, SwitchTemplate, DeleteTemplate, ClearCurrentTemplate } from '../../wailsjs/go/main/App'

const { t } = useI18n()
// hasUnsavedChanges covers in-memory edits not yet Applied to disk at all. Once a
// template is active, every Apply() folds straight into it (see syncCurrentTemplate on
// the Go side), so there's normally nothing left to lose on switch except these —
// unapplied edits that would be discarded by the reload a real template switch causes.
const props = defineProps({ hasUnsavedChanges: { type: Boolean, default: false } })
const emit = defineEmits(['switched'])

const templates = ref([])
const state = ref({ Current: '', Dirty: false })
const open = ref(false)
const loading = ref(false)
const error = ref(null)

const creating = ref(false)
const newName = ref('')

// Название шаблона, на который пользователь хочет переключиться, пока не решено,
// что делать с несохранёнными изменениями (см. pickTemplate).
const pendingSwitch = ref(null)
const switchSaveName = ref('')
const switchSaving = ref(false)

async function refresh() {
  try {
    const [list, st] = await Promise.all([ListTemplates(), GetTemplateState()])
    templates.value = list ?? []
    state.value = st
  } catch (e) {
    error.value = String(e)
  }
}
defineExpose({ refresh })

onMounted(refresh)

function toggleOpen() {
  open.value = !open.value
  if (open.value) { creating.value = false; error.value = null }
}

function startCreate() {
  creating.value = true
  newName.value = ''
  error.value = null
}

async function submitCreate() {
  const name = newName.value.trim()
  if (!name) return
  loading.value = true
  error.value = null
  try {
    await CreateTemplate(name)
    creating.value = false
    open.value = false
    await refresh()
  } catch (e) {
    error.value = String(e)
  } finally {
    loading.value = false
  }
}

// name === '' means "no template" — that switch never touches game files (see
// ClearTemplate on the Go side), so it's always safe and never needs confirmation.
function pickTemplate(name) {
  if (name === (state.value.Current || '')) { open.value = false; return }
  open.value = false
  if (name === '') {
    clearTemplate()
    return
  }
  if (props.hasUnsavedChanges) {
    pendingSwitch.value = name
    switchSaveName.value = ''
  } else {
    doSwitch(name)
  }
}

async function clearTemplate() {
  loading.value = true
  error.value = null
  try {
    await ClearCurrentTemplate()
    await refresh()
  } catch (e) {
    error.value = String(e)
  } finally {
    loading.value = false
  }
}

async function doSwitch(name) {
  loading.value = true
  error.value = null
  try {
    await SwitchTemplate(name)
    await refresh()
    emit('switched')
  } catch (e) {
    error.value = String(e)
  } finally {
    loading.value = false
    pendingSwitch.value = null
  }
}

async function saveThenSwitch() {
  const name = switchSaveName.value.trim()
  if (!name) return
  switchSaving.value = true
  error.value = null
  try {
    await CreateTemplate(name)
    await doSwitch(pendingSwitch.value)
  } catch (e) {
    error.value = String(e)
  } finally {
    switchSaving.value = false
  }
}

function discardThenSwitch() {
  doSwitch(pendingSwitch.value)
}

function cancelSwitch() {
  pendingSwitch.value = null
}

async function removeTemplate(name) {
  if (!confirm(t('templates.delete_confirm', { name }))) return
  try {
    await DeleteTemplate(name)
    await refresh()
  } catch (e) {
    error.value = String(e)
  }
}
</script>

<template>
  <div class="tpl-bar">
    <button class="tpl-trigger" @click="toggleOpen">
      <span class="tpl-label">{{ $t('templates.label') }}:</span>
      <span class="tpl-name" :class="{ 'tpl-name--active': state.Current }">
        {{ state.Current || $t('templates.none') }}
      </span>
      <span v-if="state.Current && hasUnsavedChanges" class="tpl-dirty-dot" :title="$t('templates.dirty_hint')"></span>
      <span class="tpl-chevron">▾</span>
    </button>

    <!-- Прозрачная подложка только чтобы ловить клик вне поповера — сам поповер
         позиционируется абсолютно относительно .tpl-bar (см. ниже), а не внутри
         этой fixed-подложки: иначе она стала бы контейнером для position:absolute
         и "top: 100%" считался бы от высоты всего окна, а не от кнопки. -->
    <div v-if="open" class="tpl-popover-backdrop" @click="open = false"></div>

    <div v-if="open" class="tpl-popover" @click.stop>
      <div v-if="error" class="tpl-error">{{ error }}</div>

      <template v-if="!creating">
        <ul class="tpl-list">
          <li :class="{ active: !state.Current }" @click="pickTemplate('')">
            <span class="tpl-item-name tpl-item-none">{{ $t('templates.none_option') }}</span>
          </li>
          <li
            v-for="tpl in templates"
            :key="tpl.Name"
            :class="{ active: tpl.Name === state.Current }"
            @click="pickTemplate(tpl.Name)"
          >
            <span class="tpl-item-name">{{ tpl.Name }}</span>
            <button class="tpl-item-delete" @click.stop="removeTemplate(tpl.Name)" title="✕">✕</button>
          </li>
        </ul>
        <div v-if="templates.length === 0" class="tpl-empty">{{ $t('templates.empty') }}</div>
        <button class="btn-add-template" @click="startCreate">+ {{ $t('templates.create') }}</button>
      </template>

      <div v-else class="tpl-create">
        <input
          v-model="newName"
          class="tpl-input"
          :placeholder="$t('templates.name_placeholder')"
          autofocus
          @keydown.enter="submitCreate"
        />
        <div class="tpl-create-actions">
          <button class="btn-small-add" :disabled="loading" @click="submitCreate">{{ $t('common.save') }}</button>
          <button class="btn-cancel-inline" @click="creating = false">✕</button>
        </div>
      </div>
    </div>

    <!-- Полноэкранное подтверждение при переключении с несохранёнными изменениями -->
    <div v-if="pendingSwitch" class="tpl-switch-overlay">
      <div class="tpl-switch-modal">
        <div class="tpl-switch-title">{{ $t('templates.switch_confirm_title') }}</div>
        <p class="tpl-switch-text">{{ $t('templates.switch_confirm_text', { name: pendingSwitch }) }}</p>

        <label class="tpl-switch-field">
          <span>{{ $t('templates.name_placeholder') }}</span>
          <input v-model="switchSaveName" class="tpl-input" @keydown.enter="saveThenSwitch" autofocus />
        </label>

        <div class="tpl-switch-actions">
          <button class="btn-create" :disabled="switchSaving || !switchSaveName.trim()" @click="saveThenSwitch">
            {{ switchSaving ? $t('common.saving') : $t('templates.save_and_switch') }}
          </button>
          <button class="btn-restore-solid" :disabled="switchSaving" @click="discardThenSwitch">
            {{ $t('templates.discard_and_switch') }}
          </button>
          <button class="btn-cancel" :disabled="switchSaving" @click="cancelSwitch">{{ $t('common.cancel') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tpl-bar { position: relative; }
.tpl-trigger {
  display: flex;
  align-items: center;
  gap: 6px;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  padding: 5px 9px;
  cursor: pointer;
  color: var(--color-text-dim);
  font-size: 11px;
  transition: background 0.15s;
}
.tpl-trigger:hover { background: var(--color-cell); }
.tpl-label { color: var(--color-text-muted); }
.tpl-name { font-weight: 600; color: var(--color-text-dim); }
.tpl-name--active { color: var(--color-accent); }
.tpl-dirty-dot {
  width: 6px; height: 6px; border-radius: 50%;
  background: var(--color-error);
  flex-shrink: 0;
}
.tpl-chevron { font-size: 9px; color: var(--color-text-muted); }

.tpl-popover-backdrop { position: fixed; inset: 0; z-index: 250; }
.tpl-popover {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  width: 240px;
  background: var(--color-cell);
  border: 1px solid var(--color-border-soft);
  border-radius: 8px;
  box-shadow: 0 12px 32px rgba(0,0,0,0.35);
  padding: 10px;
  z-index: 251;
}
.tpl-error { font-size: 11px; color: var(--color-error); margin-bottom: 6px; }
.tpl-empty { font-size: 11px; color: var(--color-text-muted); text-align: center; padding: 4px 0 8px; }
.tpl-item-none { font-style: italic; color: var(--color-text-muted); }
.tpl-list li.active .tpl-item-none { color: var(--color-accent); font-style: normal; }

.tpl-list { list-style: none; max-height: 220px; overflow-y: auto; margin-bottom: 8px; border-bottom: 1px solid var(--color-border-soft); padding-bottom: 6px; }
.tpl-list li {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 8px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  color: var(--color-text-dim);
}
.tpl-list li:hover { background: var(--color-input-bg); }
.tpl-list li.active { color: var(--color-accent); font-weight: 600; }
.tpl-item-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tpl-item-delete {
  visibility: hidden;
  background: none; border: none; color: var(--color-text-muted);
  cursor: pointer; font-size: 11px; padding: 0 2px; flex-shrink: 0;
}
.tpl-list li:hover .tpl-item-delete { visibility: visible; }
.tpl-item-delete:hover { color: var(--color-error); }

.btn-add-template {
  width: 100%;
  box-sizing: border-box;
  background: var(--color-primary);
  border: none;
  color: var(--color-text);
  border-radius: 4px;
  padding: 6px 0;
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
}
.btn-add-template:hover { background: var(--color-primary-hover); }

.tpl-create { display: flex; flex-direction: column; gap: 6px; }
.tpl-input {
  width: 100%;
  box-sizing: border-box;
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text);
  font-size: 12px;
  padding: 5px 8px;
}
.tpl-input:focus { outline: none; border-color: var(--color-accent); }
.tpl-create-actions { display: flex; gap: 4px; }
.btn-small-add { background: none; border: 1px dashed var(--color-border-soft); color: var(--color-text-muted); border-radius: 3px; padding: 3px 10px; font-size: 11px; cursor: pointer; }
.btn-small-add:hover { border-color: #3aba80; color: #3aba80; }
.btn-cancel-inline { background: none; border: none; color: var(--color-text-muted); cursor: pointer; font-size: 12px; padding: 2px 4px; }
.btn-cancel-inline:hover { color: var(--color-accent); }

/* Полноэкранный диалог переключения */
.tpl-switch-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.65);
  display: flex; align-items: center; justify-content: center; z-index: 260;
}
.tpl-switch-modal {
  background: var(--color-cell);
  border: 1px solid var(--color-border-soft);
  border-radius: 10px;
  padding: 20px 22px;
  width: 360px;
  box-shadow: 0 24px 60px rgba(0,0,0,0.5);
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.tpl-switch-title { font-size: 14px; font-weight: 600; color: var(--color-text); }
.tpl-switch-text { font-size: 12px; color: var(--color-text-dim); line-height: 1.4; }
.tpl-switch-field { display: flex; flex-direction: column; gap: 4px; font-size: 11px; color: var(--color-text-muted); }
.tpl-switch-actions { display: flex; flex-direction: column; gap: 6px; margin-top: 4px; }
.btn-create {
  background: var(--color-primary); color: var(--color-text); border: none;
  border-radius: 5px; padding: 8px 14px; font-size: 12px; cursor: pointer;
}
.btn-create:disabled { opacity: 0.5; cursor: default; }
.btn-create:not(:disabled):hover { background: var(--color-primary-hover); }
.btn-restore-solid {
  background: transparent; color: var(--color-error); border: 1px solid var(--color-error);
  border-radius: 5px; padding: 8px 14px; font-size: 12px; cursor: pointer;
}
.btn-restore-solid:hover { background: var(--color-error); color: var(--color-text); }
.btn-restore-solid:disabled { opacity: 0.5; cursor: default; }
.btn-cancel {
  background: transparent; border: 1px solid var(--color-border-soft);
  color: var(--color-text-dim); border-radius: 5px; padding: 8px 14px; font-size: 12px; cursor: pointer;
}
.btn-cancel:hover { border-color: var(--color-text-dim); }
.btn-cancel:disabled { opacity: 0.5; cursor: default; }
</style>
