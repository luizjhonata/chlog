package internal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrGitDirNotFound = errors.New(".git directory not found")

func FindGitDir(startDir string) (string, error) {
	dir := startDir

	for {
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			return gitPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrGitDirNotFound
		}
		dir = parent
	}
}

func FindGitDirUpward() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	return FindGitDir(dir)
}
