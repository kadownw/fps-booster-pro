package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procGlobalMem    = kernel32.NewProc("GlobalMemoryStatusEx")
	procGetSysTimes  = kernel32.NewProc("GetSystemTimes")
	gpuClassKeyPath  = `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}`
	cpuKeyPath       = `HARDWARE\DESCRIPTION\System\CentralProcessor\0`
	winVersionKeyPth = `SOFTWARE\Microsoft\Windows NT\CurrentVersion`
)

type memStatus struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys, AvailPhys uint64
	TotalPage, AvailPage uint64
	TotalVirt, AvailVirt uint64
	AvailExtVirt         uint64
}

func memory() (total, avail uint64, load uint32) {
	var m memStatus
	m.Length = uint32(unsafe.Sizeof(m))
	r, _, _ := procGlobalMem.Call(uintptr(unsafe.Pointer(&m)))
	if r == 0 {
		return 0, 0, 0
	}
	return m.TotalPhys, m.AvailPhys, m.MemoryLoad
}

type SysInfo struct {
	CPU     string    `json:"cpu"`
	Cores   int       `json:"cores"`
	GPUs    []string  `json:"gpus"`
	RAMGB   float64   `json:"ramGB"`
	Windows string    `json:"windows"`
	Admin   bool      `json:"admin"`
	User    string    `json:"user"`
	CPUInfo CPUDetail `json:"cpuInfo"`
}

func regString(root registry.Key, path, name string) string {
	k, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	v, _, _ := k.GetStringValue(name)
	return strings.TrimSpace(v)
}

func gpuAdapters() []Adapter {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, gpuClassKeyPath, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil
	}
	defer k.Close()
	names, _ := k.ReadSubKeyNames(-1)
	var out []Adapter
	for _, n := range names {
		if _, err := strconv.Atoi(n); err != nil {
			continue // "Properties", etc.
		}
		sub := gpuClassKeyPath + `\` + n
		desc := regString(registry.LOCAL_MACHINE, sub, "DriverDesc")
		if desc == "" || strings.Contains(desc, "Basic") || strings.Contains(desc, "Remote") {
			continue
		}
		label := desc
		if sk, err := registry.OpenKey(registry.LOCAL_MACHINE, sub, registry.QUERY_VALUE); err == nil {
			if v, _, err := sk.GetIntegerValue("HardwareInformation.qwMemorySize"); err == nil && v > 0 {
				label = fmt.Sprintf("%s (%.0f GB VRAM)", desc, float64(v)/(1<<30))
			}
			sk.Close()
		}
		out = append(out, Adapter{Sub: n, Label: label, Vendor: vendorOf(desc)})
	}
	return out
}

func gpus() []string {
	var out []string
	for _, a := range gpuAdapters() {
		out = append(out, a.Label)
	}
	return out
}

// Adapter es un adaptador de video tal como figura en la clase Display del registro.
type Adapter struct {
	Sub    string // "0000", "0001"...
	Label  string
	Vendor string
}

func vendorOf(desc string) string {
	d := strings.ToLower(desc)
	switch {
	case strings.Contains(d, "nvidia"), strings.Contains(d, "geforce"), strings.Contains(d, "quadro"):
		return "nvidia"
	case strings.Contains(d, "amd"), strings.Contains(d, "radeon"):
		return "amd"
	case strings.Contains(d, "intel"):
		return "intel"
	}
	return ""
}

func windowsBuild() int {
	b, _ := strconv.Atoi(regString(registry.LOCAL_MACHINE, winVersionKeyPth, "CurrentBuild"))
	return b
}

// supported indica si el tweak aplica a esta versión de Windows y a este hardware.
func supported(t Tweak) bool {
	b := windowsBuild()
	if (t.MinBuild > 0 && b < t.MinBuild) || (t.MaxBuild > 0 && b > t.MaxBuild) {
		return false
	}
	return t.Power || len(expand(t)) > 0 || len(existingTasks(t)) > 0
}

func windowsName() string {
	name := regString(registry.LOCAL_MACHINE, winVersionKeyPth, "ProductName")
	build := regString(registry.LOCAL_MACHINE, winVersionKeyPth, "CurrentBuild")
	if b, err := strconv.Atoi(build); err == nil && b >= 22000 {
		name = strings.Replace(name, "Windows 10", "Windows 11", 1)
	}
	ver := regString(registry.LOCAL_MACHINE, winVersionKeyPth, "DisplayVersion")
	return strings.TrimSpace(fmt.Sprintf("%s %s (build %s)", name, ver, build))
}

func systemInfo() SysInfo {
	total, _, _ := memory()
	return SysInfo{
		CPU:     regString(registry.LOCAL_MACHINE, cpuKeyPath, "ProcessorNameString"),
		Cores:   runtime.NumCPU(),
		GPUs:    gpus(),
		RAMGB:   float64(total) / (1 << 30),
		Windows: windowsName(),
		Admin:   isAdmin(),
		User:    os.Getenv("USERNAME"),
		CPUInfo: cpuDetailWithBase(),
	}
}

func runtimeCPUs() int { return runtime.NumCPU() }

func cpuDetailWithBase() CPUDetail {
	d := cpuDetail()
	if k, err := registry.OpenKey(registry.LOCAL_MACHINE, cpuKeyPath, registry.QUERY_VALUE); err == nil {
		if v, _, err := k.GetIntegerValue("~MHz"); err == nil {
			d.BaseMHz = int(v)
		}
		k.Close()
	}
	return d
}

// ---- uso en vivo ----

type Live struct {
	CPU      float64    `json:"cpu"`
	RAM      float64    `json:"ram"`
	RAMUsed  float64    `json:"ramUsedGB"`
	RAMTotal float64    `json:"ramTotalGB"`
	CPUMHz   int        `json:"cpuMHz"`
	DiskBusy float64    `json:"diskBusy"`
	Disks    []DiskInfo `json:"disks"`
	GPU      *GPULive   `json:"gpu"`
}

var (
	liveMu                 sync.Mutex
	prevIdle, prevK, prevU uint64
)

type fileTime struct{ Lo, Hi uint32 }

func (f fileTime) u64() uint64 { return uint64(f.Hi)<<32 | uint64(f.Lo) }

func liveUsage() Live {
	var idle, kern, user fileTime
	procGetSysTimes.Call(uintptr(unsafe.Pointer(&idle)), uintptr(unsafe.Pointer(&kern)), uintptr(unsafe.Pointer(&user)))
	liveMu.Lock()
	defer liveMu.Unlock()
	i, k, u := idle.u64(), kern.u64(), user.u64()
	var cpu float64
	if prevK != 0 {
		dIdle, dTotal := i-prevIdle, (k-prevK)+(u-prevU) // kernel incluye idle
		if dTotal > 0 {
			cpu = 100 * (1 - float64(dIdle)/float64(dTotal))
		}
	}
	prevIdle, prevK, prevU = i, k, u
	total, avail, load := memory()
	return Live{CPU: cpu, RAM: float64(load), RAMTotal: float64(total) / (1 << 30), RAMUsed: float64(total-avail) / (1 << 30)}
}
