package stage

import (
	"os"
	"path/filepath"
)

const Dir = "stages"
const FileExt = ".md"

// Exists reports whether a stage markdown file exists for the given name.
func Exists(workspaceRoot, name string) bool {
	_, err := os.Stat(filepath.Join(workspaceRoot, Dir, name+FileExt))
	return err == nil
}

// Read returns the markdown content of a stage file.
func Read(workspaceRoot, name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(workspaceRoot, Dir, name+FileExt))
}

// List returns all stage names (without extension) found in stages/.
func List(workspaceRoot string) []string {
	entries, err := os.ReadDir(filepath.Join(workspaceRoot, Dir))
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != FileExt {
			continue
		}
		names = append(names, e.Name()[:len(e.Name())-len(FileExt)])
	}
	return names
}
