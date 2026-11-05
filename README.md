# ArchGuard 🛡️

[![Arch Linux](https://img.shields.io/badge/Arch%20Linux-Compliant-1793d1?logo=arch-linux&logoColor=white)](https://archlinux.org)
[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go&logoColor=white)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Built with Bubble Tea](https://img.shields.io/badge/TUI-Bubble%20Tea-F25D94)](https://github.com/charmbracelet/bubbletea)

> **ArchGuard** is an aesthetic, lightweight Terminal User Interface (TUI) companion designed for Arch Linux enthusiasts. It guarantees safe system updates by checking official breaking advisories, monitoring critical pending updates, identifying orphaned packages, and analyzing cache bloat—all in a single lightning-fast dashboard.

---

## 🌟 Key Features

* 📰 **Arch News & Intervention Sentinel:**
  Fetches official Arch Linux news announcements and automatically warns you if an advisory mentions packages currently installed on your system.
* 📦 **Pending Updates Inspector:**
  Safely detects pending updates (via `checkupdates`), highlighting critical core components like the Linux Kernel, Systemd, Mesa, or Nvidia drivers.
* 🧹 **Interactive Orphan Manager:**
  Identifies orphaned dependencies (`pacman -Qtdq`) with size breakdowns and interactive multi-select checkboxes.
* 💾 **Cache & Health Auditor:**
  Calculates pacman cache footprint (`/var/cache/pacman/pkg`) and tracks unmerged `.pacnew` configuration files.
* ⚡ **Blazing Fast Concurrency:**
  Written in Go; fetches RSS, system databases, and file systems concurrently in parallel goroutines with zero UI freeze.

---

## 🖥️ Preview

```text
 ARCHGUARD    System Health, News & Maintenance Companion

 [ 1. Arch News ]  [ 2. Pending Updates ]  [ 3. Orphan Packages ]  [ 4. System & Cache Audit ]

╭────────────────────────────────────────────────────────────────────────────────────────────╮
│ Official Arch Linux News & Advisories:                                                     │
│                                                                                            │
│ ▶ MANUAL ACTION REQUIRED Mkinitcpio >=42 requires manual intervention [Installed: mkinit]  │
│   Starting with package version 42-1, the mkinitcpio systemd hook now includes...          │
│                                                                                            │
│   Update to ca-certificates-mozilla ≥ 3.131.1-1 to prevent man-in-the-middle attacks      │
│   Update to php ≥ 8.5.12 to prevent remote code execution                                  │
╰────────────────────────────────────────────────────────────────────────────────────────────╯

 [Tab/1-4] Switch Tab • [j/k/↑/↓] Navigate • [Space] Toggle • [r] Refresh • [q] Quit
```

---

## 🚀 Installation

### Building from Source
Ensure you have Go installed:

```bash
git clone https://github.com/SulthanAndyno/archguard.git
cd archguard
make build
sudo make install
```

### From AUR (Arch User Repository)
*(Once published)*:
```bash
yay -S archguard-bin
# or
paru -S archguard-bin
```

---

## ⌨️ Keybindings

| Key | Action |
|---|---|
| `Tab` / `Shift+Tab` | Cycle through tabs |
| `1` - `4` | Jump directly to specific tab |
| `j` / `k` or `↑` / `↓` | Navigate list items |
| `Space` | Toggle selection (in Orphan Packages tab) |
| `r` | Refresh system telemetry and news feed |
| `q` or `Ctrl+C` | Exit ArchGuard |

---

## 🛠️ Tech Stack & Architecture

* **Language:** Go (1.27+)
* **TUI Framework:** [Bubble Tea](https://github.com/charmbracelet/bubbletea) (The Elm Architecture for CLI)
* **Styling Engine:** [Lip Gloss](https://github.com/charmbracelet/lipgloss)
* **Feed Parser:** [gofeed](https://github.com/mmcdole/gofeed)
* See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed design decisions.

---

## 👤 Author

* **Sulthan Andyno**
  * GitHub: [@SulthanAndyno](https://github.com/SulthanAndyno)
  * Email: sulthandyno@gmail.com

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
