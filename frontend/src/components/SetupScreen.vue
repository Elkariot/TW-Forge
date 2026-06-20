<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import {
  OpenDirectoryDialog, InitGameFolder, GetMods,
  GetBaseGameDataPath, AddModPath, InitGame,
  LoadAppConfig, SaveAppConfig,
} from '../../wailsjs/go/main/App'

import romeImg from '../assets/images/Rome.jpg'
import medievalImg from '../assets/images/Medieval.png'

const emit = defineEmits(['ready'])

// 0 = Medieval, 1 = Rome (соответствует config.GameVersion)
const GAMES = [
  { id: 1, label: 'Rome: Total War',       img: romeImg     },
  { id: 0, label: 'Medieval II: Total War', img: medievalImg },
]

const selectedGame   = ref(null)   // 0 | 1 | null
const gamePath       = ref('')
const pathError      = ref('')
const pathLoading    = ref(false)
const mods           = ref([])     // [{ name, dataPath }]
const selectedMod    = ref(null)   // { name, dataPath }
const launching      = ref(false)
const launchError    = ref('')
const isRestoring    = ref(false)

const modsReady = computed(() => mods.value.length > 0)
// TODO: также требовать что game folder валидирована через InitGameFolder
const canLaunch = computed(() =>
  selectedGame.value !== null && selectedMod.value !== null && !launching.value
)

function saveConfig() {
  if (selectedGame.value === null || !gamePath.value) return
  SaveAppConfig(selectedGame.value, gamePath.value, selectedMod.value?.dataPath ?? '').catch(() => {})
}

watch(selectedMod, (mod) => {
  if (isRestoring.value || !mod) return
  saveConfig()
})

onMounted(async () => {
  const cfg = await LoadAppConfig()
  if (cfg.game < 0 || !cfg.gamePath) return

  isRestoring.value = true
  selectedGame.value = cfg.game
  gamePath.value = cfg.gamePath
  await validatePath()

  // Восстановить вручную добавленные моды.
  if (cfg.manualMods && Object.keys(cfg.manualMods).length > 0) {
    for (const [name, path] of Object.entries(cfg.manualMods)) {
      try { await AddModPath(path, name) } catch { /* путь мог исчезнуть */ }
    }
    // Перестроить список модов после восстановления.
    const baseDataPath = await GetBaseGameDataPath()
    const modsMap = await GetMods()
    const list = [{ name: 'Base game', dataPath: baseDataPath }]
    for (const [n, dataPath] of Object.entries(modsMap)) {
      list.push({ name: prettify(n), dataPath })
    }
    mods.value = list
  }

  if (cfg.selectedModPath && mods.value.length > 0) {
    const found = mods.value.find(m => m.dataPath === cfg.selectedModPath)
    if (found) selectedMod.value = found
  }
  isRestoring.value = false
})

function selectGame(game) {
  if (selectedGame.value === game.id) return
  selectedGame.value = game.id
  gamePath.value = ''
  pathError.value = ''
  mods.value = []
  selectedMod.value = null
}

async function browse() {
  const dir = await OpenDirectoryDialog()
  if (!dir) return
  gamePath.value = dir
  await validatePath()
}

async function validatePath() {
  if (!gamePath.value || selectedGame.value === null) return
  pathLoading.value = true
  pathError.value = ''
  mods.value = []
  selectedMod.value = null
  try {
    await InitGameFolder(selectedGame.value, gamePath.value)
    const modsMap = await GetMods()
    const baseDataPath = await GetBaseGameDataPath()

    const list = [{ name: 'Base game', dataPath: baseDataPath }]
    for (const [name, dataPath] of Object.entries(modsMap)) {
      list.push({ name: prettify(name), dataPath })
    }
    mods.value = list
    if (list.length === 1) selectedMod.value = list[0]
    saveConfig()
  } catch (e) {
    pathError.value = String(e)
  } finally {
    pathLoading.value = false
  }
}

async function browseAddMod() {
  const dir = await OpenDirectoryDialog()
  if (!dir) return
  const name = dir.split(/[\\/]/).pop() || dir
  try {
    await AddModPath(dir, name)
    const baseDataPath = await GetBaseGameDataPath()
    const modsMap = await GetMods()
    const list = []
    if (baseDataPath) list.push({ name: 'Base game', dataPath: baseDataPath })
    for (const [n, dataPath] of Object.entries(modsMap)) {
      list.push({ name: prettify(n), dataPath })
    }
    mods.value = list
  } catch (e) {
    pathError.value = String(e)
  }
}

async function launch() {
  if (!canLaunch.value) return
  launching.value = true
  launchError.value = ''
  try {
    await InitGame(selectedMod.value.dataPath, selectedGame.value)
    emit('ready', selectedGame.value)
  } catch (e) {
    launchError.value = String(e)
  } finally {
    launching.value = false
  }
}

function prettify(name) {
  return name.replace(/[_-]/g, ' ').replace(/\b\w/g, c => c.toUpperCase())
}
</script>

<template>
  <div class="setup">

    <div class="setup-header">
      <div class="setup-title">⚔ Total War Mod Editor</div>
    </div>

    <div class="setup-body">

      <!-- Левая колонка: выбор игры + путь -->
      <div class="col-left">

        <div class="section-label">Выберите игру</div>
        <div class="game-cards">
          <button
            v-for="game in GAMES"
            :key="game.id"
            class="game-card"
            :class="{ selected: selectedGame === game.id }"
            @click="selectGame(game)"
          >
            <img :src="game.img" :alt="game.label" class="game-img" />
            <div class="game-name">{{ game.label }}</div>
          </button>
        </div>

        <template v-if="selectedGame !== null">
          <div class="section-label" style="margin-top: 28px">Путь к папке игры</div>
          <div class="path-row">
            <input
              v-model="gamePath"
              class="path-input"
              placeholder="C:\Games\Rome Total War"
              @change="validatePath"
            />
            <button class="btn-browse" :disabled="pathLoading" @click="browse">
              {{ pathLoading ? '...' : 'Обзор' }}
            </button>
          </div>
          <div v-if="pathError" class="path-error">{{ pathError }}</div>
        </template>

      </div>

      <!-- Правая колонка: список модов -->
      <div class="col-right">

        <template v-if="modsReady">
          <div class="section-label">Выберите мод</div>
          <div class="mods-list">
            <button
              v-for="mod in mods"
              :key="mod.dataPath"
              class="mod-entry"
              :class="{ selected: selectedMod?.dataPath === mod.dataPath }"
              @click="selectedMod = mod"
            >
              <div class="mod-radio">
                <div class="mod-radio-dot" v-if="selectedMod?.dataPath === mod.dataPath" />
              </div>
              <div class="mod-name">{{ mod.name }}</div>
            </button>
          </div>
        </template>

        <div v-else-if="selectedGame !== null && !pathLoading && !pathError" class="mods-hint">
          Моды появятся после выбора папки игры<br/>или добавьте мод вручную
        </div>

        <!-- TODO: сделать неактивной пока не выбрана игра и не указан путь к папке -->
        <button class="btn-add-mod" @click="browseAddMod">+ Добавить мод</button>

      </div>

    </div>

    <!-- Нижняя панель -->
    <div class="setup-footer">
      <span v-if="launchError" class="launch-error">{{ launchError }}</span>
      <button class="btn-launch" :disabled="!canLaunch" @click="launch">
        <span v-if="launching">Загрузка...</span>
        <span v-else>Начать редактирование →</span>
      </button>
    </div>

  </div>
</template>

<style scoped>
.setup {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: var(--color-bg);
  color: var(--color-text);
}

/* Шапка */
.setup-header {
  padding: 20px 32px 16px;
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.setup-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--color-text);
  letter-spacing: 0.03em;
}

/* Тело */
.setup-body {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.col-left {
  flex: 1;
  padding: 32px;
  border-right: 1px solid var(--color-border);
  overflow-y: auto;
}

.col-right {
  flex: 1;
  padding: 32px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

/* Метка секции */
.section-label {
  font-size: 11px;
  text-transform: uppercase;
  color: var(--color-heading);
  letter-spacing: 0.08em;
  margin-bottom: 14px;
}

/* Карточки игр */
.game-cards {
  display: flex;
  gap: 16px;
}

.game-card {
  position: relative;
  width: 200px;
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  border: 2px solid var(--color-border);
  background: var(--color-cell);
  padding: 0;
  transition: border-color 0.15s, transform 0.15s;
}
.game-card:hover { border-color: #e898d0; transform: translateY(-2px); }
.game-card.selected { border-color: var(--color-accent); }

.game-img {
  width: 100%;
  height: 120px;
  object-fit: cover;
  display: block;
}
.game-name {
  font-size: 12px;
  font-weight: 600;
  padding: 8px 10px;
  color: var(--color-text-dim);
  text-align: center;
  background: var(--color-input-bg);
}
.game-card.selected .game-name { color: var(--color-text); }

/* Путь */
.path-row {
  display: flex;
  gap: 8px;
}
.path-input {
  flex: 1;
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text);
  font-size: 13px;
  padding: 8px 10px;
}
.path-input:focus { outline: none; border-color: var(--color-accent); }
.btn-browse {
  background: var(--color-cell);
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text-dim);
  font-size: 13px;
  padding: 8px 16px;
  cursor: pointer;
  white-space: nowrap;
}
.btn-browse:hover { border-color: var(--color-text-dim); color: var(--color-text); }
.btn-browse:disabled { opacity: 0.5; cursor: default; }
.path-error {
  margin-top: 8px;
  font-size: 12px;
  color: var(--color-error);
}

/* Моды */
.mods-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 12px;
}

.mod-entry {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  background: var(--color-cell);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  text-align: left;
  transition: border-color 0.12s, background 0.12s;
  color: var(--color-text-dim);
}
.mod-entry:hover { border-color: #e898d0; background: #f5e0ef; }
.mod-entry.selected { border-color: var(--color-accent); background: var(--color-bg); color: var(--color-text); }

.mod-radio {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 2px solid #444;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}
.mod-entry.selected .mod-radio { border-color: var(--color-accent); }
.mod-radio-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-primary);
}
.mod-name { font-size: 13px; font-weight: 500; }

.btn-add-mod {
  align-self: flex-start;
  background: transparent;
  border: 1px dashed var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text-muted);
  font-size: 12px;
  padding: 6px 14px;
  cursor: pointer;
  margin-top: 4px;
}
.btn-add-mod:hover { border-color: var(--color-text-dim); color: var(--color-text-dim); }

.mods-hint {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted);
  font-size: 13px;
}

/* Подвал */
.setup-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 14px;
  padding: 16px 32px;
  border-top: 1px solid var(--color-border);
  flex-shrink: 0;
}
.launch-error { font-size: 12px; color: var(--color-error); }
.btn-launch {
  background: var(--color-primary);
  color: var(--color-text);
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 600;
  padding: 10px 28px;
  cursor: pointer;
  letter-spacing: 0.02em;
  transition: background 0.15s;
}
.btn-launch:hover { background: var(--color-primary-hover); }
.btn-launch:disabled { background: #f0d0e8; color: var(--color-text-muted); cursor: default; }
</style>
