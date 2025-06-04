package watcher

import (
	"creation-date-saver/domain"
	"creation-date-saver/internal/filter"
	"log"
	"os"
	"path/filepath"

	"github.com/djherbis/times"
	"github.com/fsnotify/fsnotify"
)

type timeRepo interface {
	Get(relPath string) (domain.Metadata, bool)
	Upsert(meta domain.Metadata) error
	Delete(relPath string) error
}

// WatchFolder monitors the specified folder and handles file system events.
func WatchFolder(folder string, includeSubfolders bool, repo timeRepo) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	done := make(chan bool)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Skip temporary or unwanted files.
				if filter.IsTemporaryFile(event.Name) {
					continue
				}

				// Calculate relative path once for all operations.
				relPath, err := filepath.Rel(folder, event.Name)
				if err != nil {
					log.Printf("Error calculating relative path: %v\n", err)
					continue
				}

				switch event.Op {
				case fsnotify.Create:
					log.Println("File created:", relPath)

					// Check if it's a directory, if so, add it to the watcher
					if isDir(event.Name) {
						err := watcher.Add(event.Name)
						if err != nil {
							log.Printf("Error adding directory to watcher: %v\n", err)
						}
						log.Printf("New folder added to watcher: %s\n", relPath)
					} else {
						t, err := times.Stat(event.Name)
						if err == nil && t.HasBirthTime() {
							// Use birth time if available.
							repo.Upsert(domain.Metadata{
								Patch:        relPath,
								CreationTime: t.BirthTime(),
							})
						} else {
							// Fall back to modification time if birth time is not available.
							fileInfo, err := os.Stat(event.Name)
							if err == nil {
								repo.Upsert(domain.Metadata{
									Patch:        relPath,
									CreationTime: fileInfo.ModTime(),
								})
							}
						}
					}

				case fsnotify.Rename, fsnotify.Remove:
					log.Println("File removed or renamed:", relPath, event.Op)
					repo.Delete(relPath)

				case fsnotify.Write:
					log.Println("File written:", relPath)

				default:
					log.Println("Unhandled event:", relPath, event.Op)
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Watcher error:", err)
			}
		}
	}()

	// Add the folder to the watcher.
	err = watcher.Add(folder)
	if err != nil {
		return err
	}

	// Add subfolders if the option is enabled.
	if includeSubfolders {
		filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				err := watcher.Add(path)
				if err != nil {
					log.Printf("Error adding directory to watcher: %v\n", err)
				}
			}
			return nil
		})
	}

	<-done
	return nil
}

// Helper function to check if a path is a directory
func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
