// Programa de prueba: abre una ventana OpenGL y presenta cuadros sin parar. Sirve para validar que la
// medición de FPS (fps.go) también ve juegos que no usan DXGI. Uso: gltest.exe <segundos>
package main

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

var (
	u32 = syscall.NewLazyDLL("user32.dll")
	g32 = syscall.NewLazyDLL("gdi32.dll")
	k32 = syscall.NewLazyDLL("kernel32.dll")
	gl  = syscall.NewLazyDLL("opengl32.dll")
)

type wndclass struct {
	Size, Style                        uint32
	WndProc                            uintptr
	ClsExtra, WndExtra                 int32
	Instance, Icon, Cursor, Background uintptr
	MenuName, ClassName                *uint16
	IconSm                             uintptr
}
type pfd struct {
	Size, Version                      uint16
	Flags                              uint32
	PixelType, ColorBits               byte
	a                                  [13]byte
	DepthBits, StencilBits, AuxBuffers byte
	LayerType, Reserved                byte
	b                                  [12]byte
}
type msg struct {
	Hwnd    uintptr
	Message uint32
	W, L    uintptr
	Time    uint32
	X, Y    int32
}

func main() {
	runtime.LockOSThread() // el contexto GL pertenece al hilo que lo creó
	secs := 10
	if len(os.Args) > 1 {
		fmt.Sscan(os.Args[1], &secs)
	}
	fmt.Println("PID", os.Getpid())
	hinst, _, _ := k32.NewProc("GetModuleHandleW").Call(0)
	cls, _ := syscall.UTF16PtrFromString("gltest")
	proc := syscall.NewCallback(func(h, m, w, l uintptr) uintptr {
		r, _, _ := u32.NewProc("DefWindowProcW").Call(h, m, w, l)
		return r
	})
	wc := wndclass{Style: 0x20, WndProc: proc, Instance: hinst, ClassName: cls}
	wc.Size = uint32(unsafe.Sizeof(wc))
	u32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc)))
	title, _ := syscall.UTF16PtrFromString("gltest")
	hwnd, _, _ := u32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(title)), 0x10CF0000, 100, 100, 640, 480, 0, 0, hinst, 0)
	u32.NewProc("ShowWindow").Call(hwnd, 5)
	hdc, _, _ := u32.NewProc("GetDC").Call(hwnd)
	p := pfd{Version: 1, Flags: 0x25, PixelType: 0, ColorBits: 32, DepthBits: 24}
	p.Size = uint16(unsafe.Sizeof(p))
	f, _, _ := g32.NewProc("ChoosePixelFormat").Call(hdc, uintptr(unsafe.Pointer(&p)))
	g32.NewProc("SetPixelFormat").Call(hdc, f, uintptr(unsafe.Pointer(&p)))
	ctx, _, _ := gl.NewProc("wglCreateContext").Call(hdc)
	gl.NewProc("wglMakeCurrent").Call(hdc, ctx)
	clear, cc := gl.NewProc("glClear"), gl.NewProc("glClearColor")
	swap := g32.NewProc("SwapBuffers")
	peek, disp := u32.NewProc("PeekMessageW"), u32.NewProc("DispatchMessageW")
	end := time.Now().Add(time.Duration(secs) * time.Second)
	n := 0
	var m msg
	for time.Now().Before(end) {
		for {
			r, _, _ := peek.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, 1)
			if r == 0 {
				break
			}
			disp.Call(uintptr(unsafe.Pointer(&m)))
		}
		v := float32(n%100) / 100
		cc.Call(uintptr(*(*uint32)(unsafe.Pointer(&v))), 0, 0, 0)
		clear.Call(0x4000)
		swap.Call(hdc)
		n++
	}
	fmt.Println("frames", n, "fps", float64(n)/float64(secs))
}
