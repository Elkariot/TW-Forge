<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  GetProjectiles, GetProjectileDelays, UpdateProjectile,
  GetProjectileChangeType, RevertProjectile, RevertAllProjectiles, ProjectilesAvailable,
  DeleteProjectile,
} from '../../wailsjs/go/main/App'
import { PROJECTILE_FLAGS } from '../enums.js'
import ProjectileCreateModal from './ProjectileCreateModal.vue'

const { t } = useI18n()
const emit = defineEmits(['changed'])

const projectiles = ref([])
const delays = ref([])
const loading = ref(true)
const available = ref(true)
const error = ref(null)
const search = ref('')

const selectedName = ref(null)
const projectile = ref(null)   // last saved snapshot of the selected entry
const edited = ref(null)       // working copy bound to the form
const dirty = ref(false)
const saving = ref(false)
const changeType = ref('none')

onMounted(load)

async function load() {
  loading.value = true
  error.value = null
  try {
    available.value = await ProjectilesAvailable()
    if (!available.value) return
    const [list, d] = await Promise.all([GetProjectiles(), GetProjectileDelays()])
    projectiles.value = [...(list ?? [])].sort((a, b) => a.Name.localeCompare(b.Name))
    delays.value = d ?? []
  } catch (e) {
    error.value = String(e)
  } finally {
    loading.value = false
  }
}

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return projectiles.value
  return projectiles.value.filter(p => p.Name.toLowerCase().includes(q))
})

// ── Create / delete ─────────────────────────────────────────────────────────

const showCreateModal = ref(false)

async function onProjectileCreated(name) {
  showCreateModal.value = false
  await load()
  const created = projectiles.value.find(p => p.Name === name)
  if (created) await select(created)
  emit('changed')
}

async function deleteProjectile() {
  if (!confirm(t('projectile.delete_projectile_confirm', { name: selectedName.value }))) return
  try {
    await DeleteProjectile(selectedName.value)
    await load()
    selectedName.value = null
    projectile.value = null
    edited.value = null
    emit('changed')
  } catch (e) {
    error.value = String(e)
  }
}

// ── Effect fields: free-choice dropdowns backed by values already used
// elsewhere in the loaded file (see feedback: no new-animation authoring yet,
// only picking among existing effect names) ───────────────────────────────

const EFFECT_FIELDS = [
  'Effect', 'EndEffect', 'EndManEffect', 'EndPackageEffect',
  'EndShatterEffect', 'EndShatterManEffect', 'EndShatterPackageEffect',
]

const effectOptions = computed(() => {
  const set = new Set()
  for (const p of projectiles.value) {
    for (const f of EFFECT_FIELDS) {
      if (p[f]) set.add(p[f])
    }
  }
  return [...set].sort()
})

function cloneEdit(p) {
  const c = JSON.parse(JSON.stringify(p))
  c.Velocity = c.Velocity && c.Velocity.length ? c.Velocity : [0]
  // Accuracy fields are optional in the file (absent = 0% for that target type),
  // but shown as plain always-present number inputs, same as unit weapon stats.
  c.AccuracyVsUnits = c.AccuracyVsUnits ?? 0
  c.AccuracyVsBuildings = c.AccuracyVsBuildings ?? 0
  c.AccuracyVsTowers = c.AccuracyVsTowers ?? 0
  return c
}

async function select(p) {
  selectedName.value = p.Name
  projectile.value = p
  edited.value = cloneEdit(p)
  dirty.value = false
  changeType.value = await GetProjectileChangeType(p.Name)
}

function markDirty() { dirty.value = true }

// ── Optional fields: checkbox toggles a nullable value on/off ──────────────

function toggleOptional(field, defaultValue) {
  edited.value[field] = edited.value[field] == null ? defaultValue : null
  markDirty()
}

function toggleRandomVelocity(checked) {
  const v = edited.value.Velocity ?? [0]
  edited.value.Velocity = checked ? [v[0] ?? 0, v[1] ?? v[0] ?? 0] : [v[0] ?? 0]
  markDirty()
}

function toggleFlag(key, checked) {
  const current = edited.value.Flags ?? []
  edited.value.Flags = checked ? [...current, key] : current.filter(f => f !== key)
  markDirty()
}

// ── Range calculator: d = v² · sin(2×angle) / g, matches the header comment
// in vanilla descr_projectile.txt files (g = 9.81, max range at 45°) ───────

const GRAVITY = 9.81

function rangeAtAngle(v, angleDeg) {
  const rad = (angleDeg * Math.PI) / 180
  return (v * v * Math.sin(2 * rad)) / GRAVITY
}

function optimalAngle(min, max) {
  if (min <= 45 && 45 <= max) return 45
  return Math.abs(45 - min) < Math.abs(45 - max) ? min : max
}

function fmtRange(n) {
  if (!isFinite(n) || n <= 0) return '—'
  return n.toFixed(1) + ' m'
}

const rangeEstimates = computed(() => {
  if (!edited.value) return []
  const { MinAngle: min, MaxAngle: max, Velocity } = edited.value
  const vMax = Velocity[Velocity.length - 1] ?? 0
  const vMin = Velocity[0] ?? 0
  const opt = optimalAngle(min, max)

  const rows = [
    { label: t('projectile.range_at_optimal', { angle: opt }), value: fmtRange(rangeAtAngle(vMax, opt)) },
    { label: t('projectile.range_at_min_angle', { angle: min }), value: fmtRange(rangeAtAngle(vMax, min)) },
    { label: t('projectile.range_at_max_angle', { angle: max }), value: fmtRange(rangeAtAngle(vMax, max)) },
  ]
  if (Velocity.length > 1 && vMin !== vMax) {
    rows.push({ label: t('projectile.range_at_optimal_min_velocity', { angle: opt }), value: fmtRange(rangeAtAngle(vMin, opt)) })
  }
  return rows
})

async function save() {
  saving.value = true
  error.value = null
  try {
    await UpdateProjectile(selectedName.value, edited.value)
    const idx = projectiles.value.findIndex(p => p.Name === selectedName.value)
    const saved = JSON.parse(JSON.stringify(edited.value))
    if (idx !== -1) projectiles.value[idx] = saved
    if (selectedName.value !== saved.Name) {
      projectiles.value.sort((a, b) => a.Name.localeCompare(b.Name))
    }
    selectedName.value = saved.Name
    projectile.value = saved
    dirty.value = false
    changeType.value = await GetProjectileChangeType(saved.Name)
    emit('changed')
  } catch (e) {
    error.value = String(e)
  } finally {
    saving.value = false
  }
}

function cancel() {
  edited.value = cloneEdit(projectile.value)
  dirty.value = false
  error.value = null
}

async function revertOne() {
  try {
    await RevertProjectile(selectedName.value)
    await load()
    const restored = projectiles.value.find(p => p.Name === selectedName.value)
    if (restored) await select(restored)
    emit('changed')
  } catch (e) {
    error.value = String(e)
  }
}

async function revertAll() {
  if (!confirm(t('projectile.revert_all_confirm'))) return
  await RevertAllProjectiles()
  await load()
  selectedName.value = null
  edited.value = null
  emit('changed')
}
</script>

<template>
  <div v-if="loading" class="pe-root pe-root--empty">{{ $t('common.loading') }}</div>
  <div v-else-if="!available" class="pe-root pe-root--empty">{{ $t('projectile.no_file') }}</div>
  <div v-else class="pe-root">

    <!-- Список снарядов -->
    <div class="pe-list">
      <div class="pe-list-header">
        <input v-model="search" class="pe-search" :placeholder="$t('projectile.search_placeholder')" />
      </div>

      <div v-if="delays.length" class="pe-delays">
        <div class="pe-delays-title">{{ $t('projectile.delays_title') }}</div>
        <div class="pe-delay-chip" v-for="d in delays" :key="d.Type">
          {{ $t('projectile.delay_' + d.Type) }}: {{ d.Seconds }}s
        </div>
      </div>

      <div v-if="filtered.length === 0" class="pe-hint">{{ $t('common.nothing_found') }}</div>
      <ul v-else class="pe-items">
        <li
          v-for="p in filtered"
          :key="p.Name"
          :class="{ active: selectedName === p.Name }"
          @click="select(p)"
        >{{ p.Name }}</li>
      </ul>

      <div class="pe-add-row">
        <button class="btn-add-projectile" @click="showCreateModal = true">+ {{ $t('projectile.add_projectile') }}</button>
      </div>

      <button class="pe-revert-all" @click="revertAll">{{ $t('projectile.revert_all') }}</button>
    </div>

    <ProjectileCreateModal
      v-if="showCreateModal"
      :projectiles="projectiles"
      @close="showCreateModal = false"
      @created="onProjectileCreated"
    />

    <!-- Детали снаряда -->
    <div class="pe-detail">
      <template v-if="edited">

        <div class="pe-detail-header">
          <div class="pe-detail-title">
            <span class="pe-name">{{ edited.Name }}</span>
            <span v-if="changeType === 'modified'" class="pe-badge">{{ $t('projectile.modified_badge') }}</span>
            <span v-else-if="changeType === 'added'" class="pe-badge pe-badge-added">{{ $t('projectile.added_badge') }}</span>
          </div>
          <div class="pe-toolbar">
            <template v-if="dirty">
              <button class="btn-save" :disabled="saving" @click="save">
                {{ saving ? $t('common.saving') : $t('common.save') }}
              </button>
              <button class="btn-cancel" @click="cancel">{{ $t('common.cancel') }}</button>
            </template>
            <button v-else-if="changeType !== 'none'" class="btn-revert" @click="revertOne">↺</button>
            <button class="btn-delete-projectile" @click="deleteProjectile">{{ $t('projectile.delete_projectile') }}</button>
            <span v-if="error" class="pe-error">{{ error }}</span>
          </div>
        </div>

        <div class="pe-scroll">
          <div class="sections">

            <section class="stat-section">
              <h3>{{ $t('projectile.section_identity') }}</h3>
              <div class="grid">
                <label>{{ $t('projectile.field_name') }}<input v-model="edited.Name" @input="markDirty" /></label>
                <label>{{ $t('projectile.field_flaming_of') }}
                  <input v-model="edited.FlamingOf" @input="markDirty" :title="$t('projectile.field_flaming_of_hint')" />
                </label>
                <label>{{ $t('projectile.field_exploding_of') }}
                  <input v-model="edited.ExplodingOf" @input="markDirty" :title="$t('projectile.field_exploding_of_hint')" />
                </label>
              </div>
            </section>

            <section class="stat-section">
              <h3>{{ $t('projectile.section_damage') }}</h3>
              <div class="grid">
                <label>{{ $t('projectile.field_damage') }}<input type="number" v-model.number="edited.Damage" @input="markDirty" /></label>
                <label class="full checkbox-row">
                  <input type="checkbox" :checked="edited.DamageToTroops != null" @change="e => toggleOptional('DamageToTroops', e.target.checked ? 0 : null)" />
                  {{ $t('projectile.field_has_damage_to_troops') }}
                </label>
                <label v-if="edited.DamageToTroops != null">
                  {{ $t('projectile.field_damage_to_troops') }}
                  <input type="number" v-model.number="edited.DamageToTroops" @input="markDirty" />
                </label>
              </div>
            </section>

            <section class="stat-section">
              <h3>{{ $t('projectile.section_physics') }}</h3>
              <div class="physics-row">
                <div class="grid physics-grid">
                  <label>{{ $t('projectile.field_radius') }}<input type="number" step="0.01" v-model.number="edited.Radius" @input="markDirty" /></label>
                  <label>{{ $t('projectile.field_mass') }}<input type="number" step="0.01" v-model.number="edited.Mass" @input="markDirty" /></label>
                  <label>{{ $t('projectile.field_min_angle') }}<input type="number" v-model.number="edited.MinAngle" @input="markDirty" /></label>
                  <label>{{ $t('projectile.field_max_angle') }}<input type="number" v-model.number="edited.MaxAngle" @input="markDirty" /></label>

                  <label class="full checkbox-row">
                    <input type="checkbox" :checked="edited.Velocity.length > 1" @change="e => toggleRandomVelocity(e.target.checked)" />
                    {{ $t('projectile.field_velocity_random') }}
                  </label>
                  <label v-if="edited.Velocity.length <= 1">
                    {{ $t('projectile.field_velocity') }}
                    <input type="number" step="0.1" :value="edited.Velocity[0]" @input="e => { edited.Velocity = [Number(e.target.value)]; markDirty() }" />
                  </label>
                  <template v-else>
                    <label>
                      {{ $t('projectile.field_velocity_min') }}
                      <input type="number" step="0.1" :value="edited.Velocity[0]" @input="e => { edited.Velocity[0] = Number(e.target.value); markDirty() }" />
                    </label>
                    <label>
                      {{ $t('projectile.field_velocity_max') }}
                      <input type="number" step="0.1" :value="edited.Velocity[1]" @input="e => { edited.Velocity[1] = Number(e.target.value); markDirty() }" />
                    </label>
                  </template>
                </div>

                <!-- Живой расчёт дальности по физике снаряда, для сверки со stat_pri в EDU -->
                <div class="range-calc">
                  <div class="range-calc-title">{{ $t('projectile.range_calc_title') }}</div>
                  <div v-for="r in rangeEstimates" :key="r.label" class="range-calc-row">
                    <span class="range-calc-label">{{ r.label }}</span>
                    <span class="range-calc-value">{{ r.value }}</span>
                  </div>
                  <div class="range-hint">{{ $t('projectile.range_formula_hint') }}</div>
                </div>
              </div>
            </section>

            <section class="stat-section">
              <h3>{{ $t('projectile.section_accuracy') }}</h3>
              <div class="grid">
                <label class="full checkbox-row">
                  <input type="checkbox" :checked="edited.Area != null" @change="e => toggleOptional('Area', e.target.checked ? 1 : null)" />
                  {{ $t('projectile.field_has_area') }}
                </label>
                <label v-if="edited.Area != null">
                  {{ $t('projectile.field_area') }}
                  <input type="number" step="0.1" v-model.number="edited.Area" @input="markDirty" />
                </label>

                <label>
                  {{ $t('projectile.field_accuracy_units') }}
                  <input type="number" step="0.001" v-model.number="edited.AccuracyVsUnits" @input="markDirty" />
                </label>
                <label>
                  {{ $t('projectile.field_accuracy_buildings') }}
                  <input type="number" step="0.001" v-model.number="edited.AccuracyVsBuildings" @input="markDirty" />
                </label>
                <label>
                  {{ $t('projectile.field_accuracy_towers') }}
                  <input type="number" step="0.001" v-model.number="edited.AccuracyVsTowers" @input="markDirty" />
                </label>
              </div>
            </section>

            <section class="stat-section">
              <h3>{{ $t('projectile.section_flags') }}</h3>
              <div class="flag-list">
                <label v-for="f in PROJECTILE_FLAGS" :key="f.key" class="flag-check">
                  <input type="checkbox"
                    :checked="(edited.Flags ?? []).includes(f.key)"
                    @change="e => toggleFlag(f.key, e.target.checked)"
                  />
                  {{ f.label }} <span class="flag-key">({{ f.key }})</span>
                </label>
              </div>
            </section>

            <section class="stat-section">
              <h3>{{ $t('projectile.section_effects') }}</h3>
              <div class="grid">
                <label class="full" :title="$t('projectile.field_effect_hint')">
                  {{ $t('projectile.field_effect') }}
                  <input v-model="edited.Effect" list="pe-effect-options" @input="markDirty" />
                </label>
                <label>{{ $t('projectile.field_end_effect') }}<input v-model="edited.EndEffect" list="pe-effect-options" @input="markDirty" /></label>
                <label>{{ $t('projectile.field_end_man_effect') }}<input v-model="edited.EndManEffect" list="pe-effect-options" @input="markDirty" /></label>
                <label>{{ $t('projectile.field_end_package_effect') }}<input v-model="edited.EndPackageEffect" list="pe-effect-options" @input="markDirty" /></label>
                <label>{{ $t('projectile.field_end_shatter_effect') }}<input v-model="edited.EndShatterEffect" list="pe-effect-options" @input="markDirty" /></label>
                <label>{{ $t('projectile.field_end_shatter_man_effect') }}<input v-model="edited.EndShatterManEffect" list="pe-effect-options" @input="markDirty" /></label>
                <label>{{ $t('projectile.field_end_shatter_package_effect') }}<input v-model="edited.EndShatterPackageEffect" list="pe-effect-options" @input="markDirty" /></label>
              </div>
              <datalist id="pe-effect-options">
                <option v-for="opt in effectOptions" :key="opt" :value="opt" />
              </datalist>
            </section>

          </div>
        </div>
      </template>

      <div v-else class="pe-empty">
        <span v-if="error">{{ error }}</span>
        <span v-else>{{ $t('projectile.select_hint') }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.pe-root { display: flex; height: 100%; overflow: hidden; }
.pe-root--empty { align-items: center; justify-content: center; color: var(--color-text-muted); font-size: 13px; }

/* Список */
.pe-list {
  width: 240px;
  flex-shrink: 0;
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 12px;
}
.pe-list-header { margin-bottom: 8px; }
.pe-search {
  width: 100%;
  box-sizing: border-box;
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text);
  font-size: 12px;
  padding: 5px 8px;
}
.pe-search:focus { outline: none; border-color: var(--color-accent); }
.pe-filter-select {
  background: var(--color-cell); border: 1px solid var(--color-border-soft); border-radius: 4px;
  color: var(--color-text-dim); font-size: 11px; padding: 4px 6px; flex: 1;
}
.pe-filter-select:focus { outline: none; border-color: var(--color-accent); }

.pe-delays { margin-bottom: 10px; }
.pe-delays-title { font-size: 10px; text-transform: uppercase; color: var(--color-text-muted); letter-spacing: 0.05em; margin-bottom: 4px; }
.pe-delay-chip { font-size: 11px; color: var(--color-text-dim); }

.pe-hint { font-size: 12px; color: var(--color-text-muted); text-align: center; margin-top: 16px; }

.pe-items { list-style: none; flex: 1; overflow-y: auto; }
.pe-items li {
  padding: 6px 9px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  color: var(--color-text-dim);
}
.pe-items li:hover { background: var(--color-cell); }
.pe-items li.active { background: var(--color-primary); color: var(--color-text); }

.pe-revert-all {
  margin-top: 8px;
  background: transparent;
  border: 1px dashed var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text-muted);
  font-size: 11px;
  padding: 6px 0;
  cursor: pointer;
  flex-shrink: 0;
}
.pe-revert-all:hover { border-color: var(--color-accent); color: var(--color-accent); }

.pe-add-row { margin-top: 8px; }
.btn-add-projectile {
  width: 100%;
  box-sizing: border-box;
  background: var(--color-primary);
  border: none;
  color: var(--color-text);
  border-radius: 4px;
  padding: 7px 10px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}
.btn-add-projectile:hover { background: var(--color-primary-hover); }
.btn-delete-projectile { background: transparent; border: 1px solid var(--color-error); color: var(--color-error); border-radius: 4px; padding: 5px 10px; font-size: 11px; cursor: pointer; }
.btn-delete-projectile:hover { background: var(--color-error); color: var(--color-text); }

/* Детали */
.pe-detail { flex: 1; overflow: hidden; display: flex; flex-direction: column; }

.pe-detail-header {
  flex-shrink: 0;
  padding: 8px 14px;
  background: var(--color-bg);
  border-bottom: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.pe-detail-title { display: flex; align-items: center; gap: 8px; min-width: 0; }
.pe-name { font-size: 14px; font-weight: 600; color: var(--color-text); font-family: monospace; }
.pe-badge {
  font-size: 10px;
  color: var(--color-accent);
  border: 1px solid var(--color-accent);
  border-radius: 10px;
  padding: 1px 8px;
}
.pe-badge-added { color: #3aba80; border-color: #3aba80; }

.pe-toolbar { display: flex; align-items: center; gap: 6px; flex-shrink: 0; }
.btn-save {
  background: var(--color-primary); color: var(--color-text); border: none;
  padding: 5px 12px; border-radius: 4px; cursor: pointer; font-size: 12px;
}
.btn-save:disabled { opacity: 0.6; cursor: default; }
.btn-cancel {
  background: transparent; color: var(--color-text-dim); border: 1px solid #444;
  padding: 5px 10px; border-radius: 4px; cursor: pointer; font-size: 12px;
}
.btn-cancel:hover { border-color: var(--color-text-dim); }
.btn-revert {
  background: transparent; color: var(--color-text-muted); border: 1px solid var(--color-border-soft);
  padding: 4px 9px; border-radius: 4px; cursor: pointer; font-size: 13px; line-height: 1;
}
.btn-revert:hover { border-color: var(--color-accent); color: var(--color-accent); }
.pe-error { color: var(--color-error); font-size: 11px; max-width: 220px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.pe-scroll { flex: 1; overflow-y: auto; padding: 12px 20px 20px; }
.sections { display: flex; flex-direction: column; gap: 12px; max-width: 700px; }

.stat-section { background: var(--color-cell); border-radius: 6px; padding: 12px 14px; }
.stat-section h3 { font-size: 11px; text-transform: uppercase; color: var(--color-heading); margin-bottom: 10px; letter-spacing: 0.05em; }

.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 8px; }
.grid label { display: flex; flex-direction: column; gap: 3px; font-size: 11px; color: var(--color-text-dim); }
.grid .full { grid-column: 1 / -1; }
.grid input {
  background: var(--color-input-bg); border: 1px solid var(--color-border-soft);
  border-radius: 3px; color: var(--color-text); font-size: 12px; padding: 4px 6px; width: 100%;
  box-sizing: border-box;
}
.grid input:focus { outline: none; border-color: var(--color-accent); }
.checkbox-row { flex-direction: row !important; align-items: center; gap: 6px !important; cursor: pointer; }
.checkbox-row input[type="checkbox"] { width: auto; accent-color: var(--color-accent); cursor: pointer; }

.physics-row { display: flex; gap: 18px; align-items: flex-start; }
.physics-grid { flex: 1; min-width: 280px; }
.range-calc {
  flex: 1;
  min-width: 220px;
  background: var(--color-input-bg);
  border: 1px dashed var(--color-border-soft);
  border-radius: 6px;
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.range-calc-title { font-size: 10px; text-transform: uppercase; color: var(--color-text-muted); letter-spacing: 0.05em; margin-bottom: 2px; }
.range-calc-row { display: flex; justify-content: space-between; gap: 10px; font-size: 12px; color: var(--color-text-dim); }
.range-calc-value { font-family: monospace; color: var(--color-text); }
.range-hint { font-size: 10px; color: var(--color-text-muted); font-style: italic; margin-top: 4px; }

.flag-list { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 6px 10px; }
.flag-check { display: flex; align-items: center; gap: 6px; font-size: 11px; color: var(--color-text-dim); cursor: pointer; }
.flag-check input[type="checkbox"] { width: auto; margin: 0; accent-color: var(--color-accent); cursor: pointer; }
.flag-key { color: var(--color-text-muted); font-size: 10px; }

.pe-empty {
  flex: 1; display: flex; align-items: center; justify-content: center;
  color: var(--color-text-muted); font-size: 13px; padding: 20px; text-align: center;
}
</style>
