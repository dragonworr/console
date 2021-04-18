package filesystem

import (
	"fmt"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

// Service provides functionality to serve files from a git repository. The contents are stored in memory.
type Service struct {
	Cfg    Config
	logger *zap.Logger

	// In memory cache for markdowns. Map key is the filename with stripped ".md" suffix.
	filesByName map[string]File
	mutex       sync.RWMutex

	OnFilesUpdatedHook func()
}

// NewService creates a new Git service with preconfigured Auth
func NewService(cfg Config, logger *zap.Logger, onFilesUpdatedHook func()) (*Service, error) {
	childLogger := logger.With(zap.String("source", "file_provider"))

	return &Service{
		Cfg:    cfg,
		logger: childLogger,

		filesByName:        make(map[string]File),
		OnFilesUpdatedHook: onFilesUpdatedHook,
	}, nil
}

// Start to pull contents from configured paths and setup watchers to support hot reloading upon modified files.
func (c *Service) Start() error {
	if !c.Cfg.Enabled {
		return nil
	}

	filesByName := make(map[string]File, 0)
	foundFiles := 0
	loadedFiles := 0
	for _, p := range c.Cfg.Paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			return fmt.Errorf("failed to get abs path for given path '%v': %w", p, err)
		}

		err = filepath.Walk(absPath, func(currentPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			foundFiles++
			isValid, trimmedFileName := c.isValidFileExtension(currentPath)
			if !isValid {
				return nil
			}

			if info.Size() > c.Cfg.MaxFileSize {
				c.logger.Info("skipped file because it is too large",
					zap.String("currentPath", currentPath),
					zap.Int64("file_size", info.Size()),
					zap.Int64("max_allowed_file_size", c.Cfg.MaxFileSize))
				return nil
			}
			loadedFiles++

			payload, err := ioutil.ReadFile(currentPath)
			if err != nil {
				return fmt.Errorf("failed to load file '%v' from filesystem: %w", currentPath, err)
			}

			// The proto files may refer to each other. Therefore it's important to strip the base path, so that an imported
			// proto file uses the correct relative path.
			pathWithoutBasepath := strings.Trim(path.Clean(strings.Replace(currentPath, absPath, "", 1)), "\\")
			filesByName[trimmedFileName] = File{
				Path:            pathWithoutBasepath,
				Filename:        info.Name(),
				TrimmedFilename: trimmedFileName,
				Payload:         payload,
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to load files from file system: %w", err)
		}
	}
	c.setFileContents(filesByName)
	c.logger.Info("loaded all files from filesystem", zap.Int("found_files", foundFiles), zap.Int("loaded_files", loadedFiles))

	return nil
}

// setFileContents saves file contents into memory, so that they are accessible at any time.
// filesByName is a map where the key is the filename without extension suffix (e.g. "README" instead of "README.md")
func (c *Service) setFileContents(filesByName map[string]File) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.filesByName = filesByName
}

// GetFileByFilename returns the cached content for a given filename (without extension).
// The parameter must match the filename in the git repository (case sensitive).
// If there's no match nil will be returned.
func (c *Service) GetFileByFilename(fileName string) File {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	contents, exists := c.filesByName[fileName]
	if !exists {
		return File{}
	}

	return contents
}

// GetFilesByFilename returns the cached content in a map where the filename is the key (with trimmed file extension).
func (c *Service) GetFilesByFilename() map[string]File {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.filesByName
}
