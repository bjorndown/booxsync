package booxsync

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/imdario/mergo"
	log "github.com/sirupsen/logrus"
	"io/fs"
	"net/url"
	"os"
)

const (
	configFilePath = "./config.json"
)

type SyncConfig struct {
	Host        string
	SyncRoot    string
	PathsToSkip []string
	DryRun      bool
	Debug       bool
	syncUrl     url.URL
}

func getCmdLineConfig() SyncConfig {
	dryRun := flag.Bool("dry-run", false, "do everything except write to Boox library")
	debug := flag.Bool("debug", false, "enabled debug output")
	host := flag.String("host", "", "host to sync against")
	syncRoot := flag.String("sync-root", "", "path of folder to sync")

	log.Debug("parsing commandline flags")
	flag.Parse()

	return SyncConfig{DryRun: *dryRun, Host: *host, SyncRoot: *syncRoot, Debug: *debug}
}

func getFileConfig() (SyncConfig, error) {
	if _, err := os.Stat(configFilePath); errors.Is(err, fs.ErrNotExist) {
		log.Debugf("config file %q does not exist", configFilePath)
		return SyncConfig{}, nil
	}

	b, err := os.ReadFile(configFilePath)

	if err != nil {
		return SyncConfig{}, fmt.Errorf("config: could not open file: %w", err)
	}

	log.Debugf("config file %q exists", configFilePath)
	var config SyncConfig

	err = json.Unmarshal(b, &config)
	log.Debugf("read config: %v", config)

	if err != nil {
		return SyncConfig{}, fmt.Errorf("config: unmarshalling failed: %w", err)
	}

	return config, nil
}

func checkConfig(config SyncConfig) error {
	if len(config.Host) == 0 {
		return fmt.Errorf("check config: host cannot be empty")
	}
	if _, err := os.Stat(config.SyncRoot); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("check config: sync root %q does not exist", config.SyncRoot)
	}
	return nil
}

func GetConfig() (SyncConfig, error) {
	cmdLineConfig := getCmdLineConfig()
	config, err := getFileConfig()

	if err != nil {
		return SyncConfig{}, err
	}

	if config.Debug || cmdLineConfig.Debug {
		log.SetLevel(log.DebugLevel)
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
		log.Debug("debug mode enabled")
	}

	log.Debugf("command line config %#v", cmdLineConfig)
	log.Debugf("file config %#v", config)

	err = mergo.Merge(&config, cmdLineConfig, mergo.WithOverride)

	if err = checkConfig(config); err == nil {
		config.syncUrl = url.URL{Scheme: "http", Host: fmt.Sprintf("%s:%d", config.Host, 8085)}
		log.Debugf("final config %#v", config)
		return config, nil
	} else {
		return SyncConfig{}, err
	}
}
