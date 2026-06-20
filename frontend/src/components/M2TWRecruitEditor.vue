<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import {
  GetM2TWAllUnits, GetM2TWBuildings, GetM2TWFactions,
  UpdateM2TWBuildingLevel, RevertM2TWBuildings,
} from '../../wailsjs/go/main/App'

const subTab = ref('units') // 'units' | 'buildings'

// ─── State ───────────────────────────────────────────────────────────────────

const allUnits     = ref([])
const allBuildings = ref([])
const factions     = ref([{ Name: 'all', DisplayName: 'Все фракции' }])
const loading      = ref(true)
const error        = ref(null)

// По юнитам
const searchQuery    = ref('')
const selectedFactionName = ref('all')
const selectedUnit   = ref(null)
const selectedEntry  = ref(null) // выбранный блок здания

// По зданиям
const buildingSearch   = ref('')
const settlementFilter = ref('all')   // 'all' | 'city' | 'castle'
const bldFactionFilter = ref('all')
const expandedGroups   = ref(new Set())
const selectedGroup    = ref(null)
const selectedLevel    = ref(null)
const selectedPool     = ref(null)    // выбранный пул для редактирования

// Общее
const opError    = ref(null)
const saving     = ref(false)
const localDirty = ref(false)

// Пикер
const pickerMode   = ref(null)  // null | 'addUnit' | 'addBuilding'
const pickerSearch = ref('')
const pickerFactionFilter = ref('all')

// ─── Load ─────────────────────────────────────────────────────────────────────

onMounted(async () => {
  try {
    const [units, buildings, factionList] = await Promise.all([
      GetM2TWAllUnits(),
      GetM2TWBuildings(),
      GetM2TWFactions(),
    ])
    allUnits.value     = units
    allBuildings.value = buildings
    factions.value     = [{ Name: 'all', DisplayName: 'Все фракции' }, ...factionList]
    if (buildings.length > 0) expandedGroups.value = new Set([buildings[0].Name])
  } catch (e) {
    error.value = String(e)
  } finally {
    loading.value = false
  }
})

// ─── Helpers ─────────────────────────────────────────────────────────────────

function bldName(level) {
  return (level.DisplayName || level.Name).replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}
function grpName(group) {
  return (group.DisplayName || group.Name).replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}
function getLevel(groupName, levelName) {
  return allBuildings.value.find(g => g.Name === groupName)
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

// Карточки зданий для выбранного юнита
const selectedUnitBuildings = computed(() => {
  if (!selectedUnit.value) return []
  const type = selectedUnit.value.Type
  const result = []
  for (const group of allBuildings.value) {
    for (const level of group.Levels ?? []) {
      for (const pool of level.RecruitPools ?? []) {
        if (pool.UnitType === type) {
          result.push({
            groupName:    group.Name,
            groupDisplay: grpName(group),
            levelName:    level.Name,
            levelDisplay: bldName(level),
            settlementType: level.SettlementType ?? '',
            pool: { ...pool },   // копия для редактирования
            cost:         level.Cost,
          })
        }
      }
    }
  }
  return result
})

// ─── Computed: По зданиям ────────────────────────────────────────────────────

const filteredBuildings = computed(() => {
  const q = buildingSearch.value.toLowerCase()
  return allBuildings.value
    .map(group => {
      const gMatch = grpName(group).toLowerCase().includes(q) || group.Name.toLowerCase().includes(q)
      const levels = (group.Levels ?? []).filter(l => {
        // фильтр city/castle
        if (settlementFilter.value !== 'all') {
          const st = l.SettlementType ?? ''
          if (st && st !== settlementFilter.value) return false
        }
        // фильтр по фракции
        if (bldFactionFilter.value !== 'all') {
          const rc = l.RequiredCultures ?? []
          if (rc.length > 0 && !rc.includes(bldFactionFilter.value)) return false
        }
        if (gMatch) return true
        return bldName(l).toLowerCase().includes(q) || l.Name.toLowerCase().includes(q)
      })
      return levels.length ? { ...group, Levels: levels } : null
    })
    .filter(Boolean)
})

watch(buildingSearch, q => {
  if (q) expandedGroups.value = new Set(filteredBuildings.value.map(g => g.Name))
})

// Пулы выбранного уровня с данными юнита
const selectedLevelPools = computed(() => {
  if (!selectedLevel.value) return []
  return (selectedLevel.value.RecruitPools ?? []).map((pool, idx) => {
    const unit = allUnits.value.find(u => u.Type === pool.UnitType)
    return { idx, pool, unitName: unit?.Name || pool.UnitType, category: unit?.Category ?? '', cls: unit?.Class ?? '' }
  })
})

// ─── Computed: Пикер ─────────────────────────────────────────────────────────

const pickerUnits = computed(() => {
  if (pickerMode.value !== 'addUnit' || !selectedLevel.value) return []
  const existing = new Set((selectedLevel.value.RecruitPools ?? []).map(p => p.UnitType))
  const q = pickerSearch.value.toLowerCase()
  return allUnits.value.filter(u => {
    if (existing.has(u.Type)) return false
    const matchSearch = !q || (u.Name || u.Type).toLowerCase().includes(q) || u.Type.toLowerCase().includes(q)
    const matchFaction = pickerFactionFilter.value === 'all' || (u.Ownership ?? []).includes(pickerFactionFilter.value)
    return matchSearch && matchFaction
  })
})

const pickerBuildings = computed(() => {
  if (pickerMode.value !== 'addBuilding' || !selectedUnit.value) return []
  const q = pickerSearch.value.toLowerCase()
  const result = []
  for (const group of allBuildings.value) {
    const levels = (group.Levels ?? []).filter(l => {
      const already = (l.RecruitPools ?? []).some(p => p.UnitType === selectedUnit.value.Type)
      if (already) return false
      if (!q) return true
      return bldName(l).toLowerCase().includes(q) || grpName(group).toLowerCase().includes(q)
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
  selectedEntry.value = null
  closePicker()
}

function selectLevel(group, level) {
  selectedGroup.value = group
  selectedLevel.value = level
  selectedPool.value  = null
  closePicker()
}

function openPicker(mode) {
  pickerMode.value   = mode
  pickerSearch.value = ''
  pickerFactionFilter.value = 'all'
  selectedEntry.value = null
  selectedPool.value  = null
}

function closePicker() {
  pickerMode.value = null
}

async function refreshBuildings() {
  allBuildings.value = await GetM2TWBuildings()
  if (selectedGroup.value && selectedLevel.value) {
    const freshGroup = allBuildings.value.find(g => g.Name === selectedGroup.value.Name)
    selectedGroup.value = freshGroup ?? null
    selectedLevel.value = freshGroup?.Levels?.find(l => l.Name === selectedLevel.value.Name) ?? null
  }
}

async function callUpdate(groupName, levelName, newPools, bonusLines) {
  saving.value  = true
  opError.value = null
  try {
    await UpdateM2TWBuildingLevel(groupName, levelName, newPools, bonusLines ?? [])
    await refreshBuildings()
    localDirty.value = true
  } catch (e) {
    opError.value = String(e)
  } finally {
    saving.value = false
  }
}

async function revertBuildings() {
  saving.value  = true
  opError.value = null
  try {
    await RevertM2TWBuildings()
    allBuildings.value = await GetM2TWBuildings()
    selectedGroup.value = null
    selectedLevel.value = null
    selectedPool.value  = null
    selectedEntry.value = null
    localDirty.value    = false
  } catch (e) {
    opError.value = String(e)
  } finally {
    saving.value = false
  }
}

// ─── По юнитам ───────────────────────────────────────────────────────────────

function selectEntry(entry) {
  selectedEntry.value = selectedEntry.value === entry ? null : entry
}

async function removeUnitFromBuilding(entry) {
  const level = getLevel(entry.groupName, entry.levelName)
  if (!level) return
  const bonusLines = level.BonusLines ?? []
  const newPools = (level.RecruitPools ?? []).filter(p => p.UnitType !== selectedUnit.value.Type)
  if (selectedEntry.value === entry) selectedEntry.value = null
  await callUpdate(entry.groupName, entry.levelName, newPools, bonusLines)
}

async function saveEntryPool(entry) {
  const level = getLevel(entry.groupName, entry.levelName)
  if (!level) return
  const bonusLines = level.BonusLines ?? []
  const newPools = (level.RecruitPools ?? []).map(p =>
    p.UnitType === entry.pool.UnitType ? { ...entry.pool } : p
  )
  await callUpdate(entry.groupName, entry.levelName, newPools, bonusLines)
}

async function pickBuilding(group, level) {
  if (!selectedUnit.value) return
  const lvl = getLevel(group.Name, level.Name)
  if (!lvl) return
  const newPool = {
    UnitType:      selectedUnit.value.Type,
    InitialPool:   1,
    ReplenishRate: 0.15,
    MaxPool:       3,
    ExpGained:     0,
    Factions:      [],
    Conditions:    '',
  }
  await callUpdate(group.Name, level.Name, [...(lvl.RecruitPools ?? []), newPool], lvl.BonusLines ?? [])
  closePicker()
}

// ─── По зданиям ──────────────────────────────────────────────────────────────

function selectPool(item) {
  selectedPool.value = selectedPool.value?.idx === item.idx ? null : { ...item, pool: { ...item.pool } }
}

async function removePool(item) {
  if (!selectedGroup.value || !selectedLevel.value) return
  if (selectedPool.value?.idx === item.idx) selectedPool.value = null
  const bonusLines = selectedLevel.value.BonusLines ?? []
  const newPools = (selectedLevel.value.RecruitPools ?? []).filter((_, i) => i !== item.idx)
  await callUpdate(selectedGroup.value.Name, selectedLevel.value.Name, newPools, bonusLines)
}

async function savePoolEdit() {
  if (!selectedGroup.value || !selectedLevel.value || !selectedPool.value) return
  const bonusLines = selectedLevel.value.BonusLines ?? []
  const newPools = (selectedLevel.value.RecruitPools ?? []).map((p, i) =>
    i === selectedPool.value.idx ? { ...selectedPool.value.pool } : p
  )
  await callUpdate(selectedGroup.value.Name, selectedLevel.value.Name, newPools, bonusLines)
  selectedPool.value = null
}

async function pickUnit(unit) {
  if (!selectedGroup.value || !selectedLevel.value) return
  const bonusLines = selectedLevel.value.BonusLines ?? []
  const newPool = {
    UnitType:      unit.Type,
    InitialPool:   1,
    ReplenishRate: 0.15,
    MaxPool:       3,
    ExpGained:     0,
    Factions:      [],
    Conditions:    '',
  }
  await callUpdate(selectedGroup.value.Name, selectedLevel.value.Name,
    [...(selectedLevel.value.RecruitPools ?? []), newPool], bonusLines)
  closePicker()
}
</script>

<template>
  <div class="recruit-root">

    <!-- Sub-tab switcher -->
    <div class="recruit-tabs">
      <button :class="['rtab', { active: subTab === 'units' }]"     @click="subTab = 'units'">По юнитам</button>
      <button :class="['rtab', { active: subTab === 'buildings' }]" @click="subTab = 'buildings'">По зданиям</button>
      <div class="rtab-spacer"></div>
      <span v-if="opError" class="tab-error">{{ opError }}</span>
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
                :class="['bblock', { selected: selectedEntry === entry }]"
                @click="selectEntry(entry)"
              >
                <div class="bb-group-label">{{ entry.groupDisplay }}</div>
                <div class="bb-level-name">
                  {{ entry.levelDisplay }}
                  <span v-if="entry.settlementType" :class="['bb-settle-tag', entry.settlementType]">{{ entry.settlementType }}</span>
                </div>
                <div class="bb-row"><span class="bb-key">Нач. пул</span><span class="bb-val">{{ entry.pool.InitialPool }}</span></div>
                <div class="bb-row"><span class="bb-key">Пополнение</span><span class="bb-val">{{ entry.pool.ReplenishRate }}</span></div>
                <div class="bb-row"><span class="bb-key">Макс. пул</span><span class="bb-val">{{ entry.pool.MaxPool }}</span></div>
                <div class="bb-row"><span class="bb-key">Опыт</span><span class="bb-val">{{ entry.pool.ExpGained || '—' }}</span></div>
                <button class="bb-del" title="Убрать" @click.stop="removeUnitFromBuilding(entry)">✕</button>
              </div>
              <div v-if="selectedUnitBuildings.length === 0" class="area-hint">
                Юнит нигде не нанимается. Нажмите «+ Добавить здание».
              </div>
            </div>

            <!-- Панель редактирования выбранного блока -->
            <transition name="slide-up">
              <div v-if="selectedEntry && !pickerMode" class="edit-panel">
                <div class="ep-title">{{ selectedEntry.groupDisplay }} — {{ selectedEntry.levelDisplay }}</div>
                <div class="ep-fields">
                  <label>Нач. пул<input type="number" min="0" v-model.number="selectedEntry.pool.InitialPool" :disabled="saving" /></label>
                  <label>Пополнение<input type="number" step="0.01" min="0" v-model.number="selectedEntry.pool.ReplenishRate" :disabled="saving" /></label>
                  <label>Макс. пул<input type="number" min="0" v-model.number="selectedEntry.pool.MaxPool" :disabled="saving" /></label>
                  <label>Опыт<input type="number" min="0" v-model.number="selectedEntry.pool.ExpGained" :disabled="saving" /></label>
                  <label class="ep-wide">Условия<input v-model="selectedEntry.pool.Conditions" :disabled="saving" class="ep-cond" /></label>
                  <button class="ep-save" :disabled="saving" @click="saveEntryPool(selectedEntry)">{{ saving ? '...' : 'Сохранить' }}</button>
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
                    <div v-for="level in item.levels" :key="level.Name" class="pk-item" @click="pickBuilding(item.group, level)">
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
            <select v-model="bldFactionFilter" class="recruit-select">
              <option v-for="f in factions" :key="f.Name" :value="f.Name">{{ f.DisplayName || f.Name }}</option>
            </select>
            <div class="settle-tabs">
              <button
                v-for="opt in [{ v: 'all', l: 'Все' }, { v: 'city', l: 'City' }, { v: 'castle', l: 'Castle' }]"
                :key="opt.v"
                :class="['settle-tab', { active: settlementFilter === opt.v }]"
                @click="settlementFilter = opt.v"
              >{{ opt.l }}</button>
            </div>
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
                  :class="['bld-level', {
                    active: selectedLevel?.Name === level.Name && selectedGroup?.Name === group.Name,
                    'type-city':   level.SettlementType === 'city',
                    'type-castle': level.SettlementType === 'castle',
                  }]"
                  @click="selectLevel(group, level)"
                >
                  <div class="bld-level-left">
                    <span class="bld-level-name">{{ bldName(level) }}</span>
                    <span v-if="level.SettlementType" class="bld-settle-badge">{{ level.SettlementType }}</span>
                  </div>
                  <span class="bld-level-meta">{{ (level.RecruitPools ?? []).length }} юн.</span>
                </div>
              </div>
            </div>
            <div v-if="filteredBuildings.length === 0" class="list-empty">Нет зданий</div>
          </div>
        </div>

        <!-- Правая панель: пулы выбранного уровня -->
        <div class="recruit-right">
          <template v-if="selectedLevel">

            <div class="right-header">
              <span class="rh-title">{{ bldName(selectedLevel) }}</span>
              <span class="rh-sub">
                {{ grpName(selectedGroup) }} ·
                {{ selectedLevelPools.length ? `${selectedLevelPools.length} юн.` : 'нет юнитов' }}
              </span>
              <button class="btn-add" @click="openPicker('addUnit')">+ Добавить юнита</button>
            </div>

            <div class="blocks-area">
              <div
                v-for="item in selectedLevelPools"
                :key="item.idx"
                :class="['bblock', { selected: selectedPool?.idx === item.idx }]"
                @click="selectPool(item)"
              >
                <div class="bb-group-label">{{ item.category }}</div>
                <div class="bb-level-name">{{ item.unitName }}</div>
                <div class="bb-row"><span class="bb-key">Нач. пул</span><span class="bb-val">{{ item.pool.InitialPool }}</span></div>
                <div class="bb-row"><span class="bb-key">Пополнение</span><span class="bb-val">{{ item.pool.ReplenishRate }}</span></div>
                <div class="bb-row"><span class="bb-key">Макс. пул</span><span class="bb-val">{{ item.pool.MaxPool }}</span></div>
                <div class="bb-row"><span class="bb-key">Опыт</span><span class="bb-val">{{ item.pool.ExpGained || '—' }}</span></div>
                <div v-if="item.pool.Conditions" class="bb-cond">{{ item.pool.Conditions }}</div>
                <button class="bb-del" title="Убрать" :disabled="saving" @click.stop="removePool(item)">✕</button>
              </div>
              <div v-if="selectedLevelPools.length === 0" class="area-hint">
                На этом уровне здания никто не нанимается.
              </div>
            </div>

            <!-- Панель редактирования выбранного пула -->
            <transition name="slide-up">
              <div v-if="selectedPool && !pickerMode" class="edit-panel">
                <div class="ep-title">{{ selectedPool.unitName }}</div>
                <div class="ep-fields">
                  <label>Нач. пул<input type="number" min="0" v-model.number="selectedPool.pool.InitialPool" :disabled="saving" /></label>
                  <label>Пополнение<input type="number" step="0.01" min="0" v-model.number="selectedPool.pool.ReplenishRate" :disabled="saving" /></label>
                  <label>Макс. пул<input type="number" min="0" v-model.number="selectedPool.pool.MaxPool" :disabled="saving" /></label>
                  <label>Опыт<input type="number" min="0" v-model.number="selectedPool.pool.ExpGained" :disabled="saving" /></label>
                  <label class="ep-wide">Условия<input v-model="selectedPool.pool.Conditions" :disabled="saving" class="ep-cond" /></label>
                  <button class="ep-save" :disabled="saving" @click="savePoolEdit">{{ saving ? '...' : 'Сохранить' }}</button>
                  <span v-if="opError" class="ep-error">{{ opError }}</span>
                </div>
              </div>
            </transition>

          </template>
          <div v-else class="right-hint">Выберите уровень здания слева</div>

          <!-- Пикер: добавить юнита к уровню -->
          <transition name="picker-slide">
            <div v-if="pickerMode === 'addUnit'" class="picker-overlay">
              <div class="picker-head">
                <span class="picker-title">Выбрать юнита</span>
                <button class="picker-close" @click="closePicker">✕</button>
              </div>
              <div class="picker-search-row">
                <input v-model="pickerSearch" class="recruit-search" placeholder="Поиск юнита..." autofocus />
                <select v-model="pickerFactionFilter" class="recruit-select">
                  <option v-for="f in factions" :key="f.Name" :value="f.Name">{{ f.DisplayName || f.Name }}</option>
                </select>
              </div>
              <div class="picker-list">
                <div v-for="unit in pickerUnits" :key="unit.Type" class="pk-item pk-unit" @click="pickUnit(unit)">
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
.recruit-root { display: flex; flex-direction: column; height: 100%; overflow: hidden; }

/* Sub-tabs */
.recruit-tabs {
  display: flex; align-items: center; flex-shrink: 0;
  background: var(--color-secondary); border-bottom: 1px solid var(--color-border); padding: 0 16px;
}
.rtab {
  padding: 10px 18px; background: transparent;
  border: none; border-bottom: 2px solid transparent;
  color: var(--color-text-muted); cursor: pointer; font-size: 13px;
  margin-bottom: -1px; transition: color 0.15s;
  align-self: stretch; display: flex; align-items: center;
}
.rtab:hover { color: var(--color-text-dim); }
.rtab.active { color: var(--color-text); border-bottom-color: var(--color-accent); }
.rtab-spacer { flex: 1; }
.tab-error { font-size: 11px; color: var(--color-error); margin-right: 8px; }
.btn-revert {
  background: transparent; border: 1px solid var(--color-border-soft); color: var(--color-text-muted);
  padding: 4px 12px; border-radius: 4px; cursor: pointer; font-size: 11px;
  white-space: nowrap; transition: all 0.15s;
}
.btn-revert:hover:not(:disabled) { border-color: var(--color-accent); color: var(--color-accent); background: rgba(149,232,225,0.2); }
.btn-revert:disabled { opacity: 0.4; cursor: not-allowed; }

/* Body */
.recruit-body { display: flex; flex: 1; overflow: hidden; }

/* Left panel */
.recruit-left {
  width: 270px; flex-shrink: 0;
  display: flex; flex-direction: column;
  border-right: 1px solid var(--color-border); overflow: hidden;
}
.recruit-filters {
  padding: 10px; display: flex; flex-direction: column; gap: 6px;
  border-bottom: 1px solid var(--color-border); flex-shrink: 0;
}
.recruit-search, .recruit-select {
  background: var(--color-secondary); border: 1px solid var(--color-border);
  color: var(--color-text); padding: 6px 10px; border-radius: 4px;
  font-size: 12px; width: 100%; box-sizing: border-box;
}
.recruit-search:focus, .recruit-select:focus { outline: none; border-color: var(--color-primary); }

/* Settlement tabs */
.settle-tabs { display: flex; gap: 2px; }
.settle-tab {
  flex: 1; background: var(--color-cell); border: 1px solid var(--color-border); border-radius: 3px;
  color: var(--color-text-muted); font-size: 10px; padding: 3px 0; cursor: pointer; text-align: center;
}
.settle-tab:hover { color: var(--color-text-dim); border-color: var(--color-border-soft); }
.settle-tab.active { background: var(--color-primary); border-color: var(--color-primary-hover); color: var(--color-accent); }

/* Unit list */
.recruit-list { flex: 1; overflow-y: auto; padding: 8px; display: flex; flex-direction: column; gap: 2px; }
.recruit-unit-item { padding: 8px 10px; border-radius: 4px; cursor: pointer; }
.recruit-unit-item:hover { background: var(--color-cell); }
.recruit-unit-item.active { background: var(--color-primary); }
.rui-name { font-size: 13px; color: var(--color-text); }
.rui-meta { font-size: 11px; color: var(--color-text-muted); margin-top: 2px; }
.rui-faction { font-size: 10px; color: var(--color-text-muted); margin-top: 1px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.list-empty { color: var(--color-text-muted); font-size: 12px; text-align: center; padding: 20px; }

/* Building tree */
.bld-tree { flex: 1; overflow-y: auto; padding: 8px; }
.bld-group { margin-bottom: 2px; }
.bld-group-head {
  display: flex; align-items: center; gap: 6px;
  padding: 8px 10px; border-radius: 4px; cursor: pointer;
  font-size: 13px; color: var(--color-text-dim); font-weight: 500; user-select: none;
}
.bld-group-head:hover { background: var(--color-cell); }
.bld-arrow { font-size: 10px; color: var(--color-text-muted); width: 10px; }
.bld-levels { padding-left: 16px; margin-bottom: 4px; }
.bld-level {
  display: flex; align-items: center; justify-content: space-between;
  padding: 6px 10px; border-radius: 4px; cursor: pointer;
}
.bld-level:hover { background: var(--color-cell); }
.bld-level.active { background: var(--color-primary); }
.bld-level-left { display: flex; align-items: center; gap: 6px; min-width: 0; }
.bld-level-name { font-size: 12px; color: var(--color-text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.bld-level.active .bld-level-name { color: var(--color-text); }
.bld-level-meta { font-size: 10px; color: var(--color-text-muted); flex-shrink: 0; }
.bld-settle-badge {
  font-size: 8px; text-transform: uppercase; letter-spacing: 0.05em;
  padding: 1px 5px; border-radius: 8px; flex-shrink: 0;
}
.type-city  .bld-settle-badge { background: #e0f5e8; color: #3aba80; border: 1px solid #6aba8a; }
.type-castle .bld-settle-badge { background: #f0e0f8; color: #9060ba; border: 1px solid #a06aba; }

/* Right panel */
.recruit-right { flex: 1; display: flex; flex-direction: column; overflow: hidden; position: relative; }

.right-header {
  display: flex; align-items: center; gap: 12px;
  padding: 14px 20px; border-bottom: 1px solid var(--color-border); flex-shrink: 0;
}
.rh-title { font-size: 15px; font-weight: 600; }
.rh-sub { font-size: 12px; color: var(--color-text-muted); flex: 1; }
.btn-add {
  background: transparent; border: 1px solid var(--color-border); color: var(--color-text-muted);
  padding: 5px 12px; border-radius: 4px; cursor: pointer; font-size: 12px; white-space: nowrap;
}
.btn-add:hover { border-color: var(--color-primary); color: var(--color-text-dim); background: var(--color-primary); }

/* Blocks area */
.blocks-area {
  flex: 1; overflow-y: auto; padding: 16px 20px;
  display: flex; flex-wrap: wrap; align-content: flex-start; gap: 12px;
}
.bblock {
  position: relative; background: var(--color-cell); border: 1px solid var(--color-border);
  border-radius: 6px; padding: 12px 14px; width: 174px; cursor: pointer;
  transition: border-color 0.15s;
}
.bblock:hover { border-color: var(--color-border-soft); }
.bblock.selected { border-color: var(--color-accent); background: #fce8f5; }
.bb-group-label { font-size: 10px; text-transform: uppercase; color: var(--color-heading); letter-spacing: 0.05em; margin-bottom: 4px; }
.bb-level-name { font-size: 13px; font-weight: 600; margin-bottom: 8px; line-height: 1.2; display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.bb-settle-tag {
  font-size: 8px; text-transform: uppercase; letter-spacing: 0.05em; padding: 1px 5px; border-radius: 8px;
}
.bb-settle-tag.city   { background: #e0f5e8; color: #3aba80; border: 1px solid #6aba8a; }
.bb-settle-tag.castle { background: #f0e0f8; color: #9060ba; border: 1px solid #a06aba; }
.bb-row { display: flex; justify-content: space-between; font-size: 11px; }
.bb-key { color: var(--color-text-muted); }
.bb-val { color: var(--color-text-dim); }
.bb-cond { font-size: 10px; color: var(--color-text-muted); margin-top: 5px; word-break: break-all; }
.bb-del {
  position: absolute; top: 8px; right: 8px;
  background: none; border: none; color: var(--color-border-soft); cursor: pointer;
  font-size: 11px; padding: 2px 5px; border-radius: 2px; line-height: 1;
}
.bb-del:hover { color: var(--color-accent) !important; background: rgba(149,232,225,0.2); }
.bb-del:disabled { opacity: 0.3; pointer-events: none; }

/* Edit panel */
.edit-panel {
  flex-shrink: 0; border-top: 1px solid var(--color-border);
  padding: 14px 20px; background: var(--color-cell);
}
.ep-title {
  font-size: 11px; text-transform: uppercase; color: var(--color-accent);
  letter-spacing: 0.05em; margin-bottom: 10px;
}
.ep-fields { display: flex; align-items: flex-end; gap: 12px; flex-wrap: wrap; }
.ep-fields label {
  display: flex; flex-direction: column; gap: 4px;
  font-size: 11px; color: var(--color-text-dim);
}
.ep-fields input {
  background: var(--color-secondary); border: 1px solid var(--color-border);
  color: var(--color-text); padding: 5px 8px; border-radius: 4px;
  font-size: 12px; width: 72px;
}
.ep-fields input:focus { outline: none; border-color: var(--color-primary); }
.ep-wide { flex: 1; min-width: 160px; }
.ep-cond { width: 100% !important; font-family: monospace; font-size: 11px; }
.ep-save {
  background: var(--color-primary); color: var(--color-text); border: none;
  border-radius: 4px; padding: 5px 16px; font-size: 12px; cursor: pointer;
}
.ep-save:hover { background: var(--color-primary-hover); }
.ep-save:disabled { opacity: 0.5; cursor: default; }
.ep-error { font-size: 11px; color: var(--color-error); }

/* Hints */
.area-hint { color: var(--color-border-soft); font-size: 13px; width: 100%; padding-top: 8px; }
.right-hint { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--color-border-soft); font-size: 13px; }

/* Picker */
.picker-overlay {
  position: absolute; inset: 0; background: var(--color-secondary);
  display: flex; flex-direction: column; z-index: 10; border-left: 1px solid var(--color-border);
}
.picker-head {
  display: flex; align-items: center; padding: 12px 16px;
  border-bottom: 1px solid var(--color-border); flex-shrink: 0; gap: 10px;
}
.picker-title { font-size: 13px; font-weight: 600; color: var(--color-text); flex: 1; }
.picker-close { background: none; border: none; color: var(--color-text-muted); cursor: pointer; font-size: 14px; padding: 2px 6px; border-radius: 3px; }
.picker-close:hover { color: var(--color-accent); background: rgba(149,232,225,0.2); }
.picker-search-row {
  padding: 10px 14px; display: flex; flex-direction: column; gap: 8px;
  border-bottom: 1px solid var(--color-border); flex-shrink: 0;
}
.picker-list { flex: 1; overflow-y: auto; padding: 8px; }
.pk-group { margin-bottom: 4px; }
.pk-group-label { font-size: 10px; text-transform: uppercase; letter-spacing: 0.05em; color: var(--color-heading); padding: 6px 10px 4px; }
.pk-item { display: flex; align-items: center; justify-content: space-between; padding: 7px 12px; border-radius: 4px; cursor: pointer; gap: 10px; }
.pk-item:hover { background: var(--color-cell); }
.pk-unit { flex-direction: column; align-items: flex-start; gap: 2px; }
.pk-name { font-size: 13px; color: var(--color-text); }
.pk-meta { font-size: 11px; color: var(--color-text-muted); }
.pk-empty { color: var(--color-text-muted); font-size: 12px; text-align: center; padding: 20px; }

/* Animations */
.slide-up-enter-active, .slide-up-leave-active { transition: opacity 0.15s, transform 0.15s; }
.slide-up-enter-from, .slide-up-leave-to { opacity: 0; transform: translateY(8px); }
.picker-slide-enter-active, .picker-slide-leave-active { transition: opacity 0.15s, transform 0.15s; }
.picker-slide-enter-from, .picker-slide-leave-to { opacity: 0; transform: translateX(12px); }

/* Loading / error */
.recruit-loading, .recruit-error { flex: 1; display: flex; align-items: center; justify-content: center; font-size: 13px; color: var(--color-border-soft); }
.recruit-error { color: var(--color-error); }
</style>
