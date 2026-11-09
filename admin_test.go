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
	for _, id := range []string{"usb_recovery", "svc_xbox_manual", "svc_unneeded", "update_pause", "error_tasks", "telemetry_tasks", "svchost_split", "mpo_off", "flip_immediate", "nic_power_off", "bg_apps_policy", "updates_pause", "trim_on", "store_autoupdate_off", "timer_resolution", "startup_sound_off", "memory_compression_off", "gpu_msi_mode", "nv_driver_prefs", "nv_hdcp_off", "svc_max_performance", "bg_apps_policy"} {
		tw, _ := findTweak(id)
		if !supported(tw) {
			t.Logf("%s: no aplica a este equipo, se omite", id)
			continue
		}
		regs := expand(tw)
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
		if err := e.Revert(id); err != nil {
			t.Fatalf("revert %s: %v", id, err)
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
