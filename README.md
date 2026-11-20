# ⚡ ImmichPulse — Ultra-Light Terminal Dashboard (TUI) for Immich

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)
![Bubble Tea](https://img.shields.io/badge/UI-Bubble_Tea-FF79C6?style=flat)
![License](https://img.shields.io/badge/License-MIT-00FF87?style=flat)

**ImmichPulse** is a lightweight, statically linked Terminal UI (TUI) dashboard built in Go for self-hosted **Immich** servers.

![ImmichPulse Screenshot](screenshot.png)

## 🚀 Quick Install (One-Line)
```bash
git clone https://github.com/merobiju/immich-pulse.git && cd immich-pulse && ./install.sh
```

## 💻 Usage
```bash
immich
# or
immich-pulse
```
- `r` — Force refresh metrics
- `q` or `Ctrl+C` — Exit

## 🔄 Auto-Launch on SSH Login (Optional)
```bash
echo -e '\n# Immich Dashboard\nif [[ \$- == *i* ]]; then\n  immich\nfi' >> ~/.bashrc
```

## 🗑️ Uninstall
```bash
sudo rm -f /usr/local/bin/immich /usr/local/bin/immich-pulse
```

## 👨‍💻 Author
Built in Nepal by **Bijaya Pokhrel**.
