package main

import (
	"creation-date-saver/config"
	"creation-date-saver/internal/filter"
	"creation-date-saver/internal/fs/linux"
	"creation-date-saver/internal/processor"
	"creation-date-saver/internal/storage/jsonstore"
	"creation-date-saver/internal/watcher"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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
			SaveDelay:   conf.SaveDelaySeconds * time.Second,
			DeleteDelay: conf.DeleteDelaySeconds * time.Second,
		},
	)
	if err != nil {
		log.Fatalf("Error creating JSON repository: %v", err)
	}

	fsRepo := linux.NewFileSystem()

	// Load ignore patterns from ./.crignore located in the watched folder.
	var flt *filter.Filter
	ignoreFile := filepath.Join(conf.WatchFolder, ".crignore")
	if data, err := os.ReadFile(ignoreFile); err == nil {
		lines := splitLines(string(data))
		if f, ferr := filter.NewFilter(lines); ferr != nil {
			log.Printf("Failed to compile .crignore patterns: %v", ferr)
		} else {
			flt = f
		}
	}

	processor := processor.New(datetimeRepo, fsRepo, 100*time.Millisecond)
	processor.SyncFolderMetadata(conf.WatchFolder, conf.IncludeSubfolders, flt)

	// Start watching for changes in the folder
	err = watcher.WatchFolder(conf.WatchFolder, conf.IncludeSubfolders, processor, flt)
	if err != nil {
		log.Fatalf("Error folder watching: %v", err)
	}

	fmt.Println("The script is running, changes in the folder are being monitored:", conf.WatchFolder)
}

// splitLines splits on CRLF/CR/LF and trims whitespace; ignores empty and comment lines starting with #.
func splitLines(s string) []string {
	raw := strings.Split(strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n"), "\n")
	out := make([]string, 0, len(raw))
	for _, ln := range raw {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		out = append(out, ln)
	}
	return out
}
