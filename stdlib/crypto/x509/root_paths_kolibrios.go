//go:build kolibrios && gccgo

package x509

// Native installation paths for the unchanged upstream root-store loader.
var certFiles = []string{
	"/sys/certs/ca-bundle.crt",
	"/hd0/1/opencode/certs/ca-bundle.crt",
}

var certDirectories = []string{"/sys/certs", "/hd0/1/opencode/certs"}
