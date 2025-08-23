package processor

import (
	"creation-date-saver/domain"
	"creation-date-saver/internal/filter"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/djherbis/times"
)

type timeRepo interface {
	Get(relPath string) (domain.Metadata, bool)
	Upsert(meta domain.Metadata) error
	Delete(relPath string) error
	DeleteByPrefix(prefix string) error
}

type fsRepo interface {
	SetCreationTime(filePath string, newCrtime time.Time) error
}

type Processor struct {
	timeRepo timeRepo
	fsRepo   fsRepo

	ignoreMu    sync.Mutex
	ignoreMap   map[string]time.Time
	ignoreDelay time.Duration
}

// New creates a new Processor instance.
func New(repo timeRepo, fsRepo fsRepo, ignoreDelay time.Duration) *Processor {
	return &Processor{
		timeRepo:    repo,
		fsRepo:      fsRepo,
		ignoreMap:   make(map[string]time.Time),
		ignoreDelay: ignoreDelay,
	}
}

func (p *Processor) shouldIgnore(relPath string) bool {
	p.ignoreMu.Lock()
	defer p.ignoreMu.Unlock()
	last, ok := p.ignoreMap[relPath]
	if !ok {
		return false
	}
	if time.Since(last) < p.ignoreDelay {
		return true
	}
	return false
}

func (p *Processor) markIgnored(relPath string) {
	p.ignoreMu.Lock()
	defer p.ignoreMu.Unlock()
	p.ignoreMap[relPath] = time.Now()
}

func (p *Processor) HandleCreate(file domain.File) {
	if p.shouldIgnore(file.RelPath) {
		return
	}
	// Check if metadata already exists for this file (possible quick delete-create scenario)
	meta, ok := p.timeRepo.Get(file.RelPath)
	if ok {
		// Use the stored creation time as the true creation time
		p.markIgnored(file.RelPath)
		err := p.fsRepo.SetCreationTime(file.Path, meta.CreationTime)
		if err != nil {
			fmt.Println("Error setting creation time:", err)
		}
		// Upsert to ensure metadata is up to date
		p.timeRepo.Upsert(domain.Metadata{
			Patch:        file.RelPath,
			CreationTime: meta.CreationTime,
		})
		return
	}

	t, err := times.Stat(file.Path)

	if t == nil {
		return
	}
	crTime := t.BirthTime()
	if err != nil || !t.HasBirthTime() {
		crTime = t.ModTime()
	}

	p.timeRepo.Upsert(domain.Metadata{
		Patch:        file.RelPath,
		CreationTime: crTime,
	})
}

func (p *Processor) HandleRemove(file domain.File) {
	if p.shouldIgnore(file.RelPath) {
		return
	}
	_, ok := p.timeRepo.Get(file.RelPath)
	if ok {
		p.timeRepo.Delete(file.RelPath)
	} else {
		// If metadata does not exist, maybe it is a folder
		p.timeRepo.DeleteByPrefix(file.RelPath + string(os.PathSeparator))
	}
}

// SyncFolderMetadata updates the metadata of all files in the root folder (and subfolders if recursive).
// If the date in the repository is less than the file's date, the file's creation date is also updated.
func (p *Processor) SyncFolderMetadata(root string, recursive bool, flt *filter.Filter) error {
	const allowedDrift = 2 * time.Second
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

		// Skip files matching filter rules provided for initial sync.
		if flt != nil && (flt.ShouldFilter(relPath) || flt.ShouldFilter(path)) {
			return nil
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

		// If the difference between dates is less than the allowed drift, do not update the file time
		if meta.CreationTime.Add(allowedDrift).Before(fileCreationTime) {
			// Update the file's modification time to meta.CreationTime
			p.markIgnored(relPath)
			err := p.fsRepo.SetCreationTime(path, meta.CreationTime)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
