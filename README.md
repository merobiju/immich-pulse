# ⚡ ImmichPulse — Ultra-Light Terminal Dashboard (TUI) for Immich

**ImmichPulse** is a lightweight, statically linked Terminal UI (TUI) dashboard built in Go for self-hosted **Immich** servers.

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
