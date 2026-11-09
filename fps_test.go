package main

import (
	"os"
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
