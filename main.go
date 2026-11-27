package main
import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
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
const appVersion = "v3.0.0"
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
	HasBat     bool
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
	PrevRx     uint64
	PrevTx     uint64
	PrevNetTs  time.Time
	WanCache   string
	WanCacheTs time.Time
}
type animMsg time.Time
type dataMsg SysData
type model struct {
	data        SysData
	tick        int
	lastFetch   time.Time
	fetching    bool
	privacy     bool
	refreshSecs int
	cpuHist     []float64
	netHist     []float64
	width       int
	height      int
}
func runCmdTimeout(timeout time.Duration, cmd string) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "sh", "-c", cmd).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
func collectData(prev SysData) SysData {
	d := SysData{
		PrevRx:     prev.PrevRx,
		PrevTx:     prev.PrevTx,
		PrevNetTs:  prev.PrevNetTs,
		WanCache:   prev.WanCache,
		WanCacheTs: prev.WanCacheTs,
	}
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		if hn, err := os.Hostname(); err == nil && hn != "" {
			d.Hostname = hn
		} else {
			d.Hostname = "HOST"
		}
		d.OSName = runCmdTimeout(time.Second, `grep PRETTY_NAME /etc/os-release 2>/dev/null | cut -d'"' -f2`)
		if d.OSName == "" {
			d.OSName = "Linux"
		}
		d.Kernel = runCmdTimeout(time.Second, "uname -r")
		if hInfo, err := host.Info(); err == nil && hInfo != nil {
			hrs := hInfo.Uptime / 3600
			mins := (hInfo.Uptime % 3600) / 60
			d.Uptime = fmt.Sprintf("%dh %dm", hrs, mins)
		} else {
			d.Uptime = "N/A"
		}
		d.CPUModel = runCmdTimeout(time.Second, `lscpu 2>/dev/null | grep 'Model name:' | sed 's/Model name:[[:space:]]*//'`)
		d.CPUModel = strings.ReplaceAll(d.CPUModel, " with Radeon Graphics", "")
		if len(d.CPUModel) > 28 {
			d.CPUModel = d.CPUModel[:28]
		}
		if d.CPUModel == "" {
			d.CPUModel = "CPU"
		}
		if cores, err := cpu.Counts(true); err == nil {
			d.CPUCores = cores
		}
		d.CPUGov = runCmdTimeout(500*time.Millisecond, "cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor 2>/dev/null")
		if d.CPUGov == "" {
			d.CPUGov = "default"
		}
		if cPcts, err := cpu.Percent(0, false); err == nil && len(cPcts) > 0 {
			d.CPUPct = cPcts[0]
		}
		tempStr := runCmdTimeout(500*time.Millisecond, "cat /sys/class/hwmon/hwmon*/temp1_input 2>/dev/null | sort -nr | head -1")
		if tVal, err := strconv.ParseFloat(tempStr, 64); err == nil && tVal > 0 {
			d.CPUTemp = tVal / 1000.0
			d.HasTemp = true
		}
		batStr := runCmdTimeout(500*time.Millisecond, "cat /sys/class/power_supply/BAT*/capacity 2>/dev/null | head -1")
		if batStr != "" {
			d.HasBat = true
			d.BatCap, _ = strconv.Atoi(batStr)
			d.BatStat = runCmdTimeout(500*time.Millisecond, "cat /sys/class/power_supply/BAT*/status 2>/dev/null | head -1")
			if d.BatStat == "" {
				d.BatStat = "AC"
			}
			d.BatLim = runCmdTimeout(500*time.Millisecond, "cat /sys/class/power_supply/BAT*/charge_control_end_threshold 2>/dev/null | head -1")
			if d.BatLim == "" {
				d.BatLim = "100"
			}
		}
	}()
	go func() {
		defer wg.Done()
		if vMem, err := mem.VirtualMemory(); err == nil && vMem != nil {
			d.MemPct = vMem.UsedPercent
			d.MemUsedGB = float64(vMem.Used) / (1024 * 1024 * 1024)
			d.MemTotGB = float64(vMem.Total) / (1024 * 1024 * 1024)
			d.MemAvailGB = float64(vMem.Available) / (1024 * 1024 * 1024)
			d.MemCacheGB = float64(vMem.Cached) / (1024 * 1024 * 1024)
		}
		if sMem, err := mem.SwapMemory(); err == nil && sMem != nil {
			d.SwapUsedGB = float64(sMem.Used) / (1024 * 1024 * 1024)
			d.SwapTotGB = float64(sMem.Total) / (1024 * 1024 * 1024)
		}
		if dUsage, err := disk.Usage("/"); err == nil && dUsage != nil {
			d.DiskPct = dUsage.UsedPercent
			d.DiskUsedGB = float64(dUsage.Used) / (1024 * 1024 * 1024)
			d.DiskTotGB = float64(dUsage.Total) / (1024 * 1024 * 1024)
			d.DiskFreeGB = float64(dUsage.Free) / (1024 * 1024 * 1024)
		}
		if dIO, err := disk.IOCounters(); err == nil {
			for _, v := range dIO {
				d.DiskReadGB += float64(v.ReadBytes) / (1024 * 1024 * 1024)
				d.DiskWritGB += float64(v.WriteBytes) / (1024 * 1024 * 1024)
			}
		}
		if avg, err := load.Avg(); err == nil && avg != nil {
			d.LoadAvg = fmt.Sprintf("%.2f / %.2f / %.2f", avg.Load1, avg.Load5, avg.Load15)
		} else {
			d.LoadAvg = "0.00 / 0.00 / 0.00"
		}
	}()
	go func() {
		defer wg.Done()
		d.Iface = runCmdTimeout(time.Second, "ip route 2>/dev/null | grep default | awk '{print $5}' | head -1")
		if d.Iface != "" {
			d.LanIP = runCmdTimeout(time.Second, fmt.Sprintf(`ip -4 addr show %s 2>/dev/null | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | head -1`, d.Iface))
		}
		if d.LanIP == "" {
			d.LanIP = "N/A"
		}
		d.SSID = runCmdTimeout(time.Second, "nmcli -t -f active,ssid dev wifi 2>/dev/null | grep '^yes' | cut -d':' -f2")
		if d.SSID == "" {
			if d.Iface != "" {
				d.SSID = d.Iface
			} else {
				d.SSID = "Ethernet"
			}
		}
		if time.Since(d.WanCacheTs) > 5*time.Minute || d.WanCache == "" {
			wan := runCmdTimeout(2*time.Second, "curl -4 -s --max-time 2 ifconfig.me || curl -4 -s --max-time 2 icanhazip.com")
			if wan != "" && len(wan) < 45 {
				d.WanCache = wan
				d.WanCacheTs = time.Now()
			}
		}
		d.WanIP = d.WanCache
		if d.WanIP == "" {
			d.WanIP = "N/A"
		}
		if nIO, err := net.IOCounters(true); err == nil {
			now := time.Now()
			for _, nic := range nIO {
				if nic.Name == d.Iface || (d.Iface == "" && nic.Name != "lo" && !strings.HasPrefix(nic.Name, "docker") && !strings.HasPrefix(nic.Name, "br-") && !strings.HasPrefix(nic.Name, "veth")) {
					if !d.PrevNetTs.IsZero() && nic.BytesRecv >= d.PrevRx && nic.BytesSent >= d.PrevTx {
						dt := now.Sub(d.PrevNetTs).Seconds()
						if dt > 0 {
							d.RxKbps = float64(nic.BytesRecv-d.PrevRx) * 8 / 1000 / dt
							d.TxKbps = float64(nic.BytesSent-d.PrevTx) * 8 / 1000 / dt
						}
					}
					d.PrevRx = nic.BytesRecv
					d.PrevTx = nic.BytesSent
					d.PrevNetTs = now
					break
				}
			}
		}
		u := os.Getenv("USER")
		if u != "" {
			d.LastSSH = runCmdTimeout(time.Second, "who 2>/dev/null | grep "+u+" | tail -1 | awk '{print $3, $4, $5}' | tr -d '()'")
		}
		if d.LastSSH == "" {
			d.LastSSH = "Active Session"
		}
		fStr := runCmdTimeout(2*time.Second, "journalctl -u ssh -u sshd --since '24 hours ago' 2>/dev/null | grep -Ec 'Failed password|Invalid user'")
		d.FailSSH, _ = strconv.Atoi(fStr)

		d.TsIP = runCmdTimeout(time.Second, "tailscale ip -4 2>/dev/null | head -1")
		if d.TsIP == "" {
			d.TsIP = "Disconnected"
		}
		tsRaw := runCmdTimeout(2*time.Second, "tailscale status --json 2>/dev/null")
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
	}()
go func() {
defer wg.Done()
srvCont := runCmdTimeout(time.Second, `docker ps --format "{{.Names}}" 2>/dev/null | grep -E "immich[_-]server" | head -1`)
if srvCont != "" {
verJSON := runCmdTimeout(2*time.Second, fmt.Sprintf(`docker exec %s sh -c 'curl -sf http://localhost:2283/api/server/version || wget -qO- http://localhost:2283/api/server/version' 2>/dev/null`, srvCont))
var vObj struct {
Major int `json:"major"`
Minor int `json:"minor"`
Patch int `json:"patch"`
}
if json.Unmarshal([]byte(verJSON), &vObj) == nil && (vObj.Major > 0 || vObj.Minor > 0) {
d.ImmichVer = fmt.Sprintf("v%d.%d.%d", vObj.Major, vObj.Minor, vObj.Patch)
}
}
if d.ImmichVer == "" {
d.ImmichVer = "N/A"
}
d.DockerVer = runCmdTimeout(time.Second, "docker --version 2>/dev/null | awk '{print $3}' | tr -d ','")
if d.DockerVer == "" {
d.DockerVer = "N/A"
}
pgCont := runCmdTimeout(time.Second, `docker ps --format "{{.Names}}" 2>/dev/null | grep -E "immich[_-]postgres" | head -1`)
if pgCont != "" {
dbU := runCmdTimeout(time.Second, fmt.Sprintf(`docker exec %s env 2>/dev/null | grep POSTGRES_USER | cut -d= -f2`, pgCont))
if dbU == "" {
dbU = "postgres"
}
dbN := runCmdTimeout(time.Second, fmt.Sprintf(`docker exec %s env 2>/dev/null | grep POSTGRES_DB | cut -d= -f2`, pgCont))
if dbN == "" {
dbN = "immich"
}
sqlCmd := fmt.Sprintf(`printf 'SELECT (SELECT count(*) FROM asset WHERE type='\''IMAGE'\'' AND "deletedAt" IS NULL), (SELECT count(*) FROM asset WHERE type='\''VIDEO'\'' AND "deletedAt" IS NULL), (SELECT count(*) FROM "user" WHERE "deletedAt" IS NULL);\n' | docker exec -i %s psql -U %s -d %s -t -A -F'|' 2>/dev/null`, pgCont, dbU, dbN)
res := runCmdTimeout(2*time.Second, sqlCmd)
parts := strings.Split(res, "|")
if len(parts) == 3 {
d.PhotoCount, d.VideoCount, d.UserCount = parts[0], parts[1], parts[2]
}
}
		if d.PhotoCount == "" {
			d.PhotoCount = "0"
		}
		if d.VideoCount == "" {
			d.VideoCount = "0"
		}
		if d.UserCount == "" {
			d.UserCount = "0"
		}
		dStatsRaw := runCmdTimeout(3*time.Second, "docker stats --no-stream --format '{{.Name}}|{{.CPUPerc}}|{{.MemUsage}}|{{.NetIO}}' 2>/dev/null")
		statMap := make(map[string][]string)
		for _, ln := range strings.Split(dStatsRaw, "\n") {
			parts := strings.Split(ln, "|")
			if len(parts) == 4 {
				statMap[parts[0]] = parts[1:]
			}
		}
		baseNames := []string{"immich_server", "immich_postgres", "immich_redis", "immich_machine_learning"}
		d.AllHealthy = true
		for _, bn := range baseNames {
			alt := strings.ReplaceAll(bn, "_", "-")
			cn := runCmdTimeout(time.Second, fmt.Sprintf(`docker ps -a --format "{{.Names}}" 2>/dev/null | grep -E "^(%s|%s)(-[0-9]+)?$" | head -1`, bn, alt))
			if cn == "" {
				cn = bn
			}
			st := runCmdTimeout(time.Second, fmt.Sprintf("docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' %s 2>/dev/null", cn))
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
	}()
	wg.Wait()
	return d
}
func fetchCmd(prev SysData) tea.Cmd {
	return func() tea.Msg {
		return dataMsg(collectData(prev))
	}
}
func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return animMsg(t)
	})
}
func (m model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), fetchCmd(m.data))
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
		case "p":
			m.privacy = !m.privacy
		case "r":
			if !m.fetching {
				m.fetching = true
				m.lastFetch = time.Now()
				return m, fetchCmd(m.data)
			}
		}
	case animMsg:
		m.tick++
		if time.Since(m.lastFetch) >= time.Duration(m.refreshSecs)*time.Second && !m.fetching {
			m.fetching = true
			return m, tea.Batch(tickCmd(), fetchCmd(m.data))
		}
		return m, tickCmd()
	case dataMsg:
		m.data = SysData(msg)
		m.fetching = false
		m.lastFetch = time.Now()
		if len(m.cpuHist) > 0 {
			m.cpuHist = append(m.cpuHist[1:], m.data.CPUPct)
		}
		if len(m.netHist) > 0 {
			m.netHist = append(m.netHist[1:], m.data.RxKbps+m.data.TxKbps)
		}
		return m, nil
	}
	return m, nil
}
func makeBar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
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
		if v > m {
			m = v
		}
	}
	if m <= 0 {
		m = 1
	}
	var sb strings.Builder
	for _, v := range vals {
		idx := int((v / m) * 7.99)
		if idx < 0 {
			idx = 0
		}
		if idx > 7 {
			idx = 7
		}
		sb.WriteRune(sparks[idx])
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(sb.String())
}
func maskIP(s string, hide bool) string {
	if !hide || s == "" || s == "N/A" || s == "Disconnected" {
		return s
	}
	return "•••.•••.•••.•••"
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
	if !d.HasTemp {
		tCol = dim
		tTag = "○ N/A"
	} else if d.CPUTemp >= 75 {
		tCol = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Bold(true)
		tTag = "🔥 HOT"
	} else if d.CPUTemp >= 62 {
		tCol = yellow
		tTag = "▲ WARM"
	}
	batRow := padCol(yellow.Render("🔋 Power"), 12) + padCol(green.Render("⚡ AC Mains"), 24) + dim.Render("Desktop / VPS")
	if d.HasBat {
		batRow = padCol(yellow.Render("🔋 Battery"), 12) + padCol(makeBar(float64(d.BatCap), 14), 24) + cyan.Render(fmt.Sprintf("🔌 %s • %s%%", d.BatStat, d.BatLim))
	}
	cpuContent := strings.Join([]string{
		cyan.Render("⚡ PROCESSOR & POWER"),
		padCol(yellow.Render("🧠 Model"), 12) + padCol(white.Render(d.CPUModel), 24) + dim.Render("x86_64"),
		padCol(yellow.Render("⚡ Cores"), 12) + padCol(cyan.Render(fmt.Sprintf("%d Threads", d.CPUCores)), 24) + d.CPUGov,
		padCol(yellow.Render("📊 Usage"), 12) + padCol(makeBar(d.CPUPct, 14), 24) + dim.Render("Load ") + makeSpark(m.cpuHist, 100, "#00FFFF"),
		padCol(yellow.Render("🌡  Temp"), 12) + padCol(tCol.Render(fmt.Sprintf("%.1f °C", d.CPUTemp)), 24) + tCol.Render(tTag),
		batRow,
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
		padCol(yellow.Render("🏠 Local IP"), 14) + padCol(maskIP(d.LanIP, m.privacy), 29) + dim.Render("LAN IPv4"),
		padCol(yellow.Render("🌍 Public IP"), 14) + padCol(maskIP(d.WanIP, m.privacy), 29) + dim.Render("WAN IPv4"),
		padCol(yellow.Render("🚀 Traffic"), 14) + padCol(fmt.Sprintf("▼ %s  ▲ %s", fmtSpeed(d.RxKbps), fmtSpeed(d.TxKbps)), 29) + makeSpark(m.netHist, 1000, "#00FF87"),
		padCol(yellow.Render("🔒 Last SSH"), 14) + padCol(d.LastSSH, 29) + fBadge,
	}, "\n")
	tsStatus := green.Render("● CONNECTED")
	if d.TsIP == "Disconnected" {
		tsStatus = dim.Render("○ OFFLINE")
	}
	tsLines := []string{
		magenta.Render("🔒 TAILSCALE VPN"),
		padCol(yellow.Render("🔗 Status"), 14) + padCol(tsStatus, 25) + white.Render(maskIP(d.TsIP, m.privacy)),
		padCol(yellow.Render("🌐 MagicDNS"), 14) + padCol(cyan.Render(d.MagicDNS), 33) + dim.Render("Tailnet"),
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
			tsLines = append(tsLines, " "+padCol(dot+" "+p.Name, 15)+padCol(maskIP(p.IP, m.privacy), 27)+st)
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
	fwR := []string{
		cyan.Render("▸") + dim.Render("▸▸"),
		cyan.Render("▸") + white.Render("▶") + dim.Render("▸"),
		green.Render("▸") + cyan.Render("▶") + white.Render("▶"),
		dim.Render("▸") + green.Render("▶") + white.Render("▶"),
		dim.Render("▸▸") + white.Render("▶"),
		dim.Render("▸▸▸"),
	}
	fwL := []string{
		dim.Render("◂◂") + cyan.Render("◂"),
		dim.Render("◂") + white.Render("◀") + cyan.Render("◂"),
		white.Render("◀") + cyan.Render("◀") + green.Render("◂"),
		white.Render("◀") + green.Render("◀") + dim.Render("◂"),
		white.Render("◀") + dim.Render("◂◂"),
		dim.Render("◂◂◂"),
	}
	sysStat := green.Render("● OPERATIONAL")
	if m.fetching {
		sysStat = cyan.Render("⚡ SYNCING")
	} else if !d.AllHealthy {
		sysStat = yellow.Render("▲ DEGRADED")
	}
	leftFtr := fmt.Sprintf(" %s %s %s │ %s", lStr, fwR[m.tick%6], sysStat, yellow.Render("'r' sync • 'p' hide • 'q' quit"))
	rightFtr := fmt.Sprintf("%s • %s • %s %s %s ", yellow.Render("by Bijaya Pokhrel"), cyan.Render("Immich "+d.ImmichVer), magenta.Render("Docker "+d.DockerVer), fwL[m.tick%6], lStr)
	gap := 118 - lipgloss.Width(leftFtr) - lipgloss.Width(rightFtr)
	if gap < 1 {
		gap = 1
	}
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
	showVer := flag.Bool("version", false, "Print ImmichPulse version")
	refresh := flag.Int("refresh", 15, "Metrics refresh interval in seconds")
	flag.Parse()
	if *showVer {
		fmt.Println("ImmichPulse", appVersion)
		os.Exit(0)
	}
	if *refresh < 1 {
		*refresh = 1
	}
	m := model{
		cpuHist:     make([]float64, 12),
		netHist:     make([]float64, 12),
		lastFetch:   time.Now(),
		refreshSecs: *refresh,
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
