package main

import (
	"fmt"
	"log"

	"github.com/bjorm/booxsync/pkg/booxsync"
)

func main() {
	config, err := booxsync.GetConfig()

	if err != nil {
		panic(fmt.Sprintf("invalid config: %s", err))
	}

	log.Printf("syncing %s to %s", config.SyncRoot, config.Host)

	uploadedFiles, err := booxsync.Sync(*config)

	if err != nil {
		panic(fmt.Sprintf("sync failed: %s", err))
	}

	log.Printf("uploaded files: %s", uploadedFiles)

	log.Println("sync complete")
}
