//go:build !windows

package os_specific

func FixRootDirectory(p string) string {
	return p
}
