// Значения, жёстко зашитые в движок RTW/M2TW
export const UNIT_CATEGORIES = ['infantry', 'cavalry', 'siege', 'handler', 'ship', 'non_combatant']
export const UNIT_CLASSES = ['heavy', 'light', 'missile', 'spearmen']
export const VOICE_TYPES = ['Heavy_1', 'Medium_1', 'Light_1', 'General_1', 'Female_1']

export const WEAPON_TYPES = ['melee', 'missile', 'thrown', 'no']
export const TECH_TYPES = ['blade', 'archery', 'simple', 'no']
export const DAMAGE_TYPES = ['slashing', 'piercing', 'blunt', 'fire', 'no']
export const SOUND_TYPES = ['sword', 'spear', 'axe', 'mace', 'knife', 'none']
export const ARMOUR_SOUNDS = ['flesh', 'leather', 'metal']

export const DISCIPLINE_VALUES = ['disciplined', 'impetuous', 'berserker', 'low', 'normal']
export const TRAINING_VALUES = ['highly_trained', 'trained', 'untrained']

export const UNIT_ATTRIBUTES = [
  { key: 'hide_forest',          label: 'Скрытность в лесу' },
  { key: 'hide_long_grass',      label: 'Скрытность в высокой траве' },
  { key: 'hide_improved_forest', label: 'Улучш. скрытность в лесу' },
  { key: 'hide_anywhere',        label: 'Скрытность везде' },
  { key: 'hardy',                label: 'Выносливый' },
  { key: 'very_hardy',           label: 'Очень выносливый' },
  { key: 'can_sap',              label: 'Может рыть подкопы' },
  { key: 'sea_faring',           label: 'Морской переход' },
  { key: 'frighten_foot',        label: 'Устрашает пехоту' },
  { key: 'frighten_mounted',     label: 'Устрашает конницу' },
  { key: 'cantabrian_circle',    label: 'Кантабрийский круг' },
  { key: 'warcry',               label: 'Боевой клич' },
  { key: 'screeching_women',     label: 'Крики женщин' },
  { key: 'command',              label: 'Командный' },
  { key: 'druid',                label: 'Друид' },
  { key: 'general_unit',         label: 'Генеральский отряд' },
  { key: 'general_unit_upgrade', label: 'Улучш. генеральский' },
  { key: 'mercenary_unit',       label: 'Наёмники' },
  { key: 'no_custom',            label: 'Недоступен в конструкторе' },
  { key: 'can_run_amok',         label: 'Впадает в панику' },
]

// Атрибуты оружия (stat_pri_attr / stat_sec_attr), "no" означает отсутствие атрибутов
// spear_bonus_N хранится отдельно (числовое поле)
export const WEAPON_ATTRIBUTES = [
  { key: 'spear',      label: 'Длинное копьё' },
  { key: 'light_spear',label: 'Дротик / ополч. копьё' },
  { key: 'long_pike',  label: 'Сарисса (очень длинная)' },
  { key: 'ap',         label: 'Бронебойный' },
  { key: 'bp',         label: 'Против укреплений' },
  { key: 'fire',       label: 'Огненный' },
  { key: 'thrown',     label: 'Метательный' },
  { key: 'prec',       label: 'Точность (метат.)' },
  { key: 'launching',  label: 'Катапультный' },
  { key: 'area',       label: 'Площадной урон' },
]

// Строй: primary (всегда), secondary (опционально)
export const FORMATION_PRIMARY   = ['square', 'horde']
export const FORMATION_SECONDARY = ['', 'phalanx', 'testudo', 'wedge']

// Флаги-переключатели descr_projectile.txt (бинарные ключевые слова без значения).
// Должно соответствовать domain.ManagedProjectileFlags в Go.
export const PROJECTILE_FLAGS = [
  { key: 'fiery',            label: 'Зажигательный' },
  { key: 'affected_by_rain', label: 'Точность падает от дождя' },
  { key: 'ground_shatter',   label: 'Разрушается о землю' },
  { key: 'body_piercing',    label: 'Пробивает насквозь' },
  { key: 'grapeshot',        label: 'Картечь' },
  { key: 'prefer_high',      label: 'ИИ предпочитает навесную траекторию' },
  { key: 'no_ae_on_ram',     label: 'Не действует на таран' },
  { key: 'effect_only',      label: 'Только визуальный эффект (без объекта)' },
  { key: 'cow_carcass',      label: 'Коровья туша' },
]
