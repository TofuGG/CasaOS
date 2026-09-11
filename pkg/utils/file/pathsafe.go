package file

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// resolvePath returns the canonical, symlink-resolved absolute form of p.
//
// If the final path component does not exist yet (e.g. a file about to be
// created), the deepest existing ancestor is resolved and the non-existent
// remainder is re-appended. This prevents symlink-escape attacks where a
// symlink inside an allowed root points into a blocked or disallowed area.
func resolvePath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return ""
	}
	abs = filepath.Clean(abs)

	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}

	// Walk up to the nearest existing ancestor and resolve it.
	dir := abs
	rest := ""
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root — nothing resolves (defensive; / always exists on linux).
			break
		}
		if resolvedDir, err := filepath.EvalSymlinks(dir); err == nil {
			return filepath.Join(resolvedDir, rest)
		}
		rest = filepath.Join(filepath.Base(dir), rest)
		dir = parent
	}
	return abs
}

// allowedRoots returns the directory roots the CasaOS file API may touch.
// Anything outside these is rejected (both for reads and writes).
func allowedRoots(homeDir string) []string {
	roots := []string{
		"/DATA",
		"/mnt",
		"/media",
		"/tmp",
		"/var/lib/casaos",
	}
	if homeDir != "" {
		roots = append(roots, homeDir)
	}
	return roots
}

func isWithinAnyAllowedRoot(resolved, homeDir string, roots []string) bool {
	for _, root := range roots {
		if resolved == root || strings.HasPrefix(resolved, root+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// systemProtectedPrefixes are directories that must never be writable through
// the file API — checked against the symlink-resolved path.
var systemProtectedPrefixes = []string{
	"/etc",
	"/boot",
	"/dev",
	"/proc",
	"/sys",
	"/sbin",
	"/bin",
	"/usr",
	"/lib",
	"/lib64",
	"/var/lib/dpkg",
	"/var/lib/apt",
	"/var/cache/apt",
}

func isProtectedSystemPath(resolved string) bool {
	for _, prefix := range systemProtectedPrefixes {
		if resolved == prefix || strings.HasPrefix(resolved, prefix+"/") {
			return true
		}
	}
	return false
}

// sensitiveReadPaths are files/directories whose contents must never be
// served through the file API, even with a valid JWT.
var sensitiveReadPaths = []string{
	"/etc/shadow",
	"/etc/gshadow",
	"/etc/sudoers",
	"/etc/passwd",
	"/etc/group",
	"/etc/ssh/sshd_config",
	"/etc/ssl/private",
	"/root/.ssh/id_rsa",
	"/root/.ssh/id_ed25519",
	"/root/.ssh/id_ecdsa",
	"/root/.ssh/authorized_keys",
}

func isSensitiveReadPath(resolved string) bool {
	for _, sensitive := range sensitiveReadPaths {
		if resolved == sensitive || strings.HasPrefix(resolved, sensitive+"/") {
			return true
		}
	}
	return false
}

// IsPathSafeForWrite checks if a path is in a location where CasaOS is allowed
// to write/delete.
//
// The check is performed on the symlink-resolved path, so a symlink inside an
// allowed root that points into a system or out-of-roots location is rejected.
// Note: a residual time-of-check/time-of-use race remains between this check
// and the subsequent filesystem operation.
func IsPathSafeForWrite(p string) bool {
	resolved := resolvePath(p)
	if resolved == "" {
		return false
	}

	// Never allow writes into system-protected prefixes (checked on the
	// symlink-resolved path).
	if isProtectedSystemPath(resolved) {
		return false
	}

	if runtime.GOOS == "linux" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = ""
		}
		roots := allowedRoots(homeDir)
		if !isWithinAnyAllowedRoot(resolved, homeDir, roots) {
			return false
		}
	}

	return true
}

// IsPathSafeForRead checks if a file can be served through the file API.
//
// Like IsPathSafeForWrite, the check runs against the symlink-resolved path so
// symlink escapes into sensitive locations are blocked. Kept permissive enough
// for legitimate file browsing: home, /DATA, /mnt, /media, /tmp and
// /var/lib/casaos are all readable, but nothing outside those roots and no
// sensitive system files.
func IsPathSafeForRead(p string) bool {
	resolved := resolvePath(p)
	if resolved == "" {
		return false
	}

	// Never serve credentials, keys or auth data even with a valid token.
	if isSensitiveReadPath(resolved) {
		return false
	}

	if runtime.GOOS == "linux" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = ""
		}
		roots := allowedRoots(homeDir)
		if !isWithinAnyAllowedRoot(resolved, homeDir, roots) {
			return false
		}
	}

	return true
}