<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import {
  GetAllUnits, GetBuildings, GetFactions,
  UpdateBuildingLevel, RevertBuildings,
} from '../../wailsjs/go/main/App'

const subTab = ref('units') // 'units' | 'buildings'

// ─── State ───────────────────────────────────────────────────────────────────

const allUnits = ref([])
const allBuildings = ref([])
const factions = ref([{ Name: 'all', DisplayName: 'Все фракции' }])
const loading = ref(true)
const error = ref(null)

// По юнитам
const searchQuery = ref('')
const selectedFactionName = ref('all')
const selectedUnit = ref(null)
const selectedBuildingEntry = ref(null)

// По зданиям
const buildingSearch = ref('')
const expandedGroups = ref(new Set())
const selectedGroup = ref(null)
const selectedLevel = ref(null)
const selectedLevelUnit = ref(null) // выбранный юнит для редактирования найма

// Общее
const opError = ref(null)
const saving = ref(false)
const localDirty = ref(false) // были ли изменения в зданиях

// Пикер
const pickerMode = ref(null) // null | 'addUnit' | 'addBuilding'
const pickerSearch = ref('')
const pickerFaction = ref('all')

// ─── Load ─────────────────────────────────────────────────────────────────────

onMounted(async () => {
  try {
    const [units, buildings, factionList] = await Promise.all([
      GetAllUnits(),
      GetBuildings(),
      GetFactions(),
    ])
    allUnits.value = units
    allBuildings.value = buildings
    factions.value = [{ Name: 'all', DisplayName: 'Все фракции' }, ...factionList]
    if (buildings.length > 0) expandedGroups.value = new Set([buildings[0].Name])
  } catch (e) {
    error.value = String(e)
  } finally {
    loading.value = false
  }
})

// ─── Helpers ─────────────────────────────────────────────────────────────────

function bldName(level) {
  if (level.DisplayName) return level.DisplayName
  return level.Name.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}

function grpName(group) {
  if (group.DisplayName) return group.DisplayName
  return group.Name.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}

function getLevel(groupName, levelName) {
  return allBuildings.value
    .find(g => g.Name === groupName)
    ?.Levels.find(l => l.Name === levelName) ?? null
}

// ─── Computed: По юнитам ─────────────────────────────────────────────────────

const filteredUnits = computed(() => {
  const q = searchQuery.value.toLowerCase()
  return allUnits.value.filter(u => {
    const matchSearch = !q
      || (u.Name ?? '').toLowerCase().includes(q)
      || u.Type.toLowerCase().includes(q)
    const matchFaction = selectedFactionName.value === 'all'
      || (u.Ownership ?? []).includes(selectedFactionName.value)
    return matchSearch && matchFaction
  })
})

const selectedUnitBuildings = computed(() => {
  if (!selectedUnit.value) return []
  const type = selectedUnit.value.Type
  const result = []
  for (const group of allBuildings.value) {
    for (const level of group.Levels) {
      const slot = (level.RecruitSlots ?? []).find(s => s.UnitType === type)
      if (slot) {
        result.push({
          groupName: group.Name,
          groupDisplay: grpName(group),
          levelName: level.Name,
          levelDisplay: bldName(level),
          exp: slot.Level,
          cost: level.Cost,
          turns: level.Construction,
        })
      }
    }
  }
  return result
})

// ─── Computed: По зданиям ────────────────────────────────────────────────────

const filteredBuildings = computed(() => {
  const q = buildingSearch.value.toLowerCase()
  if (!q) return allBuildings.value
  return allBuildings.value
    .map(group => {
      const gMatch = grpName(group).toLowerCase().includes(q) || group.Name.toLowerCase().includes(q)
      if (gMatch) return group
      const levels = group.Levels.filter(l =>
        bldName(l).toLowerCase().includes(q) || l.Name.toLowerCase().includes(q)
      )
      return levels.length ? { ...group, Levels: levels } : null
    })
    .filter(Boolean)
})

watch(buildingSearch, q => {
  if (q) expandedGroups.value = new Set(filteredBuildings.value.map(g => g.Name))
})

const selectedLevelUnits = computed(() => {
  if (!selectedLevel.value) return []
  const seen = new Set()
  const result = []
  for (const slot of (selectedLevel.value.RecruitSlots ?? [])) {
    if (seen.has(slot.UnitType)) continue
    seen.add(slot.UnitType)
    const unit = allUnits.value.find(u => u.Type === slot.UnitType)
    result.push({
      type: slot.UnitType,
      name: unit?.Name || slot.UnitType,
      category: unit?.Category ?? '',
      cls: unit?.Class ?? '',
      exp: slot.Level,
      factions: unit?.Ownership ?? [],
    })
  }
  return result
})

// ─── Computed: Пикер ─────────────────────────────────────────────────────────

const pickerUnits = computed(() => {
  if (pickerMode.value !== 'addUnit' || !selectedLevel.value) return []
  const existing = new Set((selectedLevel.value.RecruitSlots ?? []).map(s => s.UnitType))
  const q = pickerSearch.value.toLowerCase()
  return allUnits.value.filter(u => {
    if (existing.has(u.Type)) return false
    const matchSearch = !q || (u.Name || u.Type).toLowerCase().includes(q) || u.Type.toLowerCase().includes(q)
    const matchFaction = pickerFaction.value === 'all' || (u.Ownership ?? []).includes(pickerFaction.value)
    return matchSearch && matchFaction
  })
})

const pickerBuildings = computed(() => {
  if (pickerMode.value !== 'addBuilding' || !selectedUnit.value) return []
  const q = pickerSearch.value.toLowerCase()
  const result = []
  for (const group of allBuildings.value) {
    const levels = group.Levels.filter(l => {
      const hasUnit = (l.RecruitSlots ?? []).some(s => s.UnitType === selectedUnit.value.Type)
      if (hasUnit) return false
      if (!q) return true
      return bldName(l).toLowerCase().includes(q)
        || l.Name.toLowerCase().includes(q)
        || grpName(group).toLowerCase().includes(q)
    })
    if (levels.length) result.push({ group, levels })
  }
  return result
})

// ─── Actions ─────────────────────────────────────────────────────────────────

function toggleGroup(name) {
  const s = new Set(expandedGroups.value)
  s.has(name) ? s.delete(name) : s.add(name)
  expandedGroups.value = s
}

function selectUnit(unit) {
  selectedUnit.value = unit
  selectedBuildingEntry.value = null
  closePicker()
}

function selectLevel(group, level) {
  selectedGroup.value = group
  selectedLevel.value = level
  selectedLevelUnit.value = null
  closePicker()
}

function selectBuildingEntry(entry) {
  selectedBuildingEntry.value = selectedBuildingEntry.value === entry ? null : entry
}

function selectLevelUnit(unit) {
  selectedLevelUnit.value = selectedLevelUnit.value?.type === unit.type ? null : unit
}

function openPicker(mode) {
  pickerMode.value = mode
  pickerSearch.value = ''
  pickerFaction.value = 'all'
  selectedBuildingEntry.value = null
  selectedLevelUnit.value = null
}

function closePicker() {
  pickerMode.value = null
}

async function refreshBuildings() {
  allBuildings.value = await GetBuildings()
  if (selectedGroup.value && selectedLevel.value) {
    const freshGroup = allBuildings.value.find(g => g.Name === selectedGroup.value.Name)
    selectedGroup.value = freshGroup ?? null
    selectedLevel.value = freshGroup?.Levels.find(l => l.Name === selectedLevel.value.Name) ?? null
  }
}

async function callUpdate(groupName, levelName, newSlots) {
  saving.value = true
  opError.value = null
  try {
    await UpdateBuildingLevel(groupName, levelName, newSlots)
    await refreshBuildings()
    localDirty.value = true
  } catch (e) {
    opError.value = String(e)
  } finally {
    saving.value = false
  }
}

async function revertBuildings() {
  saving.value = true
  opError.value = null
  try {
    await RevertBuildings()
    allBuildings.value = await GetBuildings()
    selectedGroup.value = null
    selectedLevel.value = null
    selectedLevelUnit.value = null
    selectedBuildingEntry.value = null
    localDirty.value = false
  } catch (e) {
    opError.value = String(e)
  } finally {
    saving.value = false
  }
}

// ─── Мутации: По юнитам ──────────────────────────────────────────────────────

async function removeUnitFromBuilding(entry) {
  const level = getLevel(entry.groupName, entry.levelName)
  if (!level) return
  const newSlots = level.RecruitSlots.filter(s => s.UnitType !== selectedUnit.value.Type)
  if (selectedBuildingEntry.value === entry) selectedBuildingEntry.value = null
  await callUpdate(entry.groupName, entry.levelName, newSlots)
}

async function setEntryExp(entry, newExp) {
  const level = getLevel(entry.groupName, entry.levelName)
  if (!level) return
  const unitType = selectedUnit.value.Type
  const newSlots = level.RecruitSlots.map(s =>
    s.UnitType === unitType ? { ...s, Level: newExp } : s
  )
  await callUpdate(entry.groupName, entry.levelName, newSlots)
  entry.exp = newExp
}

async function pickBuilding(group, level) {
  if (!selectedUnit.value) return
  const lvl = getLevel(group.Name, level.Name)
  if (!lvl) return
  const newSlot = {
    UnitType: selectedUnit.value.Type,
    Level: 0,
    Requirements: [...(selectedUnit.value.Ownership ?? [])],
    Conditions: '',
  }
  await callUpdate(group.Name, level.Name, [...(lvl.RecruitSlots ?? []), newSlot])
  closePicker()
}

// ─── Мутации: По зданиям ─────────────────────────────────────────────────────

async function removeUnitFromLevel(unitType) {
  if (!selectedGroup.value || !selectedLevel.value) return
  if (selectedLevelUnit.value?.type === unitType) selectedLevelUnit.value = null
  const newSlots = selectedLevel.value.RecruitSlots.filter(s => s.UnitType !== unitType)
  await callUpdate(selectedGroup.value.Name, selectedLevel.value.Name, newSlots)
}

async function setUnitExp(unitType, newExp) {
  if (!selectedGroup.value || !selectedLevel.value) return
  const newSlots = selectedLevel.value.RecruitSlots.map(s =>
    s.UnitType === unitType ? { ...s, Level: newExp } : s
  )
  await callUpdate(selectedGroup.value.Name, selectedLevel.value.Name, newSlots)
  if (selectedLevelUnit.value?.type === unitType) {
    selectedLevelUnit.value = { ...selectedLevelUnit.value, exp: newExp }
  }
}

async function pickUnit(unit) {
  if (!selectedGroup.value || !selectedLevel.value) return
  const newSlot = {
    UnitType: unit.Type,
    Level: 0,
    Requirements: [...(unit.Ownership ?? [])],
    Conditions: '',
  }
  await callUpdate(selectedGroup.value.Name, selectedLevel.value.Name,
    [...(selectedLevel.value.RecruitSlots ?? []), newSlot])
  closePicker()
}
</script>

<template>
  <div class="recruit-root">

    <!-- Sub-tab switcher -->
    <div class="recruit-tabs">
      <button :class="['rtab', { active: subTab === 'units' }]" @click="subTab = 'units'">По юнитам</button>
      <button :class="['rtab', { active: subTab === 'buildings' }]" @click="subTab = 'buildings'">По зданиям</button>
      <div class="rtab-spacer"></div>
      <button v-if="localDirty" class="btn-revert" :disabled="saving" @click="revertBuildings">
        Отменить изменения
      </button>
    </div>

    <div v-if="loading" class="recruit-loading">Загрузка...</div>
    <div v-else-if="error" class="recruit-error">{{ error }}</div>
    <div v-else class="recruit-body">

      <!-- ══ По юнитам ══════════════════════════════════════════════════════════ -->
      <template v-if="subTab === 'units'">

        <!-- Левая панель: список юнитов -->
        <div class="recruit-left">
          <div class="recruit-filters">
            <input v-model="searchQuery" class="recruit-search" placeholder="Поиск юнита..." />
            <select v-model="selectedFactionName" class="recruit-select">
              <option v-for="f in factions" :key="f.Name" :value="f.Name">{{ f.DisplayName || f.Name }}</option>
            </select>
          </div>
          <div class="recruit-list">
            <div
              v-for="unit in filteredUnits"
              :key="unit.Type"
              :class="['recruit-unit-item', { active: selectedUnit?.Type === unit.Type }]"
              @click="selectUnit(unit)"
            >
              <div class="rui-name">{{ unit.Name || unit.Type }}</div>
              <div class="rui-meta">{{ unit.Category }} · {{ unit.Class }}</div>
              <div class="rui-faction">{{ (unit.Ownership ?? []).join(', ') }}</div>
            </div>
            <div v-if="filteredUnits.length === 0" class="list-empty">Нет юнитов</div>
          </div>
        </div>

        <!-- Правая панель: здания для выбранного юнита -->
        <div class="recruit-right">
          <template v-if="selectedUnit">

            <div class="right-header">
              <span class="rh-title">{{ selectedUnit.Name || selectedUnit.Type }}</span>
              <span class="rh-sub">
                {{ selectedUnitBuildings.length ? `${selectedUnitBuildings.length} зд.` : 'нигде не нанимается' }}
              </span>
              <button class="btn-add" @click="openPicker('addBuilding')">+ Добавить здание</button>
            </div>

            <div class="blocks-area">
              <div
                v-for="(entry, i) in selectedUnitBuildings"
                :key="i"
                :class="['bblock', { selected: selectedBuildingEntry === entry }]"
                @click="selectBuildingEntry(entry)"
              >
                <div class="bb-group-label">{{ entry.groupDisplay }}</div>
                <div class="bb-level-name">{{ entry.levelDisplay }}</div>
                <div class="bb-row">
                  <span class="bb-key">Опыт</span>
                  <span class="bb-val">{{ entry.exp > 0 ? `+${entry.exp}` : '—' }}</span>
                </div>
                <div class="bb-row">
                  <span class="bb-key">Стоимость</span>
                  <span class="bb-val">{{ entry.cost }}</span>
                </div>
                <button class="bb-del" title="Убрать" @click.stop="removeUnitFromBuilding(entry)">✕</button>
              </div>
              <div v-if="selectedUnitBuildings.length === 0" class="area-hint">
                Юнит нигде не нанимается. Нажмите «+ Добавить здание».
              </div>
            </div>

            <transition name="slide-up">
              <div v-if="selectedBuildingEntry && !pickerMode" class="edit-panel">
                <div class="ep-title">{{ selectedBuildingEntry.groupDisplay }} — {{ selectedBuildingEntry.levelDisplay }}</div>
                <div class="ep-fields">
                  <label>
                    Опыт при найме
                    <input
                      type="number"
                      :value="selectedBuildingEntry.exp"
                      min="0" max="9"
                      :disabled="saving"
                      @change="e => setEntryExp(selectedBuildingEntry, +e.target.value)"
                    />
                  </label>
                  <span class="ep-hint">M2TW: пул и скорость пополнения будут здесь</span>
                  <span v-if="opError" class="ep-error">{{ opError }}</span>
                </div>
              </div>
            </transition>

          </template>
          <div v-else class="right-hint">Выберите юнита слева</div>

          <!-- Пикер: добавить здание к юниту -->
          <transition name="picker-slide">
            <div v-if="pickerMode === 'addBuilding'" class="picker-overlay">
              <div class="picker-head">
                <span class="picker-title">Выбрать здание</span>
                <button class="picker-close" @click="closePicker">✕</button>
              </div>
              <div class="picker-search-row">
                <input v-model="pickerSearch" class="recruit-search" placeholder="Поиск здания..." autofocus />
              </div>
              <div class="picker-list">
                <template v-if="pickerBuildings.length">
                  <div v-for="item in pickerBuildings" :key="item.group.Name" class="pk-group">
                    <div class="pk-group-label">{{ grpName(item.group) }}</div>
                    <div
                      v-for="level in item.levels"
                      :key="level.Name"
                      class="pk-item"
                      @click="pickBuilding(item.group, level)"
                    >
                      <span class="pk-name">{{ bldName(level) }}</span>
                      <span class="pk-meta">{{ level.Cost }} зол · {{ level.Construction }} ход.</span>
                    </div>
                  </div>
                </template>
                <div v-else class="pk-empty">Нет доступных зданий</div>
              </div>
            </div>
          </transition>
        </div>

      </template>

      <!-- ══ По зданиям ════════════════════════════════════════════════════════ -->
      <template v-else>

        <!-- Левая панель: дерево зданий -->
        <div class="recruit-left">
          <div class="recruit-filters">
            <input v-model="buildingSearch" class="recruit-search" placeholder="Поиск здания..." />
          </div>
          <div class="bld-tree">
            <div v-for="group in filteredBuildings" :key="group.Name" class="bld-group">
              <div class="bld-group-head" @click="toggleGroup(group.Name)">
                <span class="bld-arrow">{{ expandedGroups.has(group.Name) ? '▾' : '▸' }}</span>
                {{ grpName(group) }}
              </div>
              <div v-if="expandedGroups.has(group.Name)" class="bld-levels">
                <div
                  v-for="level in group.Levels"
                  :key="level.Name"
                  :class="['bld-level', { active: selectedLevel?.Name === level.Name && selectedGroup?.Name === group.Name }]"
                  @click="selectLevel(group, level)"
                >
                  <span class="bld-level-name">{{ bldName(level) }}</span>
                  <span class="bld-level-meta">{{ level.Cost }} зол · {{ level.Construction }} ход.</span>
                </div>
              </div>
            </div>
            <div v-if="filteredBuildings.length === 0" class="list-empty">Нет зданий</div>
          </div>
        </div>

        <!-- Правая панель: юниты на выбранном уровне -->
        <div class="recruit-right">
          <template v-if="selectedLevel">

            <div class="right-header">
              <span class="rh-title">{{ bldName(selectedLevel) }}</span>
              <span class="rh-sub">
                {{ grpName(selectedGroup) }} ·
                {{ selectedLevelUnits.length ? `${selectedLevelUnits.length} юн.` : 'нет юнитов' }}
              </span>
              <button class="btn-add" @click="openPicker('addUnit')">+ Добавить юнита</button>
            </div>

            <div class="blocks-area">
              <div
                v-for="unit in selectedLevelUnits"
                :key="unit.type"
                :class="['bblock', { selected: selectedLevelUnit?.type === unit.type }]"
                @click="selectLevelUnit(unit)"
              >
                <div class="bb-group-label">{{ unit.category }}</div>
                <div class="bb-level-name">{{ unit.name }}</div>
                <div class="bb-row">
                  <span class="bb-key">Класс</span>
                  <span class="bb-val">{{ unit.cls }}</span>
                </div>
                <div class="bb-row">
                  <span class="bb-key">Опыт</span>
                  <span class="bb-val">{{ unit.exp > 0 ? `+${unit.exp}` : '—' }}</span>
                </div>
                <div v-if="unit.factions.length" class="bb-factions">{{ unit.factions.join(', ') }}</div>
                <button class="bb-del" title="Убрать" :disabled="saving" @click.stop="removeUnitFromLevel(unit.type)">✕</button>
              </div>
              <div v-if="selectedLevelUnits.length === 0" class="area-hint">
                На этом уровне здания никто не нанимается.
              </div>
            </div>

            <!-- Панель редактирования найма выбранного юнита -->
            <transition name="slide-up">
              <div v-if="selectedLevelUnit && !pickerMode" class="edit-panel">
                <div class="ep-title">{{ selectedLevelUnit.name }}</div>
                <div class="ep-fields">
                  <label>
                    Опыт при найме
                    <input
                      type="number"
                      :value="selectedLevelUnit.exp"
                      min="0" max="9"
                      :disabled="saving"
                      @change="e => setUnitExp(selectedLevelUnit.type, +e.target.value)"
                    />
                  </label>
                  <span class="ep-hint">M2TW: пул и скорость пополнения будут здесь</span>
                  <span v-if="opError" class="ep-error">{{ opError }}</span>
                </div>
              </div>
            </transition>

          </template>
          <div v-else class="right-hint">Выберите уровень здания слева</div>

          <!-- Пикер: добавить юнита к уровню здания -->
          <transition name="picker-slide">
            <div v-if="pickerMode === 'addUnit'" class="picker-overlay">
              <div class="picker-head">
                <span class="picker-title">Выбрать юнита</span>
                <button class="picker-close" @click="closePicker">✕</button>
              </div>
              <div class="picker-search-row">
                <input v-model="pickerSearch" class="recruit-search" placeholder="Поиск юнита..." autofocus />
                <select v-model="pickerFaction" class="recruit-select">
                  <option v-for="f in factions" :key="f.Name" :value="f.Name">{{ f.DisplayName || f.Name }}</option>
                </select>
              </div>
              <div class="picker-list">
                <div
                  v-for="unit in pickerUnits"
                  :key="unit.Type"
                  class="pk-item pk-unit"
                  @click="pickUnit(unit)"
                >
                  <div class="pk-name">{{ unit.Name || unit.Type }}</div>
                  <div class="pk-meta">{{ unit.Category }} · {{ unit.Class }}</div>
                </div>
                <div v-if="pickerUnits.length === 0" class="pk-empty">Нет доступных юнитов</div>
              </div>
            </div>
          </transition>
        </div>

      </template>

    </div>
  </div>
</template>

<style scoped>
.recruit-root {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

/* Sub-tabs */
.recruit-tabs {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  background: #0d1117;
  border-bottom: 1px solid #1e2e50;
  padding: 0 16px;
}
.rtab {
  padding: 10px 18px;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: #555;
  cursor: pointer;
  font-size: 13px;
  margin-bottom: -1px;
  transition: color 0.15s;
  align-self: stretch;
  display: flex;
  align-items: center;
}
.rtab:hover { color: #aaa; }
.rtab.active { color: #e0e0e0; border-bottom-color: #e94560; }
.rtab-spacer { flex: 1; }

.btn-revert {
  background: transparent;
  border: 1px solid #2a3a5e;
  color: #668;
  padding: 4px 12px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 11px;
  white-space: nowrap;
  transition: all 0.15s;
}
.btn-revert:hover:not(:disabled) { border-color: #e94560; color: #e94560; background: rgba(233,69,96,0.08); }
.btn-revert:disabled { opacity: 0.4; cursor: not-allowed; }

/* Body */
.recruit-body {
  display: flex;
  flex: 1;
  overflow: hidden;
}

/* Left panel */
.recruit-left {
  width: 260px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid #1e2e50;
  overflow: hidden;
}

.recruit-filters {
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-bottom: 1px solid #1e2e50;
  flex-shrink: 0;
}
.recruit-search, .recruit-select {
  background: #0d1117;
  border: 1px solid #1e2e50;
  color: #e0e0e0;
  padding: 6px 10px;
  border-radius: 4px;
  font-size: 12px;
  width: 100%;
  box-sizing: border-box;
}
.recruit-search:focus, .recruit-select:focus { outline: none; border-color: #0f3460; }

.recruit-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.recruit-unit-item {
  padding: 8px 10px;
  border-radius: 4px;
  cursor: pointer;
}
.recruit-unit-item:hover { background: #16213e; }
.recruit-unit-item.active { background: #0f3460; }
.rui-name { font-size: 13px; color: #e0e0e0; }
.rui-meta { font-size: 11px; color: #555; margin-top: 2px; }
.rui-faction { font-size: 10px; color: #3a4a6e; margin-top: 1px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.list-empty { color: #333; font-size: 12px; text-align: center; padding: 20px; }

/* Building tree */
.bld-tree {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}
.bld-group { margin-bottom: 2px; }
.bld-group-head {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
  color: #aaa;
  font-weight: 500;
  user-select: none;
}
.bld-group-head:hover { background: #16213e; }
.bld-arrow { font-size: 10px; color: #555; width: 10px; }
.bld-levels { padding-left: 16px; margin-bottom: 4px; }
.bld-level {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px;
  border-radius: 4px;
  cursor: pointer;
}
.bld-level:hover { background: #16213e; }
.bld-level.active { background: #0f3460; }
.bld-level-name { font-size: 12px; color: #ccc; }
.bld-level-meta { font-size: 10px; color: #444; }
.bld-level.active .bld-level-name { color: #e0e0e0; }
.bld-level.active .bld-level-meta { color: #668; }

/* Right panel */
.recruit-right {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
}

.right-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 20px;
  border-bottom: 1px solid #1e2e50;
  flex-shrink: 0;
}
.rh-title { font-size: 15px; font-weight: 600; }
.rh-sub { font-size: 12px; color: #555; flex: 1; }
.btn-add {
  background: transparent;
  border: 1px solid #1e2e50;
  color: #668;
  padding: 5px 12px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  white-space: nowrap;
}
.btn-add:hover { border-color: #0f3460; color: #aac; background: #0f3460; }

/* Blocks area */
.blocks-area {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
  display: flex;
  flex-wrap: wrap;
  align-content: flex-start;
  gap: 12px;
}

.bblock {
  position: relative;
  background: #16213e;
  border: 1px solid #1e2e50;
  border-radius: 6px;
  padding: 12px 14px;
  width: 168px;
  cursor: pointer;
  transition: border-color 0.15s;
}
.bblock:hover { border-color: #2a3a6e; }
.bblock.selected { border-color: #e94560; background: #1c2540; }

.bb-group-label {
  font-size: 10px;
  text-transform: uppercase;
  color: #e94560;
  letter-spacing: 0.05em;
  margin-bottom: 4px;
}
.bb-level-name { font-size: 14px; font-weight: 600; margin-bottom: 10px; line-height: 1.2; }
.bb-row {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
}
.bb-key { color: #555; }
.bb-val { color: #aaa; }
.bb-factions { font-size: 10px; color: #3a4a6e; margin-top: 6px; line-height: 1.3; }

.bb-del {
  position: absolute;
  top: 8px; right: 8px;
  background: none;
  border: none;
  color: #2a3a5e;
  cursor: pointer;
  font-size: 11px;
  padding: 2px 5px;
  border-radius: 2px;
  line-height: 1;
  transition: color 0.1s, background 0.1s;
}
.bb-del:hover { color: #e94560 !important; background: rgba(233,69,96,0.1); }
.bb-del:disabled { opacity: 0.3; pointer-events: none; }

/* Edit panel */
.edit-panel {
  flex-shrink: 0;
  border-top: 1px solid #1e2e50;
  padding: 14px 20px;
  background: #16213e;
}
.ep-title {
  font-size: 11px;
  text-transform: uppercase;
  color: #e94560;
  letter-spacing: 0.05em;
  margin-bottom: 12px;
}
.ep-fields {
  display: flex;
  align-items: flex-end;
  gap: 20px;
  flex-wrap: wrap;
}
.ep-fields label {
  display: flex;
  flex-direction: column;
  gap: 5px;
  font-size: 11px;
  color: #888;
}
.ep-fields input {
  background: #0d1117;
  border: 1px solid #1e2e50;
  color: #e0e0e0;
  padding: 5px 10px;
  border-radius: 4px;
  font-size: 13px;
  width: 80px;
}
.ep-fields input:focus { outline: none; border-color: #0f3460; }
.ep-hint { font-size: 11px; color: #333; font-style: italic; }
.ep-error { font-size: 11px; color: #e94560; }

/* Hints */
.area-hint { color: #2a3a5e; font-size: 13px; width: 100%; padding-top: 8px; }
.right-hint {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #2a3a5e;
  font-size: 13px;
}

/* ─── Picker overlay ─────────────────────────────────────────────────────── */
.picker-overlay {
  position: absolute;
  inset: 0;
  background: #0d1117;
  display: flex;
  flex-direction: column;
  z-index: 10;
  border-left: 1px solid #1e2e50;
}

.picker-head {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #1e2e50;
  flex-shrink: 0;
  gap: 10px;
}
.picker-title {
  font-size: 13px;
  font-weight: 600;
  color: #e0e0e0;
  flex: 1;
}
.picker-close {
  background: none;
  border: none;
  color: #555;
  cursor: pointer;
  font-size: 14px;
  padding: 2px 6px;
  border-radius: 3px;
  line-height: 1;
}
.picker-close:hover { color: #e94560; background: rgba(233,69,96,0.1); }

.picker-search-row {
  padding: 10px 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-bottom: 1px solid #1e2e50;
  flex-shrink: 0;
}

.picker-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.pk-group { margin-bottom: 4px; }
.pk-group-label {
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #e94560;
  padding: 6px 10px 4px;
}
.pk-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 7px 12px;
  border-radius: 4px;
  cursor: pointer;
  gap: 10px;
}
.pk-item:hover { background: #16213e; }
.pk-unit { flex-direction: column; align-items: flex-start; gap: 2px; }
.pk-name { font-size: 13px; color: #e0e0e0; }
.pk-meta { font-size: 11px; color: #555; }
.pk-empty { color: #333; font-size: 12px; text-align: center; padding: 20px; }

/* Animations */
.slide-up-enter-active, .slide-up-leave-active { transition: opacity 0.15s, transform 0.15s; }
.slide-up-enter-from, .slide-up-leave-to { opacity: 0; transform: translateY(8px); }

.picker-slide-enter-active, .picker-slide-leave-active { transition: opacity 0.15s, transform 0.15s; }
.picker-slide-enter-from, .picker-slide-leave-to { opacity: 0; transform: translateX(12px); }

/* Loading / error */
.recruit-loading, .recruit-error {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  color: #2a3a5e;
}
.recruit-error { color: #e94560; }
</style>
