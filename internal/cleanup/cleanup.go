// Package cleanup runs a daily background job that removes files in the uploads
// directory that no paste references (orphans left behind by failed deletes,
// interrupted uploads, or manual tampering).
package cleanup

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"
)

// minAge protects in-flight uploads: a file is created on disk before its paste
// row records the name, so we never touch very recent files.
const minAge = time.Hour

// partialDir mirrors handler.PartialDir — the subdir holding in-progress
// resumable uploads (.part files). Kept as a literal to avoid importing handler.
const partialDir = ".partial"

// partialMaxAge is how long an untouched .part file lingers before it's treated
// as abandoned. Generous so a paused/slow resumable upload isn't reaped mid-flight.
const partialMaxAge = 24 * time.Hour

// FileLister yields the file names still referenced by some paste.
type FileLister interface {
	ReferencedFileNames(ctx context.Context) ([]string, error)
}

// Start launches the job in the background: once now, then every 24h until ctx
// is cancelled.
func Start(ctx context.Context, lister FileLister, uploadDir string) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			if err := run(ctx, lister, uploadDir); err != nil {
				log.Printf("uploads cleanup: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func run(ctx context.Context, lister FileLister, uploadDir string) error {
	names, err := lister.ReferencedFileNames(ctx)
	if err != nil {
		return err
	}
	referenced := make(map[string]struct{}, len(names))
	for _, n := range names {
		referenced[n] = struct{}{}
	}

	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		return err
	}

	var removed int
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if _, ok := referenced[e.Name()]; ok {
			continue
		}
		// Skip files created moments ago — likely an upload mid-flight.
		if info, err := e.Info(); err != nil || time.Since(info.ModTime()) < minAge {
			continue
		}
		if err := os.Remove(filepath.Join(uploadDir, e.Name())); err != nil {
			log.Printf("uploads cleanup: remove %s: %v", e.Name(), err)
			continue
		}
		removed++
	}
	if removed > 0 {
		log.Printf("uploads cleanup: removed %d orphaned file(s)", removed)
	}

	sweepPartials(filepath.Join(uploadDir, partialDir))
	return nil
}

// sweepPartials removes resumable-upload .part files that haven't been touched
// in partialMaxAge — uploads the user abandoned (closed the tab, gave up). A
// live or merely paused upload keeps a recent mtime and is left alone. The dir
// may not exist yet (no upload ever started), which is not an error.
func sweepPartials(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("uploads cleanup: read partials: %v", err)
		}
		return
	}
	var removed int
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil || time.Since(info.ModTime()) < partialMaxAge {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			log.Printf("uploads cleanup: remove partial %s: %v", e.Name(), err)
			continue
		}
		removed++
	}
	if removed > 0 {
		log.Printf("uploads cleanup: removed %d abandoned partial upload(s)", removed)
	}
}
