package main

import (
	"os"
	"testing"
)

// Pruebas que SÍ modifican el sistema (servicios, HKLM, tareas programadas) y lo restauran.
// Solo corren como administrador y con FBP_ADMIN_TESTS=1.
func TestAdminRoundTrip(t *testing.T) {
	if os.Getenv("FBP_ADMIN_TESTS") != "1" || !isAdmin() {
		t.Skip("define FBP_ADMIN_TESTS=1 y ejecuta como administrador")
	}
	e, err := NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"usb_recovery", "svc_xbox_manual", "telemetry", "error_reporting", "bg_apps", "search_local", "mouse_raw_curve", "nvidia_profile", "svchost_split", "mpo_off", "flip_immediate", "nic_power_off", "updates_pause", "trim_on", "store_autoupdate_off", "timer_resolution", "startup_sound_off", "memory_compression_off", "gpu_msi_mode", "nv_driver_prefs", "nv_hdcp_off", "svc_max_performance"} {
		tw, _ := findTweak(id)
		if !supported(tw) {
			t.Logf("%s: no aplica a este equipo, se omite", id)
			continue
		}
		regs := expand(tw)
		var nvBefore []NvOrig
		if len(tw.Nvidia) > 0 {
			if nvBefore, err = nvReadOriginals(tw.Nvidia); err != nil {
				t.Fatal(err)
			}
		}
		before := make([]RegState, len(regs))
		for i, c := range regs {
			if before[i], err = readState(c); err != nil {
				t.Fatalf("%s: %v", id, err)
			}
		}
		tasksBefore := existingTasks(tw)
		invalidateTasks()

		if err := e.Apply(id); err != nil {
			t.Fatalf("apply %s: %v", id, err)
		}
		for _, c := range regs {
			if st, _ := readState(c); !sameState(st, desired(c)) {
				t.Fatalf("%s: %s no quedó aplicado", id, c.Name)
			}
		}
		for p := range tasksBefore {
			if st := existingTasks(tw)[p]; st != "Disabled" {
				t.Fatalf("%s: la tarea %s quedó en %q", id, p, st)
			}
		}
		if len(tw.Nvidia) > 0 && !nvIsActive(tw.Nvidia) {
			t.Fatalf("%s: el perfil NVIDIA no quedó aplicado", id)
		}
		if err := e.Revert(id); err != nil {
			t.Fatalf("revert %s: %v", id, err)
		}
		if len(tw.Nvidia) > 0 {
			after, _ := nvReadOriginals(tw.Nvidia)
			for i := range nvBefore {
				if after[i] != nvBefore[i] {
					t.Fatalf("%s: el ajuste NVIDIA 0x%X no volvió: %+v vs %+v", id, nvBefore[i].ID, after[i], nvBefore[i])
				}
			}
		}
		for i, c := range regs {
			if st, _ := readState(c); !sameState(st, before[i]) {
				t.Fatalf("%s: %s no volvió al original (%+v vs %+v)", id, c.Name, st, before[i])
			}
		}
		invalidateTasks()
		for p, was := range tasksBefore {
			if now := existingTasks(tw)[p]; now != was && !(was == "Running" && now == "Ready") {
				t.Fatalf("%s: la tarea %s era %q y quedó %q", id, p, was, now)
			}
		}
		t.Logf("%s: aplicado y revertido OK (%d valores, %d tareas)", id, len(regs), len(tasksBefore))
	}
	if left := len(e.st.Items); left != 0 {
		t.Fatalf("quedaron %d backups sin revertir", left)
	}
}
