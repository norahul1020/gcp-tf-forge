package verifier

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PruneImportFiles locates all *_imports.tf files and safely archives or deletes them.
func PruneImportFiles(dir string, keepBackup bool) (int, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return 0, fmt.Errorf("invalid directory path: %w", err)
	}

	var importFiles []string
	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), "_imports.tf") {
			importFiles = append(importFiles, path)
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed scanning directory for import files: %w", err)
	}

	if len(importFiles) == 0 {
		return 0, nil
	}

	var backupDir string
	if keepBackup {
		backupDir = filepath.Join(absDir, ".archive", "imports")
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return 0, fmt.Errorf("failed creating archive directory: %w", err)
		}
	}

	pruned := 0
	for _, file := range importFiles {
		if keepBackup {
			destFile := filepath.Join(backupDir, filepath.Base(file))
			if err := copyFile(file, destFile); err != nil {
				return pruned, fmt.Errorf("failed backing up %s: %w", file, err)
			}
		}

		if err := os.Remove(file); err != nil {
			return pruned, fmt.Errorf("failed removing %s: %w", file, err)
		}
		pruned++
	}

	return pruned, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
