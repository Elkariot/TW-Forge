<script setup>
import { ref, onMounted } from 'vue'
import { GetFactions, GetUnitsByFaction } from '../wailsjs/go/main/App'

const factions = ref([])
const selectedFaction = ref(null)
const unitGroups = ref({})

onMounted(async () => {
  factions.value = await GetFactions()
})

async function selectFaction(faction) {
  selectedFaction.value = faction.Name
  unitGroups.value = await GetUnitsByFaction(faction.Name)
}
</script>

<template>
  <div class="layout">
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

    <main class="content">
      <template v-if="selectedFaction">
        <h2>{{ selectedFaction }}</h2>
        <div v-for="(classes, category) in unitGroups" :key="category" class="category">
          <h3>{{ category }}</h3>
          <div v-for="(units, cls) in classes" :key="cls" class="class-group">
            <h4>{{ cls }}</h4>
            <ul>
              <li v-for="unit in units" :key="unit.Type">
                {{ unit.Name || unit.Type }}
              </li>
            </ul>
          </div>
        </div>
      </template>
      <p v-else class="hint">Выберите фракцию слева</p>
    </main>
  </div>
</template>

<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: sans-serif; background: #1a1a2e; color: #e0e0e0; }

.layout { display: flex; height: 100vh; }

.sidebar {
  width: 220px;
  background: #16213e;
  padding: 16px;
  overflow-y: auto;
  flex-shrink: 0;
}
.sidebar h2 { font-size: 14px; text-transform: uppercase; color: #888; margin-bottom: 12px; }
.sidebar ul { list-style: none; }
.sidebar li {
  padding: 8px 10px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
}
.sidebar li:hover { background: #0f3460; }
.sidebar li.active { background: #e94560; color: #fff; }

.content { flex: 1; padding: 24px; overflow-y: auto; }
.content h2 { margin-bottom: 16px; }

.category { margin-bottom: 24px; }
.category h3 {
  font-size: 12px;
  text-transform: uppercase;
  color: #e94560;
  border-bottom: 1px solid #333;
  padding-bottom: 4px;
  margin-bottom: 12px;
}

.class-group { margin-bottom: 12px; }
.class-group h4 { font-size: 12px; color: #888; margin-bottom: 6px; }
.class-group ul { list-style: none; display: flex; flex-wrap: wrap; gap: 6px; }
.class-group li {
  background: #16213e;
  padding: 4px 10px;
  border-radius: 3px;
  font-size: 12px;
}

.hint { color: #555; margin-top: 40px; text-align: center; }
</style>
