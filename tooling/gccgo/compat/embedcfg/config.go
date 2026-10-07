package embedcfg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Config is the upstream compiler's embedcfg JSON format.
type Config struct {
	Patterns map[string][]string
	Files    map[string]string
}

// Generate reads directives from the original, selected compiler inputs.
// Resource paths remain rooted at those inputs when generics are lowered
// into a separate directory. No source or resource is rewritten.
func Generate(sources []string) (*Config, error) {
	var patterns []string
	var directory string
	positions := token.NewFileSet()
	for _, source := range sources {
		absolute, err := filepath.Abs(source)
		if err != nil {
			return nil, err
		}
		if directory == "" {
			directory = filepath.Dir(absolute)
		} else if directory != filepath.Dir(absolute) {
			return nil, fmt.Errorf("embed sources must belong to one package directory")
		}
		data, err := os.ReadFile(absolute)
		if err != nil {
			return nil, err
		}
		if !bytes.Contains(data, []byte("//go:embed")) {
			continue
		}
		file, err := parser.ParseFile(positions, absolute, data, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		for _, group := range file.Comments {
			for _, comment := range group.List {
				text := comment.Text
				if text != "//go:embed" && !strings.HasPrefix(text, "//go:embed ") && !strings.HasPrefix(text, "//go:embed\t") {
					continue
				}
				position := positions.Position(comment.Slash)
				position.Offset += len("//go:embed")
				position.Column += len("//go:embed")
				embeds, err := parseGoEmbed(text[len("//go:embed"):], position)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", position, err)
				}
				for _, embed := range embeds {
					patterns = append(patterns, embed.pattern)
				}
			}
		}
	}
	if len(patterns) == 0 {
		return nil, nil
	}
	files, mapping, err := resolveEmbed(directory, patterns)
	if err != nil {
		return nil, err
	}
	config := &Config{Patterns: mapping, Files: make(map[string]string)}
	for _, name := range files {
		config.Files[name] = filepath.Join(directory, filepath.FromSlash(name))
	}
	return config, nil
}

// Write produces a compiler input in the build cache, leaving resources at
// their original locations. An empty configuration also supports scans of
// files containing documentation examples of embed directives.
func Write(output string, sources []string) error {
	config, err := Generate(sources)
	if err != nil {
		return err
	}
	if config == nil {
		config = &Config{Patterns: map[string][]string{}, Files: map[string]string{}}
	}
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	return os.WriteFile(output, append(data, '\n'), 0644)
}
