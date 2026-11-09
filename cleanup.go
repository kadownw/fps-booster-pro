package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Limpieza de archivos temporales. A diferencia de los ajustes, borrar archivos NO se puede
// deshacer, así que: (1) solo se limpian carpetas de una lista fija, (2) primero se calcula el
// tamaño (vista previa), (3) no se siguen enlaces simbólicos ni puntos de montaje, (4) se
// respetan archivos modificados hace menos de una hora (pueden estar en uso).

const cleanMinAge = time.Hour

type cleanTarget struct {
	ID    string
	Title string
	Desc  string
	Dirs  func() []string
}

func envDir(env string, rel ...string) string {
	b := os.Getenv(env)
	if b == "" {
		return ""
	}
	return filepath.Join(append([]string{b}, rel...)...)
}

// userTempDirs devuelve %TEMP% solo si de verdad parece una carpeta de temporales: si TEMP/TMP apuntan a una
// carpeta de documentos, de un proyecto o a la raíz de un disco, NO se limpia (se borrarían archivos propios).
func userTempDirs() []string {
	d := os.TempDir()
	if !looksLikeTemp(d) {
		return nil
	}
	return []string{d}
}

func looksLikeTemp(p string) bool {
	clean := strings.ToLower(filepath.Clean(p))
	if vol := filepath.VolumeName(clean); clean == vol || clean == vol+`\` {
		return false
	}
	for _, part := range strings.Split(clean, `\`) {
		if part == "temp" || part == "tmp" {
			return true
		}
	}
	return false
}

func nonEmpty(s ...string) []string {
	var out []string
	for _, v := range s {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

var cleanTargets = []cleanTarget{
	{"temp_user", "Archivos temporales del usuario", "Archivos que las apps dejan en %TEMP%.",
		func() []string { return userTempDirs() }},
	{"temp_windows", "Archivos temporales de Windows", "Carpeta C:\\Windows\\Temp (requiere administrador).",
		func() []string { return nonEmpty(envDir("SystemRoot", "Temp")) }},
	{"crash_dumps", "Volcados de errores", "Volcados de memoria de programas que se cerraron con error.",
		func() []string {
			return nonEmpty(envDir("LOCALAPPDATA", "CrashDumps"), envDir("SystemRoot", "Minidump"))
		}},
	{"wer_reports", "Informes de errores de Windows", "Reportes pendientes de enviar a Microsoft.",
		func() []string {
			return nonEmpty(envDir("ProgramData", "Microsoft", "Windows", "WER", "ReportArchive"),
				envDir("ProgramData", "Microsoft", "Windows", "WER", "ReportQueue"))
		}},
	{"shader_cache", "Caché de shaders (NVIDIA/DirectX)", "Se regenera sola; el primer juego que abras puede cargar un poco más lento.",
		func() []string {
			return nonEmpty(envDir("LOCALAPPDATA", "D3DSCache"), envDir("LOCALAPPDATA", "NVIDIA", "DXCache"),
				envDir("LOCALAPPDATA", "NVIDIA", "GLCache"), envDir("LOCALAPPDATA", "AMD", "DxCache"))
		}},
	{"web_cache", "Caché de internet de Windows", "Archivos temporales de internet del sistema.",
		func() []string { return nonEmpty(envDir("LOCALAPPDATA", "Microsoft", "Windows", "INetCache")) }},
}

type CleanItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Desc  string `json:"desc"`
	Bytes int64  `json:"bytes"`
	Files int    `json:"files"`
	Admin bool   `json:"admin"`
}

func isReparse(fi fs.FileInfo) bool {
	return fi.Mode()&fs.ModeSymlink != 0 || fi.Mode()&fs.ModeIrregular != 0
}

// scanDir suma tamaño y cantidad de archivos que se podrían borrar.
func scanDir(root string, minAge time.Duration) (bytes int64, files int) {
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // sin permiso o desapareció: se ignora
		}
		if d.Type()&fs.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if err != nil || isReparse(fi) || time.Since(fi.ModTime()) < minAge {
			return nil
		}
		bytes += fi.Size()
		files++
		return nil
	})
	return
}

// cleanDir borra los archivos de root (y las carpetas que queden vacías). Devuelve bytes liberados.
func cleanDir(root string, minAge time.Duration) (freed int64, errs int) {
	var dirs []string
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if p != root {
				dirs = append(dirs, p)
			}
			return nil
		}
		fi, err := d.Info()
		if err != nil || isReparse(fi) || time.Since(fi.ModTime()) < minAge {
			return nil
		}
		if err := os.Remove(p); err != nil {
			errs++ // en uso o sin permiso: se deja
			return nil
		}
		freed += fi.Size()
		return nil
	})
	for i := len(dirs) - 1; i >= 0; i-- {
		os.Remove(dirs[i]) // solo borra si está vacía
	}
	return
}

func ScanCleanup() []CleanItem {
	var out []CleanItem
	for _, t := range cleanTargets {
		it := CleanItem{ID: t.ID, Title: t.Title, Desc: t.Desc, Admin: t.ID == "temp_windows" || t.ID == "wer_reports" || t.ID == "crash_dumps"}
		for _, d := range t.Dirs() {
			b, f := scanDir(d, cleanMinAge)
			it.Bytes += b
			it.Files += f
		}
		out = append(out, it)
	}
	return out
}

type CleanResult struct {
	Freed  int64    `json:"freed"`
	Errors []string `json:"errors"`
}

// RunCleanup limpia solo las categorías pedidas que existan en la lista fija.
func RunCleanup(ids []string) (CleanResult, error) {
	var res CleanResult
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	found := 0
	for _, t := range cleanTargets {
		if !want[t.ID] {
			continue
		}
		found++
		for _, d := range t.Dirs() {
			if fi, err := os.Stat(d); err != nil || !fi.IsDir() {
				continue
			}
			freed, skipped := cleanDir(d, cleanMinAge)
			res.Freed += freed
			if skipped > 0 {
				res.Errors = append(res.Errors, fmt.Sprintf("%s: %d archivos en uso o protegidos se dejaron", t.Title, skipped))
			}
		}
	}
	if found == 0 {
		return res, errors.New("no se eligió ninguna categoría válida")
	}
	return res, nil
}
