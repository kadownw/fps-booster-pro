package main

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"unsafe"
)

// Perfil global de NVIDIA (DRS = Driver Settings) vía NvAPI, la API oficial del driver.
// IDs de interfaz y de ajustes tomados del SDK oficial NVIDIA/nvapi (licencia MIT):
// nvapi_interface.h y NvApiDriverSettings.h. Se modifica solo el perfil base (global),
// guardando el valor previo de cada ajuste para poder restaurarlo.

const (
	nvidInitialize              = 0x0150e828
	nvidDRSCreateSession        = 0x0694d52e
	nvidDRSDestroySession       = 0xdad9cff8
	nvidDRSLoadSettings         = 0x375dbd6b
	nvidDRSSaveSettings         = 0xfcbc7e14
	nvidDRSGetBaseProfile       = 0xda8466a0
	nvidDRSGetSetting           = 0x73bf8338
	nvidDRSSetSetting           = 0x577dd202
	nvidDRSDeleteProfileSetting = 0xe4a26362
)

// IDs de ajustes (NvApiDriverSettings.h).
const (
	nvSettingPreferredPState = 0x1057EB71 // Power management mode
	nvSettingPrerenderLimit  = 0x007BA09E // Maximum pre-rendered frames (Low Latency Mode)
	nvSettingShaderCache     = 0x00198FFF // Shader Cache

	nvPStatePreferMax = 0x1 // PREFERRED_PSTATE_PREFER_MAX
	nvShaderCacheOn   = 0x1 // PS_SHADERDISKCACHE_ON
)

// NVDRS_SETTING_V1: version, nombre (2048 UTF-16), id, tipo, ubicación, flags y dos uniones
// de 4100 bytes (valor predefinido y actual). Tamaño total 12320 bytes (0x3020).
type nvdrsSetting struct {
	Version             uint32
	SettingName         [2048]uint16
	SettingID           uint32
	SettingType         uint32
	SettingLocation     uint32
	IsCurrentPredefined uint32
	IsPredefinedValid   uint32
	Predefined          [4100]byte
	Current             [4100]byte
}

const nvdrsSettingVer1 = uint32(unsafe.Sizeof(nvdrsSetting{})) | 1<<16

func (s *nvdrsSetting) currentDWORD() uint32 {
	return *(*uint32)(unsafe.Pointer(&s.Current[0]))
}

type nvAPI struct {
	once    sync.Once
	err     error
	query   *syscall.LazyProc
	cache   map[uint32]uintptr
	cacheMu sync.Mutex
}

var nv nvAPI

func (n *nvAPI) init() error {
	n.once.Do(func() {
		dll := syscall.NewLazyDLL("nvapi64.dll")
		if err := dll.Load(); err != nil {
			n.err = errors.New("no hay driver de NVIDIA (nvapi64.dll no encontrada)")
			return
		}
		n.query = dll.NewProc("nvapi_QueryInterface")
		if err := n.query.Find(); err != nil {
			n.err = errors.New("NvAPI no disponible")
			return
		}
		n.cache = map[uint32]uintptr{}
		if st := n.call(nvidInitialize); st != 0 {
			n.err = fmt.Errorf("NvAPI_Initialize falló (%d)", st)
		}
	})
	return n.err
}

func (n *nvAPI) fn(id uint32) uintptr {
	n.cacheMu.Lock()
	defer n.cacheMu.Unlock()
	if p, ok := n.cache[id]; ok {
		return p
	}
	p, _, _ := n.query.Call(uintptr(id))
	n.cache[id] = p
	return p
}

// call devuelve el NvAPI_Status (0 = OK; negativo = error).
func (n *nvAPI) call(id uint32, args ...uintptr) int32 {
	p := n.fn(id)
	if p == 0 {
		return -3 // NVAPI_NO_IMPLEMENTATION
	}
	r, _, _ := syscall.SyscallN(p, args...)
	return int32(r)
}

type nvSession struct{ h, profile uintptr }

func openBaseProfile() (*nvSession, error) {
	if err := nv.init(); err != nil {
		return nil, err
	}
	s := &nvSession{}
	if st := nv.call(nvidDRSCreateSession, uintptr(unsafe.Pointer(&s.h))); st != 0 {
		return nil, fmt.Errorf("NvAPI_DRS_CreateSession falló (%d)", st)
	}
	if st := nv.call(nvidDRSLoadSettings, s.h); st != 0 {
		s.close()
		return nil, fmt.Errorf("NvAPI_DRS_LoadSettings falló (%d)", st)
	}
	if st := nv.call(nvidDRSGetBaseProfile, s.h, uintptr(unsafe.Pointer(&s.profile))); st != 0 {
		s.close()
		return nil, fmt.Errorf("NvAPI_DRS_GetBaseProfile falló (%d)", st)
	}
	return s, nil
}

func (s *nvSession) close() { nv.call(nvidDRSDestroySession, s.h) }

// get devuelve el valor actual, si es el predefinido del driver y el estado de la llamada.
func (s *nvSession) get(id uint32) (value uint32, predefined bool, status int32) {
	set := new(nvdrsSetting)
	set.Version = nvdrsSettingVer1
	st := nv.call(nvidDRSGetSetting, s.h, s.profile, uintptr(id), uintptr(unsafe.Pointer(set)))
	if st != 0 {
		return 0, true, st
	}
	return set.currentDWORD(), set.IsCurrentPredefined != 0, 0
}

func (s *nvSession) set(id, value uint32) error {
	set := new(nvdrsSetting)
	set.Version = nvdrsSettingVer1
	set.SettingID = id
	set.SettingType = 0 // NVDRS_DWORD_TYPE
	*(*uint32)(unsafe.Pointer(&set.Current[0])) = value
	if st := nv.call(nvidDRSSetSetting, s.h, s.profile, uintptr(unsafe.Pointer(set))); st != 0 {
		return fmt.Errorf("NvAPI_DRS_SetSetting(0x%08X) falló (%d)", id, st)
	}
	return nil
}

func (s *nvSession) reset(id uint32) error {
	st := nv.call(nvidDRSDeleteProfileSetting, s.h, s.profile, uintptr(id))
	if st != 0 && st != -160 { // -160 = el ajuste ya no estaba definido en el perfil
		return fmt.Errorf("NvAPI_DRS_DeleteProfileSetting(0x%08X) falló (%d)", id, st)
	}
	return nil
}

func (s *nvSession) save() error {
	if st := nv.call(nvidDRSSaveSettings, s.h); st != 0 {
		return fmt.Errorf("NvAPI_DRS_SaveSettings falló (%d); ¿falta administrador?", st)
	}
	return nil
}

// NvSetting es un ajuste DWORD del perfil global con su valor deseado.
type NvSetting struct {
	ID    uint32 `json:"id"`
	Value uint32 `json:"value"`
	Name  string `json:"name"`
}

// NvOrig guarda el estado previo de un ajuste para restaurarlo.
type NvOrig struct {
	ID         uint32 `json:"id"`
	Value      uint32 `json:"value"`
	Predefined bool   `json:"predefined"` // era el valor por defecto del driver: se restaura borrando el ajuste
}

func nvAvailable() bool { return nv.init() == nil }

func nvReadOriginals(list []NvSetting) ([]NvOrig, error) {
	s, err := openBaseProfile()
	if err != nil {
		return nil, err
	}
	defer s.close()
	out := make([]NvOrig, 0, len(list))
	for _, x := range list {
		v, pre, _ := s.get(x.ID)
		out = append(out, NvOrig{ID: x.ID, Value: v, Predefined: pre})
	}
	return out, nil
}

func nvApply(list []NvSetting) error {
	s, err := openBaseProfile()
	if err != nil {
		return err
	}
	defer s.close()
	for _, x := range list {
		if err := s.set(x.ID, x.Value); err != nil {
			return err
		}
	}
	if err := s.save(); err != nil {
		return err
	}
	for _, x := range list { // verificación
		if v, _, st := s.get(x.ID); st != 0 || v != x.Value {
			return fmt.Errorf("verificación fallida en el ajuste NVIDIA %s", x.Name)
		}
	}
	return nil
}

func nvRestore(orig []NvOrig) error {
	s, err := openBaseProfile()
	if err != nil {
		return err
	}
	defer s.close()
	var errs []error
	for _, o := range orig {
		if o.Predefined {
			errs = append(errs, s.reset(o.ID))
		} else {
			errs = append(errs, s.set(o.ID, o.Value))
		}
	}
	errs = append(errs, s.save())
	return errors.Join(errs...)
}

func nvIsActive(list []NvSetting) bool {
	s, err := openBaseProfile()
	if err != nil {
		return false
	}
	defer s.close()
	for _, x := range list {
		if v, _, st := s.get(x.ID); st != 0 || v != x.Value {
			return false
		}
	}
	return true
}
