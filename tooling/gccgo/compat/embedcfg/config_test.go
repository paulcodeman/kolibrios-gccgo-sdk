package embedcfg

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func write(t *testing.T, directory, name, contents string) string {
	t.Helper()
	file := filepath.Join(directory, name)
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	return file
}

// Compare the selected-source scanner and resolver with the actual Go
// command, including directory traversal, quoted names and hidden files.
func TestMatchesGoCommand(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", "module embedfixture\n\ngo 1.23\n")
	source := write(t, dir, "main.go", "package embedfixture\nimport \"embed\"\n"+
		"//go:embed assets \"space file.txt\"\nvar FS embed.FS\n"+
		"//go:embed all:assets\nvar All embed.FS\n"+
		"var example = `//go:embed nonexistent`\n"+
		"/* //go:embed also-nonexistent */\n")
	for _, name := range []string{"assets/a.txt", "assets/nested/b.bin", "assets/.hidden", "assets/_skip/c.txt", "space file.txt", "assets/.git/ignored.txt", "assets/module/go.mod", "assets/module/ignored.txt"} {
		write(t, dir, name, name)
	}
	config, err := Generate([]string{source})
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "list", "-json", ".")
	command.Dir = dir
	data, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	var upstream struct {
		EmbedPatterns []string
		EmbedFiles    []string
	}
	if err := json.Unmarshal(data, &upstream); err != nil {
		t.Fatal(err)
	}
	for _, pattern := range upstream.EmbedPatterns {
		if _, ok := config.Patterns[pattern]; !ok {
			t.Fatalf("upstream pattern missing: %q", pattern)
		}
	}
	if len(upstream.EmbedFiles) != len(config.Files) {
		t.Fatalf("upstream %v; generated %v", upstream.EmbedFiles, config.Files)
	}
	for _, name := range upstream.EmbedFiles {
		if config.Files[name] != filepath.Join(dir, name) {
			t.Fatalf("resource path changed: %q", name)
		}
	}
	if !reflect.DeepEqual(config.Patterns["assets"], []string{"assets/a.txt", "assets/nested/b.bin"}) {
		t.Fatalf("directory filtering: %v", config.Patterns)
	}
}

func TestInvalidResources(t *testing.T) {
	for _, pattern := range []string{"missing", "../outside", "/absolute", ".", "empty", "symlink", "boundary/data.txt"} {
		t.Run(pattern, func(t *testing.T) {
			dir := t.TempDir()
			write(t, dir, "data.txt", "data")
			if err := os.Mkdir(filepath.Join(dir, "empty"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("data.txt", filepath.Join(dir, "symlink")); err != nil {
				t.Fatal(err)
			}
			write(t, dir, "boundary/go.mod", "module boundary")
			write(t, dir, "boundary/data.txt", "data")
			source := write(t, dir, "main.go", "package fixture\nimport _ \"embed\"\n//go:embed "+pattern+"\nvar Data string\n")
			if _, err := Generate([]string{source}); err == nil {
				t.Fatalf("accepted invalid pattern or resource %q", pattern)
			}
		})
	}
}
