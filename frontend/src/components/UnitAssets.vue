<script setup>
import { ref, watch } from 'vue'
import {
  GetUnitIconInfo, GetUnitModelFiles, GetUnitTextureFiles,
  GetDataPaths, PickFile, SaveIconFile, CopyAssetToRoot,
  UploadAssetFile, AbsAssetPath, GetUnitByType,
} from '../../wailsjs/go/main/App'

const props = defineProps(['unitType', 'faction'])

const iconInfo   = ref(null)    // { Data, Source, RelPath }
const modelFiles = ref([])
const texFiles   = ref([])
const dataPaths  = ref({})      // { mod, base? }
const modelName  = ref('')      // Soldier.Model — префикс для поиска файлов
const loading    = ref(false)
const error      = ref(null)

// Диалог сохранения файла
const pending = ref(null)       // { type: 'icon'|'asset', srcPath, relPath, srcAbsPath }
const showSaveDlg = ref(false)

watch(() => [props.unitType, props.faction], async ([type, faction]) => {
  if (!type || !faction) { reset(); return }
  loading.value = true
  error.value = null
  try {
    const [icon, models, textures, paths, unit] = await Promise.all([
      GetUnitIconInfo(type, faction),
      GetUnitModelFiles(type),
      GetUnitTextureFiles(type),
      GetDataPaths(),
      GetUnitByType(type),
    ])
    iconInfo.value   = icon
    modelFiles.value = models || []
    texFiles.value   = textures || []
    dataPaths.value  = paths
    modelName.value  = unit?.Soldier?.Model ?? ''
  } catch (e) {
    error.value = String(e)
  } finally {
    loading.value = false
  }
}, { immediate: true })

function reset() {
  iconInfo.value = null; modelFiles.value = []; texFiles.value = []; dataPaths.value = {}; modelName.value = ''
}

// ── Иконка ──────────────────────────────────────────────────────────────────
async function editIcon() {
  const src = await PickFile('Выберите иконку (.tga)', 'TGA files', '*.tga;*.TGA')
  if (!src) return
  openSaveDlg({ type: 'icon', srcPath: src })
}

// ── Файлы моделей / текстур ──────────────────────────────────────────────────
async function uploadAsset(subdir) {
  const ext = subdir.includes('texture') ? '*.tga;*.TGA;*.dds;*.DDS' : '*.cas;*.CAS'
  const label = subdir.includes('texture') ? 'Texture files' : 'CAS model files'
  const src = await PickFile(`Выберите файл (${subdir})`, label, ext)
  if (!src) return
  openSaveDlg({ type: 'upload', srcPath: src, subdir })
}

async function copyToMod(file) {
  const absPath = await AbsAssetPath(file.RelPath, file.Source)
  openSaveDlg({ type: 'copy', srcPath: absPath, relPath: file.RelPath, origSource: file.Source })
}

// ── Диалог выбора куда сохранять ─────────────────────────────────────────────
function openSaveDlg(data) {
  pending.value = data
  showSaveDlg.value = true
}
function closeSaveDlg() { showSaveDlg.value = false; pending.value = null }

const hasMod  = () => !!dataPaths.value.mod
const hasBase = () => !!dataPaths.value.base
// Только один путь (mod == base или мод не выбран) — предупреждение всегда
const singlePath = () => !hasBase()

async function saveToPath(destRoot, isBase) {
  if (!pending.value) return
  const p = pending.value
  closeSaveDlg()
  error.value = null
  try {
    if (p.type === 'icon') {
      await SaveIconFile(props.unitType, props.faction, p.srcPath, destRoot)
      iconInfo.value = await GetUnitIconInfo(props.unitType, props.faction)
    } else if (p.type === 'copy') {
      await CopyAssetToRoot(p.srcPath, destRoot, p.relPath)
      await refreshAssets()
    } else if (p.type === 'upload') {
      await UploadAssetFile(p.srcPath, destRoot, p.subdir)
      await refreshAssets()
    }
  } catch (e) {
    error.value = String(e)
  }
}

async function refreshAssets() {
  const [models, textures] = await Promise.all([
    GetUnitModelFiles(props.unitType),
    GetUnitTextureFiles(props.unitType),
  ])
  modelFiles.value = models || []
  texFiles.value   = textures || []
}

function sourceLabel(src) {
  if (src === 'mod')  return 'МОД'
  if (src === 'base') return 'ИГРА'
  return ''
}
function sourceClass(src) {
  if (src === 'mod')  return 'badge-mod'
  if (src === 'base') return 'badge-base'
  return ''
}
</script>

<template>
  <div class="ua-root">
    <div v-if="loading" class="ua-loading">Загрузка...</div>
    <div v-if="error" class="ua-error">{{ error }}</div>

    <!-- Иконка -->
    <div class="ua-section">
      <div class="ua-section-head">
        <span class="ua-section-title">Иконка</span>
        <button class="ua-action-btn" @click="editIcon">Изменить</button>
      </div>
      <div class="ua-icon-row">
        <div class="ua-icon-wrap">
          <img v-if="iconInfo?.Data" :src="iconInfo.Data" class="ua-icon-img" />
          <div v-else class="ua-icon-placeholder">?</div>
        </div>
        <div class="ua-icon-meta">
          <span v-if="iconInfo?.Source" :class="['ua-badge', sourceClass(iconInfo.Source)]">
            {{ sourceLabel(iconInfo.Source) }}
          </span>
          <span v-else class="ua-badge badge-missing">НЕТ</span>
          <span class="ua-rel-path">{{ iconInfo?.RelPath || '—' }}</span>
        </div>
      </div>
    </div>

    <!-- Модели -->
    <div class="ua-section">
      <div class="ua-section-head">
        <span class="ua-section-title">Модели (.cas)</span>
        <button class="ua-action-btn" @click="uploadAsset('models_unit')">+ Загрузить</button>
      </div>
      <div v-if="modelName" class="ua-model-hint">Префикс: <code>{{ modelName }}</code></div>
      <div v-if="modelFiles.length === 0" class="ua-empty">Файлы не найдены</div>
      <div v-for="f in modelFiles" :key="f.RelPath" class="ua-file-row">
        <span :class="['ua-badge', sourceClass(f.Source)]">{{ sourceLabel(f.Source) }}</span>
        <span class="ua-file-name" :title="f.RelPath">{{ f.Name }}</span>
        <button v-if="f.Source === 'base'" class="ua-copy-btn" @click="copyToMod(f)" title="Скопировать в мод">
          → МОД
        </button>
      </div>
    </div>

    <!-- Текстуры -->
    <div class="ua-section">
      <div class="ua-section-head">
        <span class="ua-section-title">Текстуры (.tga/.dds)</span>
        <button class="ua-action-btn" @click="uploadAsset('models_unit/textures')">+ Загрузить</button>
      </div>
      <div v-if="modelName" class="ua-model-hint">Префикс: <code>{{ modelName }}</code></div>
      <div v-if="texFiles.length === 0" class="ua-empty">Файлы не найдены</div>
      <div v-for="f in texFiles" :key="f.RelPath" class="ua-file-row">
        <span :class="['ua-badge', sourceClass(f.Source)]">{{ sourceLabel(f.Source) }}</span>
        <span class="ua-file-name" :title="f.RelPath">{{ f.Name }}</span>
        <button v-if="f.Source === 'base'" class="ua-copy-btn" @click="copyToMod(f)" title="Скопировать в мод">
          → МОД
        </button>
      </div>
    </div>

    <!-- Диалог выбора куда сохранять -->
    <div v-if="showSaveDlg" class="ua-dlg-overlay" @click.self="closeSaveDlg">
      <div class="ua-dlg">
        <div class="ua-dlg-title">Куда сохранить?</div>

        <button
          v-if="dataPaths.mod"
          class="ua-dlg-btn ua-dlg-btn-mod"
          @click="saveToPath(dataPaths.mod, false)"
        >
          Папка мода
          <span class="ua-dlg-hint">{{ dataPaths.mod }}</span>
        </button>

        <button
          v-if="dataPaths.base || singlePath()"
          class="ua-dlg-btn ua-dlg-btn-base"
          @click="saveToPath(dataPaths.base || dataPaths.mod, true)"
        >
          <span class="ua-dlg-warn">⚠</span> Папка оригинальной игры
          <span class="ua-dlg-hint ua-dlg-hint-warn">
            Резервная копия НЕ создаётся. Изменение затронет все моды, использующие этот файл.
          </span>
        </button>

        <button class="ua-dlg-cancel" @click="closeSaveDlg">Отмена</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ua-root { display: flex; flex-direction: column; gap: 16px; padding: 12px 0; }
.ua-loading { font-size: 11px; color: var(--color-text-muted); }
.ua-error { font-size: 11px; color: var(--color-error); }

.ua-section { display: flex; flex-direction: column; gap: 6px; }
.ua-section-head { display: flex; align-items: center; justify-content: space-between; }
.ua-section-title {
  font-size: 10px;
  text-transform: uppercase;
  color: var(--color-accent);
  letter-spacing: 0.06em;
}
.ua-action-btn {
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-soft);
  border-radius: 3px;
  color: var(--color-text-dim);
  font-size: 10px;
  padding: 2px 7px;
  cursor: pointer;
}
.ua-action-btn:hover { border-color: var(--color-accent); color: var(--color-accent); }

/* Icon */
.ua-icon-row { display: flex; align-items: center; gap: 10px; }
.ua-icon-wrap {
  width: 40px;
  height: 40px;
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  overflow: hidden;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-input-bg);
}
.ua-icon-img { width: 100%; height: 100%; object-fit: contain; }
.ua-icon-placeholder { font-size: 18px; color: var(--color-border-soft); }
.ua-icon-meta { display: flex; flex-direction: column; gap: 3px; min-width: 0; }
.ua-rel-path { font-size: 10px; color: var(--color-text-muted); word-break: break-all; }

/* Badges */
.ua-badge {
  display: inline-block;
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.04em;
  padding: 1px 5px;
  border-radius: 3px;
}
.badge-mod  { background: rgba(64,160,100,0.15); color: #4a9; border: 1px solid rgba(64,160,100,0.3); }
.badge-base { background: rgba(233,160,0,0.1);  color: #e9a000; border: 1px solid rgba(233,160,0,0.3); }
.badge-missing { background: rgba(149,232,225,0.2); color: var(--color-accent); border: 1px solid rgba(149,232,225,0.3); }

/* File rows */
.ua-empty { font-size: 11px; color: var(--color-text-muted); }
.ua-model-hint { font-size: 10px; color: #445; margin-bottom: 2px; }
.ua-model-hint code { color: #4a9; font-size: 10px; }
.ua-file-row { display: flex; align-items: center; gap: 6px; }
.ua-file-name { flex: 1; font-size: 11px; color: var(--color-text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ua-copy-btn {
  flex-shrink: 0;
  background: none;
  border: 1px solid var(--color-border-soft);
  border-radius: 3px;
  color: #4a9;
  font-size: 10px;
  padding: 1px 5px;
  cursor: pointer;
}
.ua-copy-btn:hover { border-color: #4a9; background: rgba(64,160,100,0.08); }

/* Save dialog */
.ua-dlg-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}
.ua-dlg {
  background: var(--color-cell);
  border: 1px solid var(--color-border-soft);
  border-radius: 8px;
  padding: 20px;
  width: 360px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.ua-dlg-title { font-size: 14px; font-weight: 600; color: var(--color-text); margin-bottom: 4px; }
.ua-dlg-btn {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 3px;
  background: var(--color-input-bg);
  border: 1px solid var(--color-border-soft);
  border-radius: 6px;
  color: var(--color-text);
  font-size: 13px;
  padding: 10px 14px;
  cursor: pointer;
  text-align: left;
  width: 100%;
}
.ua-dlg-btn-mod:hover  { border-color: #4a9; }
.ua-dlg-btn-base { border-color: #664400; }
.ua-dlg-btn-base:hover { border-color: #e9a000; }
.ua-dlg-warn { color: #e9a000; }
.ua-dlg-hint {
  font-size: 10px;
  color: var(--color-text-muted);
  word-break: break-all;
}
.ua-dlg-hint-warn { color: #7a5500; }
.ua-dlg-cancel {
  background: none;
  border: 1px solid var(--color-border-soft);
  border-radius: 4px;
  color: var(--color-text-dim);
  font-size: 12px;
  padding: 6px;
  cursor: pointer;
  align-self: flex-end;
}
.ua-dlg-cancel:hover { color: var(--color-text-dim); }
</style>
