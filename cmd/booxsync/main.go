package main

import (
	"github.com/bjorm/booxsync/pkg/booxsync"
	log "github.com/sirupsen/logrus"
	"strings"
)

func main() {
	config, err := booxsync.GetConfig()

	if err != nil {
		log.Infof("invalid config: %s", err)
		return
	}

	log.Infof("syncing %s to %s", config.SyncRoot, config.Host)

	uploadedFiles, err := booxsync.Sync(config)

	if err != nil {
		log.Infof("sync failed: %s", err)
		return
	}

	if len(uploadedFiles) > 0 {
		log.Infof("uploaded files:\n%s", strings.Join(uploadedFiles, "\n"))
	} else {
		log.Infoln("nothing uploaded")
	}

	log.Infoln("sync complete")
}
