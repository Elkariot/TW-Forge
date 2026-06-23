# TW Forge

[Русская версия](README_RU.md) · [![Download](https://img.shields.io/github/v/release/Elkariot/TW-Forge?label=Download&logo=github)](https://github.com/Elkariot/TW-Forge/releases/latest)

A visual mod editor for **Rome: Total War** and **Medieval II: Total War**.

Edit units, buildings, and recruitment queues without manually parsing game text files.

| | |
|---|---|
| ![Setup screen](frontend/src/assets/screens/main_screen.png) | ![Unit editor](frontend/src/assets/screens/units_screen.png) |
| ![Building editor](frontend/src/assets/screens/building_screen.png) | ![Recruit editor](frontend/src/assets/screens/recruit_screen.png) |

---

## Features

### Units

- View and edit all unit stats: weapons, armour, formation, cost, attributes
- Edit unit name and description
- Create a new unit from a template (copy of an existing unit)
- Copy a unit to another faction — icon and `battle_models.modeldb` entry are copied automatically
- Delete a unit: soft delete (remove from faction) or full delete (remove from EDU + delete icons)
- Revert changes for an individual unit
- Manage icons and 3D model files (.cas / .tga / .dds) via the Assets tab (experimental, use with caution)

### Buildings

- View and edit the building tree
- Edit level parameters: cost, construction time, settlement type, bonuses
- Manage building dependencies
- Revert all building changes

### Recruitment

A dual-mode editor:

- **By unit** — see which buildings recruit the selected unit, add/remove buildings, configure pool parameters
- **By building** — see which units are recruited at each level, add/remove units, configure pool parameters

Pool parameters: initial pool, replenish rate, max pool, experience gained, conditions.

### Applying Changes

All changes are accumulated in memory and do not touch game files until explicitly confirmed.  
The **Apply to Game** button writes the modified files to the mod folder.

---

## Supported Games

| Game | Status |
|------|--------|
| Rome: Total War | ✓ |
| Medieval II: Total War | ✓ |

---

## Installation

1. Download `tw-forge.exe` from the [Releases](https://github.com/Elkariot/TW-Forge/releases) page
2. Run it — no installation required

**Requirements:** Windows 10/11, WebView2 (installed automatically on first launch if missing)

> **Note:** Windows may show a SmartScreen warning because the executable is not code-signed. Click **More info → Run anyway** to proceed. The app does not connect to the internet and only reads/writes your local mod files.

---

## Getting Started

1. On the setup screen, select a game (RTW or M2TW)
2. Set the path to the game folder
3. Select a mod from the list (or add a mod folder manually)
4. Click **Start Editing**

The editor reads only the selected mod's files and the base game files — original files are not modified. All original text files are backed up to `_modding_editor/backup` inside the mod folder before the first change.

---

## Interface

- Language toggle: **RU / EN** button in the top-right corner
- Theme switcher: palette icon (5 themes available)
- Unit search: search bar in the left panel with faction filter

---

## Known Limitations

- Not all game file fields are supported — rare or highly specific parameters may be missing
- Reordering units within a faction (EDU order) is not supported
- The "create new unit" feature exists but is experimental — it is essentially an advanced copy operation. For RTW you can try loading models, textures, and icons, but stability is not guaranteed
- Asset editing (models, textures, icons) is unstable — use with caution
- Assets (models, textures, icons) are **NOT backed up**. Modifying original assets is at your own risk
- There is currently no validation for text field input — double-check your changes before saving

---

## Roadmap

| Version | Planned |
|---------|---------|
| v1.1 | Mercenaries, projectile types |
| v1.2 | Unit comparison card, diff mode |
| v2.0 | Campaign starting configuration editor |

---

## Building from Source

Requirements: Go 1.21+, Node.js 18+, [Wails v2](https://wails.io)

```bash
git clone <repo>
cd modding-utils
wails build
```

---

## License

[MIT](LICENSE) © 2026 Daniil Belokon

## Contact

For suggestions and bug reports:  
e-mail: amareyrey@gmail.com
