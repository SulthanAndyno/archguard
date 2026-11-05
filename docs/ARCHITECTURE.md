# ArchGuard Architecture & Design Decisions

This document outlines the architectural patterns, technology stack rationales, and safety guarantees underpinning **ArchGuard**.

---

## 1. Architectural Philosophy

Arch Linux follows the **rolling-release model**. While this provides bleeding-edge software, it carries two operational risks:
1. **Unchecked Breaking Changes:** Updating packages without reviewing official advisories requiring manual intervention.
2. **System Degradation:** Accumulating unneeded dependency trees (orphans), bloated cache directories, and unmerged `.pacnew` configuration files.

ArchGuard addresses both issues through a **zero-overhead, high-observability TUI companion**.

```
+-------------------------------------------------------------+
|                      ArchGuard TUI                          |
|             (Bubble Tea + Lip Gloss Engine)                 |
+-------------------------------------------------------------+
        │                   │                   │
        ▼                   ▼                   ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────┐
│  Arch News    │   │ Pacman Engine │   │ System Audit  │
│  (RSS / Web)  │   │(checkupdates) │   │ (Cache / etc) │
└───────────────┘   └───────────────┘   └───────────────┘
        │                   │                   │
  archlinux.org       /usr/bin/pacman      /var/cache/pacman
   RSS Feed              alpm / DB             /etc/*.pacnew
```

---

## 2. Technology Rationale

### Why Go over Python or Bash?
* **Zero Dependency Distribution:** Compiles to a single static binary. Users on minimal Arch installs do not need a Python virtual environment or specific pip wheels.
* **Predictable Concurrency:** Goroutines and channels allow concurrent querying of package databases, news RSS feeds, and disk space without blocking the UI thread.
* **Low Memory Footprint:** Starts in ~5ms with memory consumption < 15MB RSS.

### Why Charmbracelet (Bubble Tea & Lip Gloss)?
* Implements **The Elm Architecture (TEA)**:
  - `Model`: Encapsulates the entire application state.
  - `Update`: Pure state transitions triggered by events (`tea.Msg`).
  - `View`: Declarative rendering of UI frames.
* Built-in support for terminal alternate screens (`tea.WithAltScreen()`) to ensure no clutter is left in the user's terminal scrollback buffer after exiting.

---

## 3. Concurrency Model: Non-Blocking Startup

When ArchGuard launches, it invokes `fetchSystemDataCmd()` which spawns **4 parallel goroutines** synchronized with a `sync.WaitGroup`:

1. **Goroutine A (News Collector):** Downloads and parses the Arch Linux RSS feed via `gofeed`.
2. **Goroutine B (Package Inventory):** Queries local package database via `pacman -Qq` into an in-memory hash set `map[string]bool`.
3. **Goroutine C (Pending Updates):** Runs `checkupdates` to parse pending upgrades without acquiring database write locks.
4. **Goroutine D (Storage & Orphans):** Gathers orphan packages (`pacman -Qtdq`), disk usage in `/var/cache/pacman/pkg`, and locates unmerged `.pacnew` files.

Once all goroutines resolve, a single `InitialDataMsg` is dispatched to the TEA event loop, transitioning the UI from the spinner to the interactive state instantly.

---

## 4. Safety First Principle

* **Non-Destructive by Default:** Reading updates and reviewing news runs completely unprivileged. No `sudo` is required to run ArchGuard.
* **Explicit Action:** Destructive actions (like orphan purging or cache trimming) generate the exact shell command for the user to review, preventing accidental system breakage.
