<script setup>
import { ref, computed, onMounted } from 'vue'
import { GetBuildings, UpdateBuildingLevelProps, RevertBuildings, GetFactions } from '../../wailsjs/go/main/App'

const emit = defineEmits(['changed'])

const buildings = ref([])
const factions = ref([])
const selectedFaction = ref(null)
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

// Шаблоны бонусов, сгруппированные по категориям
const BONUS_TEMPLATES = [
  { cat: 'Население',  items: ['happiness_bonus bonus ', 'law_bonus bonus ', 'population_health_bonus bonus ', 'population_growth_bonus bonus '] },
  { cat: 'Экономика',  items: ['trade_base_income_bonus bonus ', 'farming_level bonus ', 'farming_level ', 'trade_fleet ', 'mine_resource '] },
  { cat: 'Войска',     items: ['recruits_exp_bonus bonus ', 'recruits_morale_bonus bonus ', 'armour bonus ', 'weapon_simple bonus ', 'weapon_bladed bonus ', 'weapon_missile bonus ', 'upgrade_bodyguard ', 'siege_engineer', 'shipwright'] },
  { cat: 'Укрепления', items: ['wall_level ', 'gate_strength ', 'gate_defences ', 'tower_level '] },
  { cat: 'Дороги',     items: ['road_level ', 'paved_roads', 'highways'] },
  { cat: 'Прочее',     items: ['stage_games ', 'stage_races ', 'agent spy 0 requires factions { }', 'agent diplomat 0 requires factions { }', 'agent assassin 0 requires factions { }'] },
]

onMounted(async () => {
  ;[buildings.value, factions.value] = await Promise.all([GetBuildings(), GetFactions()])
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
  const f = selectedFaction.value
  return rc.includes(f.Culture) || rc.includes(f.Name)
}

const filteredGroups = computed(() => {
  const q = search.value.trim().toLowerCase()
  return buildings.value
    .map(g => ({
      ...g,
      Levels: g.Levels.filter(l => {
        if (!levelMatchesFaction(l)) return false
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

const allCultures = computed(() => {
  const set = new Set()
  for (const f of factions.value) if (f.Culture) set.add(f.Culture)
  return [...set].sort()
})

// ── BonusItems ─────────────────────────────────────────────────────────────
// { text: string, original: string|null }
// original === null → новая строка (можно удалить сразу)
// original !== null → из оригинального файла (можно откатить к original)
const bonusItems = ref([])

function bonusesFromLevel(level) {
  return (level.BonusLines || []).map(line => ({ text: line, original: line }))
}

function isNewBonus(item) { return item.original === null }
function isModifiedBonus(item) { return item.original !== null && item.text !== item.original }

function revertBonus(item) {
  item.text = item.original
  markDirty()
}
function removeBonus(i) {
  bonusItems.value.splice(i, 1)
  markDirty()
}

// ── Dependency ──────────────────────────────────────────────────────────────
const originalDep = ref({ group: '', level: '' })

const depGroupLevels = computed(() => {
  const g = buildings.value.find(b => b.Name === edited.value?.dependencyGroup)
  return g ? g.Levels.map(l => l.Name) : []
})

const depChanged = computed(() =>
  edited.value?.dependencyGroup !== originalDep.value.group ||
  edited.value?.dependencyLevel !== originalDep.value.level
)

function revertDep() {
  edited.value.dependencyGroup = originalDep.value.group
  edited.value.dependencyLevel = originalDep.value.level
  markDirty()
}

// ── Level selection ─────────────────────────────────────────────────────────
function selectLevel(group, level) {
  selectedGroup.value = group
  selectedLevel.value = level
  bonusItems.value = bonusesFromLevel(level)
  const depGroup = level.Dependency?.Group || ''
  const depLevel = level.Dependency?.Level || ''
  originalDep.value = { group: depGroup, level: depLevel }
  edited.value = {
    cost: level.Cost,
    construction: level.Construction,
    settlementMin: level.SettlementMin || '',
    requiredCultures: [...(level.RequiredCultures || [])],
    dependencyGroup: depGroup,
    dependencyLevel: depLevel,
    upgrades: [...(level.Upgrades || [])],
  }
  localDirty.value = false
  error.value = null
}

function markDirty() { localDirty.value = true }

// ── RequiredCultures ────────────────────────────────────────────────────────
function addCulture(c) {
  if (!c || edited.value.requiredCultures.includes(c)) return
  edited.value.requiredCultures.push(c)
  markDirty()
}
function removeCulture(c) {
  edited.value.requiredCultures = edited.value.requiredCultures.filter(x => x !== c)
  markDirty()
}

// ── New bonus input ─────────────────────────────────────────────────────────
const newBonus = ref('')
const showTemplates = ref(false)

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
  // если шаблон не требует значения — добавляем сразу
  if (!tpl.endsWith(' ')) addBonus()
}

// ── Upgrades ────────────────────────────────────────────────────────────────
const newUpgrade = ref('')
const upgradePickerOpen = ref(false)

const availableUpgrades = computed(() =>
  allLevelNames.value.filter(n => !edited.value?.upgrades.includes(n))
)

function addUpgrade(name) {
  if (!name || edited.value.upgrades.includes(name)) return
  edited.value.upgrades.push(name)
  newUpgrade.value = ''
  upgradePickerOpen.value = false
  markDirty()
}
function removeUpgrade(name) {
  edited.value.upgrades = edited.value.upgrades.filter(u => u !== name)
  markDirty()
}
function closeUpgradePicker() {
  setTimeout(() => { upgradePickerOpen.value = false }, 150)
}

// ── Save / Revert ───────────────────────────────────────────────────────────
async function save() {
  if (!selectedGroup.value || !selectedLevel.value || !edited.value) return
  saving.value = true
  error.value = null
  try {
    const bonusLines = bonusItems.value.map(b => b.text)
    await UpdateBuildingLevelProps(
      selectedGroup.value.Name,
      selectedLevel.value.Name,
      edited.value.cost,
      edited.value.construction,
      edited.value.settlementMin,
      edited.value.requiredCultures,
      edited.value.dependencyGroup,
      edited.value.dependencyLevel,
      edited.value.upgrades,
      bonusLines,
    )
    // Обновить local level object
    selectedLevel.value.Cost = edited.value.cost
    selectedLevel.value.Construction = edited.value.construction
    selectedLevel.value.SettlementMin = edited.value.settlementMin
    selectedLevel.value.RequiredCultures = [...edited.value.requiredCultures]
    selectedLevel.value.Dependency = edited.value.dependencyGroup
      ? { Group: edited.value.dependencyGroup, Level: edited.value.dependencyLevel }
      : null
    selectedLevel.value.Upgrades = [...edited.value.upgrades]
    selectedLevel.value.BonusLines = [...bonusLines]
    // Обновить originalDep и bonusItems.original после сохранения
    originalDep.value = { group: edited.value.dependencyGroup, level: edited.value.dependencyLevel }
    bonusItems.value.forEach(b => { b.original = b.text })
    localDirty.value = false
    emit('changed')
  } catch (e) {
    error.value = String(e)
  } finally {
    saving.value = false
  }
}

async function revert() {
  try {
    await RevertBuildings()
    buildings.value = await GetBuildings()
    selectedGroup.value = null
    selectedLevel.value = null
    edited.value = null
    bonusItems.value = []
    localDirty.value = false
    emit('changed')
  } catch (e) {
    error.value = String(e)
  }
}
</script>

<template>
  <div class="be-root">

    <!-- Левая панель: дерево зданий -->
    <div class="be-tree">
      <div class="be-tree-header">
        <input v-model="search" class="be-search" placeholder="Поиск зданий..." />
        <select
          class="be-faction-select"
          :value="selectedFaction?.Name ?? ''"
          @change="e => selectedFaction = factions.find(f => f.Name === e.target.value) ?? null"
        >
          <option value="">Все фракции</option>
          <option v-for="f in factions" :key="f.Name" :value="f.Name">
            {{ f.DisplayName || f.Name }}
          </option>
        </select>
        <button v-if="localDirty" class="be-revert-btn" @click="revert">
          Отменить изменения
        </button>
      </div>

      <div class="be-groups">
        <div v-for="g in filteredGroups" :key="g.Name" class="be-group">
          <div class="be-group-name">{{ grpLabel(g) }}</div>
          <button
            v-for="l in g.Levels"
            :key="l.Name"
            class="be-level-btn"
            :class="{ active: selectedLevel === l }"
            @click="selectLevel(g, l)"
          >{{ lvlLabel(l) }}</button>
        </div>
        <div v-if="filteredGroups.length === 0" class="be-empty">Ничего не найдено</div>
      </div>
    </div>

    <!-- Правая панель: свойства уровня -->
    <div v-if="edited" class="be-detail">

      <div class="be-detail-header">
        <div class="be-detail-title">
          <span class="be-detail-group">{{ grpLabel(selectedGroup) }}</span>
          <span class="be-detail-sep">›</span>
          <span class="be-detail-level">{{ lvlLabel(selectedLevel) }}</span>
        </div>
        <div class="be-toolbar">
          <span v-if="error" class="be-error">{{ error }}</span>
          <button v-if="localDirty" class="be-btn-save" :disabled="saving" @click="save">
            {{ saving ? 'Сохранение...' : 'Сохранить' }}
          </button>
          <button v-if="localDirty" class="be-btn-cancel" @click="selectLevel(selectedGroup, selectedLevel)">
            Отмена
          </button>
        </div>
      </div>

      <div class="be-form">

        <!-- Экономика -->
        <div class="be-section">
          <h3>Экономика</h3>
          <div class="be-fields">
            <label class="be-field">
              <span>Стоимость строительства</span>
              <input type="number" v-model.number="edited.cost" min="0" @input="markDirty" />
            </label>
            <label class="be-field">
              <span>Ходов на строительство</span>
              <input type="number" v-model.number="edited.construction" min="0" @input="markDirty" />
            </label>
            <label class="be-field">
              <span>Минимальный тип поселения</span>
              <select v-model="edited.settlementMin" @change="markDirty">
                <option v-for="opt in SETTLEMENT_OPTIONS" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </option>
              </select>
            </label>
          </div>
        </div>

        <!-- Доступно фракциям -->
        <div class="be-section">
          <h3>Доступно фракциям</h3>
          <div class="be-chips">
            <div v-if="edited.requiredCultures.length === 0" class="be-empty-hint">Все фракции</div>
            <div v-for="c in edited.requiredCultures" :key="c" class="be-chip">
              <span>{{ c }}</span>
              <button class="be-chip-del" @click="removeCulture(c)">✕</button>
            </div>
          </div>
          <select class="be-add-select" @change="e => { addCulture(e.target.value); e.target.value = '' }">
            <option value="">+ Добавить культуру...</option>
            <option
              v-for="c in allCultures.filter(c => !edited.requiredCultures.includes(c))"
              :key="c" :value="c"
            >{{ c }}</option>
          </select>
        </div>

        <!-- Зависимость от здания -->
        <div class="be-section">
          <h3>
            Зависимость от здания
            <button v-if="depChanged" class="be-inline-revert" @click="revertDep" title="Откатить">↺</button>
          </h3>
          <div class="be-fields">
            <label class="be-field">
              <span>Группа зданий</span>
              <select
                class="be-dep-select"
                :value="edited.dependencyGroup"
                @change="e => { edited.dependencyGroup = e.target.value; edited.dependencyLevel = ''; markDirty() }"
              >
                <option value="">— нет зависимости —</option>
                <option v-for="g in buildings" :key="g.Name" :value="g.Name">{{ g.Name }}</option>
              </select>
            </label>
            <label class="be-field" v-if="edited.dependencyGroup">
              <span>Минимальный уровень</span>
              <select
                class="be-dep-select"
                v-model="edited.dependencyLevel"
                @change="markDirty"
              >
                <option value="">— выберите уровень —</option>
                <option v-for="lvl in depGroupLevels" :key="lvl" :value="lvl">{{ lvl }}</option>
              </select>
            </label>
          </div>
        </div>

        <!-- Бонусы capability -->
        <div class="be-section">
          <h3>Бонусы (capability)</h3>

          <div class="be-bonus-list">
            <div v-if="bonusItems.length === 0" class="be-empty-hint">Нет бонусов</div>
            <div v-for="(item, i) in bonusItems" :key="i" class="be-bonus-row">
              <div class="be-bonus-tag" :class="{ 'is-new': isNewBonus(item), 'is-modified': isModifiedBonus(item) }">
                {{ isNewBonus(item) ? 'new' : isModifiedBonus(item) ? 'изм' : '' }}
              </div>
              <input
                :value="item.text"
                class="be-bonus-input"
                @input="e => { item.text = e.target.value; markDirty() }"
              />
              <button
                v-if="isModifiedBonus(item)"
                class="be-bonus-action revert"
                @click="revertBonus(item)"
                title="Откатить к оригиналу"
              >↺</button>
              <button
                class="be-bonus-action remove"
                @click="removeBonus(i)"
                :title="isNewBonus(item) ? 'Удалить' : 'Убрать из файла'"
              >✕</button>
            </div>
          </div>

          <!-- Добавление бонуса -->
          <div class="be-bonus-add-area">
            <div class="be-upg-input-row">
              <input
                v-model="newBonus"
                class="be-bonus-new-input"
                placeholder="happiness_bonus bonus 3"
                @keydown.enter="addBonus"
              />
              <button class="be-upg-add-btn" @click="addBonus">+</button>
              <button
                class="be-tpl-toggle"
                :class="{ active: showTemplates }"
                @click="showTemplates = !showTemplates"
                title="Шаблоны бонусов"
              >≡</button>
            </div>

            <!-- Шаблоны -->
            <div v-if="showTemplates" class="be-templates">
              <div v-for="cat in BONUS_TEMPLATES" :key="cat.cat" class="be-tpl-cat">
                <div class="be-tpl-cat-name">{{ cat.cat }}</div>
                <div class="be-tpl-items">
                  <button
                    v-for="tpl in cat.items"
                    :key="tpl"
                    class="be-tpl-item"
                    @click="useTemplate(tpl)"
                  >{{ tpl }}</button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Улучшения -->
        <div class="be-section">
          <h3>Улучшения</h3>
          <div class="be-chips">
            <div v-if="edited.upgrades.length === 0" class="be-empty-hint">Нет улучшений</div>
            <div v-for="upg in edited.upgrades" :key="upg" class="be-chip">
              <span>{{ upg }}</span>
              <button class="be-chip-del" @click="removeUpgrade(upg)">✕</button>
            </div>
          </div>
          <div class="be-upg-add">
            <div class="be-upg-input-row">
              <input
                v-model="newUpgrade"
                class="be-upg-input"
                placeholder="Имя уровня..."
                @keydown.enter="addUpgrade(newUpgrade.trim())"
                @focus="upgradePickerOpen = true"
                @blur="closeUpgradePicker"
              />
              <button class="be-upg-add-btn" @click="addUpgrade(newUpgrade.trim())">+</button>
            </div>
            <div v-if="upgradePickerOpen && availableUpgrades.length" class="be-upg-picker">
              <button
                v-for="name in availableUpgrades.filter(n => !newUpgrade || n.includes(newUpgrade))"
                :key="name"
                class="be-upg-pick-item"
                @mousedown.prevent="addUpgrade(name)"
              >{{ name }}</button>
            </div>
          </div>
        </div>

      </div>
    </div>

    <div v-else class="be-empty-detail">
      <svg viewBox="0 0 24 24" fill="currentColor" width="40" height="40" style="opacity:0.2">
        <path d="M3 21h18v-2H3v2zM5 9.5v9.5h3V9.5H5zm5.5 0v9.5h3V9.5h-3zM16 9.5v9.5h3V9.5h-3zM2 7.5l10-5 10 5v1.5H2V7.5z"/>
      </svg>
      <span>Выберите уровень здания</span>
    </div>

  </div>
</template>

<style scoped>
.be-root { display: flex; height: 100%; overflow: hidden; }

/* ── Дерево ── */
.be-tree {
  width: 260px;
  flex-shrink: 0;
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.be-tree-header {
  padding: 12px;
  border-bottom: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex-shrink: 0;
}
.be-search, .be-faction-select {
  width: 100%;
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text);
  font-size: 12px;
  padding: 6px 8px;
}
.be-search:focus, .be-faction-select:focus { outline: none; border-color: var(--color-accent); }
.be-faction-select option { background: var(--color-input-bg); }
.be-revert-btn {
  background: transparent;
  border: 1px solid var(--color-accent);
  border-radius: 4px;
  color: var(--color-accent);
  font-size: 11px;
  padding: 4px 8px;
  cursor: pointer;
}
.be-revert-btn:hover { background: rgba(149,232,225,0.2); }
.be-groups { flex: 1; overflow-y: auto; padding: 8px; }
.be-group { margin-bottom: 12px; }
.be-group-name {
  font-size: 10px;
  text-transform: uppercase;
  color: var(--color-heading);
  letter-spacing: 0.06em;
  padding: 2px 6px;
  margin-bottom: 3px;
}
.be-level-btn {
  display: block;
  width: 100%;
  text-align: left;
  background: var(--color-cell);
  border: 1px solid transparent;
  border-radius: 3px;
  color: var(--color-text-dim);
  font-size: 12px;
  padding: 5px 10px;
  margin-bottom: 2px;
  cursor: pointer;
}
.be-level-btn:hover { border-color: var(--color-border-soft); color: var(--color-text); }
.be-level-btn.active { background: var(--color-primary); border-color: var(--color-primary-hover); color: var(--color-text); }
.be-empty { font-size: 12px; color: var(--color-text-muted); text-align: center; padding: 20px; }

/* ── Детали ── */
.be-detail { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.be-detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 20px;
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.be-detail-title { display: flex; align-items: center; gap: 6px; }
.be-detail-group { font-size: 13px; color: var(--color-text-dim); }
.be-detail-sep { color: var(--color-text-muted); }
.be-detail-level { font-size: 14px; font-weight: 600; color: var(--color-text); }
.be-toolbar { display: flex; align-items: center; gap: 8px; }
.be-error { font-size: 12px; color: var(--color-error); }
.be-btn-save {
  background: var(--color-primary);
  border: 1px solid var(--color-primary-hover);
  border-radius: 4px;
  color: var(--color-text);
  font-size: 12px;
  padding: 5px 14px;
  cursor: pointer;
}
.be-btn-save:hover { background: var(--color-primary-hover); }
.be-btn-save:disabled { opacity: 0.5; cursor: default; }
.be-btn-cancel {
  background: transparent;
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text-dim);
  font-size: 12px;
  padding: 5px 12px;
  cursor: pointer;
}
.be-btn-cancel:hover { color: var(--color-text-dim); border-color: var(--color-text-dim); }

/* ── Форма ── */
.be-form {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}
.be-section h3 {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  text-transform: uppercase;
  color: var(--color-heading);
  letter-spacing: 0.06em;
  margin-bottom: 14px;
}
.be-inline-revert {
  background: none;
  border: 1px solid var(--color-accent);
  border-radius: 3px;
  color: var(--color-accent);
  font-size: 12px;
  padding: 1px 5px;
  cursor: pointer;
  line-height: 1.2;
}
.be-inline-revert:hover { background: rgba(149,232,225,0.2); }
.be-fields { display: flex; flex-direction: column; gap: 12px; }
.be-field { display: flex; align-items: center; gap: 12px; }
.be-field span { width: 200px; flex-shrink: 0; font-size: 13px; color: var(--color-text-dim); }
.be-field input, .be-field select {
  flex: 1;
  max-width: 200px;
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text);
  font-size: 13px;
  padding: 6px 8px;
}
.be-field input:focus, .be-field select:focus { outline: none; border-color: var(--color-accent); }
.be-field select option { background: var(--color-input-bg); }

/* ── Chips ── */
.be-chips { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 10px; min-height: 26px; }
.be-empty-hint { font-size: 12px; color: var(--color-text-muted); }
.be-chip {
  display: flex;
  align-items: center;
  gap: 5px;
  background: var(--color-cell);
  border: 1px solid var(--color-border-soft);
  border-radius: 12px;
  padding: 3px 8px 3px 10px;
  font-size: 12px;
  color: var(--color-text-dim);
}
.be-chip-del {
  background: none;
  border: none;
  color: var(--color-text-muted);
  cursor: pointer;
  font-size: 10px;
  padding: 0;
  line-height: 1;
}
.be-chip-del:hover { color: var(--color-accent); }
.be-add-select {
  width: 100%;
  max-width: 260px;
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text-dim);
  font-size: 12px;
  padding: 5px 8px;
}
.be-add-select:focus { outline: none; border-color: var(--color-accent); }
.be-add-select option { background: var(--color-input-bg); color: var(--color-text); }

/* ── Dependency selects ── */
.be-dep-select {
  flex: 1;
  max-width: 220px;
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text);
  font-size: 12px;
  padding: 6px 8px;
}
.be-dep-select:focus { outline: none; border-color: var(--color-accent); }
.be-dep-select option { background: var(--color-input-bg); }

/* ── Bonus list ── */
.be-bonus-list { display: flex; flex-direction: column; gap: 4px; margin-bottom: 8px; }
.be-bonus-row { display: flex; align-items: center; gap: 5px; }
.be-bonus-tag {
  width: 28px;
  flex-shrink: 0;
  font-size: 9px;
  text-align: center;
  border-radius: 3px;
  padding: 1px 3px;
  color: transparent;
  background: transparent;
}
.be-bonus-tag.is-new { color: #3aba80; background: rgba(58,186,128,0.2); border: 1px solid rgba(58,186,128,0.3); }
.be-bonus-tag.is-modified { color: #e9a000; background: rgba(233,160,0,0.1); border: 1px solid rgba(233,160,0,0.3); }
.be-bonus-input {
  flex: 1;
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text);
  font-size: 12px;
  padding: 5px 8px;
  font-family: monospace;
}
.be-bonus-input:focus { outline: none; border-color: var(--color-accent); }
.be-bonus-action {
  flex-shrink: 0;
  background: none;
  border: none;
  cursor: pointer;
  font-size: 12px;
  padding: 2px 4px;
  border-radius: 3px;
  line-height: 1;
}
.be-bonus-action.revert { color: #e9a000; }
.be-bonus-action.revert:hover { color: #ffc000; background: rgba(233,160,0,0.1); }
.be-bonus-action.remove { color: var(--color-text-muted); }
.be-bonus-action.remove:hover { color: var(--color-accent); background: rgba(149,232,225,0.2); }

/* ── Bonus add + templates ── */
.be-bonus-add-area { display: flex; flex-direction: column; gap: 8px; }
.be-upg-input-row { display: flex; gap: 6px; }
.be-bonus-new-input {
  flex: 1;
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text);
  font-size: 12px;
  padding: 5px 8px;
  font-family: monospace;
}
.be-bonus-new-input:focus { outline: none; border-color: var(--color-accent); }
.be-upg-add-btn {
  background: var(--color-primary);
  border: 1px solid var(--color-primary-hover);
  border-radius: 4px;
  color: var(--color-text);
  font-size: 14px;
  width: 28px;
  cursor: pointer;
}
.be-upg-add-btn:hover { background: var(--color-primary-hover); }
.be-tpl-toggle {
  background: var(--color-cell);
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text-dim);
  font-size: 14px;
  width: 28px;
  cursor: pointer;
}
.be-tpl-toggle:hover, .be-tpl-toggle.active { border-color: var(--color-accent); color: var(--color-accent); }

.be-templates {
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 220px;
  overflow-y: auto;
}
.be-tpl-cat-name {
  font-size: 10px;
  text-transform: uppercase;
  color: var(--color-heading);
  letter-spacing: 0.06em;
  margin-bottom: 5px;
}
.be-tpl-items { display: flex; flex-wrap: wrap; gap: 4px; }
.be-tpl-item {
  background: var(--color-cell);
  border: 1px solid var(--color-border-soft);
  border-radius: 3px;
  color: var(--color-text-dim);
  font-size: 11px;
  font-family: monospace;
  padding: 3px 7px;
  cursor: pointer;
  white-space: nowrap;
}
.be-tpl-item:hover { border-color: var(--color-accent); color: var(--color-text); }

/* ── Upgrades (picker) ── */
.be-upg-add { position: relative; }
.be-upg-input {
  flex: 1;
  max-width: 240px;
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text);
  font-size: 12px;
  padding: 5px 8px;
}
.be-upg-input:focus { outline: none; border-color: var(--color-accent); }
.be-upg-picker {
  position: absolute;
  top: 100%;
  left: 0;
  z-index: 10;
  background: var(--color-cell);
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  max-height: 180px;
  overflow-y: auto;
  min-width: 200px;
  margin-top: 4px;
}
.be-upg-pick-item {
  display: block;
  width: 100%;
  text-align: left;
  background: none;
  border: none;
  color: var(--color-text-dim);
  font-size: 12px;
  padding: 6px 10px;
  cursor: pointer;
}
.be-upg-pick-item:hover { background: var(--color-primary); color: var(--color-text); }

/* ── Empty ── */
.be-empty-detail {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--color-border-soft);
  font-size: 13px;
}
</style>
