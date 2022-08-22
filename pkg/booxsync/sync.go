package booxsync

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type SyncConfig struct {
	Host        string   `json:"host"`
	SyncRoot    string   `json:"syncRoot"`
	PathsToSkip []string `json:"pathsToSkip"`
	DryRun      bool
}

func GetConfig() (*SyncConfig, error) {
	dryRun := flag.Bool("dryRun", false, "just pretend")
	flag.Parse()

	b, err := os.ReadFile("config.json")

	if err != nil {
		return nil, fmt.Errorf("getConfig: could not open config file: %w", err)
	}

	var config SyncConfig

	err = json.Unmarshal(b, &config)

	if err != nil {
		return nil, fmt.Errorf("getConfig: unmarshalling config failed: %w", err)
	}

	if _, err := os.Stat(config.SyncRoot); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("getConfig: folder %q does not exist", config.SyncRoot)
	}

	config.DryRun = *dryRun

	return &config, nil
}

func Sync(config SyncConfig) ([]string, error) {
	uploadedFiles := make([]string, 0)

	boox, err := GetBooxLibrary(&config)
	if err != nil {
		return uploadedFiles, fmt.Errorf("sync: getting Boox library failed: %w", err)
	}

	fileSystem := os.DirFS(config.SyncRoot)

	err = fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("sync: walkDirFunc got error: %w", err)
		}

		if path == "." {
			return nil
		}

		for _, pathToSkip := range config.PathsToSkip {
			if strings.Contains(path, pathToSkip) {
				log.Printf("skipping %q", path)
				return fs.SkipDir
			}
		}

		exists, _ := boox.Exists(path)

		if !exists {
			parent, err := boox.Stat(filepath.Dir(path))
			if err != nil {
				return fmt.Errorf("sync: stat'ing %q: %w", path, err)
			}
			if d.IsDir() {
				err = boox.CreateFolder(d.Name(), parent)
				if err != nil {
					return fmt.Errorf("sync: creating folder %q: %w", path, err)
				}
			} else if !d.IsDir() {
				err = boox.Upload(path, parent)
				if err != nil {
					return fmt.Errorf("sync: uploading %q: %w", path, err)
				}
				uploadedFiles = append(uploadedFiles, path)
			}
		}

		return nil
	})

	if err != nil {
		return make([]string, 0), err
	}

	return uploadedFiles, nil
}
