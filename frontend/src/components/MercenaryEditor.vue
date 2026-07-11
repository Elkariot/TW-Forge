<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  GetMercenaryCampaigns, LoadMercenaryCampaign, GetMercenaryPools, GetMercenaryRegions,
  GetMercenaryPoolChangeType, UpdateMercenaryPool, RenameMercenaryPool, CreateMercenaryPool,
  DeleteMercenaryPool, RevertMercenaryPool, RevertAllMercenaries,
  GetFactions, GetAllUnits, GetM2TWFactions, GetM2TWAllUnits, GetM2TWReligions,
  AddUnitToFaction, AddM2TWUnitToFaction, HasRexEngine,
} from '../../wailsjs/go/main/App'

const props = defineProps({ isM2TW: { type: Boolean, default: false } })
const emit = defineEmits(['changed'])
const { t } = useI18n()

const subTab = ref('units') // 'units' | 'regions' (пулы) | 'factions'

// ── Campaign ──────────────────────────────────────────────────────────────
const campaigns = ref([])
const selectedCampaign = ref(null)
const loadingCampaign = ref(false)
const campaignError = ref(null)

function campaignLabel(c) {
  const last = c.split('/').pop()
  return last.replace(/_/g, ' ').replace(/\b\w/g, ch => ch.toUpperCase())
}

async function loadCampaign(name) {
  loadingCampaign.value = true
  campaignError.value = null
  try {
    await LoadMercenaryCampaign(name)
    selectedCampaign.value = name
    pools.value = await GetMercenaryPools()
    regions.value = await GetMercenaryRegions() ?? []
  } catch (e) {
    campaignError.value = String(e)
  } finally {
    loadingCampaign.value = false
  }
  selectedName.value = null
  edited.value = null
  dirty.value = false
  selectedUnitName.value = null
  selectedOccurrence.value = null
}

onMounted(async () => {
  hasRex.value = await HasRexEngine()
  if (props.isM2TW) religionsList.value = await GetM2TWReligions() ?? []
  campaigns.value = await GetMercenaryCampaigns() ?? []
  if (campaigns.value.length > 0) await loadCampaign(campaigns.value[0])
  await ensureFactionsLoaded()
})

const hasRex = ref(false)
const religionsList = ref([])

function switchTab(name) { subTab.value = name }

// ── Shared: pools + flattened per-unit index ────────────────────────────────
const pools = ref([])
const regions = ref([])

const factions = ref([])
const allUnits = ref([])
let factionsLoaded = false
const factionsLoading = ref(false)

async function ensureFactionsLoaded() {
  if (factionsLoaded) return
  factionsLoading.value = true
  try {
    const [f, units] = await Promise.all([
      props.isM2TW ? GetM2TWFactions() : GetFactions(),
      props.isM2TW ? GetM2TWAllUnits() : GetAllUnits(),
    ])
    factions.value = f ?? []
    allUnits.value = units ?? []
    factionsLoaded = true
  } finally {
    factionsLoading.value = false
  }
}

async function refreshPools() {
  pools.value = await GetMercenaryPools()
}

function normalizeMercName(name) { return (name || '').trim().replace(/,\s*$/, '') }

// Один наёмник может встречаться во множестве пулов с разными характеристиками —
// этот индекс схлопывает их в одну запись (для вкладок "По юнитам"/"По фракциям"),
// заодно собирая религии/крестовый поход по всем вхождениям сразу для фильтров.
const mercUnitIndex = computed(() => {
  const map = new Map()
  for (const p of pools.value) {
    for (const u of p.Units ?? []) {
      const key = normalizeMercName(u.Name)
      if (!key) continue
      if (!map.has(key)) map.set(key, { name: key, poolCount: 0, religions: new Set(), crusading: false })
      const entry = map.get(key)
      entry.poolCount++
      for (const r of u.Religions ?? []) entry.religions.add(r)
      if (u.Crusading) entry.crusading = true
    }
  }
  const list = [...map.values()]
  for (const entry of list) {
    entry.matched = allUnits.value.find(u => u.Type === entry.name) ?? null
    entry.displayName = entry.matched?.Name || entry.name
    entry.religions = [...entry.religions]
  }
  list.sort((a, b) => a.displayName.localeCompare(b.displayName))
  return list
})

function newUnit() {
  return {
    Name: '', Exp: 0, Cost: 100, ReplenishLow: 0.1, ReplenishHigh: 0.3, Max: 1, Initial: 1,
    Armour: null, WeaponLvl: null, EndYear: null, StartYear: null,
    Religions: [], Crusading: false, Events: [],
  }
}

function cloneEdit(p) { return JSON.parse(JSON.stringify(p)) }

// Общие геттеры/сеттеры для опциональных и списочных полей юнита-наёмника —
// используются и на вкладке "По регионам" (юниты внутри пула), и на вкладке
// "По юнитам" (одно вхождение наёмника в конкретном пуле).
function toggleOptional(unit, field, defaultValue) { unit[field] = unit[field] == null ? defaultValue : null }
function toggleUnitReligion(unit, religion, checked) {
  const cur = unit.Religions ?? []
  unit.Religions = checked ? [...cur, religion] : cur.filter(r => r !== religion)
}
function eventsText(unit) { return (unit.Events ?? []).join(', ') }
function setEventsText(unit, text) { unit.Events = text.split(',').map(s => s.trim()).filter(Boolean) }

// ═══ Вкладка "По юнитам" ═════════════════════════════════════════════════

const unitsSearch = ref('')
const unitsReligionFilter = ref('all')
const unitsCrusadingOnly = ref(false)
const unitsEduFilter = ref('all') // all | found | missing

const filteredUnitIndex = computed(() => {
  const q = unitsSearch.value.trim().toLowerCase()
  let list = mercUnitIndex.value
  if (q) list = list.filter(e => e.name.toLowerCase().includes(q) || e.displayName.toLowerCase().includes(q))
  if (props.isM2TW) {
    if (unitsReligionFilter.value !== 'all') list = list.filter(e => e.religions.includes(unitsReligionFilter.value))
    if (unitsCrusadingOnly.value) list = list.filter(e => e.crusading)
  }
  if (unitsEduFilter.value === 'found') list = list.filter(e => e.matched)
  else if (unitsEduFilter.value === 'missing') list = list.filter(e => !e.matched)

  // Юниты, не найденные в EDU — в самый низ списка (см. п.3 замечаний).
  return [...list].sort((a, b) => {
    const rank = e => e.matched ? 0 : 1
    const diff = rank(a) - rank(b)
    return diff !== 0 ? diff : a.displayName.localeCompare(b.displayName)
  })
})

const selectedUnitName = ref(null)
const selectedUnitMeta = computed(() => mercUnitIndex.value.find(e => e.name === selectedUnitName.value) ?? null)

function selectUnit(entry) {
  selectedUnitName.value = entry.name
  selectedOccurrence.value = null
}

// Все вхождения выбранного наёмника — по одному на каждый пул, где он встречается.
const unitOccurrences = computed(() => {
  if (!selectedUnitName.value) return []
  const result = []
  for (const p of pools.value) {
    (p.Units ?? []).forEach((u, idx) => {
      if (normalizeMercName(u.Name) === selectedUnitName.value) {
        result.push({ poolName: p.Name, poolRegions: p.Regions ?? [], unitIndex: idx, unit: u })
      }
    })
  }
  return result
})

const availablePoolsToAdd = computed(() => {
  const already = new Set(unitOccurrences.value.map(o => o.poolName))
  return pools.value.filter(p => !already.has(p.Name))
})

const selectedOccurrence = ref(null) // { poolName, unitIndex, unit (editable copy), regions (editable copy) }
const occurrenceSaving = ref(false)
const occurrenceError = ref(null)

function selectOccurrence(occ) {
  selectedOccurrence.value = {
    poolName: occ.poolName,
    unitIndex: occ.unitIndex,
    unit: cloneEdit(occ.unit),
    regions: [...(occ.poolRegions ?? [])],
  }
  occurrenceError.value = null
}

function closeOccurrence() { selectedOccurrence.value = null }

function addOccRegion(name) {
  const v = (name || '').trim()
  if (!v || !selectedOccurrence.value) return
  if (!selectedOccurrence.value.regions.includes(v)) selectedOccurrence.value.regions.push(v)
}
function removeOccRegion(name) {
  selectedOccurrence.value.regions = selectedOccurrence.value.regions.filter(r => r !== name)
}
const occCustomRegion = ref('')
function addOccCustomRegion() { addOccRegion(occCustomRegion.value); occCustomRegion.value = '' }

async function saveOccurrence() {
  const occ = selectedOccurrence.value
  const targetPool = pools.value.find(p => p.Name === occ.poolName)
  if (!targetPool) return
  occurrenceSaving.value = true
  occurrenceError.value = null
  try {
    const newUnits = targetPool.Units.map((u, i) => (i === occ.unitIndex ? occ.unit : u))
    await UpdateMercenaryPool(occ.poolName, occ.regions, newUnits)
    await refreshPools()
    selectedOccurrence.value = null
    emit('changed')
  } catch (e) {
    occurrenceError.value = String(e)
  } finally {
    occurrenceSaving.value = false
  }
}

async function removeOccurrence(occ) {
  if (!confirm(t('mercenary.remove_occurrence_confirm', { pool: occ.poolName }))) return
  const targetPool = pools.value.find(p => p.Name === occ.poolName)
  if (!targetPool) return
  try {
    const newUnits = targetPool.Units.filter((_, i) => i !== occ.unitIndex)
    await UpdateMercenaryPool(occ.poolName, targetPool.Regions, newUnits)
    await refreshPools()
    if (selectedOccurrence.value?.poolName === occ.poolName) selectedOccurrence.value = null
    emit('changed')
  } catch (e) {
    occurrenceError.value = String(e)
  }
}

const unitsAddFactionTarget = ref('')
const unitsAddingFaction = ref(false)
const unitsAddFactionError = ref(null)

const availableFactionsToAdd = computed(() => {
  const owned = new Set(selectedUnitMeta.value?.matched?.Ownership ?? [])
  return factions.value.filter(f => !owned.has(f.Name))
})

async function addFactionToSelectedUnit() {
  const factionName = unitsAddFactionTarget.value
  const type = selectedUnitMeta.value?.matched?.Type
  if (!factionName || !type) return
  unitsAddingFaction.value = true
  unitsAddFactionError.value = null
  try {
    if (props.isM2TW) await AddM2TWUnitToFaction(type, factionName)
    else await AddUnitToFaction(type, factionName)
    allUnits.value = props.isM2TW ? await GetM2TWAllUnits() : await GetAllUnits()
    unitsAddFactionTarget.value = ''
    emit('changed')
  } catch (e) {
    unitsAddFactionError.value = String(e)
  } finally {
    unitsAddingFaction.value = false
  }
}

const addToPoolTarget = ref('')
async function addToPool() {
  const poolName = addToPoolTarget.value
  if (!poolName || !selectedUnitName.value) return
  const targetPool = pools.value.find(p => p.Name === poolName)
  if (!targetPool) return
  try {
    const unit = { ...newUnit(), Name: selectedUnitName.value }
    await UpdateMercenaryPool(poolName, targetPool.Regions ?? [], [...(targetPool.Units ?? []), unit])
    await refreshPools()
    addToPoolTarget.value = ''
    emit('changed')
  } catch (e) {
    occurrenceError.value = String(e)
  }
}

// ═══ Вкладка "По регионам" (пулы) ════════════════════════════════════════

const search = ref('')

const selectedName = ref(null)
const pool = ref(null)     // last saved snapshot
const edited = ref(null)   // working copy
const dirty = ref(false)
const saving = ref(false)
const error = ref(null)
const changeType = ref('none')

const filteredPools = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return pools.value
  return pools.value.filter(p =>
    p.Name.toLowerCase().includes(q) || (p.Units ?? []).some(u => u.Name.toLowerCase().includes(q))
  )
})

async function selectPool(p) {
  if (dirty.value && !confirm(t('mercenary.unsaved_confirm'))) return
  selectedName.value = p.Name
  pool.value = p
  edited.value = cloneEdit(p)
  dirty.value = false
  error.value = null
  changeType.value = await GetMercenaryPoolChangeType(p.Name)
}

function markDirty() { dirty.value = true }

// ── Regions ──
function addRegion(name) {
  const v = (name || '').trim()
  if (!v) return
  if (!(edited.value.Regions ?? []).includes(v)) {
    edited.value.Regions = [...(edited.value.Regions ?? []), v]
    markDirty()
  }
}
function removeRegion(name) {
  edited.value.Regions = edited.value.Regions.filter(r => r !== name)
  markDirty()
}
const customRegion = ref('')
function addCustomRegion() {
  addRegion(customRegion.value)
  customRegion.value = ''
}

// ── Units within pool ──
function addPoolUnit() {
  edited.value.Units = [...(edited.value.Units ?? []), newUnit()]
  markDirty()
}
function removePoolUnit(idx) {
  edited.value.Units = edited.value.Units.filter((_, i) => i !== idx)
  markDirty()
}

// ── Save / Cancel / Revert / Delete / Rename / Create ──

async function save() {
  saving.value = true
  error.value = null
  try {
    await UpdateMercenaryPool(selectedName.value, edited.value.Regions ?? [], edited.value.Units ?? [])
    await refreshPools()
    const saved = pools.value.find(p => p.Name === selectedName.value)
    if (saved) { pool.value = saved; edited.value = cloneEdit(saved) }
    dirty.value = false
    changeType.value = await GetMercenaryPoolChangeType(selectedName.value)
    emit('changed')
  } catch (e) {
    error.value = String(e)
  } finally {
    saving.value = false
  }
}

function cancel() {
  edited.value = cloneEdit(pool.value)
  dirty.value = false
  error.value = null
}

async function revertOne() {
  try {
    await RevertMercenaryPool(selectedName.value)
    await refreshPools()
    const restored = pools.value.find(p => p.Name === selectedName.value)
    if (restored) await selectPool(restored)
    emit('changed')
  } catch (e) {
    error.value = String(e)
  }
}

async function revertAll() {
  if (!confirm(t('mercenary.revert_all_confirm'))) return
  await RevertAllMercenaries()
  await refreshPools()
  selectedName.value = null
  edited.value = null
  dirty.value = false
  emit('changed')
}

async function deletePool() {
  if (!confirm(t('mercenary.delete_pool_confirm', { name: selectedName.value }))) return
  try {
    await DeleteMercenaryPool(selectedName.value)
    await refreshPools()
    selectedName.value = null
    edited.value = null
    dirty.value = false
    emit('changed')
  } catch (e) {
    error.value = String(e)
  }
}

const renaming = ref(false)
const renameValue = ref('')
function startRename() { renameValue.value = selectedName.value; renaming.value = true }
async function confirmRename() {
  const v = renameValue.value.trim()
  if (!v || v === selectedName.value) { renaming.value = false; return }
  try {
    await RenameMercenaryPool(selectedName.value, v)
    await refreshPools()
    const renamed = pools.value.find(p => p.Name === v)
    if (renamed) await selectPool(renamed)
    renaming.value = false
    emit('changed')
  } catch (e) {
    error.value = String(e)
  }
}

const creatingPool = ref(false)
const newPoolName = ref('')
async function createPool() {
  const v = newPoolName.value.trim()
  if (!v) return
  try {
    await CreateMercenaryPool(v, [])
    await refreshPools()
    creatingPool.value = false
    newPoolName.value = ''
    const created = pools.value.find(p => p.Name === v)
    if (created) await selectPool(created)
    emit('changed')
  } catch (e) {
    campaignError.value = String(e)
  }
}

// ═══ Вкладка "По фракциям": добавить существующего наёмника фракции ═══════

const selectedFactionName = ref(null)
const factionSearch = ref('')
const addingType = ref(null)
const addError = ref(null)

function alreadyOwns(entry) {
  return !!(entry.matched && selectedFactionName.value && entry.matched.Ownership?.includes(selectedFactionName.value))
}

// Порядок: уже доступные этой фракции наёмники → остальные (кого можно добавить) →
// юниты, не найденные в EDU (в самый низ — по-хорошему их вообще быть не должно).
function factionSortRank(entry) {
  if (!entry.matched) return 2
  return alreadyOwns(entry) ? 0 : 1
}

const filteredMercUnits = computed(() => {
  const q = factionSearch.value.trim().toLowerCase()
  let list = mercUnitIndex.value
  if (q) {
    list = list.filter(e => e.name.toLowerCase().includes(q) || e.displayName.toLowerCase().includes(q))
  }
  return [...list].sort((a, b) => {
    const diff = factionSortRank(a) - factionSortRank(b)
    return diff !== 0 ? diff : a.displayName.localeCompare(b.displayName)
  })
})

const selectedFactionDisplay = computed(() => {
  const f = factions.value.find(f => f.Name === selectedFactionName.value)
  return f?.DisplayName || selectedFactionName.value
})

function goToUnit(entry) {
  subTab.value = 'units'
  selectUnit(entry)
}

async function addToFaction(entry) {
  if (!entry.matched || !selectedFactionName.value || alreadyOwns(entry)) return
  addingType.value = entry.name
  addError.value = null
  try {
    if (props.isM2TW) await AddM2TWUnitToFaction(entry.matched.Type, selectedFactionName.value)
    else await AddUnitToFaction(entry.matched.Type, selectedFactionName.value)
    allUnits.value = props.isM2TW ? await GetM2TWAllUnits() : await GetAllUnits()
    emit('changed')
  } catch (e) {
    addError.value = String(e)
  } finally {
    addingType.value = null
  }
}
</script>

<template>
  <div class="me-root">

    <!-- Кампания + под-вкладки -->
    <div class="me-topbar">
      <select
        v-if="campaigns.length > 0"
        class="me-campaign-select"
        :value="selectedCampaign"
        @change="e => loadCampaign(e.target.value)"
      >
        <option v-for="c in campaigns" :key="c" :value="c" :title="c">{{ campaignLabel(c) }}</option>
      </select>

      <div class="me-tabs">
        <button :class="['metab', { active: subTab === 'units' }]" @click="switchTab('units')">{{ $t('mercenary.tab_by_units') }}</button>
        <button :class="['metab', { active: subTab === 'regions' }]" @click="switchTab('regions')">{{ $t('mercenary.tab_by_regions') }}</button>
        <button :class="['metab', { active: subTab === 'factions' }]" @click="switchTab('factions')">{{ $t('mercenary.tab_by_factions') }}</button>
      </div>

      <div class="me-spacer"></div>
      <span v-if="campaignError" class="me-error">{{ campaignError }}</span>
      <button v-if="subTab === 'regions'" class="btn-revert-all" @click="revertAll">{{ $t('mercenary.revert_all') }}</button>
    </div>

    <div v-if="loadingCampaign" class="me-hint">{{ $t('common.loading') }}</div>
    <div v-else-if="campaigns.length === 0" class="me-hint">{{ $t('mercenary.no_campaigns') }}</div>

    <!-- ══ По юнитам (плоский список наёмников) ═══════════════════════════ -->
    <div v-else-if="subTab === 'units'" class="me-body">

      <div class="me-list">
        <div class="me-list-header">
          <input v-model="unitsSearch" class="me-search" :placeholder="$t('mercenary.search_placeholder')" />
        </div>

        <div class="me-filters">
          <select v-model="unitsEduFilter" class="me-filter-select">
            <option value="all">{{ $t('mercenary.filter_edu_all') }}</option>
            <option value="found">{{ $t('mercenary.filter_edu_found') }}</option>
            <option value="missing">{{ $t('mercenary.filter_edu_missing') }}</option>
          </select>
          <template v-if="isM2TW">
            <select v-model="unitsReligionFilter" class="me-filter-select">
              <option value="all">{{ $t('mercenary.filter_religion_all') }}</option>
              <option v-for="r in religionsList" :key="r" :value="r">{{ r }}</option>
            </select>
            <label class="checkbox-row me-filter-checkbox">
              <input type="checkbox" v-model="unitsCrusadingOnly" />
              {{ $t('mercenary.field_crusading') }}
            </label>
          </template>
        </div>

        <ul class="me-items">
          <li v-for="e in filteredUnitIndex" :key="e.name" :class="{ active: selectedUnitName === e.name, unmatched: !e.matched }" @click="selectUnit(e)">
            <span class="me-pool-name">{{ e.displayName }}</span>
            <span class="me-pool-count">{{ e.poolCount }}</span>
          </li>
          <li v-if="filteredUnitIndex.length === 0" class="me-list-empty">{{ $t('common.nothing_found') }}</li>
        </ul>
      </div>

      <div class="me-detail">
        <template v-if="selectedUnitName">
          <div class="me-detail-header">
            <div class="me-detail-title">
              <span class="me-pool-title">{{ selectedUnitMeta?.displayName }}</span>
              <span v-if="!selectedUnitMeta?.matched" class="me-badge me-badge-warn">{{ $t('mercenary.unmatched_hint') }}</span>
            </div>
            <div class="me-detail-actions">
              <span v-if="occurrenceError" class="me-error">{{ occurrenceError }}</span>
              <select v-if="availablePoolsToAdd.length" v-model="addToPoolTarget" class="me-filter-select" @change="addToPool">
                <option value="">{{ $t('mercenary.add_to_pool') }}</option>
                <option v-for="p in availablePoolsToAdd" :key="p.Name" :value="p.Name">{{ p.Name }}</option>
              </select>
            </div>
          </div>

          <div class="me-scroll">

            <!-- Доступные фракции (из EDU ownership) + добавление новой -->
            <section class="me-section">
              <h3>{{ $t('mercenary.section_available_factions') }}</h3>
              <div v-if="selectedUnitMeta?.matched" class="ownership-chips">
                <span v-for="f in selectedUnitMeta.matched.Ownership" :key="f" class="chip chip--readonly">{{ f }}</span>
                <span v-if="!selectedUnitMeta.matched.Ownership?.length" class="me-muted">{{ $t('mercenary.no_owners') }}</span>
              </div>
              <div v-if="selectedUnitMeta?.matched" class="me-region-add">
                <span v-if="unitsAddFactionError" class="me-error">{{ unitsAddFactionError }}</span>
                <select v-model="unitsAddFactionTarget" class="me-filter-select" :disabled="unitsAddingFaction">
                  <option value="">{{ $t('mercenary.add_to_faction') }}</option>
                  <option v-for="f in availableFactionsToAdd" :key="f.Name" :value="f.Name">{{ f.DisplayName || f.Name }}</option>
                </select>
                <button class="btn-small-add" :disabled="!unitsAddFactionTarget || unitsAddingFaction" @click="addFactionToSelectedUnit">
                  {{ unitsAddingFaction ? $t('common.saving') : $t('mercenary.add_to_faction') }}
                </button>
              </div>
              <div v-else class="me-muted">{{ $t('mercenary.unmatched_hint') }}</div>
            </section>

            <!-- Карточки по каждому пулу, где встречается наёмник -->
            <section class="me-section">
              <h3>{{ $t('mercenary.section_occurrences') }}</h3>
              <div class="occ-grid">
                <div
                  v-for="occ in unitOccurrences"
                  :key="occ.poolName + occ.unitIndex"
                  class="occ-card"
                  :class="{ selected: selectedOccurrence?.poolName === occ.poolName && selectedOccurrence?.unitIndex === occ.unitIndex }"
                  @click="selectOccurrence(occ)"
                >
                  <div class="occ-pool-name">{{ occ.poolName }}</div>
                  <div class="occ-regions">{{ (occ.poolRegions ?? []).join(', ') || $t('mercenary.no_regions') }}</div>
                  <div class="bb-row"><span class="bb-key">{{ $t('mercenary.field_cost') }}</span><span class="bb-val">{{ occ.unit.Cost }}</span></div>
                  <div class="bb-row"><span class="bb-key">{{ $t('mercenary.field_exp') }}</span><span class="bb-val">{{ occ.unit.Exp }}</span></div>
                  <button class="pool-remove occ-remove" @click.stop="removeOccurrence(occ)">✕</button>
                </div>
                <div v-if="unitOccurrences.length === 0" class="me-muted">{{ $t('mercenary.no_units_in_pool') }}</div>
              </div>
            </section>

            <!-- Панель редактирования выбранного вхождения -->
            <section v-if="selectedOccurrence" class="me-section">
              <div class="section-header-row">
                <h3>{{ $t('mercenary.section_edit_occurrence', { pool: selectedOccurrence.poolName }) }}</h3>
                <div class="me-detail-actions">
                  <button class="btn-save" :disabled="occurrenceSaving" @click="saveOccurrence">{{ occurrenceSaving ? $t('common.saving') : $t('common.save') }}</button>
                  <button class="btn-cancel" @click="closeOccurrence">{{ $t('common.cancel') }}</button>
                </div>
              </div>

              <div class="occ-regions-hint">{{ $t('mercenary.pool_regions_hint') }}</div>
              <div class="ownership-chips">
                <span v-for="r in selectedOccurrence.regions" :key="r" class="chip">
                  {{ r }}
                  <button class="chip-remove" @click="removeOccRegion(r)">✕</button>
                </span>
                <span v-if="!selectedOccurrence.regions.length" class="me-muted">{{ $t('mercenary.no_regions') }}</span>
              </div>
              <div class="me-region-add">
                <input
                  v-model="occCustomRegion"
                  list="me-regions-datalist"
                  class="me-region-input"
                  :placeholder="$t('mercenary.add_region')"
                  @keydown.enter="addOccCustomRegion"
                />
                <button class="btn-small-add" @click="addOccCustomRegion">+</button>
              </div>
              <datalist id="me-regions-datalist">
                <option v-for="r in regions.filter(r => !selectedOccurrence.regions.includes(r))" :key="r" :value="r" />
              </datalist>

              <div class="merc-card occ-edit-card">
                <div class="merc-grid">
                  <label>{{ $t('mercenary.field_exp') }}<input type="number" v-model.number="selectedOccurrence.unit.Exp" /></label>
                  <label>{{ $t('mercenary.field_cost') }}<input type="number" v-model.number="selectedOccurrence.unit.Cost" /></label>
                  <label>{{ $t('mercenary.field_replenish_low') }}<input type="number" step="0.01" v-model.number="selectedOccurrence.unit.ReplenishLow" /></label>
                  <label>{{ $t('mercenary.field_replenish_high') }}<input type="number" step="0.01" v-model.number="selectedOccurrence.unit.ReplenishHigh" /></label>
                  <label>{{ $t('mercenary.field_max') }}<input type="number" v-model.number="selectedOccurrence.unit.Max" /></label>
                  <label>{{ $t('mercenary.field_initial') }}<input type="number" v-model.number="selectedOccurrence.unit.Initial" /></label>
                </div>

                <div v-if="hasRex" class="merc-optional-row">
                  <label class="checkbox-row">
                    <input type="checkbox" :checked="selectedOccurrence.unit.Armour != null" @change="e => toggleOptional(selectedOccurrence.unit, 'Armour', e.target.checked ? 0 : null)" />
                    {{ $t('mercenary.field_armour') }}
                  </label>
                  <input v-if="selectedOccurrence.unit.Armour != null" type="number" class="merc-inline-input" v-model.number="selectedOccurrence.unit.Armour" />
                  <label class="checkbox-row">
                    <input type="checkbox" :checked="selectedOccurrence.unit.WeaponLvl != null" @change="e => toggleOptional(selectedOccurrence.unit, 'WeaponLvl', e.target.checked ? 0 : null)" />
                    {{ $t('mercenary.field_weapon_lvl') }}
                  </label>
                  <input v-if="selectedOccurrence.unit.WeaponLvl != null" type="number" class="merc-inline-input" v-model.number="selectedOccurrence.unit.WeaponLvl" />
                </div>

                <template v-if="isM2TW">
                  <div class="merc-optional-row">
                    <label class="checkbox-row">
                      <input type="checkbox" :checked="selectedOccurrence.unit.StartYear != null" @change="e => toggleOptional(selectedOccurrence.unit, 'StartYear', e.target.checked ? 1200 : null)" />
                      {{ $t('mercenary.field_start_year') }}
                    </label>
                    <input v-if="selectedOccurrence.unit.StartYear != null" type="number" class="merc-inline-input" v-model.number="selectedOccurrence.unit.StartYear" />
                    <label class="checkbox-row">
                      <input type="checkbox" :checked="selectedOccurrence.unit.EndYear != null" @change="e => toggleOptional(selectedOccurrence.unit, 'EndYear', e.target.checked ? 1300 : null)" />
                      {{ $t('mercenary.field_end_year') }}
                    </label>
                    <input v-if="selectedOccurrence.unit.EndYear != null" type="number" class="merc-inline-input" v-model.number="selectedOccurrence.unit.EndYear" />
                    <label class="checkbox-row">
                      <input type="checkbox" v-model="selectedOccurrence.unit.Crusading" />
                      {{ $t('mercenary.field_crusading') }}
                    </label>
                  </div>

                  <div v-if="religionsList.length" class="merc-religions">
                    <span class="merc-religions-label">{{ $t('mercenary.field_religions') }}</span>
                    <label v-for="r in religionsList" :key="r" class="religion-check">
                      <input type="checkbox" :checked="(selectedOccurrence.unit.Religions ?? []).includes(r)" @change="e => toggleUnitReligion(selectedOccurrence.unit, r, e.target.checked)" />
                      {{ r }}
                    </label>
                  </div>

                  <label class="merc-events-label">{{ $t('mercenary.field_events') }}
                    <input class="merc-events-input" :value="eventsText(selectedOccurrence.unit)" @change="e => setEventsText(selectedOccurrence.unit, e.target.value)" />
                  </label>
                </template>
              </div>
            </section>

          </div>
        </template>
        <div v-else class="me-empty">{{ $t('mercenary.select_unit_hint') }}</div>
      </div>
    </div>

    <!-- ══ По регионам (пулы) ═══════════════════════════════════════════════ -->
    <div v-else-if="subTab === 'regions'" class="me-body">

      <div class="me-list">
        <div class="me-list-header">
          <input v-model="search" class="me-search" :placeholder="$t('mercenary.search_placeholder')" />
        </div>

        <ul class="me-items">
          <li v-for="p in filteredPools" :key="p.Name" :class="{ active: selectedName === p.Name }" @click="selectPool(p)">
            <span class="me-pool-name">{{ p.Name }}</span>
            <span class="me-pool-count">{{ (p.Units ?? []).length }}</span>
          </li>
          <li v-if="filteredPools.length === 0" class="me-list-empty">{{ $t('common.nothing_found') }}</li>
        </ul>

        <div v-if="!creatingPool" class="me-add-pool">
          <button class="btn-small-add" @click="creatingPool = true">{{ $t('mercenary.add_pool') }}</button>
        </div>
        <div v-else class="me-create-row">
          <input v-model="newPoolName" class="me-search" :placeholder="$t('mercenary.new_pool_placeholder')" @keydown.enter="createPool" autofocus />
          <button class="btn-small-add" @click="createPool">{{ $t('mercenary.create') }}</button>
          <button class="btn-cancel-inline" @click="creatingPool = false; newPoolName = ''">✕</button>
        </div>
      </div>

      <div class="me-detail">
        <template v-if="edited">

          <div class="me-detail-header">
            <div class="me-detail-title">
              <template v-if="!renaming">
                <span class="me-pool-title" @click="startRename" :title="$t('mercenary.rename_pool_title')">{{ selectedName }}</span>
              </template>
              <template v-else>
                <input v-model="renameValue" class="me-rename-input" @keydown.enter="confirmRename" @keydown.esc="renaming = false" autofocus />
                <button class="btn-cancel-inline" @click="confirmRename">✓</button>
                <button class="btn-cancel-inline" @click="renaming = false">✕</button>
              </template>
              <span v-if="changeType === 'modified'" class="me-badge">{{ $t('mercenary.modified_badge') }}</span>
              <span v-else-if="changeType === 'added'" class="me-badge me-badge-added">{{ $t('mercenary.added_badge') }}</span>
            </div>
            <div class="me-detail-actions">
              <span v-if="error" class="me-error">{{ error }}</span>
              <template v-if="dirty">
                <button class="btn-save" :disabled="saving" @click="save">{{ saving ? $t('common.saving') : $t('common.save') }}</button>
                <button class="btn-cancel" @click="cancel">{{ $t('common.cancel') }}</button>
              </template>
              <button v-else-if="changeType !== 'none'" class="btn-revert" @click="revertOne" :title="$t('common.revert')">↺</button>
              <button class="btn-delete-pool" @click="deletePool">{{ $t('mercenary.delete_pool') }}</button>
            </div>
          </div>

          <div class="me-scroll">

            <!-- Регионы (prominently, first) -->
            <section class="me-section">
              <h3>{{ $t('mercenary.section_regions') }}</h3>
              <div class="ownership-chips">
                <span v-for="r in edited.Regions" :key="r" class="chip">
                  {{ r }}
                  <button class="chip-remove" @click="removeRegion(r)">✕</button>
                </span>
                <span v-if="!edited.Regions?.length" class="me-muted">{{ $t('mercenary.no_regions') }}</span>
              </div>
              <div class="me-region-add">
                <input
                  v-model="customRegion"
                  list="me-regions-datalist"
                  class="me-region-input"
                  :placeholder="$t('mercenary.add_region')"
                  @keydown.enter="addCustomRegion"
                />
                <button class="btn-small-add" @click="addCustomRegion">+</button>
              </div>
              <datalist id="me-regions-datalist">
                <option v-for="r in regions.filter(r => !(edited.Regions ?? []).includes(r))" :key="r" :value="r" />
              </datalist>
            </section>

            <!-- Юниты -->
            <section class="me-section">
              <div class="section-header-row">
                <h3>{{ $t('mercenary.section_units') }}</h3>
                <button class="btn-small-add" @click="addPoolUnit">{{ $t('mercenary.add_unit') }}</button>
              </div>

              <div v-for="(unit, idx) in edited.Units" :key="idx" class="merc-card">
                <div class="merc-card-header">
                  <input v-model="unit.Name" class="merc-name-input" :placeholder="$t('mercenary.field_name')" @input="markDirty" />
                  <button class="pool-remove" @click="removePoolUnit(idx)">✕</button>
                </div>

                <div class="merc-grid">
                  <label>{{ $t('mercenary.field_exp') }}<input type="number" v-model.number="unit.Exp" @input="markDirty" /></label>
                  <label>{{ $t('mercenary.field_cost') }}<input type="number" v-model.number="unit.Cost" @input="markDirty" /></label>
                  <label>{{ $t('mercenary.field_replenish_low') }}<input type="number" step="0.01" v-model.number="unit.ReplenishLow" @input="markDirty" /></label>
                  <label>{{ $t('mercenary.field_replenish_high') }}<input type="number" step="0.01" v-model.number="unit.ReplenishHigh" @input="markDirty" /></label>
                  <label>{{ $t('mercenary.field_max') }}<input type="number" v-model.number="unit.Max" @input="markDirty" /></label>
                  <label>{{ $t('mercenary.field_initial') }}<input type="number" v-model.number="unit.Initial" @input="markDirty" /></label>
                </div>

                <!-- REX/M2EX -->
                <div v-if="hasRex" class="merc-optional-row">
                  <label class="checkbox-row">
                    <input type="checkbox" :checked="unit.Armour != null" @change="e => { toggleOptional(unit, 'Armour', e.target.checked ? 0 : null); markDirty() }" />
                    {{ $t('mercenary.field_armour') }}
                  </label>
                  <input v-if="unit.Armour != null" type="number" class="merc-inline-input" v-model.number="unit.Armour" @input="markDirty" />
                  <label class="checkbox-row">
                    <input type="checkbox" :checked="unit.WeaponLvl != null" @change="e => { toggleOptional(unit, 'WeaponLvl', e.target.checked ? 0 : null); markDirty() }" />
                    {{ $t('mercenary.field_weapon_lvl') }}
                  </label>
                  <input v-if="unit.WeaponLvl != null" type="number" class="merc-inline-input" v-model.number="unit.WeaponLvl" @input="markDirty" />
                </div>

                <!-- M2TW-only -->
                <template v-if="isM2TW">
                  <div class="merc-optional-row">
                    <label class="checkbox-row">
                      <input type="checkbox" :checked="unit.StartYear != null" @change="e => { toggleOptional(unit, 'StartYear', e.target.checked ? 1200 : null); markDirty() }" />
                      {{ $t('mercenary.field_start_year') }}
                    </label>
                    <input v-if="unit.StartYear != null" type="number" class="merc-inline-input" v-model.number="unit.StartYear" @input="markDirty" />
                    <label class="checkbox-row">
                      <input type="checkbox" :checked="unit.EndYear != null" @change="e => { toggleOptional(unit, 'EndYear', e.target.checked ? 1300 : null); markDirty() }" />
                      {{ $t('mercenary.field_end_year') }}
                    </label>
                    <input v-if="unit.EndYear != null" type="number" class="merc-inline-input" v-model.number="unit.EndYear" @input="markDirty" />
                    <label class="checkbox-row">
                      <input type="checkbox" :checked="unit.Crusading" @change="e => { unit.Crusading = e.target.checked; markDirty() }" />
                      {{ $t('mercenary.field_crusading') }}
                    </label>
                  </div>

                  <div v-if="religionsList.length" class="merc-religions">
                    <span class="merc-religions-label">{{ $t('mercenary.field_religions') }}</span>
                    <label v-for="r in religionsList" :key="r" class="religion-check">
                      <input type="checkbox" :checked="(unit.Religions ?? []).includes(r)" @change="e => { toggleUnitReligion(unit, r, e.target.checked); markDirty() }" />
                      {{ r }}
                    </label>
                  </div>

                  <label class="merc-events-label">{{ $t('mercenary.field_events') }}
                    <input class="merc-events-input" :value="eventsText(unit)" @change="e => { setEventsText(unit, e.target.value); markDirty() }" />
                  </label>
                </template>
              </div>
              <div v-if="!edited.Units?.length" class="me-muted">{{ $t('mercenary.no_units_in_pool') }}</div>
            </section>

          </div>
        </template>
        <div v-else class="me-empty">{{ $t('mercenary.select_hint') }}</div>
      </div>
    </div>

    <!-- ══ По фракциям ═══════════════════════════════════════════════════ -->
    <div v-else class="me-body">

      <aside class="me-faction-list">
        <ul>
          <li v-for="f in factions" :key="f.Name" :class="{ active: selectedFactionName === f.Name }" @click="selectedFactionName = f.Name">
            {{ f.DisplayName || f.Name }}
          </li>
        </ul>
      </aside>

      <div class="me-detail">
        <div v-if="factionsLoading" class="me-hint">{{ $t('common.loading') }}</div>
        <template v-else-if="selectedFactionName">
          <div class="me-detail-header">
            <div class="me-detail-title"><span class="me-pool-title">{{ selectedFactionDisplay }}</span></div>
            <div class="me-detail-actions">
              <span v-if="addError" class="me-error">{{ addError }}</span>
              <input v-model="factionSearch" class="me-search" :placeholder="$t('mercenary.search_placeholder')" />
            </div>
          </div>
          <div class="me-scroll">
            <div class="merc-faction-grid">
              <div v-for="entry in filteredMercUnits" :key="entry.name" class="merc-faction-card" :class="{ unmatched: !entry.matched }" :title="$t('mercenary.go_to_unit_hint')" @click="goToUnit(entry)">
                <div class="mfc-name">{{ entry.displayName }}</div>
                <div class="mfc-pools">{{ $t('mercenary.pool_count', { n: entry.poolCount }) }}</div>
                <div v-if="!entry.matched" class="mfc-unmatched-hint">{{ $t('mercenary.unmatched_hint') }}</div>
                <template v-else>
                  <div class="mfc-ownership">{{ (entry.matched.Ownership ?? []).join(', ') || $t('mercenary.no_owners') }}</div>
                  <button
                    v-if="!alreadyOwns(entry)"
                    class="mfc-add-btn"
                    :disabled="addingType === entry.name"
                    @click.stop="addToFaction(entry)"
                  >{{ addingType === entry.name ? $t('common.saving') : $t('mercenary.add_to_faction') }}</button>
                  <span v-else class="mfc-owned-tag">{{ $t('mercenary.already_owned') }}</span>
                </template>
              </div>
              <div v-if="filteredMercUnits.length === 0" class="me-list-empty">{{ $t('common.nothing_found') }}</div>
            </div>
          </div>
        </template>
        <div v-else class="me-empty">{{ $t('mercenary.faction_hint_select') }}</div>
      </div>
    </div>

  </div>
</template>

<style scoped>
.me-root { display: flex; flex-direction: column; height: 100%; overflow: hidden; }

/* Topbar */
.me-topbar {
  display: flex; align-items: center; gap: 12px; flex-shrink: 0;
  background: var(--color-secondary); border-bottom: 1px solid var(--color-border); padding: 0 16px;
}
.me-campaign-select {
  background: var(--color-cell); border: 1px solid var(--color-border-soft); border-radius: 4px;
  color: var(--color-text-dim); font-size: 11px; padding: 5px 8px; margin: 8px 0; max-width: 220px;
}
.me-tabs { display: flex; align-self: stretch; }
.metab {
  padding: 10px 18px; background: transparent; border: none; border-bottom: 2px solid transparent;
  color: var(--color-text-muted); cursor: pointer; font-size: 13px; margin-bottom: -1px;
}
.metab:hover { color: var(--color-text-dim); }
.metab.active { color: var(--color-text); border-bottom-color: var(--color-accent); }
.me-spacer { flex: 1; }
.btn-revert-all {
  background: transparent; border: 1px solid var(--color-border-soft); color: var(--color-text-muted);
  padding: 4px 12px; border-radius: 4px; cursor: pointer; font-size: 11px;
}
.btn-revert-all:hover { border-color: var(--color-accent); color: var(--color-accent); }

.me-hint { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--color-text-muted); font-size: 13px; }
.me-error { font-size: 11px; color: var(--color-error); }

/* Body */
.me-body { display: flex; flex: 1; overflow: hidden; }

/* Pool / unit list */
.me-list { width: 260px; flex-shrink: 0; display: flex; flex-direction: column; border-right: 1px solid var(--color-border); overflow: hidden; padding: 10px; }
.me-list-header { margin-bottom: 8px; }
.me-search {
  width: 100%; box-sizing: border-box; background: var(--color-input-bg); border: 1px solid var(--color-border-soft);
  border-radius: 4px; color: var(--color-text); font-size: 12px; padding: 5px 8px;
}
.me-search:focus { outline: none; border-color: var(--color-accent); }
.me-filters { display: flex; flex-direction: column; gap: 5px; margin-bottom: 8px; }
.me-filter-select {
  width: 100%; background: var(--color-cell); border: 1px solid var(--color-border-soft); border-radius: 4px;
  color: var(--color-text-dim); font-size: 11px; padding: 4px 6px;
}
.me-filter-select:focus { outline: none; border-color: var(--color-accent); }
.me-filter-checkbox { font-size: 11px !important; color: var(--color-text-dim); }
.me-items { list-style: none; flex: 1; overflow-y: auto; }
.me-items li { display: flex; justify-content: space-between; padding: 6px 9px; border-radius: 4px; cursor: pointer; font-size: 12px; color: var(--color-text-dim); }
.me-items li:hover { background: var(--color-cell); }
.me-items li.active { background: var(--color-primary); color: var(--color-text); }
.me-items li.unmatched { opacity: 0.55; }
.me-pool-count { color: var(--color-text-muted); font-size: 10px; }
.me-list-empty { color: var(--color-text-muted); font-size: 12px; text-align: center; padding: 20px; list-style: none; }
.me-add-pool { margin-top: 8px; }
.me-create-row { margin-top: 8px; display: flex; gap: 4px; align-items: center; }
.btn-small-add { background: none; border: 1px dashed var(--color-border-soft); color: var(--color-text-muted); border-radius: 3px; padding: 3px 10px; font-size: 11px; cursor: pointer; white-space: nowrap; }
.btn-small-add:hover { border-color: #3aba80; color: #3aba80; }
.btn-cancel-inline { background: none; border: none; color: var(--color-text-muted); cursor: pointer; font-size: 12px; padding: 2px 4px; }
.btn-cancel-inline:hover { color: var(--color-accent); }

/* Faction list (по фракциям) */
.me-faction-list { width: 200px; flex-shrink: 0; background: var(--color-cell); padding: 10px; overflow-y: auto; border-right: 1px solid var(--color-border); }
.me-faction-list ul { list-style: none; }
.me-faction-list li { padding: 7px 10px; border-radius: 4px; cursor: pointer; font-size: 12px; }
.me-faction-list li:hover { background: var(--color-primary); }
.me-faction-list li.active { background: var(--color-primary); color: var(--color-text); }

/* Detail */
.me-detail { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.me-empty { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--color-text-muted); font-size: 13px; }

.me-detail-header { display: flex; align-items: center; justify-content: space-between; padding: 10px 16px; border-bottom: 1px solid var(--color-border); flex-shrink: 0; gap: 10px; }
.me-detail-title { display: flex; align-items: center; gap: 8px; min-width: 0; }
.me-pool-title { font-size: 14px; font-weight: 600; color: var(--color-text); cursor: pointer; }
.me-pool-title:hover { color: var(--color-accent); }
.me-rename-input { background: var(--color-input-bg); border: 1px solid var(--color-accent); border-radius: 4px; color: var(--color-text); font-size: 13px; padding: 4px 7px; }
.me-badge { font-size: 10px; color: var(--color-accent); border: 1px solid var(--color-accent); border-radius: 10px; padding: 1px 8px; }
.me-badge-added { color: #3aba80; border-color: #3aba80; }
.me-badge-warn { color: var(--color-error); border-color: var(--color-error); }
.me-detail-actions { display: flex; align-items: center; gap: 6px; flex-shrink: 0; }
.btn-save { background: var(--color-primary); color: var(--color-text); border: none; border-radius: 4px; padding: 5px 14px; font-size: 12px; cursor: pointer; }
.btn-save:hover { background: var(--color-primary-hover); }
.btn-save:disabled { opacity: 0.5; cursor: default; }
.btn-cancel { background: transparent; border: 1px solid var(--color-border-soft); color: var(--color-text-dim); border-radius: 4px; padding: 5px 10px; font-size: 12px; cursor: pointer; }
.btn-cancel:hover { border-color: var(--color-text-dim); }
.btn-revert { background: transparent; color: var(--color-text-muted); border: 1px solid var(--color-border-soft); padding: 4px 9px; border-radius: 4px; cursor: pointer; font-size: 13px; line-height: 1; }
.btn-revert:hover { border-color: var(--color-accent); color: var(--color-accent); }
.btn-delete-pool { background: transparent; border: 1px solid var(--color-error); color: var(--color-error); border-radius: 4px; padding: 5px 10px; font-size: 11px; cursor: pointer; }
.btn-delete-pool:hover { background: var(--color-error); color: var(--color-text); }

.me-scroll { flex: 1; overflow-y: auto; padding: 12px 16px; display: flex; flex-direction: column; gap: 14px; }

/* Section */
.me-section { background: var(--color-secondary); border: 1px solid var(--color-border); border-radius: 6px; padding: 12px 14px; }
.me-section h3 { font-size: 10px; text-transform: uppercase; color: var(--color-heading); letter-spacing: 0.08em; margin-bottom: 10px; }
.section-header-row { display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px; }
.section-header-row h3 { margin-bottom: 0; }
.me-muted { font-size: 11px; color: var(--color-text-muted); }

/* Regions */
.ownership-chips { display: flex; flex-wrap: wrap; gap: 5px; margin-bottom: 8px; }
.chip { display: flex; align-items: center; gap: 4px; background: var(--color-primary); border: 1px solid var(--color-primary-hover); border-radius: 12px; padding: 3px 8px 3px 10px; font-size: 11px; color: var(--color-text-dim); }
.chip--readonly { padding: 3px 10px; }
.chip-remove { background: none; border: none; color: var(--color-text-muted); cursor: pointer; font-size: 10px; line-height: 1; padding: 0; }
.chip-remove:hover { color: var(--color-accent); }
.me-region-add { display: flex; gap: 6px; margin-bottom: 10px; }
.me-region-input { flex: 1; background: var(--color-cell); border: 1px solid var(--color-border-soft); border-radius: 4px; color: var(--color-text-dim); font-size: 12px; padding: 5px 8px; }
.me-region-input:focus { outline: none; border-color: var(--color-accent); }

/* Unit cards (используются и на "По регионам", и в панели редактирования на "По юнитам") */
.merc-card { background: var(--color-input-bg); border: 1px solid var(--color-border); border-radius: 5px; padding: 10px; margin-bottom: 8px; }
.merc-card-header { display: flex; align-items: center; gap: 6px; margin-bottom: 8px; }
.merc-name-input { flex: 1; background: var(--color-cell); border: 1px solid var(--color-border-soft); border-radius: 3px; color: var(--color-text); font-size: 12px; padding: 5px 7px; }
.merc-name-input:focus { outline: none; border-color: var(--color-accent); }
.pool-remove { background: var(--color-primary); border: 1px solid var(--color-primary-hover); color: var(--color-text); border-radius: 3px; padding: 2px 6px; font-size: 10px; cursor: pointer; }
.pool-remove:hover { background: var(--color-primary-hover); }
.merc-grid { display: grid; grid-template-columns: repeat(6, 1fr); gap: 6px; margin-bottom: 8px; }
.merc-grid label, .merc-optional-row label { display: flex; flex-direction: column; gap: 3px; font-size: 10px; color: var(--color-text-muted); }
.merc-grid input, .merc-inline-input { background: var(--color-cell); border: 1px solid var(--color-border); border-radius: 3px; color: var(--color-text-dim); font-size: 12px; padding: 4px 6px; width: 100%; box-sizing: border-box; }
.merc-grid input:focus, .merc-inline-input:focus { outline: none; border-color: var(--color-accent); }
.merc-optional-row { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; margin-bottom: 6px; }
.merc-inline-input { width: 70px !important; }
.checkbox-row { flex-direction: row !important; align-items: center; gap: 5px !important; cursor: pointer; font-size: 11px !important; }
.checkbox-row input[type="checkbox"] { width: auto; accent-color: var(--color-accent); cursor: pointer; }
.merc-religions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; margin-bottom: 6px; font-size: 11px; color: var(--color-text-dim); }
.merc-religions-label { color: var(--color-text-muted); font-size: 10px; }
.religion-check { display: flex; align-items: center; gap: 4px; cursor: pointer; }
.religion-check input[type="checkbox"] { accent-color: var(--color-accent); cursor: pointer; }
.merc-events-label { display: flex; flex-direction: column; gap: 3px; font-size: 10px; color: var(--color-text-muted); }
.merc-events-input { background: var(--color-cell); border: 1px solid var(--color-border); border-radius: 3px; color: var(--color-text-dim); font-size: 11px; padding: 5px 7px; font-family: monospace; }
.merc-events-input:focus { outline: none; border-color: var(--color-accent); }

/* По юнитам: карточки вхождений в пулы */
.occ-grid { display: flex; flex-wrap: wrap; gap: 10px; margin-bottom: 4px; }
.occ-card { position: relative; width: 180px; background: var(--color-cell); border: 1px solid var(--color-border); border-radius: 6px; padding: 10px 12px; cursor: pointer; }
.occ-card:hover { border-color: var(--color-border-soft); }
.occ-card.selected { border-color: var(--color-accent); background: var(--color-primary); }
.occ-pool-name { font-size: 12px; font-weight: 600; color: var(--color-text); margin-bottom: 4px; }
.occ-regions { font-size: 10px; color: var(--color-text-muted); margin-bottom: 6px; }
.occ-remove { position: absolute; top: 8px; right: 8px; }
.occ-regions-hint { font-size: 10px; color: var(--color-text-muted); font-style: italic; margin-bottom: 6px; }
.occ-edit-card { margin-top: 10px; }
.bb-row { display: flex; justify-content: space-between; font-size: 11px; }
.bb-key { color: var(--color-text-muted); }
.bb-val { color: var(--color-text-dim); }

/* Faction tab cards */
.merc-faction-grid { display: flex; flex-wrap: wrap; gap: 10px; }
.merc-faction-card { width: 220px; background: var(--color-cell); border: 1px solid var(--color-border); border-radius: 6px; padding: 10px 12px; cursor: pointer; }
.merc-faction-card:hover { border-color: var(--color-border-soft); }
.merc-faction-card.unmatched { opacity: 0.55; }
.mfc-name { font-size: 13px; font-weight: 600; color: var(--color-text); margin-bottom: 4px; }
.mfc-pools { font-size: 10px; color: var(--color-text-muted); margin-bottom: 6px; }
.mfc-unmatched-hint { font-size: 10px; color: var(--color-error); }
.mfc-ownership { font-size: 10px; color: var(--color-text-dim); margin-bottom: 8px; min-height: 26px; }
.mfc-add-btn { width: 100%; background: var(--color-primary); border: 1px solid var(--color-primary-hover); color: var(--color-text); border-radius: 4px; padding: 5px 0; font-size: 11px; cursor: pointer; }
.mfc-add-btn:hover { background: var(--color-primary-hover); }
.mfc-add-btn:disabled { opacity: 0.6; cursor: default; }
.mfc-owned-tag { display: block; text-align: center; font-size: 10px; color: #3aba80; border: 1px solid #3aba80; border-radius: 4px; padding: 4px 0; }
</style>
