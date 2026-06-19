<script setup>
import { ref, watch, onMounted } from 'vue'
import { GetUnitByType, UpdateUnit, GetUnitIcon, GetUnitBuildings, GetProjectileTypes } from '../../wailsjs/go/main/App'
import {
  UNIT_CATEGORIES, UNIT_CLASSES, VOICE_TYPES,
  WEAPON_TYPES, TECH_TYPES, DAMAGE_TYPES, SOUND_TYPES, ARMOUR_SOUNDS,
  DISCIPLINE_VALUES, TRAINING_VALUES, UNIT_ATTRIBUTES, WEAPON_ATTRIBUTES,
  FORMATION_PRIMARY, FORMATION_SECONDARY,
} from '../enums.js'

const props = defineProps(['unitType', 'faction'])
const emit = defineEmits(['saved', 'reverted'])

const unit = ref(null)
const edited = ref(null)
const originalType = ref(null)
const dirty = ref(false)
const saving = ref(false)
const error = ref(null)
const iconData = ref('')
const unitBuildings = ref([])
const projectileTypes = ref(['no'])

onMounted(async () => {
  projectileTypes.value = await GetProjectileTypes()
})

watch(() => props.unitType, async (type) => {
  if (!type) { unit.value = null; iconData.value = ''; unitBuildings.value = []; return }
  error.value = null
  try {
    ;[unit.value, iconData.value, unitBuildings.value] = await Promise.all([
      GetUnitByType(type),
      props.faction ? GetUnitIcon(type, props.faction) : Promise.resolve(''),
      GetUnitBuildings(type),
    ])
    edited.value = JSON.parse(JSON.stringify(unit.value))
    originalType.value = type
    dirty.value = false
  } catch (e) {
    error.value = `Ошибка загрузки юнита "${type}": ${e}`
    unit.value = null
    edited.value = null
  }
})

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
  originalType.value = unit.value.Type
  dirty.value = false
  error.value = null
  emit('reverted')
}
</script>

<template>
  <div v-if="edited" class="detail-root">

    <!-- Основная область со статами -->
    <div class="detail-main">

      <!-- Описание (из export_units) -->
      <div class="desc-card">
        <div class="desc-header">
          <div class="unit-icon-wrap">
            <img v-if="iconData" :src="iconData" class="unit-icon" />
            <div v-else class="unit-icon-placeholder">
              <span>{{ (edited.Name || edited.Type).slice(0, 2).toUpperCase() }}</span>
            </div>
          </div>
          <div class="desc-texts">
            <div class="desc-name">{{ edited.Name || edited.Type }}</div>
            <div v-if="edited.DescrShort" class="desc-short">{{ edited.DescrShort.replace(/\\n/g, '\n') }}</div>
          </div>
        </div>
        <div v-if="edited.Descr" class="desc-full">{{ edited.Descr.replace(/\\n/g, '\n') }}</div>
      </div>

      <!-- Кнопки -->
      <div class="toolbar">
        <template v-if="dirty">
          <button class="btn-save" :disabled="saving" @click="save">
            {{ saving ? 'Сохранение...' : 'Сохранить' }}
          </button>
          <button class="btn-cancel" @click="cancel">Отмена</button>
        </template>
        <span v-if="error" class="error">{{ error }}</span>
      </div>

      <div class="sections">

        <!-- Основное -->
        <section class="stat-section">
          <h3>Основное</h3>
          <div class="grid">
            <label>Тип<input v-model="edited.Type" @input="markDirty" /></label>
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
            <span v-for="f in edited.Ownership" :key="f" class="chip">{{ f }}</span>
          </div>
        </section>

      </div>
    </div>

    <!-- Колонка зданий -->
    <div class="buildings-col">
      <h3>Здания для найма</h3>
      <div v-if="unitBuildings.length === 0" class="buildings-empty">не производится</div>
      <template v-else>
        <div v-for="loc in unitBuildings" :key="loc.GroupName + loc.LevelName" class="building-entry">
          <div class="building-icon-placeholder"></div>
          <div class="building-level">{{ loc.LevelName }}</div>
        </div>
      </template>
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
  overflow-y: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* Описание */
.desc-card {
  background: #0f3460;
  border-left: 3px solid #e94560;
  border-radius: 6px;
  padding: 14px 16px;
}
.desc-header {
  display: flex;
  gap: 14px;
  align-items: flex-start;
  margin-bottom: 8px;
}
.unit-icon-wrap { flex-shrink: 0; }
.unit-icon {
  width: 80px;
  height: 80px;
  object-fit: contain;
  border-radius: 4px;
  background: #0a1628;
}
.unit-icon-placeholder {
  width: 80px;
  height: 80px;
  border-radius: 4px;
  background: #0a1628;
  border: 1px dashed #2a3a5e;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: 700;
  color: #2a3a5e;
  letter-spacing: 1px;
}
.desc-texts { flex: 1; min-width: 0; }
.desc-name {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 6px;
}
.desc-short {
  font-size: 13px;
  color: #aac;
  font-style: italic;
  white-space: pre-line;
}
.desc-full {
  font-size: 12px;
  color: #889;
  line-height: 1.5;
  white-space: pre-line;
  margin-top: 4px;
}

/* Toolbar */
.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 30px;
}
.btn-save {
  background: #e94560;
  color: #fff;
  border: none;
  padding: 6px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
}
.btn-save:disabled { opacity: 0.6; cursor: default; }
.btn-cancel {
  background: transparent;
  color: #888;
  border: 1px solid #444;
  padding: 6px 14px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
}
.btn-cancel:hover { border-color: #888; }
.error { color: #e94560; font-size: 12px; }

/* Секции */
.sections { display: flex; flex-direction: column; gap: 12px; }

.stat-section {
  background: #16213e;
  border-radius: 6px;
  padding: 12px 14px;
}
.stat-section h3 {
  font-size: 11px;
  text-transform: uppercase;
  color: #e94560;
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
  color: #888;
}
.grid .full { grid-column: 1 / -1; }
.grid input,
.grid select {
  background: #0f1b35;
  border: 1px solid #2a3a5e;
  border-radius: 3px;
  color: #e0e0e0;
  font-size: 12px;
  padding: 4px 6px;
  width: 100%;
}
.grid input:focus,
.grid select:focus { outline: none; border-color: #e94560; }
.grid select { cursor: pointer; appearance: none; background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6'%3E%3Cpath d='M0 0l5 6 5-6z' fill='%23888'/%3E%3C/svg%3E"); background-repeat: no-repeat; background-position: right 6px center; padding-right: 20px; }

/* Атрибуты (чекбоксы) */
.attr-group { display: flex; flex-direction: column; gap: 6px; }
.attr-label { font-size: 11px; color: #888; }
.attr-display {
  background: #080f20;
  border: 1px solid #1e2e50;
  border-radius: 3px;
  color: #666;
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
  color: #ccc;
  cursor: pointer;
}
.attr-check input[type="checkbox"] {
  flex-shrink: 0;
  width: auto;
  margin: 0;
  accent-color: #e94560;
  cursor: pointer;
}
.attr-key { color: #555; font-size: 10px; }
.spear-bonus-row { display: flex; align-items: center; justify-content: space-between; gap: 8px; cursor: help; }
.spear-bonus-input { width: 52px !important; padding: 2px 4px !important; text-align: center; }

/* Строй */
.formation-group { display: flex; flex-direction: column; gap: 8px; }
.formation-pairs { display: flex; gap: 10px; align-items: flex-end; flex-wrap: wrap; }
.formation-pair {
  border: 1px solid #2a3a5e;
  border-radius: 5px;
  padding: 5px 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.formation-pair-title { font-size: 10px; color: #555; text-align: center; }
.formation-pair-inputs { display: flex; gap: 6px; }
.formation-pair-inputs label,
.formation-single,
.formation-select {
  font-size: 11px;
  color: #888;
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.formation-pair-inputs label { width: 52px; }
.formation-single { width: 68px; }
.formation-pair-inputs input,
.formation-single input {
  background: #0f1b35;
  border: 1px solid #2a3a5e;
  border-radius: 3px;
  color: #e0e0e0;
  font-size: 12px;
  padding: 4px 6px;
  width: 100%;
}
.formation-select select {
  background: #0f1b35;
  border: 1px solid #2a3a5e;
  border-radius: 3px;
  color: #e0e0e0;
  font-size: 12px;
  padding: 4px 20px 4px 6px;
  cursor: pointer;
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6'%3E%3Cpath d='M0 0l5 6 5-6z' fill='%23888'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 6px center;
  min-width: 90px;
}

/* Фракции */
.ownership-chips { display: flex; flex-wrap: wrap; gap: 6px; }
.chip {
  background: #0f3460;
  border-radius: 12px;
  padding: 3px 10px;
  font-size: 11px;
  color: #aac;
}

/* Колонка зданий */
.buildings-col {
  width: 200px;
  flex-shrink: 0;
  border-left: 1px solid #1e2e50;
  padding: 20px 14px;
  overflow-y: auto;
}
.buildings-col h3 {
  font-size: 11px;
  text-transform: uppercase;
  color: #e94560;
  margin-bottom: 12px;
  letter-spacing: 0.05em;
}
.buildings-empty {
  font-size: 12px;
  color: #444;
  text-align: center;
  margin-top: 40px;
}
.building-entry {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #16213e;
  border-radius: 4px;
  padding: 6px 10px;
  margin-bottom: 6px;
}
.building-icon-placeholder {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  border-radius: 3px;
  background: #0a1628;
  border: 1px dashed #2a3a5e;
}
.building-level {
  font-size: 12px;
  color: #e0e0e0;
}

.detail-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #444;
  font-size: 14px;
  padding: 20px;
  text-align: center;
}
.load-error {
  color: #e94560;
  font-size: 13px;
  max-width: 500px;
  word-break: break-all;
}
</style>
