package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// Telemetría en vivo sin drivers de terceros:
//   - CPU (frecuencia real) y disco: contadores de rendimiento de Windows (PDH).
//   - GPU NVIDIA: nvidia-smi (viene con el driver).
//   - Discos: GetDiskFreeSpaceEx.
// La temperatura de CPU NO se muestra: Windows no la expone sin un driver de hardware, y las
// "zonas térmicas" ACPI miden otra cosa (suelen dar ~28 °C), así que sería un dato falso.

// ---------- PDH ----------

var (
	pdhDLL        = syscall.NewLazyDLL("pdh.dll")
	pdhOpenQuery  = pdhDLL.NewProc("PdhOpenQueryW")
	pdhAddEnglish = pdhDLL.NewProc("PdhAddEnglishCounterW")
	pdhCollect    = pdhDLL.NewProc("PdhCollectQueryData")
	pdhGetValue   = pdhDLL.NewProc("PdhGetFormattedCounterValue")
)

const pdhFmtDouble = 0x200

type pdhValue struct {
	Status uint32
	_      uint32
	Double float64
}

type pdhSet struct {
	mu       sync.Mutex
	query    uintptr
	counters map[string]uintptr
	ok       bool
	tried    bool
}

var pdhs pdhSet

func (p *pdhSet) init(paths ...string) {
	if p.tried {
		return
	}
	p.tried = true
	if r, _, _ := pdhOpenQuery.Call(0, 0, uintptr(unsafe.Pointer(&p.query))); r != 0 {
		return
	}
	p.counters = map[string]uintptr{}
	for _, path := range paths {
		var c uintptr
		ptr, _ := syscall.UTF16PtrFromString(path)
		if r, _, _ := pdhAddEnglish.Call(p.query, uintptr(unsafe.Pointer(ptr)), 0, uintptr(unsafe.Pointer(&c))); r == 0 {
			p.counters[path] = c
		}
	}
	pdhCollect.Call(p.query)
	p.ok = len(p.counters) > 0
}

const (
	ctrFreq = `\Processor Information(_Total)\Processor Frequency`
	ctrPerf = `\Processor Information(_Total)\% Processor Performance`
	ctrDisk = `\PhysicalDisk(_Total)\% Disk Time`
)

// sample devuelve los valores actuales de los contadores (los de tasa necesitan dos lecturas).
func (p *pdhSet) sample() map[string]float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.init(ctrFreq, ctrPerf, ctrDisk)
	out := map[string]float64{}
	if !p.ok {
		return out
	}
	pdhCollect.Call(p.query)
	for path, c := range p.counters {
		var typ uint32
		var v pdhValue
		if r, _, _ := pdhGetValue.Call(c, pdhFmtDouble, uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&v))); r == 0 && v.Status == 0 {
			out[path] = v.Double
		}
	}
	return out
}

// ---------- GPU NVIDIA ----------

type GPULive struct {
	Name      string  `json:"name"`
	Temp      int     `json:"temp"` // °C, -1 si no hay dato
	Util      int     `json:"util"` // %
	VramUsed  int     `json:"vramUsed"`
	VramTotal int     `json:"vramTotal"` // MB
	Clock     int     `json:"clock"`     // MHz
	ClockMax  int     `json:"clockMax"`
	Power     float64 `json:"power"` // W, -1 si no hay dato
	Fan       int     `json:"fan"`   // %, -1 si no hay dato
	Driver    string  `json:"driver"`
}

var (
	smiOnce sync.Once
	smiPath string
)

func findSMI() string {
	smiOnce.Do(func() {
		if p, err := exec.LookPath("nvidia-smi"); err == nil {
			smiPath = p
			return
		}
		for _, c := range []string{
			os.Getenv("SystemRoot") + `\System32\nvidia-smi.exe`,
			os.Getenv("ProgramFiles") + `\NVIDIA Corporation\NVSMI\nvidia-smi.exe`,
		} {
			if _, err := os.Stat(c); err == nil {
				smiPath = c
				return
			}
		}
	})
	return smiPath
}

func atoiOr(s string, def int) int {
	s = strings.TrimSpace(s)
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int(f)
	}
	return def
}

func parseSMI(line string) (*GPULive, error) {
	f := strings.Split(strings.TrimSpace(line), ",")
	if len(f) < 10 {
		return nil, errors.New("salida inesperada de nvidia-smi")
	}
	power := -1.0
	if v, err := strconv.ParseFloat(strings.TrimSpace(f[7]), 64); err == nil {
		power = v
	}
	return &GPULive{
		Name: strings.TrimSpace(f[0]), Temp: atoiOr(f[1], -1), Util: atoiOr(f[2], 0),
		VramUsed: atoiOr(f[3], 0), VramTotal: atoiOr(f[4], 0), Clock: atoiOr(f[5], 0), ClockMax: atoiOr(f[6], 0),
		Power: power, Fan: atoiOr(f[8], -1), Driver: strings.TrimSpace(f[9]),
	}, nil
}

func nvidiaLive() *GPULive {
	p := findSMI()
	if p == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, p, "--query-gpu=name,temperature.gpu,utilization.gpu,memory.used,memory.total,clocks.sm,clocks.max.sm,power.draw,fan.speed,driver_version",
		"--format=csv,noheader,nounits")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	line, _, _ := strings.Cut(string(out), "\n") // primera GPU
	g, err := parseSMI(line)
	if err != nil {
		return nil
	}
	return g
}

// ---------- discos ----------

type DiskInfo struct {
	Letter  string  `json:"letter"`
	TotalGB float64 `json:"totalGB"`
	FreeGB  float64 `json:"freeGB"`
	UsedPct float64 `json:"usedPct"`
}

var procGetDriveType = kernel32.NewProc("GetDriveTypeW")
var procGetDiskFree = kernel32.NewProc("GetDiskFreeSpaceExW")

func disks() []DiskInfo {
	mask, _, _ := kernel32.NewProc("GetLogicalDrives").Call()
	var out []DiskInfo
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		root := string(rune('A'+i)) + `:\`
		rp, _ := syscall.UTF16PtrFromString(root)
		if t, _, _ := procGetDriveType.Call(uintptr(unsafe.Pointer(rp))); t != 3 { // 3 = disco fijo
			continue
		}
		var avail, total, free uint64
		r, _, _ := procGetDiskFree.Call(uintptr(unsafe.Pointer(rp)), uintptr(unsafe.Pointer(&avail)), uintptr(unsafe.Pointer(&total)), uintptr(unsafe.Pointer(&free)))
		if r == 0 || total == 0 {
			continue
		}
		out = append(out, DiskInfo{Letter: root[:2], TotalGB: float64(total) / (1 << 30), FreeGB: float64(free) / (1 << 30),
			UsedPct: 100 * (1 - float64(free)/float64(total))})
	}
	return out
}

// ---------- vivo ----------

// enrichLive completa el uso básico con frecuencia, disco, discos y GPU.
func enrichLive(l *Live) {
	s := pdhs.sample()
	if f, ok := s[ctrFreq]; ok {
		perf := 100.0
		if p, ok := s[ctrPerf]; ok && p > 0 {
			perf = p
		}
		l.CPUMHz = int(f * perf / 100)
	}
	l.DiskBusy = s[ctrDisk]
	if l.DiskBusy > 100 {
		l.DiskBusy = 100
	}
	l.Disks = disks()
	l.GPU = nvidiaLive()
}

// ---------- detalle del procesador ----------

var procGLPIEx = kernel32.NewProc("GetLogicalProcessorInformationEx")

type CPUDetail struct {
	Cores   int  `json:"cores"`   // núcleos físicos
	Threads int  `json:"threads"` // procesadores lógicos
	L2KB    int  `json:"l2KB"`
	L3KB    int  `json:"l3KB"`
	BaseMHz int  `json:"baseMHz"`
	Hybrid  bool `json:"hybrid"` // núcleos de distinta clase de eficiencia (P/E)
}

func cpuDetail() CPUDetail {
	d := CPUDetail{}
	d.Threads = runtimeCPUs()
	// núcleos y cachés
	var size uint32
	procGLPIEx.Call(0xffff, 0, uintptr(unsafe.Pointer(&size))) // RelationAll: solo para conocer el tamaño
	if size == 0 {
		return d
	}
	buf := make([]byte, size)
	if r, _, _ := procGLPIEx.Call(0xffff, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size))); r == 0 {
		return d
	}
	for off := uint32(0); off+8 <= size; {
		rel := *(*uint32)(unsafe.Pointer(&buf[off]))
		sz := *(*uint32)(unsafe.Pointer(&buf[off+4]))
		if sz == 0 {
			break
		}
		switch rel {
		case 0: // RelationProcessorCore
			d.Cores++
			if buf[off+9] > 0 { // PROCESSOR_RELATIONSHIP.EfficiencyClass
				d.Hybrid = true
			}
		case 2: // RelationCache
			level := buf[off+8]
			cacheSize := *(*uint32)(unsafe.Pointer(&buf[off+12]))
			typ := *(*uint32)(unsafe.Pointer(&buf[off+16]))
			if typ == 0 || typ == 2 { // unificada o de datos
				if level == 2 {
					d.L2KB += int(cacheSize / 1024)
				} else if level == 3 {
					d.L3KB += int(cacheSize / 1024)
				}
			}
		}
		off += sz
	}
	return d
}
