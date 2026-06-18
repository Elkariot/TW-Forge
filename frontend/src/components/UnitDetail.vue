<script setup>
import { ref, watch, computed } from 'vue'
import { GetUnitByType, UpdateUnit, GetUnitIcon, GetUnitBuildings } from '../../wailsjs/go/main/App'

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

watch(() => props.unitType, async (type) => {
  if (!type) { unit.value = null; iconData.value = ''; unitBuildings.value = []; return }
  ;[unit.value, iconData.value, unitBuildings.value] = await Promise.all([
    GetUnitByType(type),
    props.faction ? GetUnitIcon(type, props.faction) : Promise.resolve(''),
    GetUnitBuildings(type),
  ])
  edited.value = JSON.parse(JSON.stringify(unit.value))
  originalType.value = type
  dirty.value = false
  error.value = null
})

function markDirty() { dirty.value = true }

const priAttrStr = computed({
  get: () => edited.value?.StatPriAttr?.join(', ') ?? '',
  set: (v) => { edited.value.StatPriAttr = v.split(',').map(s => s.trim()).filter(Boolean); markDirty() }
})

const secAttrStr = computed({
  get: () => edited.value?.StatSecAttr?.join(', ') ?? '',
  set: (v) => { edited.value.StatSecAttr = v.split(',').map(s => s.trim()).filter(Boolean); markDirty() }
})

const attributesStr = computed({
  get: () => edited.value?.Attributes?.join(', ') ?? '',
  set: (v) => { edited.value.Attributes = v.split(',').map(s => s.trim()).filter(Boolean); markDirty() }
})

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
            <label>Категория<input v-model="edited.Category" @input="markDirty" /></label>
            <label>Класс<input v-model="edited.Class" @input="markDirty" /></label>
            <label>Голос<input v-model="edited.VoiceType" @input="markDirty" /></label>
            <label class="full">Атрибуты<input v-model="attributesStr" @input="markDirty" /></label>
            <label v-if="edited.Mount !== undefined" class="full">Маунт<input v-model="edited.Mount" @input="markDirty" /></label>
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
            <label>Снаряд<input v-model="edited.StatPri.Missile" @input="markDirty" /></label>
            <label>Дальность<input type="number" v-model.number="edited.StatPri.Range" @input="markDirty" /></label>
            <label>Боезапас<input type="number" v-model.number="edited.StatPri.Ammo" @input="markDirty" /></label>
            <label>Тип оружия<input v-model="edited.StatPri.WeaponType" @input="markDirty" /></label>
            <label>Тех. тип<input v-model="edited.StatPri.TechType" @input="markDirty" /></label>
            <label>Тип урона<input v-model="edited.StatPri.DamageType" @input="markDirty" /></label>
            <label>Звук<input v-model="edited.StatPri.SoundType" @input="markDirty" /></label>
            <label>Задержка<input type="number" step="0.1" v-model.number="edited.StatPri.MinDelay" @input="markDirty" /></label>
            <label>Фактор<input type="number" step="0.1" v-model.number="edited.StatPri.Factor" @input="markDirty" /></label>
            <label class="full">Атрибуты<input v-model="priAttrStr" /></label>
          </div>
        </section>

        <!-- Вторичное оружие -->
        <section class="stat-section">
          <h3>Вторичное оружие</h3>
          <div class="grid">
            <label>Атака<input type="number" v-model.number="edited.StatSec.Attack" @input="markDirty" /></label>
            <label>Заряд<input type="number" v-model.number="edited.StatSec.ChargeBonus" @input="markDirty" /></label>
            <label>Снаряд<input v-model="edited.StatSec.Missile" @input="markDirty" /></label>
            <label>Дальность<input type="number" v-model.number="edited.StatSec.Range" @input="markDirty" /></label>
            <label>Боезапас<input type="number" v-model.number="edited.StatSec.Ammo" @input="markDirty" /></label>
            <label>Тип оружия<input v-model="edited.StatSec.WeaponType" @input="markDirty" /></label>
            <label>Тех. тип<input v-model="edited.StatSec.TechType" @input="markDirty" /></label>
            <label>Тип урона<input v-model="edited.StatSec.DamageType" @input="markDirty" /></label>
            <label>Звук<input v-model="edited.StatSec.SoundType" @input="markDirty" /></label>
            <label>Задержка<input type="number" step="0.1" v-model.number="edited.StatSec.MinDelay" @input="markDirty" /></label>
            <label>Фактор<input type="number" step="0.1" v-model.number="edited.StatSec.Factor" @input="markDirty" /></label>
            <label class="full">Атрибуты<input v-model="secAttrStr" /></label>
          </div>
        </section>

        <!-- Броня -->
        <section class="stat-section">
          <h3>Броня</h3>
          <div class="grid">
            <label>Броня (осн.)<input type="number" v-model.number="edited.StatPriArmour.Armour" @input="markDirty" /></label>
            <label>Защита<input type="number" v-model.number="edited.StatPriArmour.DefSkill" @input="markDirty" /></label>
            <label>Щит<input type="number" v-model.number="edited.StatPriArmour.Shield" @input="markDirty" /></label>
            <label>Звук брони<input v-model="edited.StatPriArmour.Sound" @input="markDirty" /></label>
            <label>Броня (доп.)<input type="number" v-model.number="edited.StatSecArmour.Armour" @input="markDirty" /></label>
            <label>Защита (доп.)<input type="number" v-model.number="edited.StatSecArmour.DefSkill" @input="markDirty" /></label>
            <label>Звук (доп.)<input v-model="edited.StatSecArmour.Sound" @input="markDirty" /></label>
          </div>
        </section>

        <!-- Прочие статы -->
        <section class="stat-section">
          <h3>Прочее</h3>
          <div class="grid">
            <label>Жара<input type="number" v-model.number="edited.StatHeat" @input="markDirty" /></label>
            <label>Земля +пыль<input type="number" v-model.number="edited.StatGround[0]" @input="markDirty" /></label>
            <label>Земля +грязь<input type="number" v-model.number="edited.StatGround[1]" @input="markDirty" /></label>
            <label>Земля +снег<input type="number" v-model.number="edited.StatGround[2]" @input="markDirty" /></label>
            <label>Земля +лес<input type="number" v-model.number="edited.StatGround[3]" @input="markDirty" /></label>
            <label>Мораль<input type="number" v-model.number="edited.StatMental.Morale" @input="markDirty" /></label>
            <label>Дисциплина<input v-model="edited.StatMental.Discipline" @input="markDirty" /></label>
            <label>Тренировка<input v-model="edited.StatMental.Training" @input="markDirty" /></label>
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
    <span>Выберите юнита</span>
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
.grid label.full { grid-column: 1 / -1; }
.grid input {
  background: #0f1b35;
  border: 1px solid #2a3a5e;
  border-radius: 3px;
  color: #e0e0e0;
  font-size: 12px;
  padding: 4px 6px;
  width: 100%;
}
.grid input:focus { outline: none; border-color: #e94560; }

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
}
</style>
