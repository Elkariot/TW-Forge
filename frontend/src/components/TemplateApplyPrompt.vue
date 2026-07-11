<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { CreateTemplate } from '../../wailsjs/go/main/App'

const { t } = useI18n()
const emit = defineEmits(['apply', 'cancel'])

const name = ref('')
const saving = ref(false)
const error = ref(null)

async function saveAndApply() {
  const trimmed = name.value.trim()
  if (!trimmed) { error.value = t('templates.name_required'); return }
  saving.value = true
  error.value = null
  try {
    await CreateTemplate(trimmed)
    emit('apply')
  } catch (e) {
    error.value = String(e)
  } finally {
    saving.value = false
  }
}

function applyWithoutSaving() {
  emit('apply')
}
</script>

<template>
  <div class="overlay" @click.self="emit('cancel')">
    <div class="modal">
      <div class="modal-header">
        <span class="modal-title">{{ $t('app.apply_template_prompt_title') }}</span>
        <button class="modal-close" @click="emit('cancel')">✕</button>
      </div>

      <div class="modal-body">
        <p class="modal-text">{{ $t('app.apply_template_prompt_text') }}</p>
        <input
          v-model="name"
          class="field-input"
          :placeholder="$t('templates.name_placeholder')"
          autofocus
          @keydown.enter="saveAndApply"
        />
        <div v-if="error" class="modal-error">{{ error }}</div>
        <div class="modal-actions">
          <button class="btn-create" :disabled="saving" @click="saveAndApply">
            {{ saving ? $t('common.saving') : $t('app.apply_save_and_apply') }}
          </button>
          <button class="btn-apply-plain" :disabled="saving" @click="applyWithoutSaving">
            {{ $t('app.apply_without_saving') }}
          </button>
          <button class="btn-cancel" :disabled="saving" @click="emit('cancel')">{{ $t('common.cancel') }}</button>
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
  z-index: 260;
}
.modal {
  background: var(--color-cell);
  border: 1px solid var(--color-border-soft);
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  width: 400px;
  box-shadow: 0 24px 60px rgba(0,0,0,0.5);
}
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  border-bottom: 1px solid var(--color-border);
}
.modal-title { font-size: 14px; font-weight: 600; color: var(--color-text); }
.modal-close {
  background: none; border: none; color: var(--color-text-muted);
  font-size: 16px; cursor: pointer; padding: 2px 4px; line-height: 1; border-radius: 3px;
}
.modal-close:hover { color: var(--color-accent); background: rgba(149,232,225,0.2); }

.modal-body { display: flex; flex-direction: column; gap: 12px; padding: 18px; }
.modal-text { font-size: 12px; color: var(--color-text-dim); line-height: 1.4; }
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
.modal-error { font-size: 12px; color: var(--color-error); }

.modal-actions { display: flex; flex-direction: column; gap: 6px; margin-top: 4px; }
.btn-create {
  background: var(--color-primary); color: var(--color-text); border: none;
  border-radius: 5px; padding: 8px 14px; font-size: 13px; cursor: pointer;
}
.btn-create:disabled { opacity: 0.5; cursor: default; }
.btn-create:not(:disabled):hover { background: var(--color-primary-hover); }
.btn-apply-plain {
  background: transparent; border: 1px solid var(--color-border-soft);
  color: var(--color-text-dim); border-radius: 5px; padding: 8px 14px; font-size: 13px; cursor: pointer;
}
.btn-apply-plain:hover { border-color: var(--color-text-dim); }
.btn-apply-plain:disabled { opacity: 0.5; cursor: default; }
.btn-cancel {
  background: transparent; border: 1px solid var(--color-border-soft);
  color: var(--color-text-muted); border-radius: 5px; padding: 8px 14px; font-size: 13px; cursor: pointer;
}
.btn-cancel:hover { border-color: var(--color-text-dim); color: var(--color-text-dim); }
.btn-cancel:disabled { opacity: 0.5; cursor: default; }
</style>
