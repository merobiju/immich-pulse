<div align="center">

# ⚡ ImmichPulse

**The fastest, ultra-lightweight (~3.8MB) Terminal Dashboard (TUI) for self-hosted Immich servers.**

Built in **Go** with zero runtime dependencies. Monitor your entire Immich stack, hardware thermals, network speed, and Tailscale VPN from a single SSH screen.

![ImmichPulse Dashboard](screenshot.png)

</div>

---

## 🔥 Why ImmichPulse?

Heavy web dashboards eat up RAM and CPU on your home server. **ImmichPulse** gives you instant, real-time telemetry in a single **~3.8MB static binary**:

- 📸 **Live Immich DB Metrics** — Real-time Photo, Video, and Active User counts directly from PostgreSQL + server version.
- 🐳 **Docker Container Health** — Live status, CPU %, RAM usage, and Network I/O for `immich_server`, `immich_postgres`, `immich_redis`, and `immich_machine_learning`.
- ⚡ **Hardware & Power Stats** — CPU load sparklines, temperature alerts (`COOL` / `WARM` / `HOT`), laptop battery charge thresholds, RAM/Swap, and Disk I/O.
- 🌐 **Network & Tailscale VPN** — Live download/upload speed (`Kbps`/`Mbps`), SSH failed-login security counter, MagicDNS, and peer online indicators.
- 🔒 **1-Key Privacy Mode** — Press **`p`** anytime to instantly mask all Local, Public, and Tailscale IPs (`•••.•••.•••.•••`) before taking screenshots or recording demos.
- 🌍 **14 Platforms Supported** — Pre-compiled binaries ready for Linux, macOS, Windows, FreeBSD, OpenBSD, and NetBSD.

---

## 🚀 Quick Install (Linux — 1 Command)

```bash
git clone https://github.com/merobiju/immich-pulse.git && cd immich-pulse && ./install.sh
```

Or download pre-compiled binaries directly from the [**Releases Page**](https://github.com/merobiju/immich-pulse/releases/latest) on the right.

---

## ⌨️ Keyboard Shortcuts

| Key | Action |
| :--- | :--- |
| **`r`** | Force immediate metrics sync |
| **`p`** | Toggle **Privacy Mode** (Hide/Show all IPs) |
| **`q`** / `Ctrl+C` | Exit dashboard |

---

## 🔄 Auto-Launch on SSH Login (Optional)

Want your Immich dashboard to greet you every time you SSH into your server? Run this once:

```bash
echo -e '\n# ImmichPulse Dashboard\nif [[ $- == *i* ]]; then\n  immich\nfi' >> ~/.bashrc
```

---

## 🗑️ Uninstall

```bash
sudo rm -f /usr/local/bin/immich /usr/local/bin/immich-pulse
```

---

<div align="center">

### 👨‍💻 Crafted with ❤️ in Nepal 🇳🇵 by **[Bijaya Pokhrel](https://github.com/merobiju)**

**If ImmichPulse saved you time or looks cool on your server, please drop a ⭐ Star on this repo!**

</div>
