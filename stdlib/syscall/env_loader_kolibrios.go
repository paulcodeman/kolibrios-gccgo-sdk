//go:build kolibrios

package syscall

import (
	"kos"
	"path"
	"strings"
)

// KolibriOS supplies a loader path and command line, without an environment
// block. SDK applications can supply one in <executable path>.env. Loading
// here precedes dependent package initialization, including provider discovery.
// The file contains literal KEY=VALUE lines; no shell evaluation occurs.
// KOLIBRI_WORKING_DIRECTORY opts a native launch into an initial directory
// relative to the executable, and anchors standard relative path variables
// there before dependent package initialization. SDK children retain their
// explicit working directory and environment instead.
func runtime_envs() []string {
	if startup := kos.CurrentProcessStartup(); startup != nil {
		return startup.Env
	}
	path := kos.LoaderPath()
	if path == "" {
		return nil
	}
	data, status := kos.ReadAllFile(path + ".env")
	if status != kos.FileSystemOK {
		return nil
	}
	var result []string
	for start := 0; start < len(data); {
		end := start
		for end < len(data) && data[end] != '\n' {
			end++
		}
		line := data[start:end]
		start = end + 1
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		equal, valid := -1, true
		for i, value := range line {
			if value == 0 {
				valid = false
				break
			}
			if value == '=' && equal < 0 {
				equal = i
			}
		}
		if valid && equal > 0 {
			result = append(result, string(line))
		}
	}
	return sidecarEnvironmentDirectory(result, path)
}

func sidecarEnvironmentDirectory(environment []string, executable string) []string {
	directory := ""
	for _, entry := range environment {
		key, value, _ := strings.Cut(entry, "=")
		if key == "KOLIBRI_WORKING_DIRECTORY" {
			directory = value
			break
		}
	}
	if directory == "" {
		return environment
	}
	if !path.IsAbs(directory) {
		directory = path.Join(path.Dir(executable), directory)
	}
	if status := kos.ChangeCurrentFolder(directory); status != kos.FileSystemOK {
		kos.DebugString("SDK: cannot set KOLIBRI_WORKING_DIRECTORY\n")
		return environment
	}
	for index, entry := range environment {
		key, value, _ := strings.Cut(entry, "=")
		if value == "" {
			continue
		}
		switch key {
		case "HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME",
			"TMPDIR", "TMP", "TEMP", "SHELL", "SSL_CERT_FILE":
			if !path.IsAbs(value) {
				environment[index] = key + "=" + path.Join(directory, value)
			}
		case "PATH", "SSL_CERT_DIR", "XDG_CONFIG_DIRS", "XDG_DATA_DIRS":
			paths := strings.Split(value, ":")
			for index, value := range paths {
				if !path.IsAbs(value) {
					paths[index] = path.Join(directory, value)
				}
			}
			environment[index] = key + "=" + strings.Join(paths, ":")
		}
	}
	return environment
}

// Match upstream's no-cgo case. This SDK runtime does not load cgo's
// environment mirror hooks; Go callers use the upstream protected env table.
var RuntimeGodebugChanged func(string)

func setenv_c(key, value string) {
	if key == "GODEBUG" && RuntimeGodebugChanged != nil {
		RuntimeGodebugChanged(value)
	}
}
func unsetenv_c(key string) {
	if key == "GODEBUG" && RuntimeGodebugChanged != nil {
		RuntimeGodebugChanged("")
	}
}
