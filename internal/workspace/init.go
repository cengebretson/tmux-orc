package workspace

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed all:templates
var templateFS embed.FS

type InitOptions struct {
	Root              string
	WithSampleWorkers bool
	DryRun            bool
	Force             bool
}

type fileEntry struct {
	dest    string
	content string
}

func Init(opts InitOptions) error {
	root, err := filepath.Abs(opts.Root)
	if err != nil {
		return fmt.Errorf("resolving workspace path: %w", err)
	}

	entries, err := collectEntries(opts)
	if err != nil {
		return err
	}

	if opts.DryRun {
		return printDryRun(root, entries)
	}

	return writeEntries(root, entries, opts.Force)
}

func collectEntries(opts InitOptions) ([]fileEntry, error) {
	var entries []fileEntry

	err := fs.WalkDir(templateFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// The embed uses all:templates (required to include _-prefixed dirs like
		// _template), which also pulls in dotfiles such as .DS_Store. Never
		// scaffold OS junk into a workspace.
		if d.Name() == ".DS_Store" {
			return nil
		}

		isSample := strings.HasPrefix(path, "templates/workers/sample/")
		if isSample && !opts.WithSampleWorkers {
			return nil
		}

		content, err := templateFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading template %s: %w", path, err)
		}

		// strip "templates/" prefix; flatten sample workers into workers/
		rel := strings.TrimPrefix(path, "templates/")
		if isSample {
			rel = "workers/" + strings.TrimPrefix(rel, "workers/sample/")
		}

		entries = append(entries, fileEntry{dest: rel, content: string(content)})
		return nil
	})

	return entries, err
}

func printDryRun(root string, entries []fileEntry) error {
	fmt.Printf("Dry run — workspace root: %s\n\n", root)

	dirs := map[string]bool{}
	for _, e := range entries {
		dir := filepath.Dir(e.dest)
		if dir != "." {
			dirs[dir] = true
		}
	}

	// print directories first
	for dir := range dirs {
		fmt.Printf("  mkdir  %s\n", dir)
	}

	fmt.Println()

	for _, e := range entries {
		dest := filepath.Join(root, e.dest)
		if _, err := os.Stat(dest); err == nil {
			fmt.Printf("  skip   %s (already exists)\n", e.dest)
		} else {
			fmt.Printf("  create %s\n", e.dest)
		}
	}

	return nil
}

func writeEntries(root string, entries []fileEntry, force bool) error {
	created := 0
	skipped := 0

	for _, e := range entries {
		dest := filepath.Join(root, e.dest)

		if _, err := os.Stat(dest); err == nil && !force {
			fmt.Printf("  skip   %s\n", e.dest)
			skipped++
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", e.dest, err)
		}

		if err := os.WriteFile(dest, []byte(e.content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", e.dest, err)
		}

		fmt.Printf("  create %s\n", e.dest)
		created++
	}

	// always create empty dirs that hold runtime artifacts
	runtimeDirs := []string{"worktrees", "projects", "features"}
	for _, dir := range runtimeDirs {
		path := filepath.Join(root, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}

	writeGitignore(root, force)

	fmt.Printf("\nDone. %d created, %d skipped.\n", created, skipped)
	if skipped > 0 {
		fmt.Println("Use --force to overwrite existing files.")
	}
	fmt.Printf("\nWorkspace ready at: %s\n\n", root)
	fmt.Println("Next step: run setup with your agent of choice:")
	fmt.Println(`  claude "Read SETUP.md and follow the setup instructions"`)
	fmt.Println(`  codex  "Read SETUP.md and follow the setup instructions"`)

	return nil
}

func writeGitignore(root string, force bool) {
	dest := filepath.Join(root, ".gitignore")
	if _, err := os.Stat(dest); err == nil && !force {
		return
	}
	content := "worktrees/\n"
	_ = os.WriteFile(dest, []byte(content), 0644)
}
