package file

import (
	"fmt"
	"os"
	"path/filepath"
)

//
// A safer file writer that writes to a temp file first and then renames it into place.
// This should allow us to avoid partial writes and provide moderate protection against symlink races.
//

type AtomicWriter struct {
	path      string
	dir       string
	tmpPath   string
	file      *os.File
	committed bool
}

// NewAtomicWriter creates a temp file in the target directory and prepares it for streamed writing
func NewAtomicWriter(path string, mode os.FileMode) (*AtomicWriter, error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	if err := refuseIfSymlink(path); err != nil {
		return nil, err
	}

	f, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	tmpPath := f.Name()

	if err := f.Chmod(mode); err != nil {
		f.Close()
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("chmod temp file: %w", err)
	}

	return &AtomicWriter{
		path:    path,
		dir:     dir,
		tmpPath: tmpPath,
		file:    f,
	}, nil
}

func (w *AtomicWriter) Write(p []byte) (int, error) {
	if w.file == nil {
		return 0, fmt.Errorf("writer is closed")
	}
	return w.file.Write(p)
}

// Commit flushes the file to disk, closes it, and atomically renames it into place.
func (w *AtomicWriter) Commit() error {
	if w.file == nil {
		return fmt.Errorf("writer is already closed")
	}
	if w.committed {
		return fmt.Errorf("writer already committed")
	}

	if err := w.file.Sync(); err != nil {
		w.cleanup()
		return fmt.Errorf("sync temp file: %w", err)
	}

	if err := w.file.Close(); err != nil {
		w.cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}
	w.file = nil

	if err := refuseIfSymlink(w.path); err != nil {
		w.cleanup()
		return err
	}

	if err := os.Rename(w.tmpPath, w.path); err != nil {
		w.cleanup()
		return fmt.Errorf("rename temp file into place: %w", err)
	}

	if err := syncDir(w.dir); err != nil {
		_ = os.Remove(w.tmpPath)
		return err
	}

	w.committed = true
	return nil
}

func (w *AtomicWriter) Abort() {
	if w.committed {
		return
	}
	w.cleanup()
}

func (w *AtomicWriter) cleanup() {
	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}
	if w.tmpPath != "" {
		_ = os.Remove(w.tmpPath)
		w.tmpPath = ""
	}
}

func refuseIfSymlink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("lstat %s: %w", path, err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to write to symlink: %s", path)
	}

	return nil
}

func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open dir for sync: %w", err)
	}
	defer f.Close()

	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync dir: %w", err)
	}
	return nil
}
