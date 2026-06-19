<script setup>
import { ref, onMounted } from 'vue'
import { GetFactions, GetUnitsByFaction, Save, HasUnsavedChanges } from '../wailsjs/go/main/App'
import UnitDetail from './components/UnitDetail.vue'
import SetupScreen from './components/SetupScreen.vue'

const screen = ref('setup') // 'setup' | 'editor'

async function onGameReady() {
  screen.value = 'editor'
  factions.value = await GetFactions()
}

const factions = ref([])
const selectedFaction = ref(null)
const unitGroups = ref({})
const selectedUnitType = ref(null)
const hasChanges = ref(false)
const applying = ref(false)
const applyError = ref(null)

onMounted(() => {})

async function selectFaction(faction) {
  selectedFaction.value = faction.Name
  unitGroups.value = await GetUnitsByFaction(faction.Name)
  selectedUnitType.value = null
}

function selectUnit(unit) {
  selectedUnitType.value = unit.Type
}

async function onUnitSaved() {
  hasChanges.value = await HasUnsavedChanges()
}

async function applyToGame() {
  applying.value = true
  applyError.value = null
  try {
    await Save()
    hasChanges.value = await HasUnsavedChanges()
  } catch (e) {
    applyError.value = String(e)
  } finally {
    applying.value = false
  }
}
</script>

<template>
  <SetupScreen v-if="screen === 'setup'" @ready="onGameReady" />

  <div v-else class="layout">

    <!-- Топбар -->
    <div class="topbar">
      <span class="topbar-title">Total War Mod Editor</span>
      <div class="topbar-actions">
        <span v-if="applyError" class="topbar-error">{{ applyError }}</span>
        <button class="btn-apply" :disabled="applying || !hasChanges" @click="applyToGame">
          {{ applying ? 'Запись...' : 'Применить к игре' }}
        </button>
      </div>
    </div>

    <div class="below-topbar">

    <!-- Фракции -->
    <aside class="sidebar">
      <h2>Фракции</h2>
      <ul>
        <li
          v-for="faction in factions"
          :key="faction.Name"
          :class="{ active: selectedFaction === faction.Name }"
          @click="selectFaction(faction)"
        >
          {{ faction.Name }}
        </li>
      </ul>
    </aside>

    <!-- Список юнитов -->
    <div class="unit-list" v-if="selectedFaction">
      <h2>{{ selectedFaction }}</h2>
      <div v-for="(classes, category) in unitGroups" :key="category" class="category">
        <h3>{{ category }}</h3>
        <div v-for="(units, cls) in classes" :key="cls" class="class-group">
          <h4>{{ cls }}</h4>
          <ul>
            <li
              v-for="unit in units"
              :key="unit.Type"
              :class="{ active: selectedUnitType === unit.Type }"
              @click="selectUnit(unit)"
            >
              {{ unit.Name || unit.Type }}
            </li>
          </ul>
        </div>
      </div>
    </div>
    <div class="unit-list hint-panel" v-else>
      <p class="hint">Выберите фракцию слева</p>
    </div>

    <!-- Детали юнита -->
    <div class="detail-panel">
      <UnitDetail :unit-type="selectedUnitType" :faction="selectedFaction" @saved="onUnitSaved" @reverted="onUnitSaved" />
    </div>

    </div> <!-- below-topbar -->
  </div>
</template>


<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: sans-serif; background: #1a1a2e; color: #e0e0e0; }

.layout { display: flex; flex-direction: column; height: 100vh; overflow: hidden; }

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  height: 44px;
  background: #0d1117;
  border-bottom: 1px solid #1e2e50;
  flex-shrink: 0;
}
.topbar-title { font-size: 13px; font-weight: 600; color: #aaa; }
.topbar-actions { display: flex; align-items: center; gap: 10px; }
.topbar-error { font-size: 12px; color: #e94560; }
.btn-apply {
  background: #0f3460;
  color: #e0e0e0;
  border: 1px solid #1e4a8a;
  padding: 6px 14px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
}
.btn-apply:hover { background: #1a4a80; }
.btn-apply:disabled { opacity: 0.5; cursor: default; }

.below-topbar { display: flex; flex: 1; overflow: hidden; }

/* Фракции */
.sidebar {
  width: 200px;
  flex-shrink: 0;
  background: #16213e;
  padding: 16px;
  overflow-y: auto;
  border-right: 1px solid #1e2e50;
}
.sidebar h2 { font-size: 11px; text-transform: uppercase; color: #888; margin-bottom: 12px; letter-spacing: 0.05em; }
.sidebar ul { list-style: none; }
.sidebar li {
  padding: 7px 10px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
}
.sidebar li:hover { background: #0f3460; }
.sidebar li.active { background: #e94560; color: #fff; }

/* Список юнитов */
.unit-list {
  width: 240px;
  flex-shrink: 0;
  padding: 16px;
  overflow-y: auto;
  border-right: 1px solid #1e2e50;
}
.unit-list h2 { font-size: 13px; color: #aaa; margin-bottom: 14px; }

.hint-panel { display: flex; align-items: center; justify-content: center; }
.hint { color: #444; font-size: 13px; }

.category { margin-bottom: 20px; }
.category h3 {
  font-size: 10px;
  text-transform: uppercase;
  color: #e94560;
  border-bottom: 1px solid #1e2e50;
  padding-bottom: 4px;
  margin-bottom: 8px;
  letter-spacing: 0.05em;
}

.class-group { margin-bottom: 10px; }
.class-group h4 { font-size: 10px; color: #555; margin-bottom: 4px; text-transform: uppercase; }
.class-group ul { list-style: none; }
.class-group li {
  padding: 5px 8px;
  border-radius: 3px;
  font-size: 12px;
  cursor: pointer;
  color: #ccc;
}
.class-group li:hover { background: #16213e; }
.class-group li.active { background: #0f3460; color: #fff; }

/* Панель деталей */
.detail-panel {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
</style>
