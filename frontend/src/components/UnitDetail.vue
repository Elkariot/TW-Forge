<script setup>
import { ref, watch, onMounted } from 'vue'
import { GetUnitByType, UpdateUnit, GetUnitIcon, GetUnitBuildings, GetProjectileTypes, GetUnitChangeType, RevertUnit, DeleteUnit, HardDeleteUnit, IsCopyUnit, GetBuildings, UpdateBuildingLevel, GetFactions } from '../../wailsjs/go/main/App'
import UnitAssets from './UnitAssets.vue'
import {
  UNIT_CATEGORIES, UNIT_CLASSES, VOICE_TYPES,
  WEAPON_TYPES, TECH_TYPES, DAMAGE_TYPES, SOUND_TYPES, ARMOUR_SOUNDS,
  DISCIPLINE_VALUES, TRAINING_VALUES, UNIT_ATTRIBUTES, WEAPON_ATTRIBUTES,
  FORMATION_PRIMARY, FORMATION_SECONDARY,
} from '../enums.js'

const props = defineProps(['unitType', 'faction'])
const emit = defineEmits(['saved', 'reverted', 'deleted', 'restored'])

const unit = ref(null)
const edited = ref(null)
const originalType = ref(null)
const dirty = ref(false)
const saving = ref(false)
const error = ref(null)
const iconData = ref('')
const unitBuildings = ref([])
const unitChangeType = ref('none')
const projectileTypes = ref(['no'])

// Пикер зданий для найма
const bldPicker = ref(false)
const bldPickerSearch = ref('')
const allBuildings = ref([])
const bldPickerLoading = ref(false)

// Фракции
const allFactions = ref([])

onMounted(async () => {
  const [pt, factions] = await Promise.all([GetProjectileTypes(), GetFactions()])
  projectileTypes.value = pt
  allFactions.value = factions
})

watch(() => props.unitType, async (type) => {
  if (!type) { unit.value = null; edited.value = null; iconData.value = ''; unitBuildings.value = []; unitChangeType.value = 'none'; return }
  error.value = null
  try {
    const [fetchedUnit, fetchedBuildings, fetchedChangeType] = await Promise.all([
      GetUnitByType(type),
      GetUnitBuildings(type),
      GetUnitChangeType(type),
    ])
    if (props.unitType !== type) return
    unit.value = fetchedUnit
    unitBuildings.value = fetchedBuildings
    unitChangeType.value = fetchedChangeType
    edited.value = JSON.parse(JSON.stringify(unit.value))
    if (!edited.value.VoiceType) edited.value.VoiceType = 'Heavy_1'
    if (edited.value.StatPri && !edited.value.StatPri.TechType) edited.value.StatPri.TechType = 'simple'
    if (edited.value.StatSec && !edited.value.StatSec.TechType) edited.value.StatSec.TechType = 'no'
    originalType.value = type
    dirty.value = false
    if (props.faction) {
      GetUnitIcon(type, props.faction).then(icon => {
        if (props.unitType === type) iconData.value = icon
      }).catch(() => { iconData.value = '' })
    } else {
      iconData.value = ''
    }
  } catch (e) {
    if (props.unitType !== type) return
    error.value = `Ошибка загрузки юнита "${type}": ${e}`
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

// Formation helpers — работают с "N, N, N, N, N, primary[, secondary]"
function fParts() {
  return (edited.value?.Formation ?? '').split(',').map(s => s.trim())
}
function getFormationNum(idx) {
  return parseFloat(fParts()[idx]) || 0
}
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

async function save() {
  saving.value = true
  error.value = null
  try {
    await UpdateUnit(originalType.value, edited.value)
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
    const choice = window.confirm(
      `"${type}" — копия юнита.\n\nОК = Полное удаление (убрать из EDU + удалить иконки)\nОтмена = Удалить только из фракции "${props.faction}"`
    )
    try {
      if (choice) {
        await HardDeleteUnit(type)
      } else {
        await DeleteUnit(type, props.faction)
      }
      emit('deleted')
    } catch (e) {
      error.value = String(e)
    }
    return
  }

  const isNew = unitChangeType.value === 'added'
  const msg = isNew
    ? `Удалить созданный юнит "${type}"?`
    : `Удалить юнит "${type}" из фракции "${props.faction}"? Если фракция последняя — юнит будет помечен удалённым.`
  if (!confirm(msg)) return
  try {
    await DeleteUnit(type, props.faction)
    emit('deleted')
  } catch (e) {
    error.value = String(e)
  }
}

async function restoreUnit() {
  try {
    await RevertUnit(originalType.value)
    emit('restored')
  } catch (e) {
    error.value = String(e)
  }
}

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
        const alreadyHas = (lvl.RecruitSlots ?? []).some(s => s.UnitType === unitType)
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
    try { allBuildings.value = await GetBuildings() } finally { bldPickerLoading.value = false }
  }
}

async function addToBuilding(grp, lvl) {
  const newSlot = {
    UnitType: originalType.value,
    Level: 0,
    Requirements: edited.value?.Ownership ?? [],
    Conditions: '',
  }
  const updatedSlots = [...(lvl.RecruitSlots ?? []), newSlot]
  await UpdateBuildingLevel(grp.Name, lvl.Name, updatedSlots)
  lvl.RecruitSlots = updatedSlots
  unitBuildings.value = await GetUnitBuildings(originalType.value)
  bldPicker.value = false
}

async function removeFromBuilding(loc) {
  if (allBuildings.value.length === 0) {
    allBuildings.value = await GetBuildings()
  }
  const grp = allBuildings.value.find(g => g.Name === loc.GroupName)
  if (!grp) return
  const lvl = (grp.Levels ?? []).find(l => l.Name === loc.LevelName)
  if (!lvl) return
  const updatedSlots = (lvl.RecruitSlots ?? []).filter(s => s.UnitType !== originalType.value)
  await UpdateBuildingLevel(loc.GroupName, loc.LevelName, updatedSlots)
  lvl.RecruitSlots = updatedSlots
  unitBuildings.value = await GetUnitBuildings(originalType.value)
}

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
</script>

<template>
  <div v-if="edited" class="detail-root">

    <!-- Основная область со статами -->
    <div class="detail-main">

      <!-- Шапка (не прокручивается) -->
      <div class="detail-header">
        <div class="header-row">
          <!-- Мини-иконка + имя юнита -->
          <div class="header-unit-ident">
            <div class="header-icon-wrap">
              <img v-if="iconData" :src="iconData" class="header-icon-img" />
              <div v-else class="header-icon-placeholder">
                {{ (edited.Name || edited.Type).slice(0, 2).toUpperCase() }}
              </div>
            </div>
            <div class="header-unit-name">
              <span class="header-name">{{ edited.Name || edited.Type }}</span>
              <span class="header-type">{{ edited.Type }}</span>
            </div>
          </div>
          <!-- Кнопки -->
          <div class="toolbar">
            <template v-if="dirty">
              <button class="btn-save" :disabled="saving" @click="save">
                {{ saving ? '...' : 'Сохранить' }}
              </button>
              <button class="btn-cancel" @click="cancel">Отмена</button>
            </template>
            <button
              v-if="unitChangeType === 'deleted'"
              class="btn-restore"
              @click="restoreUnit"
            >↺</button>
            <button
              v-else
              class="btn-delete"
              @click="deleteUnit"
              :title="unitChangeType === 'added' ? 'Удалить копию' : 'Удалить юнит'"
            >✕</button>
            <span v-if="error" class="error">{{ error }}</span>
          </div>
        </div>
      </div> <!-- detail-header -->

      <!-- Прокручиваемое содержимое -->
      <div class="detail-scroll">
      <div class="sections">

        <!-- Описание -->
        <section class="stat-section">
          <h3>Описание</h3>
          <div class="grid">
            <label class="full">Название<input v-model="edited.Name" @input="markDirty" /></label>
            <label class="full">Краткое описание<textarea v-model="edited.DescrShort" @input="markDirty" rows="2" class="desc-textarea" /></label>
            <label class="full">Полное описание<textarea v-model="edited.Descr" @input="markDirty" rows="4" class="desc-textarea" /></label>
          </div>
        </section>

        <!-- Основное -->
        <section class="stat-section">
          <h3>Основное</h3>
          <div class="grid">
            <label>Тип<input v-model="edited.Type" @input="markDirty" /></label>
            <label>Dictionary<input v-model="edited.Dictionary" @input="markDirty" /></label>
            <label>Категория
              <select v-model="edited.Category" @change="markDirty">
                <option v-for="v in UNIT_CATEGORIES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>Класс
              <select v-model="edited.Class" @change="markDirty">
                <option v-for="v in UNIT_CLASSES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>Голос
              <select v-model="edited.VoiceType" @change="markDirty">
                <option v-for="v in VOICE_TYPES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label v-if="edited.Mount || edited.Category === 'cavalry'" class="full">Маунт<input v-model="edited.Mount" @input="markDirty" /></label>
            <label v-if="edited.Mount || edited.MountEffect || edited.Category === 'cavalry'" class="full">Эффект маунта<input v-model="edited.MountEffect" @input="markDirty" /></label>
            <div class="full formation-group">
              <div class="attr-label">Строй</div>
              <div class="formation-pairs">
                <div class="formation-pair">
                  <div class="formation-pair-title">Тесный</div>
                  <div class="formation-pair-inputs">
                    <label>Шир.<input type="number" step="0.1" :value="getFormationNum(0)" @change="e => setFormationNum(0, e.target.value)" /></label>
                    <label>Гл.<input type="number" step="0.1" :value="getFormationNum(1)" @change="e => setFormationNum(1, e.target.value)" /></label>
                  </div>
                </div>
                <div class="formation-pair">
                  <div class="formation-pair-title">Свободный</div>
                  <div class="formation-pair-inputs">
                    <label>Шир.<input type="number" step="0.1" :value="getFormationNum(2)" @change="e => setFormationNum(2, e.target.value)" /></label>
                    <label>Гл.<input type="number" step="0.1" :value="getFormationNum(3)" @change="e => setFormationNum(3, e.target.value)" /></label>
                  </div>
                </div>
                <label class="formation-single">Макс. гл.<input type="number" step="1" :value="getFormationNum(4)" @change="e => setFormationNum(4, e.target.value)" /></label>
                <label class="formation-select">Тип строя
                  <select :value="getFormationPrimary()" @change="e => setFormationPrimary(e.target.value)">
                    <option v-for="v in FORMATION_PRIMARY" :key="v" :value="v">{{ v }}</option>
                  </select>
                </label>
                <label class="formation-select">Доп. строй
                  <select :value="getFormationSecondary()" @change="e => setFormationSecondary(e.target.value)">
                    <option v-for="v in FORMATION_SECONDARY" :key="v" :value="v">{{ v || '—' }}</option>
                  </select>
                </label>
              </div>
            </div>
            <div class="full attr-group">
              <div class="attr-label">Атрибуты</div>
              <input type="text" readonly class="attr-display"
                :value="(edited.Attributes ?? []).join(', ') || '—'"
                placeholder="нет" />
              <div class="attr-list">
                <label v-for="attr in UNIT_ATTRIBUTES" :key="attr.key" class="attr-check">
                  <input type="checkbox" :value="attr.key" v-model="edited.Attributes" @change="markDirty" />
                  {{ attr.label }}
                </label>
              </div>
            </div>
          </div>
        </section>

        <!-- Солдаты -->
        <section class="stat-section">
          <h3>Солдаты</h3>
          <div class="grid">
            <label>Модель<input v-model="edited.Soldier.Model" @input="markDirty" /></label>
            <label>Кол-во<input type="number" v-model.number="edited.Soldier.Count" @input="markDirty" /></label>
            <label>Доп.<input type="number" v-model.number="edited.Soldier.Extras" @input="markDirty" /></label>
            <label>Масса<input type="number" step="0.1" v-model.number="edited.Soldier.Mass" @input="markDirty" /></label>
          </div>
        </section>

        <!-- Здоровье -->
        <section class="stat-section">
          <h3>Здоровье</h3>
          <div class="grid">
            <label>HP<input type="number" v-model.number="edited.StatHealth[0]" @input="markDirty" /></label>
            <label>Бонус HP<input type="number" v-model.number="edited.StatHealth[1]" @input="markDirty" /></label>
          </div>
        </section>

        <!-- Основное оружие -->
        <section class="stat-section">
          <h3>Основное оружие</h3>
          <div class="grid">
            <label>Атака<input type="number" v-model.number="edited.StatPri.Attack" @input="markDirty" /></label>
            <label>Заряд<input type="number" v-model.number="edited.StatPri.ChargeBonus" @input="markDirty" /></label>
            <label>Снаряд
              <select v-model="edited.StatPri.Missile" @change="markDirty">
                <option v-for="v in projectileTypes" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>Дальность<input type="number" v-model.number="edited.StatPri.Range" @input="markDirty" /></label>
            <label>Боезапас<input type="number" v-model.number="edited.StatPri.Ammo" @input="markDirty" /></label>
            <label>Тип оружия
              <select v-model="edited.StatPri.WeaponType" @change="markDirty">
                <option v-for="v in WEAPON_TYPES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>Тех. тип
              <select v-model="edited.StatPri.TechType" @change="markDirty">
                <option v-for="v in TECH_TYPES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>Тип урона
              <select v-model="edited.StatPri.DamageType" @change="markDirty">
                <option v-for="v in DAMAGE_TYPES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>Звук
              <select v-model="edited.StatPri.SoundType" @change="markDirty">
                <option v-for="v in SOUND_TYPES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>Задержка<input type="number" step="0.1" v-model.number="edited.StatPri.MinDelay" @input="markDirty" /></label>
            <label>Фактор<input type="number" step="0.1" v-model.number="edited.StatPri.Factor" @input="markDirty" /></label>
            <div class="full attr-group">
              <div class="attr-label">Атрибуты оружия</div>
              <input type="text" readonly class="attr-display"
                :value="(edited.StatPriAttr ?? []).filter(a => a !== 'no').join(', ') || '—'"
                placeholder="нет" />
              <div class="attr-list">
                <label v-for="attr in WEAPON_ATTRIBUTES" :key="attr.key" class="attr-check">
                  <input type="checkbox"
                    :checked="!!(edited.StatPriAttr?.includes(attr.key))"
                    @change="e => onWeaponAttrChange('StatPriAttr', attr.key, e.target.checked)"
                  />
                  {{ attr.label }}
                </label>
                <label class="attr-check spear-bonus-row" title="+N к атаке против кавалерии. Типовые значения: 4 — ополч. копьё, 8 — гоплиты/копейщики, 12 — тяжёлые пикинёры">
                  <span>Бонус vs конницы <span class="attr-key">(spear_bonus)</span></span>
                  <input type="number" min="0" max="20" step="1" class="spear-bonus-input"
                    :value="getSpearBonus('StatPriAttr')"
                    @change="e => setSpearBonus('StatPriAttr', Number(e.target.value))"
                  />
                </label>
              </div>
            </div>
          </div>
        </section>

        <!-- Вторичное оружие -->
        <section class="stat-section">
          <h3>Вторичное оружие</h3>
          <div class="grid">
            <label>Атака<input type="number" v-model.number="edited.StatSec.Attack" @input="markDirty" /></label>
            <label>Заряд<input type="number" v-model.number="edited.StatSec.ChargeBonus" @input="markDirty" /></label>
            <label>Снаряд
              <select v-model="edited.StatSec.Missile" @change="markDirty">
                <option v-for="v in projectileTypes" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>Дальность<input type="number" v-model.number="edited.StatSec.Range" @input="markDirty" /></label>
            <label>Боезапас<input type="number" v-model.number="edited.StatSec.Ammo" @input="markDirty" /></label>
            <label>Тип оружия
              <select v-model="edited.StatSec.WeaponType" @change="markDirty">
                <option v-for="v in WEAPON_TYPES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>Тех. тип
              <select v-model="edited.StatSec.TechType" @change="markDirty">
                <option v-for="v in TECH_TYPES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>Тип урона
              <select v-model="edited.StatSec.DamageType" @change="markDirty">
                <option v-for="v in DAMAGE_TYPES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>Звук
              <select v-model="edited.StatSec.SoundType" @change="markDirty">
                <option v-for="v in SOUND_TYPES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>Задержка<input type="number" step="0.1" v-model.number="edited.StatSec.MinDelay" @input="markDirty" /></label>
            <label>Фактор<input type="number" step="0.1" v-model.number="edited.StatSec.Factor" @input="markDirty" /></label>
            <div class="full attr-group">
              <div class="attr-label">Атрибуты оружия</div>
              <input type="text" readonly class="attr-display"
                :value="(edited.StatSecAttr ?? []).filter(a => a !== 'no').join(', ') || '—'"
                placeholder="нет" />
              <div class="attr-list">
                <label v-for="attr in WEAPON_ATTRIBUTES" :key="attr.key" class="attr-check">
                  <input type="checkbox"
                    :checked="!!(edited.StatSecAttr?.includes(attr.key))"
                    @change="e => onWeaponAttrChange('StatSecAttr', attr.key, e.target.checked)"
                  />
                  {{ attr.label }}
                </label>
                <label class="attr-check spear-bonus-row" title="+N к атаке против кавалерии. Типовые значения: 4 — ополч. копьё, 8 — гоплиты/копейщики, 12 — тяжёлые пикинёры">
                  <span>Бонус vs конницы <span class="attr-key">(spear_bonus)</span></span>
                  <input type="number" min="0" max="20" step="1" class="spear-bonus-input"
                    :value="getSpearBonus('StatSecAttr')"
                    @change="e => setSpearBonus('StatSecAttr', Number(e.target.value))"
                  />
                </label>
              </div>
            </div>
          </div>
        </section>

        <!-- Броня -->
        <section class="stat-section">
          <h3>Броня</h3>
          <div class="grid">
            <label>Броня (осн.)<input type="number" v-model.number="edited.StatPriArmour.Armour" @input="markDirty" /></label>
            <label>Защита<input type="number" v-model.number="edited.StatPriArmour.DefSkill" @input="markDirty" /></label>
            <label>Щит<input type="number" v-model.number="edited.StatPriArmour.Shield" @input="markDirty" /></label>
            <label>Звук брони
              <select v-model="edited.StatPriArmour.Sound" @change="markDirty">
                <option v-for="v in ARMOUR_SOUNDS" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>Броня (доп.)<input type="number" v-model.number="edited.StatSecArmour.Armour" @input="markDirty" /></label>
            <label>Защита (доп.)<input type="number" v-model.number="edited.StatSecArmour.DefSkill" @input="markDirty" /></label>
            <label>Звук (доп.)
              <select v-model="edited.StatSecArmour.Sound" @change="markDirty">
                <option v-for="v in ARMOUR_SOUNDS" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
          </div>
        </section>

        <!-- Прочие статы -->
        <section class="stat-section">
          <h3>Прочее</h3>
          <div class="grid">
            <label>Жара<input type="number" v-model.number="edited.StatHeat" @input="markDirty" /></label>
            <label>Кустарник<input type="number" v-model.number="edited.StatGround[0]" @input="markDirty" /></label>
            <label>Песок<input type="number" v-model.number="edited.StatGround[1]" @input="markDirty" /></label>
            <label>Лес<input type="number" v-model.number="edited.StatGround[2]" @input="markDirty" /></label>
            <label>Снег<input type="number" v-model.number="edited.StatGround[3]" @input="markDirty" /></label>
            <label>Мораль<input type="number" v-model.number="edited.StatMental.Morale" @input="markDirty" /></label>
            <label>Дисциплина
              <select v-model="edited.StatMental.Discipline" @change="markDirty">
                <option v-for="v in DISCIPLINE_VALUES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>Тренировка
              <select v-model="edited.StatMental.Training" @change="markDirty">
                <option v-for="v in TRAINING_VALUES" :key="v" :value="v">{{ v }}</option>
              </select>
            </label>
            <label>Дист. заряда<input type="number" v-model.number="edited.StatChargeDist" @input="markDirty" /></label>
            <label>Задержка огня<input type="number" v-model.number="edited.StatFireDelay" @input="markDirty" /></label>
            <label>Еда (осн.)<input type="number" v-model.number="edited.StatFood[0]" @input="markDirty" /></label>
            <label>Еда (доп.)<input type="number" v-model.number="edited.StatFood[1]" @input="markDirty" /></label>
          </div>
        </section>

        <!-- Стоимость -->
        <section class="stat-section">
          <h3>Стоимость</h3>
          <div class="grid">
            <label>Ходов<input type="number" v-model.number="edited.StatCost.Turns" @input="markDirty" /></label>
            <label>Цена<input type="number" v-model.number="edited.StatCost.Cost" @input="markDirty" /></label>
            <label>Содержание<input type="number" v-model.number="edited.StatCost.Upkeep" @input="markDirty" /></label>
            <label>Улучш. оружие<input type="number" v-model.number="edited.StatCost.WeaponUpgrade" @input="markDirty" /></label>
            <label>Улучш. броня<input type="number" v-model.number="edited.StatCost.ArmourUpgrade" @input="markDirty" /></label>
            <label>Кастом<input type="number" v-model.number="edited.StatCost.Custom" @input="markDirty" /></label>
          </div>
        </section>

        <!-- Фракции -->
        <section class="stat-section">
          <h3>Фракции</h3>
          <div class="ownership-chips">
            <span v-for="f in edited.Ownership" :key="f" class="chip">
              {{ f }}
              <button class="chip-remove" @click="removeFaction(f)" title="Убрать">✕</button>
            </span>
          </div>
          <select
            v-if="allFactions.length"
            class="faction-add-select"
            @change="e => { addFaction(e.target.value); e.target.value = '' }"
          >
            <option value="">+ Добавить фракцию</option>
            <option
              v-for="f in allFactions.filter(f => !(edited.Ownership ?? []).includes(f.Name))"
              :key="f.Name"
              :value="f.Name"
            >{{ f.DisplayName || f.Name }}</option>
          </select>
        </section>

      </div>
      </div> <!-- detail-scroll -->
    </div>

    <!-- Колонка зданий -->
    <div class="buildings-col">
      <h3>Здания для найма</h3>

      <template v-if="!bldPicker">
        <div v-if="unitBuildings.length === 0" class="buildings-empty">не производится</div>
        <template v-else>
          <div v-for="loc in unitBuildings" :key="loc.GroupName + loc.LevelName" class="building-entry">
            <div class="building-level">{{ loc.LevelName }}</div>
            <button
              v-if="unitChangeType === 'added'"
              class="bld-remove-btn"
              @click="removeFromBuilding(loc)"
              title="Убрать из этого здания"
            >✕</button>
          </div>
        </template>
        <button
          v-if="unitChangeType === 'added'"
          class="bld-add-btn"
          @click="openBldPicker"
        >+ Добавить здание</button>
      </template>

      <!-- Пикер здания -->
      <template v-else>
        <input
          v-model="bldPickerSearch"
          class="bld-search"
          placeholder="Поиск..."
          autofocus
        />
        <div class="bld-picker-list">
          <div v-if="bldPickerLoading" class="bld-picker-empty">Загрузка...</div>
          <template v-else-if="filteredBldGroups().length">
            <div v-for="grp in filteredBldGroups()" :key="grp.Name">
              <div class="bld-picker-group">{{ grp.DisplayName || grp.Name }}</div>
              <button
                v-for="lvl in grp.Levels"
                :key="lvl.Name"
                class="bld-picker-lvl"
                @click="addToBuilding(grp, lvl)"
              >{{ bldName(grp, lvl) }}</button>
            </div>
          </template>
          <div v-else class="bld-picker-empty">Нет доступных зданий</div>
        </div>
        <button class="bld-cancel-btn" @click="bldPicker = false">Отмена</button>
      </template>

    </div>

    <!-- Иконка / Модели / Текстуры -->
    <div class="assets-col">
      <h3>Медиафайлы</h3>
      <UnitAssets :unitType="props.unitType" :faction="props.faction" />
    </div>

  </div>

  <div v-else class="detail-empty">
    <span v-if="error" class="load-error">{{ error }}</span>
    <span v-else>Выберите юнита</span>
  </div>
</template>

<style scoped>
.detail-root {
  display: flex;
  height: 100%;
  gap: 0;
  overflow: hidden;
}

.detail-main {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.detail-header {
  flex-shrink: 0;
  padding: 8px 14px;
  background: var(--color-bg);
  border-bottom: 1px solid var(--color-border);
}

.header-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.header-unit-ident {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.header-icon-wrap { flex-shrink: 0; }
.header-icon-img {
  width: 36px;
  height: 36px;
  object-fit: contain;
  border-radius: 3px;
  background: var(--color-input-bg);
}
.header-icon-placeholder {
  width: 36px;
  height: 36px;
  border-radius: 3px;
  background: var(--color-input-bg);
  border: 1px dashed var(--color-border-soft);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 700;
  color: var(--color-border-soft);
}

.header-unit-name {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}
.header-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.header-type {
  font-size: 10px;
  color: var(--color-text-muted);
  font-family: monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.detail-scroll {
  flex: 1;
  overflow-y: auto;
  padding: 12px 20px 20px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* Описание (в прокручиваемой области) */
.desc-card {
  background: var(--color-primary);
  border-left: 3px solid var(--color-accent);
  border-radius: 6px;
  padding: 10px 14px;
}
.desc-header { margin-bottom: 4px; }
.desc-texts { min-width: 0; }
.desc-short {
  font-size: 12px;
  color: var(--color-text-dim);
  font-style: italic;
  white-space: pre-line;
}
.desc-full {
  font-size: 11px;
  color: var(--color-text-dim);
  line-height: 1.5;
  white-space: pre-line;
  margin-top: 4px;
}

/* Toolbar */
.toolbar {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}
.btn-save {
  background: var(--color-primary);
  color: var(--color-text);
  border: none;
  padding: 5px 12px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  white-space: nowrap;
}
.btn-save:disabled { opacity: 0.6; cursor: default; }
.btn-cancel {
  background: transparent;
  color: var(--color-text-dim);
  border: 1px solid #444;
  padding: 5px 10px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
}
.btn-cancel:hover { border-color: var(--color-text-dim); }
.btn-delete {
  background: transparent;
  color: var(--color-accent);
  border: 1px solid var(--color-accent);
  padding: 5px 9px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
}
.btn-delete:hover { background: rgba(149,232,225,0.2); }
.btn-restore {
  background: transparent;
  color: #3aba80;
  border: 1px solid #3aba80;
  padding: 5px 9px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
}
.btn-restore:hover { background: rgba(58,186,128,0.15); }
.error { color: var(--color-error); font-size: 11px; max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

/* Секции */
.sections { display: flex; flex-direction: column; gap: 12px; }

.stat-section {
  background: var(--color-cell);
  border-radius: 6px;
  padding: 12px 14px;
}
.stat-section h3 {
  font-size: 11px;
  text-transform: uppercase;
  color: var(--color-heading);
  margin-bottom: 10px;
  letter-spacing: 0.05em;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 8px;
}
.grid label {
  display: flex;
  flex-direction: column;
  gap: 3px;
  font-size: 11px;
  color: var(--color-text-dim);
}
.grid .full { grid-column: 1 / -1; }
.grid input,
.grid select {
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-soft);
  border-radius: 3px;
  color: var(--color-text);
  font-size: 12px;
  padding: 4px 6px;
  width: 100%;
}
.grid input:focus,
.grid select:focus,
.grid textarea:focus { outline: none; border-color: var(--color-accent); }
.desc-textarea {
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-soft);
  border-radius: 3px;
  color: var(--color-text);
  font-size: 12px;
  padding: 4px 6px;
  width: 100%;
  resize: vertical;
  font-family: inherit;
  line-height: 1.5;
}
.grid select { cursor: pointer; appearance: none; background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6'%3E%3Cpath d='M0 0l5 6 5-6z' fill='%233a2035'/%3E%3C/svg%3E"); background-repeat: no-repeat; background-position: right 6px center; padding-right: 20px; }

/* Атрибуты (чекбоксы) */
.attr-group { display: flex; flex-direction: column; gap: 6px; }
.attr-label { font-size: 11px; color: var(--color-text-dim); }
.attr-display {
  background: var(--color-input-bg);
  border: 1px solid var(--color-border);
  border-radius: 3px;
  color: var(--color-text-muted);
  font-size: 11px;
  font-family: monospace;
  padding: 4px 7px;
  width: 100%;
  cursor: default;
  box-sizing: border-box;
}
.attr-list {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 5px 10px;
}
.attr-check {
  display: flex;
  flex-direction: row !important;
  align-items: center;
  gap: 5px !important;
  font-size: 11px;
  color: var(--color-text-dim);
  cursor: pointer;
}
.attr-check input[type="checkbox"] {
  flex-shrink: 0;
  width: auto;
  margin: 0;
  accent-color: var(--color-accent);
  cursor: pointer;
}
.attr-key { color: var(--color-text-muted); font-size: 10px; }
.spear-bonus-row { display: flex; align-items: center; justify-content: space-between; gap: 8px; cursor: help; }
.spear-bonus-input { width: 52px !important; padding: 2px 4px !important; text-align: center; }

/* Строй */
.formation-group { display: flex; flex-direction: column; gap: 8px; }
.formation-pairs { display: flex; gap: 10px; align-items: flex-end; flex-wrap: wrap; }
.formation-pair {
  border: 1px solid var(--color-border-soft);
  border-radius: 5px;
  padding: 5px 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.formation-pair-title { font-size: 10px; color: var(--color-text-muted); text-align: center; }
.formation-pair-inputs { display: flex; gap: 6px; }
.formation-pair-inputs label,
.formation-single,
.formation-select {
  font-size: 11px;
  color: var(--color-text-dim);
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.formation-pair-inputs label { width: 52px; }
.formation-single { width: 68px; }
.formation-pair-inputs input,
.formation-single input {
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-soft);
  border-radius: 3px;
  color: var(--color-text);
  font-size: 12px;
  padding: 4px 6px;
  width: 100%;
}
.formation-select select {
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-soft);
  border-radius: 3px;
  color: var(--color-text);
  font-size: 12px;
  padding: 4px 20px 4px 6px;
  cursor: pointer;
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6'%3E%3Cpath d='M0 0l5 6 5-6z' fill='%233a2035'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 6px center;
  min-width: 90px;
}

/* Фракции */
.ownership-chips { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 8px; }
.chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: var(--color-primary);
  border-radius: 12px;
  padding: 3px 8px 3px 10px;
  font-size: 11px;
  color: var(--color-text-dim);
}
.chip-remove {
  background: none;
  border: none;
  color: var(--color-text-muted);
  cursor: pointer;
  font-size: 10px;
  padding: 0;
  line-height: 1;
}
.chip-remove:hover { color: var(--color-accent); }
.faction-add-select {
  background: var(--color-input-bg);
  border: 1px dashed var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text-muted);
  font-size: 11px;
  padding: 4px 6px;
  cursor: pointer;
  width: 100%;
}
.faction-add-select:focus { outline: none; border-color: var(--color-accent); color: var(--color-text); }

/* Колонка зданий */
.buildings-col {
  width: 200px;
  flex-shrink: 0;
  border-left: 1px solid var(--color-border);
  padding: 20px 14px;
  overflow-y: auto;
}
.buildings-col h3 {
  font-size: 11px;
  text-transform: uppercase;
  color: var(--color-accent);
  margin-bottom: 12px;
  letter-spacing: 0.05em;
}
.buildings-empty {
  font-size: 12px;
  color: var(--color-text-muted);
  text-align: center;
  margin-top: 40px;
}
.building-entry {
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--color-cell);
  border-radius: 4px;
  padding: 6px 8px;
  margin-bottom: 6px;
}
.building-level {
  flex: 1;
  font-size: 12px;
  color: var(--color-text);
}
.bld-remove-btn {
  flex-shrink: 0;
  background: none;
  border: none;
  color: var(--color-text-dim);
  cursor: pointer;
  font-size: 11px;
  padding: 0 2px;
  line-height: 1;
}
.bld-remove-btn:hover { color: var(--color-accent); }

.bld-add-btn {
  width: 100%;
  margin-top: 10px;
  background: transparent;
  border: 1px dashed var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text-muted);
  font-size: 12px;
  padding: 6px 0;
  cursor: pointer;
}
.bld-add-btn:hover { border-color: var(--color-accent); color: var(--color-accent); }

.bld-search {
  width: 100%;
  box-sizing: border-box;
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text);
  font-size: 12px;
  padding: 5px 8px;
  margin-bottom: 8px;
}
.bld-search:focus { outline: none; border-color: var(--color-accent); }

.bld-picker-list {
  flex: 1;
  overflow-y: auto;
  max-height: 340px;
}
.bld-picker-group {
  font-size: 10px;
  text-transform: uppercase;
  color: var(--color-accent);
  letter-spacing: 0.06em;
  margin: 8px 0 4px;
}
.bld-picker-lvl {
  display: block;
  width: 100%;
  text-align: left;
  background: var(--color-cell);
  border: 1px solid var(--color-border);
  border-radius: 3px;
  color: var(--color-text-dim);
  font-size: 12px;
  padding: 5px 8px;
  margin-bottom: 3px;
  cursor: pointer;
}
.bld-picker-lvl:hover { border-color: var(--color-accent); color: var(--color-text); }
.bld-picker-empty { font-size: 12px; color: var(--color-text-muted); text-align: center; margin-top: 20px; }

.bld-cancel-btn {
  width: 100%;
  margin-top: 8px;
  background: transparent;
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text-muted);
  font-size: 12px;
  padding: 5px 0;
  cursor: pointer;
}
.bld-cancel-btn:hover { color: var(--color-text-dim); }

/* Колонка медиафайлов */
.assets-col {
  width: 220px;
  flex-shrink: 0;
  border-left: 1px solid var(--color-border);
  padding: 20px 14px;
  overflow-y: auto;
}
.assets-col h3 {
  font-size: 11px;
  text-transform: uppercase;
  color: var(--color-accent);
  margin-bottom: 4px;
  letter-spacing: 0.05em;
}

.detail-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--color-text-muted);
  font-size: 14px;
  padding: 20px;
  text-align: center;
}
.load-error {
  color: var(--color-error);
  font-size: 13px;
  max-width: 500px;
  word-break: break-all;
}
</style>
