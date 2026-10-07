package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"kos"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var initialDirectory, initialDirectoryError = os.Getwd()
var initialHome = os.Getenv("HOME")
var initialCertificate = os.Getenv("SSL_CERT_FILE")

func require(ok bool, why string) {
	if !ok {
		fmt.Fprintln(os.Stderr, "PORTABLE_FAIL:", why, "initial cwd:", initialDirectory, "HOME:", initialHome)
		_ = os.WriteFile("/hd0/1/PORTABLE.FAIL", []byte(why+"\ninitial cwd="+initialDirectory+"\nHOME="+initialHome+"\n"), 0600)
		panic(why)
	}
}

func copyDirectory(source, target string) error {
	if err := os.MkdirAll(target, 0700); err != nil {
		return err
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		from, to := filepath.Join(source, entry.Name()), filepath.Join(target, entry.Name())
		if entry.IsDir() {
			err = copyDirectory(from, to)
		} else {
			var data []byte
			data, err = os.ReadFile(from)
			if err == nil {
				err = os.WriteFile(to, data, 0600)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func diagnoseRAM(console kos.Console) {
	base := "/tmp0/1/opencode-portable-4bc3253a2a67/opencode"
	err := copyDirectory("/hd0/1/archive/opencode", base)
	require(err == nil, fmt.Sprintf("copy full package to RAM: %v", err))
	binary := base + "/opencode.kex"
	// An explicit SDK child environment tests the original CLI error while
	// retaining its output; the next launch uses the actual native sidecar.
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	command := exec.CommandContext(ctx, binary, "-d")
	command.Dir = base
	command.Env = []string{"HOME=" + base, "XDG_CONFIG_HOME=" + base + "/.config", "USER=kolibri",
		"SHELL=" + base + "/gosh.kex", "PATH=" + base + ":/sys:/sys/develop",
		"SSL_CERT_FILE=" + base + "/certs/ca-bundle.crt"}
	output, err := command.CombinedOutput()
	cancel()
	report := fmt.Sprintf("RAM executable=%s\nUnconfigured provider exit=%v\n%s\n", binary, err, output)
	require(os.WriteFile("/hd0/1/RAM-DIAG.txt", []byte(report), 0600) == nil, "RAM diagnostic output")
	require(err != nil && strings.Contains(string(output), "agent coder not found"), "expected original missing-provider error from RAM")
	environment, err := os.ReadFile(binary + ".env")
	require(err == nil, "RAM sidecar")
	environment = append(environment, []byte("LOCAL_ENDPOINT=http://10.0.2.2:18081/v1\nOPENCODE_DEV_DEBUG=true\n")...)
	require(os.WriteFile(binary+".env", environment, 0600) == nil, "configure RAM provider")
	pid, status := kos.StartApplication(binary, "-d", false)
	report += fmt.Sprintf("Native sidecar launch with local provider: PID=%d status=%d\n", pid, status)
	require(os.WriteFile("/hd0/1/RAM-DIAG.txt", []byte(report), 0600) == nil, "native RAM launch report")
	require(status == kos.FileSystemOK && pid > 0, "native RAM executable load")
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		log, err := os.ReadFile(base + "/.opencode/debug.log")
		if err == nil && strings.Contains(string(log), "LSP clients initialization started in background") {
			require(os.WriteFile("/hd0/1/RAM-OPENCODE.log", log, 0600) == nil, "RAM TUI log")
			report += "PASS: original TUI initialized from RAM with relative sidecar paths\n"
			require(os.WriteFile("/hd0/1/RAM-DIAG.txt", []byte(report), 0600) == nil, "RAM TUI result")
			console.WriteString(report)
			if len(os.Args) > 1 && os.Args[1] == "diagnose-ram-chat" {
				// Preserve evidence on the writable HDD while the full native TUI
				// receives real keyboard input and stores its history on RAM.
				end := time.Now().Add(300 * time.Second)
				for time.Now().Before(end) {
					for _, name := range []string{"debug.log", "opencode.db", "opencode.db-wal"} {
						if data, err := os.ReadFile(base + "/.opencode/" + name); err == nil {
							_ = os.WriteFile("/hd0/1/RAM-"+name, data, 0600)
						}
					}
					entries, _ := os.ReadDir(base)
					for _, entry := range entries {
						if strings.HasPrefix(entry.Name(), "opencode-panic-") {
							data, _ := os.ReadFile(base + "/" + entry.Name())
							_ = os.WriteFile("/hd0/1/RAM-PANIC.log", data, 0600)
						}
					}
					time.Sleep(time.Second)
				}
			}
			return
		}
		time.Sleep(time.Second)
	}
	require(false, "RAM TUI initialization timed out")
}

func main() {
	if len(os.Args) > 2 && os.Args[1] == "child" {
		require(initialDirectoryError == nil && initialDirectory == os.Args[2], "child working directory overwritten")
		require(filepath.IsAbs(initialHome), "relative HOME inherited by child")
		fmt.Print("CHILD_OK")
		return
	}
	console, ok := kos.OpenConsole("Portable native launch")
	require(ok, "console")
	defer console.Exit(false)
	if len(os.Args) > 1 && os.Args[1] == "diagnose-entropy" {
		var bytes [16]byte
		n, err := rand.Read(bytes[:])
		report := fmt.Sprintf("crypto/rand.Read: bytes=%d err=%v\n", n, err)
		if err != nil {
			base := "/hd0/1/archive/opencode"
			command := exec.Command(base+"/opencode.kex", "-d", "-p", "hi", "-q")
			command.Dir = base
			command.Env = []string{"HOME=" + base, "XDG_CONFIG_HOME=" + base + "/.config",
				"SHELL=" + base + "/gosh.kex", "PATH=" + base + ":/sys:/sys/develop",
				"LOCAL_ENDPOINT=http://10.0.2.2:18081/v1", "OPENCODE_DEV_DEBUG=true"}
			output, err := command.CombinedOutput()
			report += fmt.Sprintf("Original CLI exit=%v\n%s\n", err, output)
		}
		_ = os.WriteFile("/hd0/1/ENTROPY.txt", []byte(report), 0600)
		console.WriteString(report)
		return
	}
	if len(os.Args) > 1 && (os.Args[1] == "diagnose-ram" || os.Args[1] == "diagnose-ram-chat") {
		diagnoseRAM(console)
		return
	}
	executable, err := os.Executable()
	require(err == nil, "executable path")
	directory := filepath.Dir(executable)
	require(initialDirectoryError == nil && initialDirectory == directory, "directory was not set before initialization")
	require(initialHome == directory, "HOME was not anchored before initialization")
	require(initialCertificate == directory+"/certs/marker.txt", "relative certificate path")
	require(os.Getenv("SHELL") == directory+"/gosh.kex", "relative shell path")
	require(os.Getenv("PATH") == directory+":/sys", "relative PATH entry")
	require(os.Getenv("LITERAL") == " $HOME = ${HOME} ", "ordinary value was expanded")
	require(os.Getenv("DUP") == "first", "duplicate precedence")
	data, err := os.ReadFile(initialCertificate)
	require(err == nil && string(data) == "certificate-path-ok", "relative certificate access")
	require(os.TempDir() == "/tmp0/1", "system RAM temporary directory")
	file, err := os.CreateTemp("", "portable-file-*")
	require(err == nil, "CreateTemp on system RAM disk")
	name := file.Name()
	require(strings.HasPrefix(name, "/tmp0/1/"), "temporary file path")
	require(file.Close() == nil && os.Remove(name) == nil, "temporary file cleanup")
	temporary, err := os.MkdirTemp("", "portable-dir-*")
	require(err == nil && strings.HasPrefix(temporary, "/tmp0/1/"), "MkdirTemp on system RAM disk")
	require(os.Remove(temporary) == nil, "temporary directory cleanup")
	project := directory + "/project"
	require(os.MkdirAll(project, 0700) == nil, "project directory")
	command := exec.Command(executable, "child", project)
	command.Dir = project
	output, err := command.CombinedOutput()
	require(err == nil && string(output) == "CHILD_OK", "inherited explicit child startup")
	if len(os.Args) > 1 && os.Args[1] == "diagnose-iso" {
		listing := ""
		cdrom := ""
		for _, name := range []string{"/", "/cd0", "/cd1", "/cd0/1", "/cd1/1", "/cd0/1/opencode", "/cd1/1/opencode"} {
			entries, err := os.ReadDir(name)
			listing += fmt.Sprintf("%s: %v\n", name, err)
			for _, entry := range entries {
				listing += entry.Name() + "\n"
				if name == "/" && strings.HasPrefix(entry.Name(), "cd") && cdrom == "" {
					cdrom = "/" + entry.Name() + "/1"
				}
			}
		}
		_ = os.WriteFile("/hd0/1/ISO-LIST.txt", []byte(listing), 0600)
		require(cdrom != "", "no native CD-ROM found")
		binary := cdrom + "/opencode/opencode.kex"
		environment, err := os.ReadFile(binary + ".env")
		require(err == nil, fmt.Sprintf("ISO environment file: %v", err))
		report := ""
		for _, workingDirectory := range []string{filepath.Dir(binary), "/sys"} {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			command := exec.CommandContext(ctx, binary, "-c", workingDirectory)
			command.Dir = workingDirectory
			command.Env = strings.Split(strings.TrimSpace(string(environment)), "\n")
			output, err := command.CombinedOutput()
			cancel()
			report += fmt.Sprintf("cwd=%s\nexit=%v\n%s\n", workingDirectory, err, output)
			require(os.WriteFile(directory+"/ISO-DIAG.txt", []byte(report), 0600) == nil, "diagnostic report")
		}
		require(os.WriteFile(directory+"/ISO-DIAG.txt", []byte(report), 0600) == nil, "diagnostic report")
		console.WriteString(report)
	}
	require(os.WriteFile(directory+"/PORTABLE.PASS", []byte("PASS: relative pre-init paths, relocated launch, /tmp0/1 CreateTemp/MkdirTemp, explicit child startup"), 0600) == nil, "result file")
	console.WriteString("PORTABLE_LAUNCH_PASS\n")
}
