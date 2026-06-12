package context

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	// MaxFullRepoSize is the threshold below which the entire repo is included verbatim.
	MaxFullRepoSize = 500 * 1024 // 500KB

	// MaxContextSize is the hard cap on total context sent to the LLM.
	MaxContextSize = 1024 * 1024 // 1MB
)

// skipDirs are directories that should never be scanned.
var skipDirs = map[string]bool{
	".git":         true,
	".revv":        true,
	".agents":      true,
	"vendor":       true,
	"node_modules": true,
	"dist":         true,
	"build":        true,
	"bin":          true,
	".next":        true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
}

// skipExtensions are file extensions to always skip (binaries, images, etc.).
var skipExtensions = map[string]bool{
	".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".ico": true, ".svg": true,
	".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
	".zip": true, ".tar": true, ".gz": true, ".bz2": true,
	".pdf": true, ".doc": true, ".docx": true,
	".mp3": true, ".mp4": true, ".wav": true, ".avi": true,
	".pyc": true, ".pyo": true,
	".DS_Store": true,
}

// priorityFiles are files that should always be included first (config, docs, entry points).
var priorityFiles = []string{
	"go.mod", "go.sum",
	"package.json", "package-lock.json", "tsconfig.json",
	"pyproject.toml", "setup.py", "setup.cfg", "requirements.txt", "Pipfile",
	"Cargo.toml", "Cargo.lock",
	"Gemfile", "Gemfile.lock",
	"pom.xml", "build.gradle", "build.gradle.kts",
	"README.md", "README.rst", "README.txt",
	"CONTRIBUTING.md", "CONTRIBUTING.rst",
	"Makefile", "CMakeLists.txt",
	"Dockerfile", "docker-compose.yml", "docker-compose.yaml",
	".github/workflows",
	".gitlab-ci.yml",
	".travis.yml",
	"Jenkinsfile",
}

type repoFile struct {
	RelPath string
	Size    int64
	Content string
}

func ReadRepositoryContext(dir string) (map[string]string, error) {
	// Phase 1: Walk the repo and collect all eligible files
	var allFiles []repoFile
	var totalSize int64
	var treeParts []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable files
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		relPath = strings.ReplaceAll(relPath, string(filepath.Separator), "/")

		if relPath == "." {
			return nil
		}

		// Skip excluded directories
		if info.IsDir() {
			baseName := filepath.Base(path)
			if skipDirs[baseName] {
				return filepath.SkipDir
			}
			treeParts = append(treeParts, relPath+"/")
			return nil
		}

		// Skip excluded extensions
		ext := strings.ToLower(filepath.Ext(path))
		if skipExtensions[ext] {
			return nil
		}

		// Skip very large individual files (> 100KB)
		if info.Size() > 100*1024 {
			treeParts = append(treeParts, fmt.Sprintf("%s (%.1fKB, skipped — too large)", relPath, float64(info.Size())/1024))
			return nil
		}

		treeParts = append(treeParts, relPath)
		totalSize += info.Size()

		allFiles = append(allFiles, repoFile{
			RelPath: relPath,
			Size:    info.Size(),
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	contextMap := make(map[string]string)

	// Always include the file tree — it's cheap and gives structural awareness
	sort.Strings(treeParts)
	contextMap["__FILE_TREE__"] = strings.Join(treeParts, "\n")

	// Phase 2: Decide strategy based on total size
	if totalSize <= MaxFullRepoSize {
		// Small repo: include everything
		for i := range allFiles {
			content, err := os.ReadFile(filepath.Join(dir, allFiles[i].RelPath))
			if err != nil {
				continue
			}
			contextMap[allFiles[i].RelPath] = string(content)
		}
	} else {
		// Large repo: prioritize important files, fill up to budget
		scored := scoreAndSort(allFiles)
		var usedSize int64

		for _, f := range scored {
			if usedSize+f.Size > MaxContextSize {
				continue
			}
			content, err := os.ReadFile(filepath.Join(dir, f.RelPath))
			if err != nil {
				continue
			}
			contextMap[f.RelPath] = string(content)
			usedSize += f.Size
		}
	}

	return contextMap, nil
}

// scoreAndSort assigns priority scores to files and sorts highest priority first.
func scoreAndSort(files []repoFile) []repoFile {
	type scoredFile struct {
		repoFile
		score int
	}

	var scored []scoredFile
	for _, f := range files {
		s := scoreFile(f.RelPath)
		scored = append(scored, scoredFile{repoFile: f, score: s})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].RelPath < scored[j].RelPath
	})

	result := make([]repoFile, len(scored))
	for i, s := range scored {
		result[i] = s.repoFile
	}
	return result
}

// scoreFile returns a priority score for a file path. Higher = more important.
func scoreFile(relPath string) int {
	base := filepath.Base(relPath)
	lower := strings.ToLower(relPath)

	// Priority config/doc files get highest score
	for _, pf := range priorityFiles {
		if base == pf || strings.HasPrefix(relPath, pf) {
			return 100
		}
	}

	// CI/CD files
	if strings.Contains(lower, ".github/workflows") || strings.Contains(lower, ".gitlab-ci") {
		return 95
	}

	// Test files are very important for understanding testing patterns
	if strings.Contains(lower, "_test.go") || strings.Contains(lower, ".test.") ||
		strings.Contains(lower, "test_") || strings.Contains(lower, "/tests/") ||
		strings.Contains(lower, "/test/") || strings.HasSuffix(lower, "_test.py") {
		return 80
	}

	// Entry points / main files
	if base == "main.go" || base == "index.ts" || base == "index.js" ||
		base == "app.py" || base == "main.py" || base == "lib.rs" {
		return 75
	}

	// Source code
	srcExts := []string{".go", ".py", ".js", ".ts", ".rs", ".java", ".rb", ".c", ".cpp", ".h"}
	ext := strings.ToLower(filepath.Ext(relPath))
	for _, srcExt := range srcExts {
		if ext == srcExt {
			return 50
		}
	}

	// Config/yaml/toml files
	if ext == ".yaml" || ext == ".yml" || ext == ".toml" || ext == ".json" || ext == ".ini" {
		return 40
	}

	// Documentation
	if ext == ".md" || ext == ".rst" || ext == ".txt" {
		return 30
	}

	return 10
}
