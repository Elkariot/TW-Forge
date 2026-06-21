<script setup>
import { ref, computed } from 'vue'
import {
  GetFactions, GetUnitsByFaction, GetDeletedUnits, Save, HasUnsavedChanges, CopyUnit,
  GetM2TWFactions, GetM2TWUnitsByFaction, GetM2TWDeletedUnits, SaveM2TW, M2TWHasUnsavedChanges,
  CopyM2TWUnit,
} from '../wailsjs/go/main/App'
import UnitDetail from './components/UnitDetail.vue'
import UnitCreateModal from './components/UnitCreateModal.vue'
import M2TWUnitDetail from './components/M2TWUnitDetail.vue'
import M2TWUnitCreateModal from './components/M2TWUnitCreateModal.vue'
import SetupScreen from './components/SetupScreen.vue'
import RecruitEditor from './components/RecruitEditor.vue'
import BuildingEditor from './components/BuildingEditor.vue'
import M2TWBuildingEditor from './components/M2TWBuildingEditor.vue'
import M2TWRecruitEditor from './components/M2TWRecruitEditor.vue'
import ThemePicker from './components/ThemePicker.vue'

const screen = ref('setup') // 'setup' | 'editor'
const gameVersion = ref(null) // 0 = Medieval, 1 = Rome
const isM2TW = computed(() => gameVersion.value === 0)

async function onGameReady(game) {
  gameVersion.value = game
  screen.value = 'editor'
  factions.value = isM2TW.value ? await GetM2TWFactions() : await GetFactions()
}

function goBack() {
  screen.value = 'setup'
  gameVersion.value = null
  factions.value = []
  selectedFaction.value = null
  unitGroups.value = {}
  selectedUnitType.value = null
  deletedUnits.value = []
  showDeleted.value = false
  hasChanges.value = false
  applyError.value = null
  copyDialog.value = null
  activeTab.value = 'units'
}

const activeTab = ref('units') // 'units' | 'buildings'

const factions = ref([])
const selectedFaction = ref(null)
const unitGroups = ref({})
const unitSearch = ref('')
const filteredUnitGroups = computed(() => {
  const q = unitSearch.value.trim().toLowerCase()
  if (!q) return unitGroups.value
  const result = {}
  for (const [category, classes] of Object.entries(unitGroups.value)) {
    const filteredClasses = {}
    for (const [cls, units] of Object.entries(classes)) {
      const matched = units.filter(u =>
        (u.Name || '').toLowerCase().includes(q) || u.Type.toLowerCase().includes(q)
      )
      if (matched.length > 0) filteredClasses[cls] = matched
    }
    if (Object.keys(filteredClasses).length > 0) result[category] = filteredClasses
  }
  return result
})
const selectedUnitType = ref(null)
const hasChanges = ref(false)
const isApplied = ref(false)
const applying = ref(false)
const applyError = ref(null)
const showDeleted = ref(false)
const deletedUnits = ref([])
const showCreateModal = ref(false)


async function selectFaction(faction) {
  selectedFaction.value = faction.Name
  unitSearch.value = ''
  unitGroups.value = isM2TW.value
    ? await GetM2TWUnitsByFaction(faction.Name)
    : await GetUnitsByFaction(faction.Name)
  selectedUnitType.value = null
  if (showDeleted.value) deletedUnits.value = isM2TW.value
    ? await GetM2TWDeletedUnits()
    : await GetDeletedUnits()
}

async function toggleShowDeleted() {
  showDeleted.value = !showDeleted.value
  if (showDeleted.value) {
    deletedUnits.value = isM2TW.value ? await GetM2TWDeletedUnits() : await GetDeletedUnits()
  } else {
    deletedUnits.value = []
    if (deletedUnits.value.includes(d => d.Type === selectedUnitType.value)) selectedUnitType.value = null
  }
}

function selectUnit(unit) {
  selectedUnitType.value = unit.Type
}

async function checkChanges() {
  hasChanges.value = isM2TW.value ? await M2TWHasUnsavedChanges() : await HasUnsavedChanges()
}

async function onUnitSaved() { isApplied.value = false; await checkChanges() }

async function onUnitDeleted() {
  isApplied.value = false
  if (selectedFaction.value) {
    unitGroups.value = isM2TW.value
      ? await GetM2TWUnitsByFaction(selectedFaction.value)
      : await GetUnitsByFaction(selectedFaction.value)
  }
  selectedUnitType.value = null
  await checkChanges()
  if (showDeleted.value) {
    deletedUnits.value = isM2TW.value ? await GetM2TWDeletedUnits() : await GetDeletedUnits()
  }
}

async function onUnitRestored() {
  isApplied.value = false
  if (selectedFaction.value) {
    unitGroups.value = isM2TW.value
      ? await GetM2TWUnitsByFaction(selectedFaction.value)
      : await GetUnitsByFaction(selectedFaction.value)
  }
  if (showDeleted.value) {
    deletedUnits.value = isM2TW.value ? await GetM2TWDeletedUnits() : await GetDeletedUnits()
  }
  selectedUnitType.value = null
  await checkChanges()
}

async function onUnitCreated(newType) {
  isApplied.value = false
  showCreateModal.value = false
  if (selectedFaction.value) {
    unitGroups.value = isM2TW.value
      ? await GetM2TWUnitsByFaction(selectedFaction.value)
      : await GetUnitsByFaction(selectedFaction.value)
  }
  selectedUnitType.value = newType
  await checkChanges()
}

// ── Диалог копирования ───────────────────────────────────────
const copyDialog = ref(null) // { unit }
const copyTargetFaction = ref('')

function openCopyDialog(unit) {
  copyTargetFaction.value = selectedFaction.value ?? factions.value[0]?.Name ?? ''
  copyDialog.value = { unit }
}

async function confirmCopy() {
  const unit = copyDialog.value?.unit
  if (!unit || !copyTargetFaction.value) return
  copyDialog.value = null
  try {
    const newType = isM2TW.value
      ? await CopyM2TWUnit(unit.Type, copyTargetFaction.value)
      : await CopyUnit(unit.Type, copyTargetFaction.value)
    if (copyTargetFaction.value === selectedFaction.value) {
      unitGroups.value = isM2TW.value
        ? await GetM2TWUnitsByFaction(selectedFaction.value)
        : await GetUnitsByFaction(selectedFaction.value)
    }
    isApplied.value = false
    selectedUnitType.value = newType
    await checkChanges()
  } catch (e) {
    applyError.value = `Ошибка копирования: ${e}`
  }
}

async function applyToGame() {
  applying.value = true
  applyError.value = null
  try {
    isM2TW.value ? await SaveM2TW() : await Save()
    await checkChanges()
    isApplied.value = true
  } catch (e) {
    applyError.value = String(e)
  } finally {
    applying.value = false
  }
}
</script>

<template>
  <SetupScreen v-if="screen === 'setup'" @ready="onGameReady" />

  <div v-else class="layout">

    <!-- Топбар -->
    <div class="topbar">
      <div class="topbar-left">
        <button class="btn-back" @click="goBack" title="Вернуться к выбору игры">
          <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16">
            <path d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z"/>
          </svg>
        </button>
        <span class="topbar-title">TW Forge</span>
      </div>
      <div class="topbar-actions">
        <ThemePicker />
        <span v-if="applyError" class="topbar-error">{{ applyError }}</span>
        <button
          class="btn-apply"
          :class="{ 'btn-apply--done': isApplied && hasChanges }"
          :disabled="applying || !hasChanges"
          @click="applyToGame"
        >
          {{ applying ? 'Запись...' : isApplied ? '✓ Применено' : 'Применить к игре' }}
        </button>
      </div>
    </div>

    <div class="below-topbar">

    <!-- Nav Rail -->
    <nav class="nav-rail">
      <button class="nav-btn" :class="{ active: activeTab === 'units' }" title="Юниты" @click="activeTab = 'units'">
        <svg viewBox="0 0 24 24" fill="currentColor" width="20" height="20">
          <path d="M12 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8zm-7 9a7 7 0 0 1 14 0H5z"/>
        </svg>
      </button>
      <button class="nav-btn" :class="{ active: activeTab === 'buildings' }" title="Здания" @click="activeTab = 'buildings'">
        <svg viewBox="0 0 24 24" fill="currentColor" width="20" height="20">
          <path d="M3 21h18v-2H3v2zM5 9.5v9.5h3V9.5H5zm5.5 0v9.5h3V9.5h-3zM16 9.5v9.5h3V9.5h-3zM2 7.5l10-5 10 5v1.5H2V7.5z"/>
        </svg>
      </button>
      <button class="nav-btn" :class="{ active: activeTab === 'recruit' }" title="Найм" @click="activeTab = 'recruit'">
        <svg viewBox="0 0 24 24" fill="currentColor" width="20" height="20">
          <path d="M12 1L3 5v6c0 5.5 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4z"/>
        </svg>
      </button>
    </nav>

    <!-- Вкладка: Юниты -->
    <template v-if="activeTab === 'units'">

    <!-- Фракции -->
    <aside class="sidebar">
      <h2>Фракции</h2>
      <ul>
        <li
          v-for="faction in factions"
          :key="faction.Name"
          :class="{ active: selectedFaction === faction.Name }"
          @click="selectFaction(faction)"
        >
          {{ faction.DisplayName || faction.Name }}
        </li>
      </ul>
    </aside>

    <!-- Список юнитов -->
    <div class="unit-list" v-if="selectedFaction">
      <div class="unit-list-header">
        <h2>{{ selectedFaction }}</h2>
        <div class="unit-list-actions">
          <button
            class="toggle-deleted-btn"
            :class="{ active: showDeleted }"
            @click="toggleShowDeleted"
            title="Показать удалённых юнитов"
          >🗑</button>
          <button
            class="unit-add-btn"
            @click="showCreateModal = true"
            title="Добавить юнита"
          >+</button>
        </div>
      </div>
      <div class="unit-search-wrap">
        <input
          v-model="unitSearch"
          class="unit-search"
          type="text"
          placeholder="Поиск юнитов..."
        />
        <button v-if="unitSearch" class="unit-search-clear" @click="unitSearch = ''" title="Очистить">✕</button>
      </div>
      <div v-if="unitSearch && Object.keys(filteredUnitGroups).length === 0" class="unit-search-empty">
        Ничего не найдено
      </div>
      <div v-for="(classes, category) in filteredUnitGroups" :key="category" class="category">
        <h3>{{ category }}</h3>
        <div v-for="(units, cls) in classes" :key="cls" class="class-group">
          <h4>{{ cls }}</h4>
          <ul>
            <li
              v-for="unit in units"
              :key="unit.Type"
              :class="{ active: selectedUnitType === unit.Type }"
              @click="selectUnit(unit)"
            >
              <span class="unit-name">{{ unit.Name || unit.Type }}</span>
              <button class="unit-copy-btn" @click.stop="openCopyDialog(unit)" title="Копировать юнит">⧉</button>
            </li>
          </ul>
        </div>
      </div>

      <!-- Удалённые юниты -->
      <div v-if="showDeleted && deletedUnits.length > 0" class="category deleted-section">
        <h3>Удалённые</h3>
        <ul class="class-group">
          <li
            v-for="unit in deletedUnits"
            :key="unit.Type"
            :class="{ active: selectedUnitType === unit.Type }"
            class="deleted-unit"
            @click="selectUnit(unit)"
          >
            <span class="unit-name">{{ unit.Name || unit.Type }}</span>
          </li>
        </ul>
      </div>
      <div v-else-if="showDeleted && deletedUnits.length === 0" class="deleted-empty">
        Нет удалённых юнитов
      </div>
    </div>
    <div class="unit-list hint-panel" v-else>
      <p class="hint">Выберите фракцию слева</p>
    </div>

    <!-- Детали юнита -->
    <div class="detail-panel">
      <M2TWUnitDetail
        v-if="isM2TW"
        :unit-type="selectedUnitType"
        :faction="selectedFaction"
        @saved="onUnitSaved"
        @reverted="onUnitSaved"
        @deleted="onUnitDeleted"
        @restored="onUnitRestored"
      />
      <UnitDetail
        v-else
        :unit-type="selectedUnitType"
        :faction="selectedFaction"
        @saved="onUnitSaved"
        @reverted="onUnitSaved"
        @deleted="onUnitDeleted"
        @restored="onUnitRestored"
      />
    </div>

    <!-- Модал создания юнита -->
    <M2TWUnitCreateModal
      v-if="showCreateModal && isM2TW"
      :faction="selectedFaction"
      @close="showCreateModal = false"
      @created="onUnitCreated"
    />
    <UnitCreateModal
      v-else-if="showCreateModal"
      :faction="selectedFaction"
      @close="showCreateModal = false"
      @created="onUnitCreated"
    />

    </template>

    <!-- Вкладка: Здания -->
    <M2TWBuildingEditor v-else-if="activeTab === 'buildings' && isM2TW" class="tab-fill" @changed="onUnitSaved" />
    <BuildingEditor v-else-if="activeTab === 'buildings'" class="tab-fill" @changed="onUnitSaved" />

    <!-- Вкладка: Найм -->
    <M2TWRecruitEditor v-else-if="activeTab === 'recruit' && isM2TW" class="tab-fill" />
    <RecruitEditor v-else-if="activeTab === 'recruit'" />

    </div> <!-- below-topbar -->
  </div>

  <!-- Диалог копирования юнита -->
  <div v-if="copyDialog" class="copy-overlay" @click.self="copyDialog = null">
    <div class="copy-modal">
      <div class="copy-modal-title">Копировать юнит</div>
      <div class="copy-modal-unit">{{ copyDialog.unit.Name || copyDialog.unit.Type }}</div>
      <label class="copy-modal-label">Скопировать во фракцию</label>
      <select v-model="copyTargetFaction" class="copy-modal-select">
        <option v-for="f in factions" :key="f.Name" :value="f.Name">
          {{ f.DisplayName || f.Name }}
        </option>
      </select>
      <div class="copy-modal-actions">
        <button class="copy-btn-confirm" @click="confirmCopy">Копировать</button>
        <button class="copy-btn-cancel" @click="copyDialog = null">Отмена</button>
      </div>
    </div>
  </div>
</template>


<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: sans-serif; background: var(--color-bg); color: var(--color-text); }

.layout { display: flex; flex-direction: column; height: 100vh; overflow: hidden; }

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  height: 44px;
  background: var(--color-secondary);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.topbar-left { display: flex; align-items: center; gap: 8px; }
.topbar-title { font-size: 13px; font-weight: 600; color: var(--color-text-dim); }
.topbar-actions { display: flex; align-items: center; gap: 10px; }
.topbar-error { font-size: 12px; color: var(--color-error); }
.btn-back {
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
.btn-back:hover { background: var(--color-cell); color: var(--color-text); }
.btn-apply {
  background: var(--color-primary);
  color: var(--color-text);
  border: 1px solid var(--color-primary-hover);
  padding: 6px 14px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
}
.btn-apply:hover { background: var(--color-primary-hover); }
.btn-apply:disabled { opacity: 0.5; cursor: default; }
.btn-apply--done { background: #4a9e6b; border-color: #3d8a5c; }
.btn-apply--done:hover { background: #3d8a5c; }

.below-topbar { display: flex; flex: 1; overflow: hidden; }

/* Nav Rail */
.nav-rail {
  width: 48px;
  flex-shrink: 0;
  background: var(--color-secondary);
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 8px 0;
  gap: 4px;
}
.nav-btn {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  border: none;
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s, color 0.15s;
}
.nav-btn:hover { background: var(--color-cell); color: #5a3050; }
.nav-btn.active { background: var(--color-primary); color: var(--color-text); }

/* Фракции */
.sidebar {
  width: 200px;
  flex-shrink: 0;
  background: var(--color-cell);
  padding: 16px;
  overflow-y: auto;
  border-right: 1px solid var(--color-border);
}
.sidebar h2 { font-size: 11px; text-transform: uppercase; color: var(--color-text-dim); margin-bottom: 12px; letter-spacing: 0.05em; }
.sidebar ul { list-style: none; }
.sidebar li {
  padding: 7px 10px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
}
.sidebar li:hover { background: var(--color-primary); }
.sidebar li.active { background: var(--color-primary); color: var(--color-text); }

/* Список юнитов */
.unit-list {
  width: 240px;
  flex-shrink: 0;
  padding: 16px;
  overflow-y: auto;
  border-right: 1px solid var(--color-border);
}
.unit-list-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px; }
.unit-list-header h2 { font-size: 13px; color: var(--color-text-dim); margin-bottom: 0; }
.unit-list-actions { display: flex; align-items: center; gap: 4px; }
.unit-search-wrap { position: relative; margin-bottom: 10px; }
.unit-search {
  width: 100%;
  box-sizing: border-box;
  padding: 5px 24px 5px 8px;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  color: var(--color-text);
  font-size: 12px;
  outline: none;
}
.unit-search:focus { border-color: var(--color-accent); }
.unit-search-clear {
  position: absolute;
  right: 5px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  color: var(--color-text-dim);
  cursor: pointer;
  font-size: 10px;
  padding: 2px;
  line-height: 1;
}
.unit-search-clear:hover { color: var(--color-text); }
.unit-search-empty { font-size: 12px; color: var(--color-text-dim); text-align: center; padding: 12px 0; }
.unit-add-btn {
  background: none;
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text-muted);
  font-size: 16px;
  line-height: 1;
  padding: 1px 6px 2px;
  cursor: pointer;
}
.unit-add-btn:hover { border-color: #3aba80; color: #3aba80; }
.unit-list h2 { font-size: 13px; color: var(--color-text-dim); margin-bottom: 14px; }
.toggle-deleted-btn {
  background: none;
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text-muted);
  font-size: 12px;
  padding: 2px 6px;
  cursor: pointer;
  line-height: 1.4;
}
.toggle-deleted-btn:hover { border-color: var(--color-accent); color: var(--color-accent); }
.toggle-deleted-btn.active { border-color: var(--color-accent); color: var(--color-accent); background: rgba(149,232,225,0.2); }
.deleted-section h3 { color: #664; border-color: #332; }
.deleted-unit { opacity: 0.55; }
.deleted-unit:hover { opacity: 0.8; }
.deleted-unit.active { opacity: 1; }
.deleted-empty { font-size: 11px; color: var(--color-text-muted); padding: 4px 0; }

.hint-panel { display: flex; align-items: center; justify-content: center; }
.hint { color: var(--color-text-muted); font-size: 13px; }

.category { margin-bottom: 20px; }
.category h3 {
  font-size: 10px;
  text-transform: uppercase;
  color: var(--color-heading);
  border-bottom: 1px solid var(--color-border);
  padding-bottom: 4px;
  margin-bottom: 8px;
  letter-spacing: 0.05em;
}

.class-group { margin-bottom: 10px; }
.class-group h4 { font-size: 10px; color: var(--color-text-muted); margin-bottom: 4px; text-transform: uppercase; }
.class-group ul { list-style: none; }
.class-group li {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 5px 8px;
  border-radius: 3px;
  font-size: 12px;
  cursor: pointer;
  color: var(--color-text-dim);
}
.class-group li:hover { background: var(--color-cell); }
.class-group li.active { background: var(--color-primary); color: var(--color-text); }
.unit-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.unit-copy-btn {
  visibility: hidden;
  flex-shrink: 0;
  background: none;
  border: none;
  color: var(--color-text-dim);
  cursor: pointer;
  font-size: 13px;
  padding: 0 2px;
  line-height: 1;
}
.class-group li:hover .unit-copy-btn { visibility: visible; }
.unit-copy-btn:hover { color: var(--color-accent); }

/* Панель деталей */
.detail-panel {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

/* Вкладки на весь экран */
.tab-fill {
  flex: 1;
  overflow: hidden;
}

/* Диалог копирования */
.copy-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.5);
  display: flex; align-items: center; justify-content: center; z-index: 300;
}
.copy-modal {
  background: var(--color-cell); border: 1px solid var(--color-border-soft);
  border-radius: 10px; padding: 20px 24px; width: 300px;
  box-shadow: 0 16px 48px rgba(0,0,0,0.3);
  display: flex; flex-direction: column; gap: 12px;
}
.copy-modal-title { font-size: 13px; font-weight: 600; color: var(--color-text); }
.copy-modal-unit {
  font-size: 12px; color: var(--color-text-dim);
  background: var(--color-input-bg); border: 1px solid var(--color-border);
  border-radius: 4px; padding: 6px 10px;
}
.copy-modal-label { font-size: 11px; color: var(--color-text-muted); }
.copy-modal-select {
  background: var(--color-input-bg); border: 1px solid var(--color-border-soft);
  border-radius: 4px; color: var(--color-text); font-size: 12px; padding: 6px 8px;
}
.copy-modal-select:focus { outline: none; border-color: var(--color-accent); }
.copy-modal-actions { display: flex; gap: 8px; margin-top: 4px; }
.copy-btn-confirm {
  flex: 1; background: var(--color-primary); border: none; border-radius: 5px;
  color: var(--color-text); font-size: 12px; padding: 8px; cursor: pointer;
}
.copy-btn-confirm:hover { background: var(--color-primary-hover); }
.copy-btn-cancel {
  background: transparent; border: 1px solid var(--color-border-soft);
  border-radius: 5px; color: var(--color-text-muted); font-size: 12px;
  padding: 8px 14px; cursor: pointer;
}
.copy-btn-cancel:hover { border-color: var(--color-text-muted); }
</style>
