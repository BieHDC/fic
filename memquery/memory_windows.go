//go:build windows
// +build windows

package memory

import (
	"syscall"
	"unsafe"
)

type memStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

var memStatusExSize uint32
var globalMemoryStatusEx *syscall.Proc

func init() {
	var msx memStatusEx
	memStatusExSize = uint32(unsafe.Sizeof(msx))

	kernel32, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		panic("kernel32.dll must exist and be loadable")
	}
	// GetPhysicallyInstalledSystemMemory is simpler, but broken on
	// older versions of windows (and uses this under the hood anyway).
	globalMemoryStatusEx, err = kernel32.FindProc("GlobalMemoryStatusEx")
	if err != nil {
		panic("GlobalMemoryStatusEx must exist")
	}
}

func GetMemInfo() *MemInfo {
	msx := &memStatusEx{
		Length: memStatusExSize,
	}
	r, _, _ := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(msx)))
	if r == 0 {
		return nil
	}

	var mi MemInfo
	mi.MemoryTotal = uint(msx.TotalPhys)
	mi.MemoryFree = uint(msx.AvailPhys)

	mi.SwapTotal = uint(msx.TotalPageFile)
	mi.SwapFree = uint(msx.AvailPageFile)
	return &mi
}
