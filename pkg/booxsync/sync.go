package booxsync

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func Sync(config SyncConfig) ([]string, error) {
	var uploadedFiles []string

	boox, err := GetBooxLibrary(&config)
	if err != nil {
		return uploadedFiles, fmt.Errorf("sync: getting Boox library failed: %w", err)
	}

	log.Debug("finished reading boox library")

	fileSystem := os.DirFS(config.SyncRoot)

	log.Debugf("about to walk sync root %q", config.SyncRoot)

	err = fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("sync: walkDirFunc got error: %w", err)
		}

		if path == "." {
			return nil
		}

		for _, pathToSkip := range config.PathsToSkip {
			if strings.Contains(path, pathToSkip) {
				log.Debugf("skipping %q", path)
				return fs.SkipDir
			}
		}

		exists, _ := boox.Exists(path)

		if !exists {
			log.Debugf("%q does not exist on boox", path)
			parent, err := boox.Stat(filepath.Dir(path))
			if err != nil {
				return fmt.Errorf("sync: stat'ing %q: %w", path, err)
			}

			if d.IsDir() {
				log.Debugf("creating %q in %q", d.Name(), parent.Name)
				err = boox.CreateFolder(d.Name(), parent)
				if err != nil {
					return fmt.Errorf("sync: creating folder %q: %w", path, err)
				}
			} else {
				log.Debugf("uploading %q to %q", filepath.Base(path), parent.Name)
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
		return []string{}, err
	}

	log.Debug("finished walking sync root")

	return uploadedFiles, nil
}
