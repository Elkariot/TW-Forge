<script setup>
import { ref, computed, onMounted } from 'vue'
import {
  GetM2TWBuildings, GetM2TWFactions,
  UpdateM2TWBuildingLevel, UpdateM2TWBuildingLevelProps,
  RevertM2TWBuildings,
} from '../../wailsjs/go/main/App'

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

const SETTLEMENT_OPTIONS = [
  { value: '', label: '— нет ограничения —' },
  { value: 'village', label: 'Деревня (village)' },
  { value: 'town', label: 'Город (town)' },
  { value: 'large_town', label: 'Большой город (large_town)' },
  { value: 'city', label: 'Мегаполис (city)' },
  { value: 'large_city', label: 'Большой мегаполис (large_city)' },
  { value: 'huge_city', label: 'Огромный мегаполис (huge_city)' },
]

const SETTLEMENT_TYPE_OPTIONS = [
  { value: '', label: '— оба типа —' },
  { value: 'city', label: 'City' },
  { value: 'castle', label: 'Castle' },
]

const M2TW_BONUS_TEMPLATES = [
  { cat: 'Население',  items: ['happiness_bonus bonus 1', 'law_bonus bonus 1', 'population_health_bonus bonus 1', 'population_growth_bonus bonus 1'] },
  { cat: 'Экономика',  items: ['trade_base_income_bonus bonus 1', 'farming_level bonus 1', 'farming_level 1', 'trade_fleet 1', 'mine_resource iron'] },
  { cat: 'Войска',     items: ['recruits_exp_bonus bonus 1', 'recruits_morale_bonus bonus 1', 'armour bonus 1', 'weapon_simple bonus 1', 'weapon_bladed bonus 1', 'weapon_missile bonus 1', 'upgrade_bodyguard 1', 'siege_engineer', 'shipwright'] },
  { cat: 'Укрепления', items: ['wall_level 1', 'gate_strength 1', 'gate_defences 1', 'tower_level 1'] },
  { cat: 'Дороги',     items: ['road_level 1', 'paved_roads', 'highways'] },
  { cat: 'Агенты',     items: ['agent spy 0 requires factions { }', 'agent diplomat 0 requires factions { }', 'agent assassin 0 requires factions { }'] },
]

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
  if (!level.SettlementType) return true // no restriction → show in both
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
  if (localDirty.value && !confirm('Есть несохранённые изменения. Продолжить?')) return
  selectedGroup.value = grp
  selectedLevel.value = lvl
  edited.value = JSON.parse(JSON.stringify(lvl))
  if (!edited.value.RecruitPools) edited.value.RecruitPools = []
  if (!edited.value.BonusLines) edited.value.BonusLines = []
  if (!edited.value.Upgrades) edited.value.Upgrades = []
  if (!edited.value.RequiredCultures) edited.value.RequiredCultures = []
  localDirty.value = false
  error.value = null
}

function markDirty() { localDirty.value = true }

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

// ── Bonus lines ──
function addBonusLine(template) {
  edited.value.BonusLines = [...(edited.value.BonusLines ?? []), template.trim()]
  markDirty()
}
function removeBonusLine(idx) {
  edited.value.BonusLines = edited.value.BonusLines.filter((_, i) => i !== idx)
  markDirty()
}
function setBonusLine(idx, val) {
  const arr = [...edited.value.BonusLines]
  arr[idx] = val
  edited.value.BonusLines = arr
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
      edited.value.BonusLines ?? [],
    )
    // Update local model
    const grp = buildings.value.find(g => g.Name === selectedGroup.value.Name)
    if (grp) {
      const idx = grp.Levels.findIndex(l => l.Name === selectedLevel.value.Name)
      if (idx !== -1) grp.Levels[idx] = JSON.parse(JSON.stringify(edited.value))
    }
    selectedLevel.value = JSON.parse(JSON.stringify(edited.value))
    localDirty.value = false
    emit('changed')
  } catch (e) {
    error.value = String(e)
  } finally {
    saving.value = false
  }
}

async function revertAll() {
  if (!confirm('Откатить все изменения зданий?')) return
  try {
    await RevertM2TWBuildings()
    ;[buildings.value, factions.value] = await Promise.all([GetM2TWBuildings(), GetM2TWFactions()])
    selectedGroup.value = null
    selectedLevel.value = null
    edited.value = null
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
        <input v-model="search" class="bld-search" placeholder="Поиск..." />
        <div class="bld-filters">
          <!-- Фракция -->
          <select class="bld-filter-select" @change="e => { selectedFaction = factions.find(f => f.Name === e.target.value) || null }">
            <option value="">Все фракции</option>
            <option v-for="f in factions" :key="f.Name" :value="f.Name">{{ f.DisplayName || f.Name }}</option>
          </select>
          <!-- City / Castle -->
          <div class="settlement-tabs">
            <button
              v-for="opt in [{ value: 'all', label: 'Все' }, { value: 'city', label: 'City' }, { value: 'castle', label: 'Castle' }]"
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
        <div v-if="filteredGroups.length === 0" class="bld-empty">Нет результатов</div>
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
          <button v-if="localDirty" class="btn-save" :disabled="saving" @click="save">{{ saving ? '...' : 'Сохранить' }}</button>
          <button v-if="localDirty" class="btn-cancel" @click="edited = JSON.parse(JSON.stringify(selectedLevel)); localDirty = false">Отмена</button>
          <button class="btn-revert-all" @click="revertAll" title="Откатить все здания">↺</button>
        </div>
      </div>

      <div class="bld-detail-scroll">

        <!-- Свойства уровня -->
        <section class="bld-section">
          <h3>Свойства</h3>
          <div class="prop-grid">
            <label>Тип поселения
              <select v-model="edited.SettlementType" @change="markDirty">
                <option v-for="o in SETTLEMENT_TYPE_OPTIONS" :key="o.value" :value="o.value">{{ o.label }}</option>
              </select>
            </label>
            <label>Мин. поселение
              <select v-model="edited.SettlementMin" @change="markDirty">
                <option v-for="o in SETTLEMENT_OPTIONS" :key="o.value" :value="o.value">{{ o.label }}</option>
              </select>
            </label>
            <label>Строительство<input type="number" v-model.number="edited.Construction" @input="markDirty" /></label>
            <label>Стоимость<input type="number" v-model.number="edited.Cost" @input="markDirty" /></label>
            <label>Convert to<input type="number" v-model.number="edited.ConvertTo" @input="markDirty" /></label>
          </div>

          <!-- Зависимость -->
          <div class="dep-row">
            <span class="dep-label">Требует здание:</span>
            <input class="dep-input" placeholder="группа" :value="edited.Dependency?.Group ?? ''"
              @input="e => { edited.Dependency = { ...(edited.Dependency ?? {}), Group: e.target.value }; markDirty() }" />
            <input class="dep-input dep-input--lvl" placeholder="уровень" :value="edited.Dependency?.Level ?? ''"
              @input="e => { edited.Dependency = { ...(edited.Dependency ?? {}), Level: e.target.value }; markDirty() }" />
          </div>

          <!-- Требуемые фракции -->
          <div class="faction-req-block">
            <div class="faction-req-label">Требуется фракция:</div>
            <div class="ownership-chips">
              <span v-for="f in edited.RequiredCultures" :key="f" class="chip">
                {{ f }}
                <button class="chip-remove" @click="removeRequiredFaction(f)">✕</button>
              </span>
            </div>
            <select class="faction-add-select" @change="e => { addRequiredFaction(e.target.value); e.target.value = '' }">
              <option value="">+ Добавить фракцию</option>
              <option v-for="f in factions.filter(f => !(edited.RequiredCultures ?? []).includes(f.Name))" :key="f.Name" :value="f.Name">
                {{ f.DisplayName || f.Name }}
              </option>
            </select>
          </div>
        </section>

        <!-- Найм (recruit_pool) -->
        <section class="bld-section">
          <div class="section-header-row">
            <h3>Найм (recruit_pool)</h3>
            <button class="btn-small-add" @click="addPool">+ Добавить</button>
          </div>

          <div v-for="(pool, idx) in edited.RecruitPools" :key="idx" class="pool-card">
            <div class="pool-header">
              <span class="pool-idx">{{ idx + 1 }}</span>
              <button class="pool-remove" @click="removePool(idx)">✕</button>
            </div>
            <div class="pool-grid">
              <label class="pool-full">Юнит<input :value="pool.UnitType" @input="e => { pool.UnitType = e.target.value; markDirty() }" /></label>
              <label>Нач. пул<input type="number" :value="pool.InitialPool" @input="e => { pool.InitialPool = +e.target.value; markDirty() }" /></label>
              <label>Пополнение<input type="number" step="0.01" :value="pool.ReplenishRate" @input="e => { pool.ReplenishRate = +e.target.value; markDirty() }" /></label>
              <label>Макс. пул<input type="number" :value="pool.MaxPool" @input="e => { pool.MaxPool = +e.target.value; markDirty() }" /></label>
              <label>Опыт<input type="number" :value="pool.ExpGained" @input="e => { pool.ExpGained = +e.target.value; markDirty() }" /></label>
            </div>
            <!-- Фракции пула -->
            <div class="pool-factions">
              <div class="pool-factions-label">Фракции <span class="pool-hint">(пусто = все)</span></div>
              <div class="ownership-chips">
                <span v-for="f in pool.Factions" :key="f" class="chip chip--small">
                  {{ f }}
                  <button class="chip-remove" @click="removePoolFaction(idx, f)">✕</button>
                </span>
              </div>
              <select class="faction-add-select" @change="e => { addPoolFaction(idx, e.target.value); e.target.value = '' }">
                <option value="">+ Фракция</option>
                <option v-for="f in factions.filter(f => !(pool.Factions ?? []).includes(f.Name))" :key="f.Name" :value="f.Name">
                  {{ f.DisplayName || f.Name }}
                </option>
              </select>
            </div>
            <!-- Условия -->
            <label class="pool-cond-label">Условия (raw)<input class="pool-cond" :value="pool.Conditions" @input="e => { pool.Conditions = e.target.value; markDirty() }" /></label>
          </div>
          <div v-if="!edited.RecruitPools?.length" class="bld-empty">Нет записей найма</div>
        </section>

        <!-- Бонусы (capability) -->
        <section class="bld-section">
          <div class="section-header-row">
            <h3>Бонусы (capability)</h3>
          </div>
          <div v-for="(line, idx) in edited.BonusLines" :key="idx" class="bonus-row">
            <input class="bonus-input" :value="line" @input="e => setBonusLine(idx, e.target.value)" />
            <button class="bonus-remove" @click="removeBonusLine(idx)">✕</button>
          </div>
          <div class="bonus-templates">
            <div v-for="cat in M2TW_BONUS_TEMPLATES" :key="cat.cat" class="bonus-cat">
              <div class="bonus-cat-label">{{ cat.cat }}</div>
              <div class="bonus-cat-items">
                <button v-for="t in cat.items" :key="t" class="bonus-tpl" @click="addBonusLine(t)">{{ t.trim() }}</button>
              </div>
            </div>
          </div>
        </section>

        <!-- Улучшения (upgrades) -->
        <section class="bld-section">
          <h3>Улучшения (upgrades)</h3>
          <div class="ownership-chips">
            <span v-for="u in edited.Upgrades" :key="u" class="chip">
              {{ u }}
              <button class="chip-remove" @click="removeUpgrade(u)">✕</button>
            </span>
          </div>
          <select class="faction-add-select" @change="e => { addUpgrade(e.target.value); e.target.value = '' }">
            <option value="">+ Добавить уровень</option>
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
.bonus-row { display: flex; gap: 4px; margin-bottom: 4px; }
.bonus-input { flex: 1; background: var(--color-cell); border: 1px solid var(--color-border); border-radius: 3px; color: var(--color-text-dim); font-size: 12px; padding: 4px 7px; font-family: monospace; }
.bonus-input:focus { outline: none; border-color: var(--color-accent); }
.bonus-remove { background: var(--color-primary); border: 1px solid var(--color-primary-hover); color: var(--color-text); border-radius: 3px; padding: 2px 6px; font-size: 10px; cursor: pointer; }
.bonus-remove:hover { background: var(--color-primary-hover); }
.bonus-templates { margin-top: 10px; border-top: 1px solid var(--color-border); padding-top: 8px; }
.bonus-cat { margin-bottom: 8px; }
.bonus-cat-label { font-size: 9px; text-transform: uppercase; color: var(--color-text-muted); letter-spacing: 0.05em; margin-bottom: 4px; }
.bonus-cat-items { display: flex; flex-wrap: wrap; gap: 4px; }
.bonus-tpl {
  background: var(--color-input-bg); border: 1px solid var(--color-border); border-radius: 3px;
  color: var(--color-text-muted); font-size: 10px; padding: 2px 7px; cursor: pointer; font-family: monospace;
}
.bonus-tpl:hover { border-color: var(--color-border-soft); color: var(--color-text-dim); }
</style>
