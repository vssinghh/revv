package context

import (
	"os"
	"path/filepath"
	"strings"
)

// ReadExistingRevvConfig reads all files from an existing .revv/ directory
// and returns them as a path→content map. Returns an empty map if .revv/ doesn't exist.
func ReadExistingRevvConfig(dir string) (map[string]string, error) {
	revvDir := filepath.Join(dir, ".revv")
	configMap := make(map[string]string)

	info, err := os.Stat(revvDir)
	if err != nil {
		if os.IsNotExist(err) {
			return configMap, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return configMap, nil
	}

	err = filepath.Walk(revvDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(revvDir, path)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Normalize path separators
		relPath = strings.ReplaceAll(relPath, string(filepath.Separator), "/")
		configMap[relPath] = string(content)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return configMap, nil
}
