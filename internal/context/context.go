package context

import (
	"os"
	"path/filepath"
)

func ReadRepositoryContext(dir string) (map[string]string, error) {
	filesToScan := []string{"README.md", "CONTRIBUTING.md", "Makefile", "Dockerfile"}
	contextMap := make(map[string]string)

	for _, filename := range filesToScan {
		path := filepath.Join(dir, filename)
		content, err := os.ReadFile(path)
		if err == nil {
			contextMap[filename] = string(content)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return contextMap, nil
}
