<script setup>
import { ref, computed, onMounted } from 'vue'
import {
  GetM2TWBuildings, GetM2TWFactions,
  UpdateM2TWBuildingLevel, UpdateM2TWBuildingLevelProps,
  RevertM2TWBuildings,
} from '../../wailsjs/go/main/App'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const emit = defineEmits(['changed'])

const buildings = ref([])
const factions = ref([])
const selectedFaction = ref(null)
const settlementFilter = ref('all') // 'all' | 'city' | 'castle'
const selectedGroup = ref(null)
const selectedLevel = ref(null)
const edited = ref(null)
const localDirty = ref(false)
const saving = ref(false)
const error = ref(null)
const search = ref('')

const bonusItems = ref([])
const newBonus = ref('')
const showTemplates = ref(false)

const SETTLEMENT_OPTIONS = computed(() => [
  { value: '', label: t('building.settlement_none') },
  { value: 'village', label: t('building.settlement_village') },
  { value: 'town', label: t('building.settlement_town') },
  { value: 'large_town', label: t('building.settlement_large_town') },
  { value: 'city', label: t('building.settlement_city') },
  { value: 'large_city', label: t('building.settlement_large_city') },
  { value: 'huge_city', label: t('building.settlement_huge_city') },
])

const SETTLEMENT_TYPE_OPTIONS = computed(() => [
  { value: '', label: t('building.field_settlement_type_both') },
  { value: 'city', label: 'City' },
  { value: 'castle', label: 'Castle' },
])

const M2TW_BONUS_TEMPLATES = computed(() => [
  { cat: 'Population',    items: ['happiness_bonus bonus ', 'law_bonus bonus ', 'population_health_bonus bonus ', 'population_growth_bonus bonus '] },
  { cat: 'Economy',       items: ['trade_base_income_bonus bonus ', 'farming_level bonus ', 'farming_level ', 'trade_fleet ', 'mine_resource '] },
  { cat: 'Military',      items: ['recruits_exp_bonus bonus ', 'recruits_morale_bonus bonus ', 'recruitment_slots ', 'armour bonus ', 'weapon_simple bonus ', 'weapon_bladed bonus ', 'weapon_missile bonus ', 'upgrade_bodyguard ', 'siege_engineer', 'shipwright'] },
  { cat: 'Fortifications',items: ['wall_level ', 'gate_strength ', 'gate_defences ', 'tower_level '] },
  { cat: 'Roads',         items: ['road_level ', 'paved_roads', 'highways'] },
  { cat: 'Agents',        items: ['agent spy 0 requires factions { }', 'agent diplomat 0 requires factions { }', 'agent assassin 0 requires factions { }'] },
])

onMounted(async () => {
  ;[buildings.value, factions.value] = await Promise.all([GetM2TWBuildings(), GetM2TWFactions()])
})

function grpLabel(g) {
  const n = g.DisplayName || g.Name
  return n.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}
function lvlLabel(l) {
  const n = l.DisplayName || l.Name
  return n.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}

function levelMatchesFaction(level) {
  if (!selectedFaction.value) return true
  const rc = level.RequiredCultures
  if (!rc || rc.length === 0) return true
  return rc.includes(selectedFaction.value.Name)
}

function levelMatchesSettlement(level) {
  if (settlementFilter.value === 'all') return true
  if (!level.SettlementType) return true
  return level.SettlementType === settlementFilter.value
}

const filteredGroups = computed(() => {
  const q = search.value.trim().toLowerCase()
  return buildings.value
    .map(g => ({
      ...g,
      Levels: (g.Levels ?? []).filter(l => {
        if (!levelMatchesFaction(l)) return false
        if (!levelMatchesSettlement(l)) return false
        if (!q) return true
        return grpLabel(g).toLowerCase().includes(q) || lvlLabel(l).toLowerCase().includes(q)
      }),
    }))
    .filter(g => g.Levels.length > 0)
})

const allLevelNames = computed(() => {
  const names = []
  for (const g of buildings.value)
    for (const l of g.Levels) names.push(l.Name)
  return names
})

function selectLevel(grp, lvl) {
  if (localDirty.value && !confirm(t('building.unsaved_confirm'))) return
  selectedGroup.value = grp
  selectedLevel.value = lvl
  edited.value = JSON.parse(JSON.stringify(lvl))
  if (!edited.value.RecruitPools) edited.value.RecruitPools = []
  if (!edited.value.BonusLines) edited.value.BonusLines = []
  if (!edited.value.Upgrades) edited.value.Upgrades = []
  if (!edited.value.RequiredCultures) edited.value.RequiredCultures = []
  bonusItems.value = bonusesFromLevel(lvl)
  newBonus.value = ''
  showTemplates.value = false
  localDirty.value = false
  error.value = null
}

function markDirty() { localDirty.value = true }

// ── Bonus lines ──
function bonusesFromLevel(level) {
  return (level.BonusLines || []).map(line => ({ text: line, original: line }))
}
function isNewBonus(item) { return item.original === null }
function isModifiedBonus(item) { return item.original !== null && item.text !== item.original }
function revertBonus(item) { item.text = item.original; markDirty() }
function removeBonus(i) { bonusItems.value.splice(i, 1); markDirty() }
function addBonus() {
  const v = newBonus.value.trim()
  if (!v) return
  bonusItems.value.push({ text: v, original: null })
  newBonus.value = ''
  markDirty()
}
function useTemplate(tpl) {
  newBonus.value = tpl
  showTemplates.value = false
  if (!tpl.endsWith(' ')) addBonus()
}

// ── Pool editing ──
function addPool() {
  edited.value.RecruitPools = [
    ...(edited.value.RecruitPools ?? []),
    { UnitType: '', InitialPool: 1, ReplenishRate: 0.15, MaxPool: 3, ExpGained: 0, Factions: [], Conditions: '' },
  ]
  markDirty()
}
function removePool(idx) {
  edited.value.RecruitPools = edited.value.RecruitPools.filter((_, i) => i !== idx)
  markDirty()
}
function addPoolFaction(poolIdx, name) {
  if (!name) return
  const pool = edited.value.RecruitPools[poolIdx]
  if (!(pool.Factions ?? []).includes(name)) {
    pool.Factions = [...(pool.Factions ?? []), name]
    markDirty()
  }
}
function removePoolFaction(poolIdx, name) {
  edited.value.RecruitPools[poolIdx].Factions = edited.value.RecruitPools[poolIdx].Factions.filter(f => f !== name)
  markDirty()
}

// ── Upgrades ──
function addUpgrade(name) {
  if (!name) return
  if (!(edited.value.Upgrades ?? []).includes(name)) {
    edited.value.Upgrades = [...(edited.value.Upgrades ?? []), name]
    markDirty()
  }
}
function removeUpgrade(n) {
  edited.value.Upgrades = edited.value.Upgrades.filter(u => u !== n)
  markDirty()
}

// ── Required factions ──
function addRequiredFaction(name) {
  if (!name) return
  if (!(edited.value.RequiredCultures ?? []).includes(name)) {
    edited.value.RequiredCultures = [...(edited.value.RequiredCultures ?? []), name]
    markDirty()
  }
}
function removeRequiredFaction(name) {
  edited.value.RequiredCultures = edited.value.RequiredCultures.filter(f => f !== name)
  markDirty()
}

// ── Save / Revert ──
async function save() {
  if (!selectedGroup.value || !selectedLevel.value) return
  saving.value = true
  error.value = null
  try {
    const bonusLines = bonusItems.value.map(b => b.text)
    await UpdateM2TWBuildingLevelProps(
      selectedGroup.value.Name,
      selectedLevel.value.Name,
      edited.value.Cost ?? 0,
      edited.value.Construction ?? 0,
      edited.value.ConvertTo ?? 0,
      edited.value.SettlementMin ?? '',
      edited.value.SettlementType ?? '',
      edited.value.RequiredCultures ?? [],
      edited.value.Dependency?.Group ?? '',
      edited.value.Dependency?.Level ?? '',
      edited.value.Upgrades ?? [],
    )
    await UpdateM2TWBuildingLevel(
      selectedGroup.value.Name,
      selectedLevel.value.Name,
      edited.value.RecruitPools ?? [],
      bonusLines,
    )
    // Update local model
    const grp = buildings.value.find(g => g.Name === selectedGroup.value.Name)
    if (grp) {
      const idx = grp.Levels.findIndex(l => l.Name === selectedLevel.value.Name)
      if (idx !== -1) grp.Levels[idx] = JSON.parse(JSON.stringify(edited.value))
    }
    selectedLevel.value = JSON.parse(JSON.stringify(edited.value))
    bonusItems.value.forEach(b => { b.original = b.text })
    localDirty.value = false
    emit('changed')
  } catch (e) {
    error.value = String(e)
  } finally {
    saving.value = false
  }
}

async function revertAll() {
  if (!confirm(t('building.revert_all_confirm'))) return
  try {
    await RevertM2TWBuildings()
    ;[buildings.value, factions.value] = await Promise.all([GetM2TWBuildings(), GetM2TWFactions()])
    selectedGroup.value = null
    selectedLevel.value = null
    edited.value = null
    bonusItems.value = []
    newBonus.value = ''
    localDirty.value = false
    emit('changed')
  } catch (e) {
    error.value = String(e)
  }
}
</script>

<template>
  <div class="bld-layout">

    <!-- Левая панель: список зданий -->
    <div class="bld-sidebar">
      <div class="bld-sidebar-top">
        <input v-model="search" class="bld-search" :placeholder="$t('building.search_placeholder')" />
        <div class="bld-filters">
          <!-- Фракция -->
          <select class="bld-filter-select" @change="e => { selectedFaction = factions.find(f => f.Name === e.target.value) || null }">
            <option value="">{{ $t('building.all_factions') }}</option>
            <option v-for="f in factions" :key="f.Name" :value="f.Name">{{ f.DisplayName || f.Name }}</option>
          </select>
          <!-- City / Castle -->
          <div class="settlement-tabs">
            <button
              v-for="opt in [{ value: 'all', label: $t('building.filter_all') }, { value: 'city', label: 'City' }, { value: 'castle', label: 'Castle' }]"
              :key="opt.value"
              class="settlement-tab"
              :class="{ active: settlementFilter === opt.value }"
              @click="settlementFilter = opt.value"
            >{{ opt.label }}</button>
          </div>
        </div>
      </div>

      <div class="bld-list">
        <div v-for="grp in filteredGroups" :key="grp.Name" class="bld-group">
          <div class="bld-group-name">{{ grpLabel(grp) }}</div>
          <button
            v-for="lvl in grp.Levels"
            :key="lvl.Name"
            class="bld-lvl-btn"
            :class="{
              active: selectedLevel?.Name === lvl.Name && selectedGroup?.Name === grp.Name,
              'type-city': lvl.SettlementType === 'city',
              'type-castle': lvl.SettlementType === 'castle',
            }"
            @click="selectLevel(grp, lvl)"
          >
            <span class="bld-lvl-name">{{ lvlLabel(lvl) }}</span>
            <span v-if="lvl.SettlementType" class="bld-lvl-type">{{ lvl.SettlementType }}</span>
          </button>
        </div>
        <div v-if="filteredGroups.length === 0" class="bld-empty">{{ $t('building.no_entries') }}</div>
      </div>
    </div>

    <!-- Правая панель: редактор уровня -->
    <div class="bld-detail" v-if="edited">
      <div class="bld-detail-header">
        <div class="bld-detail-title">
          <span class="bld-detail-grp">{{ grpLabel(selectedGroup) }}</span>
          <span class="bld-detail-sep"> › </span>
          <span class="bld-detail-lvl">{{ lvlLabel(selectedLevel) }}</span>
        </div>
        <div class="bld-detail-actions">
          <span v-if="error" class="bld-error">{{ error }}</span>
          <button v-if="localDirty" class="btn-save" :disabled="saving" @click="save">{{ saving ? '...' : $t('building.save') }}</button>
          <button v-if="localDirty" class="btn-cancel" @click="edited = JSON.parse(JSON.stringify(selectedLevel)); bonusItems = bonusesFromLevel(selectedLevel); localDirty = false">{{ $t('common.cancel') }}</button>
          <button class="btn-revert-all" @click="revertAll" :title="$t('building.revert_changes')">↺</button>
        </div>
      </div>

      <div class="bld-detail-scroll">

        <!-- Свойства уровня -->
        <section class="bld-section">
          <h3>{{ $t('building.section_properties') }}</h3>
          <div class="prop-grid">
            <label>{{ $t('building.field_settlement_type') }}
              <select v-model="edited.SettlementType" @change="markDirty">
                <option v-for="o in SETTLEMENT_TYPE_OPTIONS" :key="o.value" :value="o.value">{{ o.label }}</option>
              </select>
            </label>
            <label>{{ $t('building.field_settlement_min') }}
              <select v-model="edited.SettlementMin" @change="markDirty">
                <option v-for="o in SETTLEMENT_OPTIONS" :key="o.value" :value="o.value">{{ o.label }}</option>
              </select>
            </label>
            <label>{{ $t('building.field_construction') }}<input type="number" v-model.number="edited.Construction" @input="markDirty" /></label>
            <label>{{ $t('building.field_cost') }}<input type="number" v-model.number="edited.Cost" @input="markDirty" /></label>
            <label>{{ $t('building.field_convert_to') }}<input type="number" v-model.number="edited.ConvertTo" @input="markDirty" /></label>
          </div>

          <!-- Зависимость -->
          <div class="dep-row">
            <span class="dep-label">{{ $t('building.requires_building') }}</span>
            <input class="dep-input" :placeholder="$t('building.level_group_placeholder')" :value="edited.Dependency?.Group ?? ''"
              @input="e => { edited.Dependency = { ...(edited.Dependency ?? {}), Group: e.target.value }; markDirty() }" />
            <input class="dep-input dep-input--lvl" :placeholder="$t('building.level_lvl_placeholder')" :value="edited.Dependency?.Level ?? ''"
              @input="e => { edited.Dependency = { ...(edited.Dependency ?? {}), Level: e.target.value }; markDirty() }" />
          </div>

          <!-- Требуемые фракции -->
          <div class="faction-req-block">
            <div class="faction-req-label">{{ $t('building.requires_faction') }}</div>
            <div class="ownership-chips">
              <span v-for="f in edited.RequiredCultures" :key="f" class="chip">
                {{ f }}
                <button class="chip-remove" @click="removeRequiredFaction(f)">✕</button>
              </span>
            </div>
            <select class="faction-add-select" @change="e => { addRequiredFaction(e.target.value); e.target.value = '' }">
              <option value="">{{ $t('common.add_faction') }}</option>
              <option v-for="f in factions.filter(f => !(edited.RequiredCultures ?? []).includes(f.Name))" :key="f.Name" :value="f.Name">
                {{ f.DisplayName || f.Name }}
              </option>
            </select>
          </div>
        </section>

        <!-- Найм (recruit_pool) -->
        <section class="bld-section">
          <div class="section-header-row">
            <h3>{{ $t('building.section_recruit') }}</h3>
            <button class="btn-small-add" @click="addPool">{{ $t('building.add_pool') }}</button>
          </div>

          <div v-for="(pool, idx) in edited.RecruitPools" :key="idx" class="pool-card">
            <div class="pool-header">
              <span class="pool-idx">{{ idx + 1 }}</span>
              <button class="pool-remove" @click="removePool(idx)">✕</button>
            </div>
            <div class="pool-grid">
              <label class="pool-full">{{ $t('building.pool_unit') }}<input :value="pool.UnitType" @input="e => { pool.UnitType = e.target.value; markDirty() }" /></label>
              <label>{{ $t('building.pool_initial') }}<input type="number" :value="pool.InitialPool" @input="e => { pool.InitialPool = +e.target.value; markDirty() }" /></label>
              <label>{{ $t('building.pool_replenish') }}<input type="number" step="0.01" :value="pool.ReplenishRate" @input="e => { pool.ReplenishRate = +e.target.value; markDirty() }" /></label>
              <label>{{ $t('building.pool_max') }}<input type="number" :value="pool.MaxPool" @input="e => { pool.MaxPool = +e.target.value; markDirty() }" /></label>
              <label>{{ $t('building.pool_exp') }}<input type="number" :value="pool.ExpGained" @input="e => { pool.ExpGained = +e.target.value; markDirty() }" /></label>
            </div>
            <!-- Фракции пула -->
            <div class="pool-factions">
              <div class="pool-factions-label">{{ $t('building.pool_factions') }} <span class="pool-hint">{{ $t('building.pool_factions_hint') }}</span></div>
              <div class="ownership-chips">
                <span v-for="f in pool.Factions" :key="f" class="chip chip--small">
                  {{ f }}
                  <button class="chip-remove" @click="removePoolFaction(idx, f)">✕</button>
                </span>
              </div>
              <select class="faction-add-select" @change="e => { addPoolFaction(idx, e.target.value); e.target.value = '' }">
                <option value="">{{ $t('common.add_faction') }}</option>
                <option v-for="f in factions.filter(f => !(pool.Factions ?? []).includes(f.Name))" :key="f.Name" :value="f.Name">
                  {{ f.DisplayName || f.Name }}
                </option>
              </select>
            </div>
            <!-- Условия -->
            <label class="pool-cond-label">{{ $t('building.pool_conditions') }}<input class="pool-cond" :value="pool.Conditions" @input="e => { pool.Conditions = e.target.value; markDirty() }" /></label>
          </div>
          <div v-if="!edited.RecruitPools?.length" class="bld-empty">{{ $t('building.no_recruit_entries') }}</div>
        </section>

        <!-- Бонусы (capability) -->
        <section class="bld-section">
          <h3>{{ $t('building.section_bonuses') }}</h3>

          <div class="be-bonus-list">
            <div v-if="bonusItems.length === 0" class="bld-empty">{{ $t('building.no_bonuses_entry') }}</div>
            <div v-for="(item, i) in bonusItems" :key="i" class="be-bonus-row">
              <div class="be-bonus-tag" :class="{ 'is-new': isNewBonus(item), 'is-modified': isModifiedBonus(item) }">
                {{ isNewBonus(item) ? 'new' : isModifiedBonus(item) ? 'mod' : '' }}
              </div>
              <input :value="item.text" class="be-bonus-input" @input="e => { item.text = e.target.value; markDirty() }" />
              <button v-if="isModifiedBonus(item)" class="be-bonus-action revert" @click="revertBonus(item)" :title="$t('common.revert')">↺</button>
              <button class="be-bonus-action remove" @click="removeBonus(i)">✕</button>
            </div>
          </div>

          <div class="be-bonus-add-area">
            <div class="be-upg-input-row">
              <input v-model="newBonus" class="be-bonus-new-input" placeholder="happiness_bonus bonus 3" @keydown.enter="addBonus" />
              <button class="be-upg-add-btn" @click="addBonus">+</button>
              <button class="be-tpl-toggle" :class="{ active: showTemplates }" @click="showTemplates = !showTemplates" :title="$t('building.bonus_template_title')">≡</button>
            </div>
            <div v-if="showTemplates" class="be-templates">
              <div v-for="cat in M2TW_BONUS_TEMPLATES" :key="cat.cat" class="be-tpl-cat">
                <div class="be-tpl-cat-name">{{ cat.cat }}</div>
                <div class="be-tpl-items">
                  <button v-for="tpl in cat.items" :key="tpl" class="be-tpl-item" @click="useTemplate(tpl)">{{ tpl.trim() }}</button>
                </div>
              </div>
            </div>
          </div>
        </section>

        <!-- Улучшения (upgrades) -->
        <section class="bld-section">
          <h3>{{ $t('building.section_upgrades_header') }}</h3>
          <div class="ownership-chips">
            <span v-for="u in edited.Upgrades" :key="u" class="chip">
              {{ u }}
              <button class="chip-remove" @click="removeUpgrade(u)">✕</button>
            </span>
          </div>
          <select class="faction-add-select" @change="e => { addUpgrade(e.target.value); e.target.value = '' }">
            <option value="">{{ $t('building.add_level') }}</option>
            <option v-for="n in allLevelNames.filter(n => !(edited.Upgrades ?? []).includes(n))" :key="n" :value="n">{{ n }}</option>
          </select>
        </section>

      </div>
    </div>

    <div v-else class="bld-empty-panel">Выберите уровень здания</div>

  </div>
</template>

<style scoped>
.bld-layout { display: flex; height: 100%; overflow: hidden; }

/* Sidebar */
.bld-sidebar {
  width: 260px; flex-shrink: 0; display: flex; flex-direction: column;
  border-right: 1px solid var(--color-border); background: var(--color-secondary);
}
.bld-sidebar-top { padding: 10px; border-bottom: 1px solid var(--color-border); flex-shrink: 0; display: flex; flex-direction: column; gap: 6px; }
.bld-search {
  width: 100%; background: var(--color-cell); border: 1px solid var(--color-border);
  border-radius: 4px; color: var(--color-text-dim); font-size: 12px; padding: 6px 8px;
}
.bld-search:focus { outline: none; border-color: var(--color-accent); }
.bld-filters { display: flex; flex-direction: column; gap: 5px; }
.bld-filter-select {
  width: 100%; background: var(--color-cell); border: 1px solid var(--color-border);
  border-radius: 4px; color: var(--color-text-dim); font-size: 11px; padding: 4px 6px;
}
.bld-filter-select:focus { outline: none; border-color: var(--color-accent); }
.settlement-tabs { display: flex; gap: 2px; }
.settlement-tab {
  flex: 1; background: var(--color-cell); border: 1px solid var(--color-border); border-radius: 3px;
  color: var(--color-text-muted); font-size: 10px; padding: 3px 0; cursor: pointer; text-align: center;
}
.settlement-tab:hover { color: var(--color-text-dim); border-color: var(--color-border-soft); }
.settlement-tab.active { background: var(--color-primary); border-color: var(--color-primary-hover); color: var(--color-accent); }

.bld-list { flex: 1; overflow-y: auto; padding: 8px; }
.bld-group { margin-bottom: 10px; }
.bld-group-name { font-size: 9px; text-transform: uppercase; color: var(--color-heading); letter-spacing: 0.05em; margin-bottom: 4px; padding: 0 4px; }
.bld-lvl-btn {
  display: flex; align-items: center; justify-content: space-between;
  width: 100%; text-align: left; background: transparent;
  border: 1px solid transparent; border-radius: 3px;
  color: var(--color-text-dim); font-size: 12px; padding: 5px 8px; cursor: pointer; margin-bottom: 2px;
}
.bld-lvl-btn:hover { background: var(--color-cell); border-color: var(--color-border); }
.bld-lvl-btn.active { background: var(--color-primary); border-color: var(--color-primary-hover); color: var(--color-text); }
.bld-lvl-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.bld-lvl-type {
  font-size: 8px; text-transform: uppercase; letter-spacing: 0.05em;
  padding: 1px 5px; border-radius: 8px; margin-left: 4px; flex-shrink: 0;
}
.bld-lvl-btn.type-city .bld-lvl-type { background: #e0f5e8; color: #3aba80; border: 1px solid #6aba8a; }
.bld-lvl-btn.type-castle .bld-lvl-type { background: #f0e0f8; color: #9060ba; border: 1px solid #a06aba; }
.bld-empty { font-size: 11px; color: var(--color-text-muted); padding: 10px 4px; }

/* Detail */
.bld-detail { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.bld-empty-panel { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--color-text-muted); font-size: 14px; }

.bld-detail-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 16px; border-bottom: 1px solid var(--color-border); flex-shrink: 0;
}
.bld-detail-title { font-size: 13px; color: var(--color-text-dim); }
.bld-detail-grp { color: var(--color-accent); }
.bld-detail-sep, .bld-detail-lvl { color: var(--color-text-dim); }
.bld-detail-actions { display: flex; align-items: center; gap: 6px; }
.bld-error { font-size: 11px; color: var(--color-error); }
.btn-save { background: var(--color-primary); color: var(--color-text); border: none; border-radius: 4px; padding: 5px 14px; font-size: 12px; cursor: pointer; }
.btn-save:hover { background: var(--color-primary-hover); }
.btn-save:disabled { opacity: 0.5; cursor: default; }
.btn-cancel { background: transparent; border: 1px solid var(--color-border-soft); color: var(--color-text-dim); border-radius: 4px; padding: 5px 10px; font-size: 12px; cursor: pointer; }
.btn-cancel:hover { border-color: var(--color-text-dim); color: var(--color-text-dim); }
.btn-revert-all { background: none; border: 1px solid var(--color-border-soft); color: var(--color-text-muted); border-radius: 4px; padding: 4px 8px; font-size: 12px; cursor: pointer; }
.btn-revert-all:hover { border-color: var(--color-accent); color: var(--color-accent); }

.bld-detail-scroll { flex: 1; overflow-y: auto; padding: 12px 16px; display: flex; flex-direction: column; gap: 14px; }

/* Section */
.bld-section { background: var(--color-secondary); border: 1px solid var(--color-border); border-radius: 6px; padding: 12px 14px; }
.bld-section h3 { font-size: 10px; text-transform: uppercase; color: var(--color-heading); letter-spacing: 0.08em; margin-bottom: 10px; }
.section-header-row { display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px; }
.section-header-row h3 { margin-bottom: 0; }
.btn-small-add { background: none; border: 1px dashed var(--color-border-soft); color: var(--color-text-muted); border-radius: 3px; padding: 3px 10px; font-size: 11px; cursor: pointer; }
.btn-small-add:hover { border-color: #3aba80; color: #3aba80; }

/* Props */
.prop-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px; margin-bottom: 10px; }
label { display: flex; flex-direction: column; gap: 3px; font-size: 10px; color: var(--color-text-muted); }
label input, label select { background: var(--color-cell); border: 1px solid var(--color-border); border-radius: 3px; color: var(--color-text-dim); font-size: 12px; padding: 5px 7px; }
label input:focus, label select:focus { outline: none; border-color: var(--color-accent); }
label select option { background: var(--color-cell); }

.dep-row { display: flex; align-items: center; gap: 6px; margin-top: 8px; margin-bottom: 8px; }
.dep-label { font-size: 10px; color: var(--color-text-muted); white-space: nowrap; }
.dep-input { background: var(--color-cell); border: 1px solid var(--color-border); border-radius: 3px; color: var(--color-text-dim); font-size: 12px; padding: 4px 7px; flex: 1; }
.dep-input--lvl { max-width: 90px; flex: none; }
.dep-input:focus { outline: none; border-color: var(--color-accent); }

.faction-req-block { margin-top: 8px; }
.faction-req-label { font-size: 10px; color: var(--color-text-muted); margin-bottom: 5px; }

/* Chips */
.ownership-chips { display: flex; flex-wrap: wrap; gap: 5px; margin-bottom: 6px; }
.chip {
  display: flex; align-items: center; gap: 4px;
  background: var(--color-primary); border: 1px solid var(--color-primary-hover);
  border-radius: 12px; padding: 3px 8px 3px 10px; font-size: 11px; color: var(--color-text-dim);
}
.chip--small { font-size: 10px; padding: 2px 6px 2px 8px; }
.chip-remove { background: none; border: none; color: var(--color-text-muted); cursor: pointer; font-size: 10px; line-height: 1; padding: 0; }
.chip-remove:hover { color: var(--color-accent); }
.faction-add-select {
  background: var(--color-cell); border: 1px solid var(--color-border-soft); border-radius: 4px;
  color: var(--color-text-dim); font-size: 11px; padding: 4px 8px;
}
.faction-add-select:focus { outline: none; border-color: var(--color-accent); }

/* Pool */
.pool-card { background: var(--color-input-bg); border: 1px solid var(--color-border); border-radius: 5px; padding: 10px; margin-bottom: 8px; }
.pool-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.pool-idx { font-size: 10px; color: var(--color-accent); font-weight: 600; }
.pool-remove { background: var(--color-primary); border: 1px solid var(--color-primary-hover); color: var(--color-text); border-radius: 3px; padding: 2px 6px; font-size: 10px; cursor: pointer; }
.pool-remove:hover { background: var(--color-primary-hover); }
.pool-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 6px; margin-bottom: 8px; }
.pool-full { grid-column: 1 / -1; }
.pool-factions { margin-bottom: 6px; }
.pool-factions-label { font-size: 10px; color: var(--color-text-muted); margin-bottom: 4px; }
.pool-hint { color: var(--color-text-muted); }
.pool-cond-label { font-size: 10px; color: var(--color-text-muted); margin-top: 6px; }
.pool-cond { width: 100%; background: var(--color-cell); border: 1px solid var(--color-border); border-radius: 3px; color: var(--color-text-dim); font-size: 11px; padding: 4px 7px; font-family: monospace; }
.pool-cond:focus { outline: none; border-color: var(--color-accent); }

/* Bonus lines */
.be-bonus-list { display: flex; flex-direction: column; gap: 4px; margin-bottom: 8px; }
.be-bonus-row { display: flex; align-items: center; gap: 5px; }
.be-bonus-tag {
  width: 28px; flex-shrink: 0; font-size: 9px; text-align: center;
  border-radius: 3px; padding: 1px 3px; color: transparent; background: transparent;
}
.be-bonus-tag.is-new { color: #3aba80; background: rgba(58,186,128,0.2); border: 1px solid rgba(58,186,128,0.3); }
.be-bonus-tag.is-modified { color: #e9a000; background: rgba(233,160,0,0.1); border: 1px solid rgba(233,160,0,0.3); }
.be-bonus-input {
  flex: 1; background: var(--color-input-bg); border: 1px solid var(--color-border-soft);
  border-radius: 4px; color: var(--color-text); font-size: 12px; padding: 5px 8px; font-family: monospace;
}
.be-bonus-input:focus { outline: none; border-color: var(--color-accent); }
.be-bonus-action {
  flex-shrink: 0; background: none; border: none; cursor: pointer;
  font-size: 12px; padding: 2px 4px; border-radius: 3px; line-height: 1;
}
.be-bonus-action.revert { color: #e9a000; }
.be-bonus-action.revert:hover { color: #ffc000; background: rgba(233,160,0,0.1); }
.be-bonus-action.remove { color: var(--color-text-muted); }
.be-bonus-action.remove:hover { color: var(--color-accent); background: rgba(149,232,225,0.2); }

/* Bonus add + templates */
.be-bonus-add-area { display: flex; flex-direction: column; gap: 8px; }
.be-upg-input-row { display: flex; gap: 6px; }
.be-bonus-new-input {
  flex: 1; background: var(--color-input-bg); border: 1px solid var(--color-border-soft);
  border-radius: 4px; color: var(--color-text); font-size: 12px; padding: 5px 8px; font-family: monospace;
}
.be-bonus-new-input:focus { outline: none; border-color: var(--color-accent); }
.be-upg-add-btn {
  background: var(--color-primary); border: 1px solid var(--color-primary-hover);
  border-radius: 4px; color: var(--color-text); font-size: 14px; width: 28px; cursor: pointer;
}
.be-upg-add-btn:hover { background: var(--color-primary-hover); }
.be-tpl-toggle {
  background: var(--color-cell); border: 1px solid var(--color-border-soft);
  border-radius: 4px; color: var(--color-text-dim); font-size: 14px; width: 28px; cursor: pointer;
}
.be-tpl-toggle:hover, .be-tpl-toggle.active { border-color: var(--color-accent); color: var(--color-accent); }
.be-templates {
  background: var(--color-input-bg); border: 1px solid var(--color-border-soft);
  border-radius: 4px; padding: 10px; display: flex; flex-direction: column; gap: 10px;
  max-height: 220px; overflow-y: auto;
}
.be-tpl-cat-name { font-size: 10px; text-transform: uppercase; color: var(--color-heading); letter-spacing: 0.06em; margin-bottom: 5px; }
.be-tpl-items { display: flex; flex-wrap: wrap; gap: 4px; }
.be-tpl-item {
  background: var(--color-cell); border: 1px solid var(--color-border-soft); border-radius: 3px;
  color: var(--color-text-dim); font-size: 11px; font-family: monospace; padding: 3px 7px; cursor: pointer; white-space: nowrap;
}
.be-tpl-item:hover { border-color: var(--color-accent); color: var(--color-text); }
</style>
