package main

import (
	"creation-date-saver/config"
	"creation-date-saver/internal/storage/jsonstore"
	"creation-date-saver/internal/watcher"
	"fmt"
	"log"
	"path/filepath"
	"time"
)

func main() {
	// Загружаем конфигурацию
	conf, err := config.LoadConfig("./config.yaml")
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

	// Запускаем отслеживание изменений в папке
	err = watcher.WatchFolder(conf.WatchFolder, conf.IncludeSubfolders, datetimeRepo)
	if err != nil {
		log.Fatalf("Error folder watching: %v", err)
	}

	fmt.Println("The script is running, changes in the folder are being monitored:", conf.WatchFolder)
}
