package main

import (
	"os"
	"testing"
	"unsafe"
)

func TestNvdrsSettingLayout(t *testing.T) {
	// NVDRS_SETTING_V1 mide 0x3020 bytes; un tamaño distinto haría que el driver rechace la versión.
	if s := unsafe.Sizeof(nvdrsSetting{}); s != 0x3020 {
		t.Fatalf("tamaño de NVDRS_SETTING_V1 = 0x%X, esperado 0x3020", s)
	}
	if nvdrsSettingVer1 != 0x13020 {
		t.Fatalf("versión = 0x%X", nvdrsSettingVer1)
	}
}

// Solo lectura: valida que el layout es aceptado por el driver instalado.
func TestNvReadOnly(t *testing.T) {
	if !nvAvailable() {
		t.Skip("sin NVIDIA")
	}
	s, err := openBaseProfile()
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	v, pre, st := s.get(nvSettingPreferredPState)
	if st != 0 && st != -160 {
		t.Fatalf("GetSetting devolvió %d (¿layout incorrecto?)", st)
	}
	t.Logf("Power management mode: valor=%d predefinido=%v estado=%d", v, pre, st)
}

// Modifica el perfil global de NVIDIA y lo restaura. Solo con FBP_ADMIN_TESTS=1 y admin.
func TestNvApplyRestore(t *testing.T) {
	if os.Getenv("FBP_ADMIN_TESTS") != "1" || !isAdmin() || !nvAvailable() {
		t.Skip("define FBP_ADMIN_TESTS=1, ejecuta como administrador y con NVIDIA")
	}
	for _, id := range []string{"nvidia_profile"} {
		tw, _ := findTweak(id)
		orig, err := nvReadOriginals(tw.Nvidia)
		if err != nil {
			t.Fatal(err)
		}
		if err := nvApply(tw.Nvidia); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if !nvIsActive(tw.Nvidia) {
			t.Fatalf("%s: no quedó aplicado", id)
		}
		if err := nvRestore(orig); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		after, _ := nvReadOriginals(tw.Nvidia)
		for i := range orig {
			if after[i].Value != orig[i].Value || after[i].Predefined != orig[i].Predefined {
				t.Fatalf("%s: ajuste 0x%X no volvió: %+v vs %+v", id, orig[i].ID, after[i], orig[i])
			}
		}
	}
}
