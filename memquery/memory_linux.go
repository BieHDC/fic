//go:build linux
// +build linux

package memory

import "syscall"

func GetMemInfo() *MemInfo {
	si := &syscall.Sysinfo_t{}
	err := syscall.Sysinfo(si)
	if err != nil {
		return nil
	}

	var mi MemInfo

	mi.MemoryTotal = uint(si.Totalram) * uint(si.Unit)
	mi.MemoryFree = uint(si.Freeram+si.Bufferram+si.Sharedram) * uint(si.Unit)

	mi.SwapTotal = uint(si.Totalswap) * uint(si.Unit)
	mi.SwapFree = uint(si.Freeswap) * uint(si.Unit)

	return &mi
}
