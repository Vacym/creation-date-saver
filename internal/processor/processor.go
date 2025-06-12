package processor

import (
	"creation-date-saver/domain"
	"os"
	"path/filepath"
	"time"

	"github.com/djherbis/times"
)

type timeRepo interface {
	Get(relPath string) (domain.Metadata, bool)
	Upsert(meta domain.Metadata) error
	Delete(relPath string) error
}

type fsRepo interface {
	SetCreationTime(filePath string, newCrtime time.Time) error
}

type Processor struct {
	timeRepo timeRepo
	fsRepo   fsRepo
}

// New creates a new Processor instance.
func New(repo timeRepo, fsRepo fsRepo) *Processor {
	return &Processor{
		timeRepo: repo,
		fsRepo:   fsRepo,
	}
}

func (p *Processor) HandleCreate(file domain.File) {
	t, err := times.Stat(file.Path)
	if err == nil && t.HasBirthTime() {
		// Use birth time if available.
		p.timeRepo.Upsert(domain.Metadata{
			Patch:        file.RelPath,
			CreationTime: t.BirthTime(),
		})
	} else {
		// Fall back to modification time if birth time is not available.
		fileInfo, err := os.Stat(file.Path)
		if err == nil {
			p.timeRepo.Upsert(domain.Metadata{
				Patch:        file.RelPath,
				CreationTime: fileInfo.ModTime(),
			})
		}
	}
}

func (p *Processor) HandleRemove(file domain.File) {
	p.timeRepo.Delete(file.RelPath)
}

// SyncFolderMetadata updates the metadata of all files in the root folder (and subfolders if recursive).
// If the date in the repository is less than the file's date, the file's creation date is also updated.
func (p *Processor) SyncFolderMetadata(root string, recursive bool) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != root {
			if !recursive {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		t, terr := times.Stat(path)
		var fileCreationTime time.Time
		if terr == nil && t.HasBirthTime() {
			fileCreationTime = t.BirthTime()
		} else {
			fileCreationTime = info.ModTime()
		}

		meta, ok := p.timeRepo.Get(relPath)
		if !ok {
			// Not in repo — add it
			return p.timeRepo.Upsert(domain.Metadata{
				Patch:        relPath,
				CreationTime: fileCreationTime,
			})
		}

		// Exists in repo — compare dates
		minTime := meta.CreationTime
		if fileCreationTime.Before(meta.CreationTime) {
			minTime = fileCreationTime
		}
		if !minTime.Equal(meta.CreationTime) {
			// Update if the minimum is different
			if err := p.timeRepo.Upsert(domain.Metadata{
				Patch:        relPath,
				CreationTime: minTime,
			}); err != nil {
				return err
			}
		}

		// If the date in the repository is less than the file's date — update the file's date
		if meta.CreationTime.Before(fileCreationTime) {
			// Update the file's modification time to meta.CreationTime
			err := p.fsRepo.SetCreationTime(path, meta.CreationTime)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
