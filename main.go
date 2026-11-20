package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type PeerInfo struct {
	Name   string
	IP     string
	Online bool
}

type ContainerInfo struct {
	Name   string
	Status string
	CPU    string
	Mem    string
	Net    string
}

type SysData struct {
	Hostname   string
	OSName     string
	Kernel     string
	Uptime     string
	CPUModel   string
	CPUCores   int
	CPUGov     string
	CPUPct     float64
	CPUTemp    float64
	HasTemp    bool
	BatCap     int
	BatStat    string
	BatLim     string
	MemPct     float64
	MemUsedGB  float64
	MemTotGB   float64
	MemAvailGB float64
	MemCacheGB float64
	SwapUsedGB float64
	SwapTotGB  float64
	DiskPct    float64
	DiskUsedGB float64
	DiskTotGB  float64
	DiskFreeGB float64
	DiskReadGB float64
	DiskWritGB float64
	LoadAvg    string
	SSID       string
	Iface      string
	LanIP      string
	WanIP      string
	RxKbps     float64
	TxKbps     float64
	LastSSH    string
	FailSSH    int
	TsIP       string
	MagicDNS   string
	Peers      []PeerInfo
	Containers []ContainerInfo
	AllHealthy bool
	ImmichVer  string
	DockerVer  string
	PhotoCount string
	VideoCount string
	UserCount  string
}

type animMsg time.Time
type dataMsg SysData

type model struct {
	data       SysData
	tick       int
	lastFetch  time.Time
	fetching   bool
	cpuHist    []float64
	netHist    []float64
	width      int
	height     int
	prevRx     uint64
	prevTx     uint64
	prevNetTs  time.Time
	wanCache   string
	wanCacheTs time.Time
}

func runCmd(cmd string) string {
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (m *model) collectData() SysData {
	var d SysData
	d.Hostname, _ = os.Hostname()
	if d.Hostname == "" {
		d.Hostname = "MEROBIJU"
	}
	d.OSName = runCmd(`grep PRETTY_NAME /etc/os-release | cut -d'"' -f2`)
	if d.OSName == "" {
		d.OSName = "Ubuntu Linux"
	}
	d.Kernel = runCmd("uname -r")

	hInfo, _ := host.Info()
	hrs := hInfo.Uptime / 3600
	mins := (hInfo.Uptime % 3600) / 60
	d.Uptime = fmt.Sprintf("%dh %dm", hrs, mins)

	d.CPUModel = runCmd(`lscpu | grep 'Model name:' | sed 's/Model name:[[:space:]]*//'`)
	d.CPUModel = strings.ReplaceAll(d.CPUModel, " with Radeon Graphics", "")
	if len(d.CPUModel) > 28 {
		d.CPUModel = d.CPUModel[:28]
	}
	if d.CPUModel == "" {
		d.CPUModel = "AMD Ryzen"
	}
	d.CPUCores, _ = cpu.Counts(true)
	d.CPUGov = runCmd("cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor")
	if d.CPUGov == "" {
		d.CPUGov = "schedutil"
	}
	cPcts, _ := cpu.Percent(0, false)
	if len(cPcts) > 0 {
		d.CPUPct = cPcts[0]
	}

	tempStr := runCmd("cat /sys/class/hwmon/hwmon*/temp1_input 2>/dev/null | sort -nr | head -1")
	if tVal, err := strconv.ParseFloat(tempStr, 64); err == nil {
		d.CPUTemp = tVal / 1000.0
		d.HasTemp = true
	}

	batStr := runCmd("cat /sys/class/power_supply/BAT0/capacity")
	d.BatCap, _ = strconv.Atoi(batStr)
	d.BatStat = runCmd("cat /sys/class/power_supply/BAT0/status")
	if d.BatStat == "" {
		d.BatStat = "AC Power"
	}
	d.BatLim = runCmd("cat /sys/class/power_supply/BAT0/charge_control_end_threshold")
	if d.BatLim == "" {
		d.BatLim = "100"
	}

	vMem, _ := mem.VirtualMemory()
	sMem, _ := mem.SwapMemory()
	d.MemPct = vMem.UsedPercent
	d.MemUsedGB = float64(vMem.Used) / (1024 * 1024 * 1024)
	d.MemTotGB = float64(vMem.Total) / (1024 * 1024 * 1024)
	d.MemAvailGB = float64(vMem.Available) / (1024 * 1024 * 1024)
	d.MemCacheGB = float64(vMem.Cached) / (1024 * 1024 * 1024)
	d.SwapUsedGB = float64(sMem.Used) / (1024 * 1024 * 1024)
	d.SwapTotGB = float64(sMem.Total) / (1024 * 1024 * 1024)
	dUsage, _ := disk.Usage("/")
	d.DiskPct = dUsage.UsedPercent
	d.DiskUsedGB = float64(dUsage.Used) / (1024 * 1024 * 1024)
	d.DiskTotGB = float64(dUsage.Total) / (1024 * 1024 * 1024)
	d.DiskFreeGB = float64(dUsage.Free) / (1024 * 1024 * 1024)

	dIO, _ := disk.IOCounters()
	for _, v := range dIO {
		d.DiskReadGB += float64(v.ReadBytes) / (1024 * 1024 * 1024)
		d.DiskWritGB += float64(v.WriteBytes) / (1024 * 1024 * 1024)
	}

	avg, _ := load.Avg()
	d.LoadAvg = fmt.Sprintf("%.2f / %.2f / %.2f", avg.Load1, avg.Load5, avg.Load15)

	d.Iface = runCmd("ip route | grep default | awk '{print $5}' | head -1")
	if d.Iface != "" {
		d.LanIP = runCmd(fmt.Sprintf(`ip -4 addr show %s | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | head -1`, d.Iface))
	}
	if d.LanIP == "" {
		d.LanIP = "N/A"
	}
	d.SSID = runCmd("nmcli -t -f active,ssid dev wifi | grep '^yes' | cut -d':' -f2")
	if d.SSID == "" {
		d.SSID = d.Iface
	}

	if time.Since(m.wanCacheTs) > 5*time.Minute || m.wanCache == "" {
		wan := runCmd("curl -4 -s --max-time 2 ifconfig.me || curl -s --max-time 2 icanhazip.com")
		if wan != "" {
			m.wanCache = wan
			m.wanCacheTs = time.Now()
		}
	}
	d.WanIP = m.wanCache
	if d.WanIP == "" {
		d.WanIP = "N/A"
	}

	nIO, _ := net.IOCounters(true)
	now := time.Now()
	for _, nic := range nIO {
		if nic.Name == d.Iface {
			if !m.prevNetTs.IsZero() {
				dt := now.Sub(m.prevNetTs).Seconds()
				if dt > 0 {
					d.RxKbps = float64(nic.BytesRecv-m.prevRx) * 8 / 1000 / dt
					d.TxKbps = float64(nic.BytesSent-m.prevTx) * 8 / 1000 / dt
				}
			}
			m.prevRx = nic.BytesRecv
			m.prevTx = nic.BytesSent
			m.prevNetTs = now
			break
		}
	}

	d.LastSSH = runCmd("who | grep bijaya | tail -1 | awk '{print $3, $4, $5}' | tr -d '()'")
	if d.LastSSH == "" {
		d.LastSSH = fmt.Sprintf("Active (%s)", d.Hostname)
	}
	fStr := runCmd("journalctl -u ssh -u sshd --since '24 hours ago' 2>/dev/null | grep -Ec 'Failed password|Invalid user'")
	d.FailSSH, _ = strconv.Atoi(fStr)

	d.TsIP = runCmd("tailscale ip -4 | head -1")
	if d.TsIP == "" {
		d.TsIP = "Disconnected"
	}
	tsRaw := runCmd("tailscale status --json")
	if tsRaw != "" {
		var tsObj struct {
			Self struct {
				DNSName string `json:"DNSName"`
			} `json:"Self"`
			Peer map[string]struct {
				DNSName      string   `json:"DNSName"`
				HostName     string   `json:"HostName"`
				TailscaleIPs []string `json:"TailscaleIPs"`
				Online       bool     `json:"Online"`
			} `json:"Peer"`
		}
		if json.Unmarshal([]byte(tsRaw), &tsObj) == nil {
			d.MagicDNS = strings.TrimSuffix(tsObj.Self.DNSName, ".")
			for _, p := range tsObj.Peer {
				name := strings.Split(p.DNSName, ".")[0]
				if name == "" {
					name = p.HostName
				}
				pip := "N/A"
				if len(p.TailscaleIPs) > 0 {
					pip = p.TailscaleIPs[0]
				}
				d.Peers = append(d.Peers, PeerInfo{Name: name, IP: pip, Online: p.Online})
			}
		}
	}

	d.ImmichVer = runCmd(`docker exec immich_server cat /usr/src/app/package.json 2>/dev/null | grep -m1 '"version"' | cut -d'"' -f4`)
	if d.ImmichVer == "" {
		d.ImmichVer = "v3.4.2"
	} else if !strings.HasPrefix(d.ImmichVer, "v") {
		d.ImmichVer = "v" + d.ImmichVer
	}
	d.DockerVer = runCmd("docker --version | awk '{print $3}' | tr -d ','")
	pgCont := runCmd(`docker ps --format "{{.Names}}" | grep -E "immich[_-]postgres" | head -1`)
	if pgCont == "" { pgCont = "immich_postgres" }
	dbU := runCmd(fmt.Sprintf(`docker exec %s env | grep POSTGRES_USER | cut -d= -f2`, pgCont))
	if dbU == "" { dbU = "postgres" }
	dbN := runCmd(fmt.Sprintf(`docker exec %s env | grep POSTGRES_DB | cut -d= -f2`, pgCont))
	if dbN == "" { dbN = "immich" }
	d.PhotoCount = runCmd(fmt.Sprintf(`docker exec %s psql -U %s -d %s -t -A -c "SELECT count(*) FROM asset WHERE type='IMAGE' AND \"deletedAt\" IS NULL;" 2>/dev/null`, pgCont, dbU, dbN))
	d.VideoCount = runCmd(fmt.Sprintf(`docker exec %s psql -U %s -d %s -t -A -c "SELECT count(*) FROM asset WHERE type='VIDEO' AND \"deletedAt\" IS NULL;" 2>/dev/null`, pgCont, dbU, dbN))
	d.UserCount = runCmd(fmt.Sprintf(`docker exec %s psql -U %s -d %s -t -A -c "SELECT count(*) FROM \"user\" WHERE \"deletedAt\" IS NULL;" 2>/dev/null`, pgCont, dbU, dbN))
	if d.PhotoCount == "" { d.PhotoCount = "0" }
	if d.VideoCount == "" { d.VideoCount = "0" }
	if d.UserCount == "" { d.UserCount = "1" }

	dStatsRaw := runCmd("docker stats --no-stream --format '{{.Name}}|{{.CPUPerc}}|{{.MemUsage}}|{{.NetIO}}'")
	statMap := make(map[string][]string)
	for _, ln := range strings.Split(dStatsRaw, "\n") {
		parts := strings.Split(ln, "|")
		if len(parts) == 4 {
			statMap[parts[0]] = parts[1:]
		}
	}

	cNames := []string{"immich_server", "immich_postgres", "immich_redis", "immich_machine_learning"}
	d.AllHealthy = true
	for _, cn := range cNames {
		st := runCmd(fmt.Sprintf("docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' %s", cn))
		if st == "" {
			st = "down"
		}
		if st != "healthy" && st != "running" {
			d.AllHealthy = false
		}
		vals, ok := statMap[cn]
		if !ok {
			vals = []string{"0.00%", "0B / 0B", "0B / 0B"}
		}
		d.Containers = append(d.Containers, ContainerInfo{
			Name:   cn,
			Status: strings.ToUpper(st),
			CPU:    vals[0],
			Mem:    vals[1],
			Net:    vals[2],
		})
	}

	return d
}

func fetchCmd(m *model) tea.Cmd {
	return func() tea.Msg {
		return dataMsg(m.collectData())
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return animMsg(t)
	})
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch strings.ToLower(msg.String()) {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			if !m.fetching {
				m.fetching = true
				m.lastFetch = time.Now()
				return m, fetchCmd(&m)
			}
		}
	case animMsg:
		m.tick++
		if time.Since(m.lastFetch) >= 15*time.Second && !m.fetching {
			m.fetching = true
			return m, tea.Batch(tickCmd(), fetchCmd(&m))
		}
		return m, tickCmd()
	case dataMsg:
		m.data = SysData(msg)
		m.fetching = false
		m.lastFetch = time.Now()
		m.cpuHist = append(m.cpuHist[1:], m.data.CPUPct)
		m.netHist = append(m.netHist[1:], m.data.RxKbps+m.data.TxKbps)
		return m, nil
	}
	return m, nil
}
func makeBar(pct float64, width int) string {
	if pct < 0 { pct = 0 }
	if pct > 100 { pct = 100 }
	filled := int(pct * float64(width) / 100.0)
	empty := width - filled
	col := "#00FF87"
	if pct >= 85 {
		col = "#FF5555"
	} else if pct >= 70 {
		col = "#FFD700"
	}
	bar := lipgloss.NewStyle().Foreground(lipgloss.Color(col)).Render(strings.Repeat("█", filled))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render(strings.Repeat("░", empty))
	txt := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Render(fmt.Sprintf(" %2.0f%%", pct))
	return bar + dim + txt
}

func makeSpark(vals []float64, maxVal float64, color string) string {
	sparks := []rune("·▪▫▬▮▰▲█")
	m := maxVal
	for _, v := range vals {
		if v > m { m = v }
	}
	if m <= 0 { m = 1 }
	var sb strings.Builder
	for _, v := range vals {
		idx := int((v / m) * 7.99)
		if idx < 0 { idx = 0 }
		if idx > 7 { idx = 7 }
		sb.WriteRune(sparks[idx])
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(sb.String())
}

func fmtSpeed(kbps float64) string {
	if kbps >= 1000 {
		return fmt.Sprintf("%.2f Mbps", kbps/1000.0)
	}
	return fmt.Sprintf("%.0f Kbps", kbps)
}

func padCol(s string, w int) string {
	vw := lipgloss.Width(s)
	if vw >= w {
		return s
	}
	return s + strings.Repeat(" ", w-vw)
}

func (m model) View() string {
	d := m.data
	cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")).Bold(true)
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF87")).Bold(true)
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Bold(true)
	magenta := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6")).Bold(true)
	white := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))

	leds := []string{"▰▱▰", "▱▰▱", "▰▰▱", "▱▰▰"}
	beacons := []string{"◉", "◎", "●", "◎"}
	wR := []string{"▸  ", "▸▶ ", "▸▶▶", " ▶▶", "  ▶", "   "}
	wL := []string{"  ◂", " ◀◂", "◀◀◂", "◀◀ ", "◀  ", "   "}
	lStr := green.Render(leds[(m.tick/2)%len(leds)])
	bStr := white.Render(beacons[(m.tick/4)%len(beacons)])

	bannerTxt := fmt.Sprintf(
		"%s %s  %s  %s  %s\n%s   %s   %s   %s   %s",
		lStr, cyan.Render(wR[m.tick%6]),
		white.Render(bStr+"  I M M I C H   P U L S E   •   C L O U D   T U I  "+bStr),
		cyan.Render(wL[m.tick%6]), lStr,
		green.Render(fmt.Sprintf("📸 %s Photos", d.PhotoCount)), dim.Render("│"),
		magenta.Render(fmt.Sprintf("🎬 %s Videos", d.VideoCount)), dim.Render("│"),
		yellow.Render(fmt.Sprintf("👥 %s Users", d.UserCount)),
	)
	banner := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#00FFFF")).
		Padding(0, 3).
		Align(lipgloss.Center).
		Render(bannerTxt)

	hdrLine := fmt.Sprintf("%s    %s    %s    %s    %s    %s    %s",
		cyan.Render("◈ "+strings.ToUpper(d.Hostname)+" ◈"), dim.Render("│"),
		cyan.Render("🐧 "+d.OSName), dim.Render("│"),
		magenta.Render("⚙  Kernel "+d.Kernel), dim.Render("│"),
		green.Render("⏱  Up "+d.Uptime),
	)
	subHdr := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00FFFF")).
		Width(120).
		Align(lipgloss.Center).
		Render(hdrLine)

	quadStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00BFFF")).
		Width(59).
		Padding(0, 1)

	tCol := green
	tTag := "● COOL"
	if d.CPUTemp >= 75 {
		tCol = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Bold(true)
		tTag = "🔥 HOT"
	} else if d.CPUTemp >= 62 {
		tCol = yellow
		tTag = "▲ WARM"
	}

	cpuContent := strings.Join([]string{
		cyan.Render("⚡ PROCESSOR & POWER"),
		padCol(yellow.Render("🧠 Model"), 12) + padCol(white.Render(d.CPUModel), 24) + dim.Render("x86_64"),
		padCol(yellow.Render("⚡ Cores"), 12) + padCol(cyan.Render(fmt.Sprintf("%d Threads", d.CPUCores)), 24) + d.CPUGov,
		padCol(yellow.Render("📊 Usage"), 12) + padCol(makeBar(d.CPUPct, 14), 24) + dim.Render("Load ") + makeSpark(m.cpuHist, 100, "#00FFFF"),
		padCol(yellow.Render("🌡  Temp"), 12) + padCol(tCol.Render(fmt.Sprintf("%.1f °C", d.CPUTemp)), 24) + tCol.Render(tTag),
		padCol(yellow.Render("🔋 Battery"), 12) + padCol(makeBar(float64(d.BatCap), 14), 24) + cyan.Render(fmt.Sprintf("🔌 %s • %s%%", d.BatStat, d.BatLim)),
	}, "\n")

	memContent := strings.Join([]string{
		lipgloss.NewStyle().Foreground(lipgloss.Color("#5F87FF")).Bold(true).Render("💾 MEMORY & STORAGE"),
		padCol(yellow.Render("💾 RAM"), 13) + padCol(makeBar(d.MemPct, 18), 28) + white.Render(fmt.Sprintf("%.1f / %.1f GB", d.MemUsedGB, d.MemTotGB)),
		padCol(yellow.Render("📦 Memory"), 13) + padCol(green.Render(fmt.Sprintf("Avail %.1fG • Cache %.1fG", d.MemAvailGB, d.MemCacheGB)), 28) + fmt.Sprintf("Swap %.1f/%.1fG", d.SwapUsedGB, d.SwapTotGB),
		padCol(yellow.Render("💿 Disk (/)"), 13) + padCol(makeBar(d.DiskPct, 18), 28) + white.Render(fmt.Sprintf("%.0f/%.0fG", d.DiskUsedGB, d.DiskTotGB)),
		padCol(yellow.Render("🔄 Disk I/O"), 13) + padCol(cyan.Render(fmt.Sprintf("▼ Read %.1f GB", d.DiskReadGB)), 28) + magenta.Render(fmt.Sprintf("▲ Write %.1f GB", d.DiskWritGB)),
		padCol(yellow.Render("📈 Sys Load"), 13) + padCol(d.LoadAvg, 28) + dim.Render("1m / 5m / 15m"),
	}, "\n")
	fBadge := green.Render("✓ 0 failed")
	if d.FailSSH > 0 {
		fBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Bold(true).Render(fmt.Sprintf("⚠ %d failed", d.FailSSH))
	}
	netContent := strings.Join([]string{
		cyan.Render("🌐 LOCAL NETWORK"),
		padCol(yellow.Render("📡 WiFi / IF"), 14) + padCol(white.Render(d.SSID), 29) + green.Render("● UP"),
		padCol(yellow.Render("🏠 Local IP"), 14) + padCol(d.LanIP, 29) + dim.Render("LAN IPv4"),
		padCol(yellow.Render("🌍 Public IP"), 14) + padCol(d.WanIP, 29) + dim.Render("WAN IPv4"),
		padCol(yellow.Render("🚀 Traffic"), 14) + padCol(fmt.Sprintf("▼ %s  ▲ %s", fmtSpeed(d.RxKbps), fmtSpeed(d.TxKbps)), 29) + makeSpark(m.netHist, 1000, "#00FF87"),
		padCol(yellow.Render("🔒 Last SSH"), 14) + padCol(d.LastSSH, 29) + fBadge,
	}, "\n")

	tsLines := []string{
		magenta.Render("🔒 TAILSCALE VPN"),
		padCol(yellow.Render("🔗 Status"), 14) + padCol(green.Render("● CONNECTED"), 25) + white.Render(d.TsIP),
		padCol(yellow.Render("🌐 MagicDNS"), 14) + padCol(cyan.Render(d.MagicDNS), 33) + dim.Render("ts.net"),
	}
	for i := 0; i < 3; i++ {
		if i < len(d.Peers) {
			p := d.Peers[i]
			dot := dim.Render("○")
			st := dim.Render("offline")
			if p.Online {
				dot = green.Render("●")
				st = green.Render("online")
			}
			tsLines = append(tsLines, " "+padCol(dot+" "+p.Name, 15)+padCol(p.IP, 27)+st)
		} else {
			tsLines = append(tsLines, "")
		}
	}

	row1 := lipgloss.JoinHorizontal(lipgloss.Top, quadStyle.Render(cpuContent), quadStyle.Render(memContent))
	row2 := lipgloss.JoinHorizontal(lipgloss.Top, quadStyle.Render(netContent), quadStyle.Render(strings.Join(tsLines, "\n")))

	var dRows []string
	dRows = append(dRows, cyan.Render("🐳 DOCKER & IMMICH CONTAINERS"))
	hdrRow := "  " + padCol(dim.Render("Container"), 29) + padCol(dim.Render("Status"), 15) + padCol(dim.Render("Activity"), 19) + padCol(dim.Render("CPU"), 11) + padCol(dim.Render("Memory Usage"), 23) + dim.Render("Network I/O")
	dRows = append(dRows, hdrRow)
	dRows = append(dRows, dim.Render("  "+strings.Repeat("─", 114)))

	for i, c := range d.Containers {
		stCol := green
		if c.Status != "HEALTHY" && c.Status != "RUNNING" {
			stCol = yellow
		}
		act := cyan.Render("·····▸▶▶·····")
		if (m.tick+i)%2 == 0 {
			act = green.Render("····▸▶▶······")
		}
		icon := "📦"
		if strings.Contains(c.Name, "server") {
			icon = "📸"
		} else if strings.Contains(c.Name, "postgres") {
			icon = "🐘"
		} else if strings.Contains(c.Name, "redis") {
			icon = "⚡"
		} else if strings.Contains(c.Name, "machine") {
			icon = "🤖"
		}
		rowStr := "  " +
			padCol(icon+" "+white.Render(c.Name), 29) +
			padCol(stCol.Render("◉ "+c.Status), 15) +
			padCol(act, 19) +
			padCol(c.CPU, 11) +
			padCol(c.Mem, 23) +
			c.Net
		dRows = append(dRows, rowStr)
	}
	dockerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00FFFF")).
		Width(120).
		Padding(0, 1).
		Render(strings.Join(dRows, "\n"))

	spinners := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	spin := spinners[(m.tick/2)%len(spinners)]
	rem := 15 - int(time.Since(m.lastFetch).Seconds())
	if rem < 1 { rem = 1 }

	sysStat := green.Render("● ALL SYSTEMS OPERATIONAL")
	if !d.AllHealthy {
		sysStat = yellow.Render("▲ SOME SERVICES DEGRADED")
	}
	leftFtr := fmt.Sprintf(" %s│%s│%s",
		sysStat, cyan.Render(fmt.Sprintf(" %s %02ds ", spin, rem)), yellow.Render(" 'r' refresh • 'q' exit"),
	)
	rightFtr := fmt.Sprintf("%s • %s • %s ",
		yellow.Render("by Bijaya Pokhrel"), cyan.Render("Immich "+d.ImmichVer), magenta.Render("Docker "+d.DockerVer),
	)
	gap := 118 - lipgloss.Width(leftFtr) - lipgloss.Width(rightFtr)
	if gap < 1 { gap = 1 }

	footerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00FF87")).
		Width(120).
		Render(leftFtr + strings.Repeat(" ", gap) + rightFtr)

	ui := lipgloss.JoinVertical(lipgloss.Center, banner, subHdr, row1, row2, dockerBox, footerBox)
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, ui)
	}
	return ui
}

func main() {
	m := model{
		cpuHist:   make([]float64, 12),
		netHist:   make([]float64, 12),
		lastFetch: time.Now(),
	}
	m.data = m.collectData()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
