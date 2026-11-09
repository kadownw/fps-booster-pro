package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestETWStructLayouts(t *testing.T) {
	if s := unsafe.Sizeof(eventTraceProps{}); s != 120 {
		t.Fatalf("EVENT_TRACE_PROPERTIES = %d, esperado 120", s)
	}
	if s := unsafe.Sizeof(eventTraceLogfile{}); s != 448 {
		t.Fatalf("EVENT_TRACE_LOGFILEW = %d, esperado 448", s)
	}
	var r eventRecord
	if o := unsafe.Offsetof(r.ProcessID); o != 12 {
		t.Fatalf("ProcessId en %d, esperado 12", o)
	}
	if o := unsafe.Offsetof(r.TimeStamp); o != 16 {
		t.Fatalf("TimeStamp en %d, esperado 16", o)
	}
	if o := unsafe.Offsetof(r.ID); o != 40 {
		t.Fatalf("EventDescriptor.Id en %d, esperado 40", o)
	}
	l := eventTraceLogfile{}
	if o := unsafe.Offsetof(l.EventCB); o != 424 {
		t.Fatalf("EventRecordCallback en %d, esperado 424", o)
	}
}

func TestComputeFPS(t *testing.T) {
	d := make([]float32, 1000)
	var sum float64
	for i := range d {
		d[i] = 10 // 100 FPS
		sum += 10
	}
	d[0] = 50 // un tirón
	sum += 40
	r := computeFPS(d, sum)
	if r.Avg < 99 || r.Avg > 100 {
		t.Fatalf("media = %.1f", r.Avg)
	}
	if r.Low01 < 19.9 || r.Low01 > 20.1 {
		t.Fatalf("0,1%% low = %.1f, esperado 20", r.Low01)
	}
	if r2 := computeFPS(d[:10], 100); r2.Avg != 0 {
		t.Fatal("con pocos cuadros no se debe dar un número")
	}
}

// Valida toda la tubería ETW contra dwm.exe (el compositor de Windows presenta siempre).
func TestETWCapturesDWM(t *testing.T) {
	if os.Getenv("FBP_ADMIN_TESTS") != "1" || !isAdmin() {
		t.Skip("define FBP_ADMIN_TESTS=1 y ejecuta como administrador")
	}
	pid := pidOf("dwm.exe")
	if pid == 0 {
		t.Skip("sin dwm.exe")
	}
	c := &fpsCapture{}
	old := fpsCap
	fpsCap = c
	defer func() { fpsCap = old }()
	if err := c.Start(pid); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * time.Second)
	r := c.Stop()
	t.Logf("dwm.exe: %d cuadros, %.1f FPS, 0,1%% low %.1f", r.Frames, r.Avg, r.Low01)
	if r.Frames == 0 {
		t.Fatal("no llegó ningún evento Present")
	}
}

func pidOf(name string) uint32 {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0
	}
	defer windows.CloseHandle(snap)
	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	for err = windows.Process32First(snap, &pe); err == nil; err = windows.Process32Next(snap, &pe) {
		if windows.UTF16ToString(pe.ExeFile[:]) == name {
			return pe.ProcessID
		}
	}
	return 0
}

// OpenGL no pasa por DXGI: la medición tiene que salir del evento Present_Info de DxgKrnl.
func TestETWCapturesOpenGL(t *testing.T) {
	if os.Getenv("FBP_ADMIN_TESTS") != "1" || !isAdmin() {
		t.Skip("define FBP_ADMIN_TESTS=1 y ejecuta como administrador")
	}
	exe := filepath.Join(t.TempDir(), "gltest.exe")
	if out, err := exec.Command("go", "build", "-o", exe, "./tools/gltest").CombinedOutput(); err != nil {
		t.Fatalf("no se pudo compilar gltest: %v\n%s", err, out)
	}
	cmd := exec.Command(exe, "9")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()
	time.Sleep(2 * time.Second) // la ventana tarda en abrir
	c := &fpsCapture{}
	old := fpsCap
	fpsCap = c
	defer func() { fpsCap = old }()
	if err := c.Start(uint32(cmd.Process.Pid)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(4 * time.Second)
	t.Logf("DxgKrnl EnableTraceEx2 = %d", c.krnlErr)
	r := c.Stop()
	t.Logf("OpenGL: fuente %s, %d cuadros, %.0f FPS, 1%% low %.0f, 0,1%% low %.0f", r.Source, r.Frames, r.Avg, r.Low1, r.Low01)
	if r.Source != "DxgKrnl" || r.Frames < 100 {
		t.Fatalf("no se midió el juego OpenGL: %+v", r)
	}
	// Un presenter OpenGL sin vsync da cientos de FPS; el promedio debe ser razonable.
	if r.Avg < 20 {
		t.Fatalf("FPS demasiado bajos: %.1f", r.Avg)
	}
}
