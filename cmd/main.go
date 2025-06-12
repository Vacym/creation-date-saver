package main

import (
	"creation-date-saver/config"
	"creation-date-saver/internal/fs/linux"
	"creation-date-saver/internal/processor"
	"creation-date-saver/internal/storage/jsonstore"
	"creation-date-saver/internal/watcher"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"time"
)

func main() {
	configPath := flag.String("config", "./config.yaml", "Path to config file")
	flag.Parse()

	// Load configuration
	conf, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Error config loading: %v", err)
	}

	metadataFilePath := conf.MetadataFile
	if !filepath.IsAbs(metadataFilePath) {
		metadataFilePath = filepath.Join(conf.WatchFolder, metadataFilePath)
	}

	datetimeRepo, err := jsonstore.CreateTimeRepo(
		metadataFilePath,
		jsonstore.RepoConfig{
			SaveDelay: conf.SaveDelaySeconds * time.Second,
		},
	)
	if err != nil {
		log.Fatalf("Error creating JSON repository: %v", err)
	}

	fsRepo := linux.NewFileSystem()

	processor := processor.New(datetimeRepo, fsRepo)
	processor.SyncFolderMetadata(conf.WatchFolder, conf.IncludeSubfolders)

	// Start watching for changes in the folder
	err = watcher.WatchFolder(conf.WatchFolder, conf.IncludeSubfolders, processor)
	if err != nil {
		log.Fatalf("Error folder watching: %v", err)
	}

	fmt.Println("The script is running, changes in the folder are being monitored:", conf.WatchFolder)
}
