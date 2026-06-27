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
	return nil
}
