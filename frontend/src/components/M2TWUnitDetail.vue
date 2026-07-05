<script setup>
import { ref, watch, onMounted } from 'vue'
import {
  GetM2TWUnitByType, UpdateM2TWUnit, GetM2TWUnitChangeType,
  DeleteM2TWUnit, HardDeleteM2TWUnit, IsCopyUnit, RevertM2TWUnit,
  GetM2TWUnitBuildings, GetM2TWBuildings, UpdateM2TWBuildingLevel,
  GetM2TWFactions, GetM2TWProjectileTypes, HasRexEngine,
} from '../../wailsjs/go/main/App'
import {
  UNIT_CATEGORIES, UNIT_CLASSES, VOICE_TYPES,
  WEAPON_TYPES, TECH_TYPES, DAMAGE_TYPES, SOUND_TYPES, ARMOUR_SOUNDS,
  DISCIPLINE_VALUES, TRAINING_VALUES, UNIT_ATTRIBUTES, REX_UNIT_ATTRIBUTES, WEAPON_ATTRIBUTES,
  FORMATION_PRIMARY, FORMATION_SECONDARY,
} from '../enums.js'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps(['unitType', 'faction'])
const emit = defineEmits(['saved', 'reverted', 'deleted', 'restored'])

const unit = ref(null)
const edited = ref(null)
const originalType = ref(null)
const dirty = ref(false)
const saving = ref(false)
const error = ref(null)
const unitBuildings = ref([])
const unitChangeType = ref('none')
const projectileTypes = ref(['no'])
const allFactions = ref([])
const hasRex = ref(false)

// Building picker
const bldPicker = ref(false)
const bldPickerSearch = ref('')
const allBuildings = ref([])
const bldPickerLoading = ref(false)

onMounted(async () => {
  const [pt, factions, rex] = await Promise.all([GetM2TWProjectileTypes(), GetM2TWFactions(), HasRexEngine()])
  projectileTypes.value = ['no', ...(pt ?? [])]
  allFactions.value = factions ?? []
  hasRex.value = rex
})

watch(() => props.unitType, async (type) => {
  if (!type) { unit.value = null; edited.value = null; unitBuildings.value = []; unitChangeType.value = 'none'; return }
  error.value = null
  try {
    const [u, buildings, ct] = await Promise.all([
      GetM2TWUnitByType(type),
      GetM2TWUnitBuildings(type),
      GetM2TWUnitChangeType(type),
    ])
    if (props.unitType !== type) return
    unit.value = u
    unitBuildings.value = buildings
    unitChangeType.value = ct
    edited.value = JSON.parse(JSON.stringify(unit.value))
    if (!edited.value.Eras) edited.value.Eras = {}
    if (!edited.value.VoiceType) edited.value.VoiceType = 'Heavy_1'
    if (edited.value.StatPri && !edited.value.StatPri.TechType) edited.value.StatPri.TechType = 'simple'
    if (edited.value.StatSec && !edited.value.StatSec.TechType) edited.value.StatSec.TechType = 'no'
    originalType.value = type
    dirty.value = false
  } catch (e) {
    if (props.unitType !== type) return
    error.value = t('unit.load_error', { type, err: e })
    unit.value = null
    edited.value = null
  }
}, { immediate: true })

function markDirty() { dirty.value = true }

function onWeaponAttrChange(field, key, checked) {
  const current = (edited.value[field] ?? []).filter(a => a !== 'no')
  const updated = checked ? [...current, key] : current.filter(a => a !== key)
  edited.value[field] = updated.length ? updated : ['no']
  markDirty()
}

function fParts() {
  return (edited.value?.Formation ?? '').split(',').map(s => s.trim())
}
function getFormationNum(idx) { return parseFloat(fParts()[idx]) || 0 }
function setFormationNum(idx, val) {
  const p = fParts()
  while (p.length < 7) p.push('')
  p[idx] = String(val)
  edited.value.Formation = p.filter((v, i) => i < 5 || v !== '').join(', ')
  markDirty()
}
function getFormationPrimary() { return fParts()[5] || 'square' }
function getFormationSecondary() { return fParts()[6] || '' }
function setFormationPrimary(val) {
  const p = fParts()
  while (p.length < 6) p.push('')
  p[5] = val
  edited.value.Formation = p.filter((v, i) => i < 5 || v !== '').join(', ')
  markDirty()
}
function setFormationSecondary(val) {
  const p = fParts()
  while (p.length < 6) p.push('square')
  if (val) { p[6] = val } else { p.splice(6, 1) }
  edited.value.Formation = p.filter((v, i) => i < 5 || v !== '').join(', ')
  markDirty()
}
function getSpearBonus(field) {
  const item = (edited.value?.[field] ?? []).find(a => /^spear_bonus_\d+$/.test(a))
  return item ? parseInt(item.slice(12)) : 0
}
function setSpearBonus(field, n) {
  const current = (edited.value[field] ?? []).filter(a => !/^spear_bonus_\d+$/.test(a))
  edited.value[field] = n > 0 ? [...current, `spear_bonus_${n}`] : current
  markDirty()
}

// ── Eras ──
function eraFactions(eraNum) {
  return (edited.value?.Eras?.[eraNum] ?? [])
}
function addEraFaction(eraNum, name) {
  if (!name) return
  if (!edited.value.Eras) edited.value.Eras = {}
  const cur = edited.value.Eras[eraNum] ?? []
  if (!cur.includes(name)) {
    edited.value.Eras = { ...edited.value.Eras, [eraNum]: [...cur, name] }
    markDirty()
  }
}
function removeEraFaction(eraNum, name) {
  if (!edited.value.Eras?.[eraNum]) return
  edited.value.Eras = {
    ...edited.value.Eras,
    [eraNum]: edited.value.Eras[eraNum].filter(f => f !== name),
  }
  markDirty()
}
function availableEraFactions(eraNum) {
  const already = new Set(eraFactions(eraNum))
  return allFactions.value.filter(f => !already.has(f.Name))
}

// ── Ownership ──
function addFaction(name) {
  if (!name) return
  if (!(edited.value.Ownership ?? []).includes(name)) {
    edited.value.Ownership = [...(edited.value.Ownership ?? []), name]
    markDirty()
  }
}
function removeFaction(name) {
  edited.value.Ownership = (edited.value.Ownership ?? []).filter(f => f !== name)
  markDirty()
}

// ── Officers ──
function setOfficer(idx, val) {
  const arr = [...(edited.value.Officers ?? [])]
  arr[idx] = val
  edited.value.Officers = arr
  markDirty()
}
function removeOfficer(idx) {
  edited.value.Officers = (edited.value.Officers ?? []).filter((_, i) => i !== idx)
  markDirty()
}
function addOfficer() {
  edited.value.Officers = [...(edited.value.Officers ?? []), '']
  markDirty()
}

// ── Save / Cancel / Delete ──
async function save() {
  saving.value = true
  error.value = null
  try {
    await UpdateM2TWUnit(originalType.value, edited.value)
    originalType.value = edited.value.Type
    unit.value = JSON.parse(JSON.stringify(edited.value))
    dirty.value = false
    emit('saved')
  } catch (e) {
    error.value = String(e)
  } finally {
    saving.value = false
  }
}
function cancel() {
  edited.value = JSON.parse(JSON.stringify(unit.value))
  if (!edited.value.Eras) edited.value.Eras = {}
  if (!edited.value.VoiceType) edited.value.VoiceType = 'Heavy_1'
  if (edited.value.StatPri && !edited.value.StatPri.TechType) edited.value.StatPri.TechType = 'simple'
  if (edited.value.StatSec && !edited.value.StatSec.TechType) edited.value.StatSec.TechType = 'no'
  originalType.value = unit.value.Type
  dirty.value = false
  error.value = null
  emit('reverted')
}
async function deleteUnit() {
  const type = originalType.value
  const isCopy = await IsCopyUnit(type)

  if (isCopy) {
    const choice = window.confirm(t('unit.confirm_delete_copy', { type, faction: props.faction }))
    try {
      if (choice) {
        await HardDeleteM2TWUnit(type)
      } else {
        await DeleteM2TWUnit(type, props.faction)
      }
      emit('deleted')
    } catch (e) {
      error.value = String(e)
    }
    return
  }

  const isNew = unitChangeType.value === 'added'
  const msg = isNew
    ? t('unit.confirm_delete_new', { type })
    : t('unit.confirm_delete_faction', { type, faction: props.faction })
  if (!confirm(msg)) return
  try {
    await DeleteM2TWUnit(type, props.faction)
    emit('deleted')
  } catch (e) { error.value = String(e) }
}
async function restoreUnit() {
  try {
    await RevertM2TWUnit(originalType.value)
    emit('restored')
  } catch (e) { error.value = String(e) }
}

// ── Building picker ──
function bldName(grp, lvl) {
  const n = lvl.DisplayName || grp.DisplayName || lvl.Name
  return n.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}
function filteredBldGroups() {
  const q = bldPickerSearch.value.trim().toLowerCase()
  const unitType = originalType.value
  return allBuildings.value
    .map(grp => ({
      ...grp,
      Levels: (grp.Levels ?? []).filter(lvl => {
        const alreadyHas = (lvl.RecruitPools ?? []).some(p => p.UnitType === unitType)
        if (alreadyHas) return false
        if (!q) return true
        return bldName(grp, lvl).toLowerCase().includes(q) || (grp.Name ?? '').toLowerCase().includes(q)
      }),
    }))
    .filter(grp => grp.Levels.length > 0)
}
async function openBldPicker() {
  bldPicker.value = true
  bldPickerSearch.value = ''
  if (allBuildings.value.length === 0) {
    bldPickerLoading.value = true
    try { allBuildings.value = await GetM2TWBuildings() } finally { bldPickerLoading.value = false }
  }
}
async function addToBuilding(grp, lvl) {
  const newPool = {
    UnitType: originalType.value,
    InitialPool: 1,
    ReplenishRate: 0.15,
    MaxPool: 3,
    ExpGained: 0,
    Factions: [],
    Conditions: '',
  }
  const updatedPools = [...(lvl.RecruitPools ?? []), newPool]
  const updatedBonus = lvl.BonusLines ?? []
  await UpdateM2TWBuildingLevel(grp.Name, lvl.Name, updatedPools, updatedBonus)
  lvl.RecruitPools = updatedPools
  unitBuildings.value = await GetM2TWUnitBuildings(originalType.value)
  bldPicker.value = false
}
async function removeFromBuilding(loc) {
  if (allBuildings.value.length === 0) allBuildings.value = await GetM2TWBuildings()
  const grp = allBuildings.value.find(g => g.Name === loc.GroupName)
  if (!grp) return
  const lvl = (grp.Levels ?? []).find(l => l.Name === loc.LevelName)
  if (!lvl) return
  const updatedPools = (lvl.RecruitPools ?? []).filter(p => p.UnitType !== originalType.value)
  await UpdateM2TWBuildingLevel(loc.GroupName, loc.LevelName, updatedPools, lvl.BonusLines ?? [])
  lvl.RecruitPools = updatedPools
  unitBuildings.value = await GetM2TWUnitBuildings(originalType.value)
}
</script>

<template>
  <div v-if="edited" class="detail-root">
    <div class="detail-main">

      <!-- Шапка -->
      <div class="detail-header">
        <div class="header-row">
          <div class="header-unit-ident">
            <div class="header-icon-placeholder">
              {{ (edited.Name || edited.Type).slice(0, 2).toUpperCase() }}
            </div>
            <div class="header-unit-name">
              <span class="header-name">{{ edited.Name || edited.Type }}</span>
              <span class="header-type">{{ edited.Type }}</span>
            </div>
          </div>
          <div class="toolbar">
            <template v-if="dirty">
              <button class="btn-save" :disabled="saving" @click="save">{{ saving ? '...' : $t('unit.btn_save') }}</button>
              <button class="btn-cancel" @click="cancel">{{ $t('unit.btn_cancel') }}</button>
            </template>
            <button v-if="unitChangeType === 'deleted'" class="btn-restore" @click="restoreUnit">↺</button>
            <button v-else class="btn-delete" @click="deleteUnit" :title="unitChangeType === 'added' ? $t('unit.btn_delete_copy') : $t('unit.btn_delete_unit')">✕</button>
            <span v-if="error" class="error">{{ error }}</span>
          </div>
        </div>
      </div>

      <div class="detail-scroll">
      <div class="sections">

        <!-- Описание -->
        <section class="stat-section">
          <h3>{{ $t('unit.section_description') }}</h3>
          <div class="grid">
            <label class="full">{{ $t('unit.field_name') }}<input v-model="edited.Name" @input="markDirty" /></label>
            <label class="full">{{ $t('unit.field_descr_short') }}<textarea v-model="edited.DescrShort" @input="markDirty" rows="2" class="desc-textarea" /></label>
            <label class="full">{{ $t('unit.field_descr_full') }}<textarea v-model="edited.Descr" @input="markDirty" rows="4" class="desc-textarea" /></label>
          </div>
        </section>

        <!-- Основное -->
        <section class="stat-section">
          <h3>{{ $t('unit.section_main') }}</h3>
          <div class="grid">
            <label>{{ $t('unit.field_type') }}<input v-model="edited.Type" @input="markDirty" /></label>
            <label>Dictionary<input v-model="edited.Dictionary" @input="markDirty" /></label>
            <label>{{ $t('unit.field_category') }}
              <select v-model="edited.Category" @change="markDirty">
                <option v-for="v in UNIT_CATEGORIES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>{{ $t('unit.field_class') }}
              <select v-model="edited.Class" @change="markDirty">
                <option v-for="v in UNIT_CLASSES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>{{ $t('unit.field_voice') }}
              <select v-model="edited.VoiceType" @change="markDirty">
                <option v-if="edited.VoiceType && !VOICE_TYPES.includes(edited.VoiceType)" :value="edited.VoiceType">{{ edited.VoiceType }}</option>
                <option v-for="v in VOICE_TYPES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>{{ $t('unit.field_accent') }}<input v-model="edited.Accent" @input="markDirty" /></label>
            <label>Banner Faction<input v-model="edited.BannerFaction" @input="markDirty" /></label>
            <label>Banner Holy<input v-model="edited.BannerHoly" @input="markDirty" /></label>
            <label v-if="edited.Mount || edited.Category === 'cavalry'">{{ $t('unit.field_mount') }}<input v-model="edited.Mount" @input="markDirty" /></label>
            <label v-if="edited.Mount || edited.MountEffect || edited.Category === 'cavalry'">{{ $t('unit.field_mount_effect') }}<input v-model="edited.MountEffect" @input="markDirty" /></label>
            <label>{{ $t('unit.field_speed_mod') }}<input type="number" step="0.01" v-model.number="edited.MoveSpeedMod" @input="markDirty" /></label>

            <!-- Офицеры -->
            <div class="full attr-group">
              <div class="attr-label">{{ $t('unit.field_officers') }}</div>
              <div v-for="(o, idx) in (edited.Officers ?? [])" :key="idx" class="officer-row">
                <input :value="o" @input="e => setOfficer(idx, e.target.value)" class="officer-input" />
                <button class="officer-remove" @click="removeOfficer(idx)">✕</button>
              </div>
              <button class="btn-small-add" @click="addOfficer">+ {{ $t('unit.field_officers') }}</button>
            </div>

            <!-- Строй -->
            <div class="full formation-group">
              <div class="attr-label">{{ $t('unit.field_formation') }}</div>
              <div class="formation-pairs">
                <div class="formation-pair">
                  <div class="formation-pair-title">{{ $t('unit.field_formation_tight') }}</div>
                  <div class="formation-pair-inputs">
                    <label>{{ $t('unit.field_formation_width') }}<input type="number" step="0.1" :value="getFormationNum(0)" @change="e => setFormationNum(0, e.target.value)" /></label>
                    <label>{{ $t('unit.field_formation_depth') }}<input type="number" step="0.1" :value="getFormationNum(1)" @change="e => setFormationNum(1, e.target.value)" /></label>
                  </div>
                </div>
                <div class="formation-pair">
                  <div class="formation-pair-title">{{ $t('unit.field_formation_loose') }}</div>
                  <div class="formation-pair-inputs">
                    <label>{{ $t('unit.field_formation_width') }}<input type="number" step="0.1" :value="getFormationNum(2)" @change="e => setFormationNum(2, e.target.value)" /></label>
                    <label>{{ $t('unit.field_formation_depth') }}<input type="number" step="0.1" :value="getFormationNum(3)" @change="e => setFormationNum(3, e.target.value)" /></label>
                  </div>
                </div>
                <label class="formation-single">{{ $t('unit.field_formation_max_depth') }}<input type="number" step="1" :value="getFormationNum(4)" @change="e => setFormationNum(4, e.target.value)" /></label>
                <label class="formation-select">{{ $t('unit.field_formation_type') }}
                  <select :value="getFormationPrimary()" @change="e => setFormationPrimary(e.target.value)">
                    <option v-for="v in FORMATION_PRIMARY" :key="v" :value="v">{{ v }}</option>
                  </select>
                </label>
                <label class="formation-select">{{ $t('unit.field_formation_secondary') }}
                  <select :value="getFormationSecondary()" @change="e => setFormationSecondary(e.target.value)">
                    <option v-for="v in FORMATION_SECONDARY" :key="v" :value="v">{{ v || '—' }}</option>
                  </select>
                </label>
              </div>
            </div>

            <!-- Атрибуты -->
            <div class="full attr-group">
              <div class="attr-label">{{ $t('unit.field_attributes') }}</div>
              <input type="text" readonly class="attr-display" :value="(edited.Attributes ?? []).join(', ') || '—'" />
              <div class="attr-list">
                <label v-for="attr in UNIT_ATTRIBUTES" :key="attr.key" class="attr-check">
                  <input type="checkbox" :value="attr.key" v-model="edited.Attributes" @change="markDirty" />
                  {{ attr.label }}
                </label>
              </div>
              <template v-if="hasRex">
                <div class="attr-divider">{{ $t('unit.rex_attributes_divider') }}</div>
                <div class="attr-list-rex">
                  <label v-for="attr in REX_UNIT_ATTRIBUTES" :key="attr.key" class="attr-check-rex">
                    <input type="checkbox" :value="attr.key" v-model="edited.Attributes" @change="markDirty" />
                    <span class="attr-check-text">
                      <span class="attr-check-name">{{ attr.label }}</span>
                      <span v-if="attr.hint" class="attr-check-hint">{{ attr.hint }}</span>
                    </span>
                  </label>
                </div>
              </template>
            </div>
          </div>
        </section>

        <!-- Солдаты -->
        <section class="stat-section">
          <h3>{{ $t('unit.section_soldiers') }}</h3>
          <div class="grid">
            <label>{{ $t('unit.field_model') }}<input v-model="edited.Soldier.Model" @input="markDirty" /></label>
            <label>{{ $t('unit.field_count') }}<input type="number" v-model.number="edited.Soldier.Count" @input="markDirty" /></label>
            <label>{{ $t('unit.field_extras') }}<input type="number" v-model.number="edited.Soldier.Extras" @input="markDirty" /></label>
            <label>{{ $t('unit.field_mass') }}<input type="number" step="0.1" v-model.number="edited.Soldier.Mass" @input="markDirty" /></label>
          </div>
        </section>

        <!-- Здоровье -->
        <section class="stat-section">
          <h3>{{ $t('unit.section_health') }}</h3>
          <div class="grid">
            <label>{{ $t('unit.field_hp') }}<input type="number" v-model.number="edited.StatHealth[0]" @input="markDirty" /></label>
            <label>{{ $t('unit.field_hp_bonus') }}<input type="number" v-model.number="edited.StatHealth[1]" @input="markDirty" /></label>
          </div>
        </section>

        <!-- Основное оружие -->
        <section class="stat-section">
          <h3>{{ $t('unit.section_primary_weapon') }}</h3>
          <div class="grid">
            <label>{{ $t('unit.field_attack') }}<input type="number" v-model.number="edited.StatPri.Attack" @input="markDirty" /></label>
            <label>{{ $t('unit.field_charge') }}<input type="number" v-model.number="edited.StatPri.ChargeBonus" @input="markDirty" /></label>
            <label>{{ $t('unit.field_missile') }}
              <select v-model="edited.StatPri.Missile" @change="markDirty">
                <option v-for="v in projectileTypes" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>{{ $t('unit.field_range') }}<input type="number" v-model.number="edited.StatPri.Range" @input="markDirty" /></label>
            <label>{{ $t('unit.field_ammo') }}<input type="number" v-model.number="edited.StatPri.Ammo" @input="markDirty" /></label>
            <label>{{ $t('unit.field_weapon_type') }}
              <select v-model="edited.StatPri.WeaponType" @change="markDirty">
                <option v-for="v in WEAPON_TYPES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>{{ $t('unit.field_tech_type') }}
              <select v-model="edited.StatPri.TechType" @change="markDirty">
                <option v-for="v in TECH_TYPES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>{{ $t('unit.field_damage_type') }}
              <select v-model="edited.StatPri.DamageType" @change="markDirty">
                <option v-for="v in DAMAGE_TYPES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>{{ $t('unit.field_sound') }}
              <select v-model="edited.StatPri.SoundType" @change="markDirty">
                <option v-for="v in SOUND_TYPES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>{{ $t('unit.field_delay') }}<input type="number" step="0.1" v-model.number="edited.StatPri.MinDelay" @input="markDirty" /></label>
            <label>{{ $t('unit.field_factor') }}<input type="number" step="0.1" v-model.number="edited.StatPri.Factor" @input="markDirty" /></label>
            <div class="full attr-group">
              <div class="attr-label">{{ $t('unit.field_weapon_attributes') }}</div>
              <input type="text" readonly class="attr-display" :value="(edited.StatPriAttr ?? []).filter(a => a !== 'no').join(', ') || '—'" />
              <div class="attr-list">
                <label v-for="attr in WEAPON_ATTRIBUTES" :key="attr.key" class="attr-check">
                  <input type="checkbox" :checked="!!(edited.StatPriAttr?.includes(attr.key))" @change="e => onWeaponAttrChange('StatPriAttr', attr.key, e.target.checked)" />
                  {{ attr.label }}
                </label>
                <label class="attr-check spear-bonus-row">
                  <span>{{ $t('unit.field_spear_bonus') }} <span class="attr-key">(spear_bonus)</span></span>
                  <input type="number" min="0" max="20" step="1" class="spear-bonus-input" :value="getSpearBonus('StatPriAttr')" @change="e => setSpearBonus('StatPriAttr', Number(e.target.value))" />
                </label>
              </div>
            </div>
          </div>
        </section>

        <!-- Вторичное оружие -->
        <section class="stat-section">
          <h3>{{ $t('unit.section_secondary_weapon') }}</h3>
          <div class="grid">
            <label>{{ $t('unit.field_attack') }}<input type="number" v-model.number="edited.StatSec.Attack" @input="markDirty" /></label>
            <label>{{ $t('unit.field_charge') }}<input type="number" v-model.number="edited.StatSec.ChargeBonus" @input="markDirty" /></label>
            <label>{{ $t('unit.field_missile') }}
              <select v-model="edited.StatSec.Missile" @change="markDirty">
                <option v-for="v in projectileTypes" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>{{ $t('unit.field_range') }}<input type="number" v-model.number="edited.StatSec.Range" @input="markDirty" /></label>
            <label>{{ $t('unit.field_ammo') }}<input type="number" v-model.number="edited.StatSec.Ammo" @input="markDirty" /></label>
            <label>{{ $t('unit.field_weapon_type') }}
              <select v-model="edited.StatSec.WeaponType" @change="markDirty">
                <option v-for="v in WEAPON_TYPES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>{{ $t('unit.field_tech_type') }}
              <select v-model="edited.StatSec.TechType" @change="markDirty">
                <option v-for="v in TECH_TYPES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>{{ $t('unit.field_damage_type') }}
              <select v-model="edited.StatSec.DamageType" @change="markDirty">
                <option v-for="v in DAMAGE_TYPES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>{{ $t('unit.field_sound') }}
              <select v-model="edited.StatSec.SoundType" @change="markDirty">
                <option v-for="v in SOUND_TYPES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>{{ $t('unit.field_delay') }}<input type="number" step="0.1" v-model.number="edited.StatSec.MinDelay" @input="markDirty" /></label>
            <label>{{ $t('unit.field_factor') }}<input type="number" step="0.1" v-model.number="edited.StatSec.Factor" @input="markDirty" /></label>
            <div class="full attr-group">
              <div class="attr-label">{{ $t('unit.field_weapon_attributes') }}</div>
              <input type="text" readonly class="attr-display" :value="(edited.StatSecAttr ?? []).filter(a => a !== 'no').join(', ') || '—'" />
              <div class="attr-list">
                <label v-for="attr in WEAPON_ATTRIBUTES" :key="attr.key" class="attr-check">
                  <input type="checkbox" :checked="!!(edited.StatSecAttr?.includes(attr.key))" @change="e => onWeaponAttrChange('StatSecAttr', attr.key, e.target.checked)" />
                  {{ attr.label }}
                </label>
                <label class="attr-check spear-bonus-row">
                  <span>{{ $t('unit.field_spear_bonus') }} <span class="attr-key">(spear_bonus)</span></span>
                  <input type="number" min="0" max="20" step="1" class="spear-bonus-input" :value="getSpearBonus('StatSecAttr')" @change="e => setSpearBonus('StatSecAttr', Number(e.target.value))" />
                </label>
              </div>
            </div>
          </div>
        </section>

        <!-- Броня -->
        <section class="stat-section">
          <h3>{{ $t('unit.section_armour') }}</h3>
          <div class="grid">
            <label>{{ $t('unit.field_armour_primary') }}<input type="number" v-model.number="edited.StatPriArmour.Armour" @input="markDirty" /></label>
            <label>{{ $t('unit.field_def_skill') }}<input type="number" v-model.number="edited.StatPriArmour.DefSkill" @input="markDirty" /></label>
            <label>{{ $t('unit.field_shield') }}<input type="number" v-model.number="edited.StatPriArmour.Shield" @input="markDirty" /></label>
            <label>{{ $t('unit.field_armour_sound') }}
              <select v-model="edited.StatPriArmour.Sound" @change="markDirty">
                <option v-for="v in ARMOUR_SOUNDS" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>{{ $t('unit.field_armour_secondary') }}<input type="number" v-model.number="edited.StatSecArmour.Armour" @input="markDirty" /></label>
            <label>{{ $t('unit.field_def_skill_secondary') }}<input type="number" v-model.number="edited.StatSecArmour.DefSkill" @input="markDirty" /></label>
            <label>{{ $t('unit.field_sound_secondary') }}
              <select v-model="edited.StatSecArmour.Sound" @change="markDirty">
                <option v-for="v in ARMOUR_SOUNDS" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
          </div>
          <!-- Улучшения брони (raw строки) -->
          <div class="grid" style="margin-top: 8px">
            <label class="full">armour_ug_levels<input v-model="edited.ArmourUgLevels" @input="markDirty" class="monospace" /></label>
            <label class="full">armour_ug_models<input v-model="edited.ArmourUgModels" @input="markDirty" class="monospace" /></label>
          </div>
        </section>

        <!-- Прочие статы -->
        <section class="stat-section">
          <h3>{{ $t('unit.section_misc') }}</h3>
          <div class="grid">
            <label>{{ $t('unit.field_heat') }}<input type="number" v-model.number="edited.StatHeat" @input="markDirty" /></label>
            <label>{{ $t('unit.field_scrub') }}<input type="number" v-model.number="edited.StatGround[0]" @input="markDirty" /></label>
            <label>{{ $t('unit.field_sand') }}<input type="number" v-model.number="edited.StatGround[1]" @input="markDirty" /></label>
            <label>{{ $t('unit.field_forest') }}<input type="number" v-model.number="edited.StatGround[2]" @input="markDirty" /></label>
            <label>{{ $t('unit.field_snow') }}<input type="number" v-model.number="edited.StatGround[3]" @input="markDirty" /></label>
            <label>{{ $t('unit.field_morale') }}<input type="number" v-model.number="edited.StatMental.Morale" @input="markDirty" /></label>
            <label>{{ $t('unit.field_discipline') }}
              <select v-model="edited.StatMental.Discipline" @change="markDirty">
                <option v-for="v in DISCIPLINE_VALUES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>{{ $t('unit.field_training') }}
              <select v-model="edited.StatMental.Training" @change="markDirty">
                <option v-for="v in TRAINING_VALUES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>{{ $t('unit.field_charge_dist') }}<input type="number" v-model.number="edited.StatChargeDist" @input="markDirty" /></label>
            <label>{{ $t('unit.field_fire_delay') }}<input type="number" v-model.number="edited.StatFireDelay" @input="markDirty" /></label>
            <label>{{ $t('unit.field_food_primary') }}<input type="number" v-model.number="edited.StatFood[0]" @input="markDirty" /></label>
            <label>{{ $t('unit.field_food_secondary') }}<input type="number" v-model.number="edited.StatFood[1]" @input="markDirty" /></label>
          </div>
        </section>

        <!-- Стоимость -->
        <section class="stat-section">
          <h3>{{ $t('unit.section_cost') }}</h3>
          <div class="grid">
            <label>{{ $t('unit.field_turns') }}<input type="number" v-model.number="edited.StatCost.Turns" @input="markDirty" /></label>
            <label>{{ $t('unit.field_cost') }}<input type="number" v-model.number="edited.StatCost.Cost" @input="markDirty" /></label>
            <label>{{ $t('unit.field_upkeep') }}<input type="number" v-model.number="edited.StatCost.Upkeep" @input="markDirty" /></label>
            <label>{{ $t('unit.field_weapon_upgrade') }}<input type="number" v-model.number="edited.StatCost.WeaponUpgrade" @input="markDirty" /></label>
            <label>{{ $t('unit.field_armour_upgrade') }}<input type="number" v-model.number="edited.StatCost.ArmourUpgrade" @input="markDirty" /></label>
            <label>{{ $t('unit.field_custom') }}<input type="number" v-model.number="edited.StatCost.Custom" @input="markDirty" /></label>
            <label>Extra 1<input type="number" v-model.number="edited.StatCost.Extra1" @input="markDirty" /></label>
            <label>Extra 2<input type="number" v-model.number="edited.StatCost.Extra2" @input="markDirty" /></label>
          </div>
        </section>

        <!-- Найм / Приоритет -->
        <section class="stat-section">
          <h3>{{ $t('unit.section_recruit') }}</h3>
          <div class="grid">
            <label>{{ $t('unit.field_recruit_priority') }}<input type="number" v-model.number="edited.RecruitPriorityOffset" @input="markDirty" /></label>
            <label>info_pic_dir<input v-model="edited.InfoPicDir" @input="markDirty" /></label>
          </div>
        </section>

        <!-- Фракции (ownership) -->
        <section class="stat-section">
          <h3>{{ $t('unit.section_factions_ownership') }}</h3>
          <div class="ownership-chips">
            <span v-for="f in edited.Ownership" :key="f" class="chip">
              {{ f }}
              <button class="chip-remove" @click="removeFaction(f)">✕</button>
            </span>
          </div>
          <select v-if="allFactions.length" class="faction-add-select" @change="e => { addFaction(e.target.value); e.target.value = '' }">
            <option value="">{{ $t('common.add_faction') }}</option>
            <option v-for="f in allFactions.filter(f => !(edited.Ownership ?? []).includes(f.Name))" :key="f.Name" :value="f.Name">
              {{ f.DisplayName || f.Name }}
            </option>
          </select>
        </section>

        <!-- Эры -->
        <section class="stat-section">
          <h3>{{ $t('unit.section_eras') }}</h3>
          <div v-for="eraNum in [0, 1, 2]" :key="eraNum" class="era-block">
            <div class="era-label">{{ $t('unit.era', { n: eraNum }) }}</div>
            <div class="ownership-chips">
              <span v-for="f in eraFactions(eraNum)" :key="f" class="chip chip--era">
                {{ f }}
                <button class="chip-remove" @click="removeEraFaction(eraNum, f)">✕</button>
              </span>
            </div>
            <select v-if="allFactions.length" class="faction-add-select" @change="e => { addEraFaction(eraNum, e.target.value); e.target.value = '' }">
              <option value="">{{ $t('unit.add_to_era', { n: eraNum }) }}</option>
              <option v-for="f in availableEraFactions(eraNum)" :key="f.Name" :value="f.Name">
                {{ f.DisplayName || f.Name }}
              </option>
            </select>
          </div>
        </section>

      </div>
      </div>
    </div>

    <!-- Колонка зданий -->
    <div class="buildings-col">
      <div class="buildings-col-header">
        <span>{{ $t('unit.building_hire_col') }}</span>
        <button class="btn-add-bld" @click="openBldPicker" :title="$t('unit.add_building')">+</button>
      </div>
      <ul class="bld-list">
        <li v-for="loc in unitBuildings" :key="loc.GroupName + '/' + loc.LevelName" class="bld-item">
          <div class="bld-item-name">{{ loc.LevelDisplayName || loc.LevelName }}</div>
          <div class="bld-item-group">{{ loc.GroupDisplayName || loc.GroupName }}</div>
          <button class="bld-item-remove" @click="removeFromBuilding(loc)" :title="$t('unit.remove_from_building')">✕</button>
        </li>
        <li v-if="unitBuildings.length === 0" class="bld-empty">{{ $t('unit.not_in_any_building') }}</li>
      </ul>

      <!-- Пикер зданий -->
      <div v-if="bldPicker" class="bld-picker-overlay" @click.self="bldPicker = false">
        <div class="bld-picker">
          <div class="bld-picker-header">
            <input v-model="bldPickerSearch" class="bld-picker-search" :placeholder="$t('common.search_placeholder')" autofocus />
            <button class="bld-picker-close" @click="bldPicker = false">✕</button>
          </div>
          <div v-if="bldPickerLoading" class="bld-picker-loading">{{ $t('common.loading') }}</div>
          <div v-else class="bld-picker-list">
            <div v-for="grp in filteredBldGroups()" :key="grp.Name" class="bld-picker-grp">
              <div class="bld-picker-grp-name">{{ grp.DisplayName || grp.Name }}</div>
              <button
                v-for="lvl in grp.Levels"
                :key="lvl.Name"
                class="bld-picker-lvl"
                @click="addToBuilding(grp, lvl)"
              >{{ lvl.DisplayName || lvl.Name }}</button>
            </div>
            <div v-if="filteredBldGroups().length === 0" class="bld-picker-empty">{{ $t('unit.no_suitable_buildings') }}</div>
          </div>
        </div>
      </div>
    </div>

  </div>
  <div v-else class="detail-empty">
    <span>{{ $t('unit.select_unit_m2tw') }}</span>
  </div>
</template>

<style scoped>
/* Структура */
.detail-root { display: flex; height: 100%; overflow: hidden; }
.detail-main { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.detail-empty { flex: 1; display: flex; align-items: center; justify-content: center; color: var(--color-text-muted); font-size: 14px; }

/* Шапка */
.detail-header { padding: 12px 16px; border-bottom: 1px solid var(--color-border); flex-shrink: 0; }
.header-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.header-unit-ident { display: flex; align-items: center; gap: 10px; min-width: 0; }
.header-icon-placeholder {
  width: 36px; height: 36px; border-radius: 6px;
  background: var(--color-primary); color: var(--color-text);
  display: flex; align-items: center; justify-content: center;
  font-size: 11px; font-weight: 700; letter-spacing: 0.05em; flex-shrink: 0;
}
.header-unit-name { display: flex; flex-direction: column; min-width: 0; }
.header-name { font-size: 14px; font-weight: 600; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.header-type { font-size: 10px; color: var(--color-text-muted); }
.toolbar { display: flex; align-items: center; gap: 6px; flex-shrink: 0; }
.btn-save { background: var(--color-primary); color: var(--color-text); border: none; border-radius: 4px; padding: 5px 14px; font-size: 12px; cursor: pointer; }
.btn-save:hover { background: var(--color-primary-hover); }
.btn-save:disabled { opacity: 0.5; cursor: default; }
.btn-cancel { background: transparent; border: 1px solid var(--color-border-soft); color: var(--color-text-dim); border-radius: 4px; padding: 5px 12px; font-size: 12px; cursor: pointer; }
.btn-cancel:hover { border-color: var(--color-text-dim); color: var(--color-text-dim); }
.btn-delete { background: none; border: 1px solid var(--color-border-soft); color: var(--color-text-muted); border-radius: 4px; padding: 4px 8px; font-size: 11px; cursor: pointer; }
.btn-delete:hover { border-color: var(--color-accent); color: var(--color-accent); }
.btn-restore { background: none; border: 1px solid #3aba80; color: #3aba80; border-radius: 4px; padding: 4px 8px; font-size: 11px; cursor: pointer; }
.error { font-size: 11px; color: var(--color-error); max-width: 200px; }

/* Scroll */
.detail-scroll { flex: 1; overflow-y: auto; padding: 12px 16px; }
.sections { display: flex; flex-direction: column; gap: 16px; }

/* Секции */
.stat-section { background: var(--color-secondary); border: 1px solid var(--color-border); border-radius: 6px; padding: 12px 14px; }
.stat-section h3 { font-size: 10px; text-transform: uppercase; color: var(--color-heading); letter-spacing: 0.08em; margin-bottom: 10px; }

/* Grid */
.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(160px, 1fr)); gap: 8px; }
label { display: flex; flex-direction: column; gap: 3px; font-size: 10px; color: var(--color-text-muted); }
label input, label select, label textarea {
  background: var(--color-cell); border: 1px solid var(--color-border); border-radius: 3px;
  color: var(--color-text-dim); font-size: 12px; padding: 5px 7px;
}
label input:focus, label select:focus { outline: none; border-color: var(--color-accent); }
label select option { background: var(--color-cell); }
.full { grid-column: 1 / -1; }
.desc-textarea { resize: vertical; min-height: 40px; font-family: inherit; }
.monospace input { font-family: monospace; font-size: 11px; }

/* Attr group */
.attr-group { display: flex; flex-direction: column; gap: 4px; }
.attr-label { font-size: 10px; color: var(--color-text-muted); }
.attr-display { background: var(--color-input-bg); border: 1px solid var(--color-border); border-radius: 3px; color: var(--color-text-dim); font-size: 11px; padding: 4px 7px; width: 100%; box-sizing: border-box; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
.attr-list { display: flex; flex-wrap: wrap; gap: 4px; }
.attr-check { display: flex; align-items: center; gap: 4px; font-size: 11px; color: var(--color-text-dim); cursor: pointer; }
.attr-check input[type=checkbox] { cursor: pointer; }
.attr-divider {
  margin-top: 4px;
  padding-top: 6px;
  border-top: 1px dashed var(--color-border);
  font-size: 9px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-accent);
}
.attr-list-rex { display: grid; grid-template-columns: repeat(2, 1fr); gap: 7px 14px; }
.attr-check-rex { display: flex; align-items: flex-start; gap: 6px; cursor: pointer; }
.attr-check-rex input[type=checkbox] { flex-shrink: 0; margin-top: 1px; cursor: pointer; }
.attr-check-text { display: flex; flex-direction: column; gap: 1px; }
.attr-check-name { font-size: 11px; color: var(--color-text-dim); }
.attr-check-hint { font-size: 9px; color: var(--color-text-muted); font-style: italic; }
.attr-key { font-size: 9px; color: var(--color-text-muted); }
.spear-bonus-row { display: flex; align-items: center; justify-content: space-between; width: 100%; }
.spear-bonus-input { width: 48px; background: var(--color-cell); border: 1px solid var(--color-border); border-radius: 3px; color: var(--color-text-dim); font-size: 11px; padding: 2px 5px; text-align: right; }

/* Formation */
.formation-group { display: flex; flex-direction: column; gap: 6px; }
.formation-pairs { display: flex; flex-wrap: wrap; gap: 8px; align-items: flex-end; }
.formation-pair { display: flex; flex-direction: column; gap: 3px; }
.formation-pair-title { font-size: 9px; color: var(--color-text-muted); }
.formation-pair-inputs { display: flex; gap: 6px; }
.formation-pair-inputs label { width: 56px; }
.formation-single label { width: 72px; }
.formation-select label { width: 100px; }
.formation-select, .formation-single { display: flex; flex-direction: column; gap: 3px; font-size: 10px; color: var(--color-text-muted); }

/* Officers */
.officer-row { display: flex; gap: 4px; margin-bottom: 3px; }
.officer-input { flex: 1; background: var(--color-cell); border: 1px solid var(--color-border); border-radius: 3px; color: var(--color-text-dim); font-size: 12px; padding: 4px 7px; }
.officer-remove { background: none; border: 1px solid var(--color-border-soft); color: var(--color-text-muted); border-radius: 3px; padding: 2px 6px; font-size: 10px; cursor: pointer; }
.officer-remove:hover { border-color: var(--color-accent); color: var(--color-accent); }
.btn-small-add { background: none; border: 1px dashed var(--color-border-soft); color: var(--color-text-muted); border-radius: 3px; padding: 3px 10px; font-size: 11px; cursor: pointer; margin-top: 2px; align-self: flex-start; }
.btn-small-add:hover { border-color: #3aba80; color: #3aba80; }

/* Chips */
.ownership-chips { display: flex; flex-wrap: wrap; gap: 5px; margin-bottom: 6px; }
.chip {
  display: flex; align-items: center; gap: 4px;
  background: var(--color-primary); border: 1px solid var(--color-primary-hover);
  border-radius: 12px; padding: 3px 8px 3px 10px;
  font-size: 11px; color: var(--color-text-dim);
}
.chip--era { background: #1a2a0f; border-color: #2a4a1e; }
.chip-remove { background: none; border: none; color: var(--color-text-muted); cursor: pointer; font-size: 10px; line-height: 1; padding: 0; }
.chip-remove:hover { color: var(--color-accent); }
.faction-add-select {
  background: var(--color-cell); border: 1px solid var(--color-border-soft); border-radius: 4px;
  color: var(--color-text-dim); font-size: 11px; padding: 4px 8px;
}
.faction-add-select:focus { outline: none; border-color: var(--color-accent); }

/* Eras */
.era-block { margin-bottom: 10px; padding-bottom: 10px; border-bottom: 1px solid var(--color-border); }
.era-block:last-child { border-bottom: none; margin-bottom: 0; padding-bottom: 0; }
.era-label { font-size: 10px; color: var(--color-text-dim); text-transform: uppercase; letter-spacing: 0.05em; margin-bottom: 5px; }

/* Buildings column */
.buildings-col {
  width: 220px; flex-shrink: 0; border-left: 1px solid var(--color-border);
  display: flex; flex-direction: column; overflow: hidden;
}
.buildings-col-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 12px; font-size: 11px; color: var(--color-text-dim);
  border-bottom: 1px solid var(--color-border); text-transform: uppercase; letter-spacing: 0.05em;
  flex-shrink: 0;
}
.btn-add-bld {
  background: none; border: 1px solid var(--color-border-soft); color: var(--color-text-muted);
  border-radius: 4px; width: 22px; height: 22px; font-size: 14px; line-height: 1;
  cursor: pointer; display: flex; align-items: center; justify-content: center;
}
.btn-add-bld:hover { border-color: #3aba80; color: #3aba80; }
.bld-list { flex: 1; overflow-y: auto; padding: 8px; list-style: none; }
.bld-item {
  position: relative; padding: 7px 28px 7px 8px;
  border-bottom: 1px solid var(--color-border); cursor: default;
}
.bld-item:last-child { border-bottom: none; }
.bld-item-name { font-size: 12px; color: var(--color-text-dim); }
.bld-item-group { font-size: 10px; color: var(--color-text-muted); margin-top: 1px; }
.bld-item-remove {
  position: absolute; right: 6px; top: 50%; transform: translateY(-50%);
  background: none; border: none; color: var(--color-text-muted); font-size: 10px; cursor: pointer; padding: 2px;
}
.bld-item-remove:hover { color: var(--color-accent); }
.bld-empty { font-size: 11px; color: var(--color-text-muted); padding: 8px; }

/* Bld picker */
.bld-picker-overlay {
  position: fixed; inset: 0; z-index: 100;
  display: flex; align-items: flex-end; justify-content: flex-end;
  padding: 20px;
}
.bld-picker {
  background: var(--color-cell); border: 1px solid var(--color-border-soft); border-radius: 8px;
  width: 300px; max-height: 60vh; display: flex; flex-direction: column;
  box-shadow: 0 8px 32px rgba(0,0,0,0.5);
}
.bld-picker-header { display: flex; gap: 8px; padding: 10px; border-bottom: 1px solid var(--color-border); flex-shrink: 0; }
.bld-picker-search {
  flex: 1; background: var(--color-input-bg); border: 1px solid var(--color-border-soft);
  border-radius: 4px; color: var(--color-text); font-size: 12px; padding: 5px 8px;
}
.bld-picker-search:focus { outline: none; border-color: var(--color-accent); }
.bld-picker-close { background: none; border: none; color: var(--color-text-muted); font-size: 14px; cursor: pointer; }
.bld-picker-close:hover { color: var(--color-accent); }
.bld-picker-loading { padding: 20px; text-align: center; font-size: 12px; color: var(--color-text-muted); }
.bld-picker-list { flex: 1; overflow-y: auto; padding: 8px; }
.bld-picker-grp { margin-bottom: 8px; }
.bld-picker-grp-name { font-size: 9px; text-transform: uppercase; color: var(--color-heading); letter-spacing: 0.05em; margin-bottom: 4px; }
.bld-picker-lvl {
  display: block; width: 100%; text-align: left;
  background: var(--color-secondary); border: 1px solid var(--color-border); border-radius: 3px;
  color: var(--color-text-dim); font-size: 11px; padding: 5px 8px; cursor: pointer; margin-bottom: 2px;
}
.bld-picker-lvl:hover { background: var(--color-primary); border-color: var(--color-primary-hover); }
.bld-picker-empty { font-size: 11px; color: var(--color-text-muted); padding: 8px; }
</style>
