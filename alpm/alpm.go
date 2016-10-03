// Package alpm lets us use libalpm.  See https://www.archlinux.org/pacman/libalpm.3.html
package alpm

/*
#cgo LDFLAGS: -lalpm
#include <alpm.h>
*/
import "C"
import "unsafe"

// VerCmp is an interface to alpm_pkg_vercmp.  See
// https://www.archlinux.org/pacman/vercmp.8.html.  Also see
// https://wiki.archlinux.org/index.php/PKGBUILD#Version
func VerCmp(ver1, ver2 string) int {
	cVer1 := C.CString(ver1)
	defer C.free(unsafe.Pointer(cVer1))
	cVer2 := C.CString(ver2)
	defer C.free(unsafe.Pointer(cVer2))
	return int(C.alpm_pkg_vercmp(cVer1, cVer2))
}
