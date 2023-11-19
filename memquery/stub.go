//go:build !linux && !windows
// +build !linux,!windows

package memory

func GetMemInfo() *MemInfo {
	return nil
}
