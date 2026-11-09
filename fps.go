package main

import (
	"errors"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Medición de FPS sin programas externos: se abre una sesión ETW en tiempo real, se activa el
// proveedor Microsoft-Windows-DXGI y se cuenta el evento «Present» (id 42) del proceso del juego.
// Es la misma fuente que usan PresentMon y CapFrameX. Solo LEE eventos: no toca el juego.
// Requiere administrador (o el grupo «Performance Log Users»). Cubre juegos DirectX 9Ex/10/11/12;
// los que dibujan con Vulkan u OpenGL puro no emiten este evento y quedan sin FPS.

const (
	etwSession       = "SouthTweaksFPS"
	etwRealTime      = 0x00000100
	etwEventRecord   = 0x10000000
	wnodeTracedGUID  = 0x00020000
	dxgiPresentStart = 42
	capDeltas        = 4_000_000 // tope de cuadros guardados por sesión (~4,6 h a 240 FPS)
)

var (
	guidDXGI = windows.GUID{Data1: 0xCA11C036, Data2: 0x0102, Data3: 0x4A2D, Data4: [8]byte{0xA6, 0xAD, 0xF0, 0x3C, 0xFE, 0xD5, 0xD3, 0xC9}}

	advapiETW       = windows.NewLazySystemDLL("advapi32.dll")
	procStartTrace  = advapiETW.NewProc("StartTraceW")
	procControlTr   = advapiETW.NewProc("ControlTraceW")
	procEnableTrace = advapiETW.NewProc("EnableTraceEx2")
	procOpenTrace   = advapiETW.NewProc("OpenTraceW")
	procProcessTr   = advapiETW.NewProc("ProcessTrace")
	procCloseTrace  = advapiETW.NewProc("CloseTrace")
)

// EVENT_TRACE_PROPERTIES (x64, 120 bytes).
type eventTraceProps struct {
	BufferSize      uint32
	ProviderID      uint32
	HistoricalCtx   uint64
	TimeStamp       int64
	Guid            windows.GUID
	ClientContext   uint32
	Flags           uint32
	TBufferSize     uint32
	MinimumBuffers  uint32
	MaximumBuffers  uint32
	MaximumFileSize uint32
	LogFileMode     uint32
	FlushTimer      uint32
	EnableFlags     uint32
	AgeLimit        int32
	NumberOfBuffers uint32
	FreeBuffers     uint32
	EventsLost      uint32
	BuffersWritten  uint32
	LogBuffersLost  uint32
	RTBuffersLost   uint32
	LoggerThreadID  uintptr
	LogFileNameOff  uint32
	LoggerNameOff   uint32
}

// EVENT_TRACE_LOGFILEW (x64, 448 bytes).
type eventTraceLogfile struct {
	LogFileName   *uint16
	LoggerName    *uint16
	CurrentTime   int64
	BuffersRead   uint32
	ProcessMode   uint32
	CurrentEvent  [88]byte
	LogfileHeader [280]byte
	BufferCB      uintptr
	BufferSize    uint32
	Filled        uint32
	EventsLost    uint32
	EventCB       uintptr
	IsKernel      uint32
	Context       uintptr
}

// Primeros campos de EVENT_RECORD (lo que hace falta: proceso, hora y número de evento).
type eventRecord struct {
	Size       uint16
	HeaderType uint16
	Flags      uint16
	Property   uint16
	ThreadID   uint32
	ProcessID  uint32
	TimeStamp  int64
	Provider   windows.GUID
	ID         uint16
}

// fpsCapture acumula los intervalos entre «Present» de un proceso.
type fpsCapture struct {
	mu     sync.Mutex
	last   int64
	deltas []float32 // ms entre presentaciones consecutivas
	sumMs  float64

	pid     atomic.Uint32
	session uint64
	trace   uint64
	done    chan struct{}
	running bool
}

var fpsCap = &fpsCapture{}
var fpsCallback = windows.NewCallback(func(rec *eventRecord) uintptr {
	if rec.ID == dxgiPresentStart {
		fpsCap.onPresent(rec.ProcessID, rec.TimeStamp)
	}
	return 0
})

func (c *fpsCapture) onPresent(pid uint32, ts int64) {
	if pid == 0 || pid != c.pid.Load() {
		return
	}
	c.mu.Lock()
	if c.last != 0 {
		// El tiempo viene en unidades de 100 ns. Pausas de más de 1 s (carga, menú) no cuentan.
		if dt := float64(ts-c.last) / 10000.0; dt > 0 && dt < 1000 && len(c.deltas) < capDeltas {
			c.deltas = append(c.deltas, float32(dt))
			c.sumMs += dt
		}
	}
	c.last = ts
	c.mu.Unlock()
}

func newTraceProps() ([]byte, *eventTraceProps) {
	name, _ := windows.UTF16FromString(etwSession)
	buf := make([]byte, 120+len(name)*2+16)
	p := (*eventTraceProps)(unsafe.Pointer(&buf[0]))
	p.BufferSize = uint32(len(buf))
	p.Flags = wnodeTracedGUID
	p.ClientContext = 1 // reloj QPC
	p.LogFileMode = etwRealTime
	p.LoggerNameOff = 120
	for i, ch := range name {
		*(*uint16)(unsafe.Pointer(&buf[120+i*2])) = ch
	}
	return buf, p
}

func stopSessionByName() {
	buf, p := newTraceProps()
	name, _ := windows.UTF16PtrFromString(etwSession)
	procControlTr.Call(0, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(p)), 1) // EVENT_TRACE_CONTROL_STOP
	_ = buf
}

// Start abre la sesión ETW y empieza a contar los «Present» de pid.
func (c *fpsCapture) Start(pid uint32) error {
	if c.running {
		c.pid.Store(pid)
		return nil
	}
	c.mu.Lock()
	c.last, c.deltas, c.sumMs = 0, c.deltas[:0], 0
	c.mu.Unlock()
	c.pid.Store(pid)

	name, _ := windows.UTF16PtrFromString(etwSession)
	buf, p := newTraceProps()
	r, _, _ := procStartTrace.Call(uintptr(unsafe.Pointer(&c.session)), uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(p)))
	if r == uintptr(windows.ERROR_ALREADY_EXISTS) { // sesión huérfana de una ejecución anterior
		stopSessionByName()
		buf, p = newTraceProps()
		r, _, _ = procStartTrace.Call(uintptr(unsafe.Pointer(&c.session)), uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(p)))
	}
	_ = buf
	if r != 0 {
		return windows.Errno(r)
	}
	// Nivel 4 (informativo), todas las palabras clave.
	if r, _, _ := procEnableTrace.Call(uintptr(c.session), uintptr(unsafe.Pointer(&guidDXGI)), 1, 4, ^uintptr(0), 0, 0, 0); r != 0 {
		stopSessionByName()
		return windows.Errno(r)
	}
	lf := &eventTraceLogfile{LoggerName: name, ProcessMode: etwRealTime | etwEventRecord, EventCB: fpsCallback}
	h, _, _ := procOpenTrace.Call(uintptr(unsafe.Pointer(lf)))
	if h == ^uintptr(0) || h == 0 {
		stopSessionByName()
		return errors.New("no se pudo abrir la sesión de eventos")
	}
	c.trace = uint64(h)
	c.done = make(chan struct{})
	c.running = true
	go func(h uint64, done chan struct{}) {
		defer close(done)
		handles := []uint64{h}
		procProcessTr.Call(uintptr(unsafe.Pointer(&handles[0])), 1, 0, 0) // bloquea hasta que se cierre la sesión
	}(c.trace, c.done)
	return nil
}

// SetPID cambia el proceso observado (el juego se reinició con otro PID).
func (c *fpsCapture) SetPID(pid uint32) { c.pid.Store(pid) }

type fpsResult struct {
	Avg    float64 // FPS medio
	Low1   float64 // 1% low
	Low01  float64 // 0,1% low
	Frames int
}

// Snapshot calcula los FPS con lo capturado hasta ahora, sin detener la captura.
func (c *fpsCapture) Snapshot() fpsResult {
	c.mu.Lock()
	d := append([]float32(nil), c.deltas...)
	sum := c.sumMs
	c.mu.Unlock()
	return computeFPS(d, sum)
}

// Stop cierra la sesión y devuelve el resultado.
func (c *fpsCapture) Stop() fpsResult {
	res := c.Snapshot()
	if c.running {
		stopSessionByName()
		procCloseTrace.Call(uintptr(c.trace))
		if c.done != nil {
			<-c.done
		}
		c.running = false
	}
	c.pid.Store(0)
	return res
}

// computeFPS: media = cuadros / tiempo; «low» = FPS equivalente al promedio del peor X % de los cuadros.
func computeFPS(deltas []float32, sumMs float64) fpsResult {
	n := len(deltas)
	if n < 30 || sumMs <= 0 {
		return fpsResult{Frames: n}
	}
	s := append([]float32(nil), deltas...)
	sort.Slice(s, func(i, j int) bool { return s[i] > s[j] })
	low := func(frac float64) float64 {
		k := int(math.Ceil(float64(n) * frac))
		if k < 1 {
			k = 1
		}
		var t float64
		for _, v := range s[:k] {
			t += float64(v)
		}
		return 1000 / (t / float64(k))
	}
	return fpsResult{Avg: float64(n) / sumMs * 1000, Low1: low(0.01), Low01: low(0.001), Frames: n}
}
