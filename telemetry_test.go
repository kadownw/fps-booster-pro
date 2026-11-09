package main

import (
	"testing"
	"time"
)

func TestParseSMI(t *testing.T) {
	g, err := parseSMI("NVIDIA GeForce RTX 2060, 60, 6, 1043, 12288, 1890, 2100, 62.93, 54, 596.49\n")
	if err != nil {
		t.Fatal(err)
	}
	if g.Name != "NVIDIA GeForce RTX 2060" || g.Temp != 60 || g.Util != 6 || g.VramTotal != 12288 || g.Power != 62.93 || g.Fan != 54 || g.Driver != "596.49" {
		t.Fatalf("parseo incorrecto: %+v", g)
	}
	// sensores que no existen (portátiles, sin ventilador): nvidia-smi imprime [N/A]
	g, err = parseSMI("GeForce MX, 55, 10, 100, 2048, 900, 1400, [N/A], [N/A], 550.1")
	if err != nil {
		t.Fatal(err)
	}
	if g.Power != -1 || g.Fan != -1 || g.Temp != 55 {
		t.Fatalf("[N/A] mal manejado: %+v", g)
	}
	if _, err := parseSMI("basura"); err == nil {
		t.Fatal("aceptó una línea inválida")
	}
}

func TestLiveTelemetryOnThisMachine(t *testing.T) {
	liveUsage()
	enrichLive(&Live{})
	time.Sleep(1200 * time.Millisecond)
	l := liveUsage()
	enrichLive(&l)
	if l.RAMTotal <= 0 || l.RAM <= 0 || l.RAM > 100 {
		t.Fatalf("RAM inválida: %+v", l)
	}
	if l.CPUMHz <= 0 {
		t.Errorf("sin frecuencia de CPU (PDH): %+v", l)
	}
	if len(l.Disks) == 0 {
		t.Error("no detectó ningún disco fijo")
	}
	for _, d := range l.Disks {
		if d.UsedPct < 0 || d.UsedPct > 100 || d.TotalGB <= 0 {
			t.Errorf("disco inválido: %+v", d)
		}
	}
	t.Logf("CPU %.0f%% %d MHz | RAM %.1f/%.1f GB | discos %d | GPU %+v", l.CPU, l.CPUMHz, l.RAMUsed, l.RAMTotal, len(l.Disks), l.GPU)
	d := cpuDetailWithBase()
	if d.Threads <= 0 || d.Cores <= 0 || d.Cores > d.Threads {
		t.Errorf("detalle de CPU inválido: %+v", d)
	}
	t.Logf("CPU: %+v", d)
}
