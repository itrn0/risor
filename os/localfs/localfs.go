package localfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ros "github.com/itrn0/risor/os"
)

var _ ros.FS = (*Filesystem)(nil)

type Filesystem struct {
	ctx  context.Context
	base string
}

// Option is a configuration function for a local Filesystem.
type Option func(*Filesystem)

// WithBase sets the base directory for the filesystem.
func WithBase(dir string) Option {
	return func(fs *Filesystem) {
		fs.base = dir
	}
}

// New creates a new local filesystem with the given options.
func New(ctx context.Context, opts ...Option) (*Filesystem, error) {
	fs := &Filesystem{ctx: ctx}
	for _, opt := range opts {
		opt(fs)
	}
	if fs.base != "" {
		orig := fs.base
		fs.base = filepath.Clean(orig)
		if strings.HasPrefix(fs.base, "..") {
			return nil, fmt.Errorf("invalid base path for filesystem: %s", orig)
		}
	}
	return fs, nil
}

func (fs *Filesystem) resolvePath(path, op string) (string, error) {
	return ros.ResolvePath(fs.base, path, op)
}

func (fs *Filesystem) Create(name string) (ros.File, error) {
	path, err := fs.resolvePath(name, "create")
	if err != nil {
		return nil, err
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, ros.MassagePathError(fs.base, err)
	}
	return f, nil
}

func (fs *Filesystem) Mkdir(name string, perm ros.FileMode) error {
	path, err := fs.resolvePath(name, "mkdir")
	if err != nil {
		return err
	}
	return os.Mkdir(path, perm)
}

func (fs *Filesystem) MkdirAll(path string, perm ros.FileMode) error {
	resolvedPath, err := fs.resolvePath(path, "mkdir")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(resolvedPath, perm); err != nil {
		return ros.MassagePathError(fs.base, err)
	}
	return nil
}

func (fs *Filesystem) MkdirTemp(dir, pattern string) (string, error) {
	if dir != "" {
		var err error
		dir, err = fs.resolvePath(dir, "mkdir")
		if err != nil {
			return "", err
		}
	}
	result, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		return "", ros.MassagePathError(fs.base, err)
	}
	return result, nil
}

func (fs *Filesystem) Open(name string) (ros.File, error) {
	resolvedPath, err := fs.resolvePath(name, "open")
	if err != nil {
		return nil, err
	}
	f, err := os.Open(resolvedPath)
	if err != nil {
		return nil, ros.MassagePathError(fs.base, err)
	}
	return f, nil
}

func (fs *Filesystem) OpenFile(name string, flag int, perm ros.FileMode) (ros.File, error) {
	path, err := fs.resolvePath(name, "open")
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, flag, perm)
	if err != nil {
		return nil, ros.MassagePathError(fs.base, err)
	}
	return f, nil
}

func (fs *Filesystem) ReadFile(name string) ([]byte, error) {
	resolvedPath, err := fs.resolvePath(name, "read")
	if err != nil {
		return nil, err
	}
	f, err := os.Open(resolvedPath)
	if err != nil {
		return nil, ros.MassagePathError(fs.base, err)
	}
	defer f.Close()
	return io.ReadAll(f)
}

func (fs *Filesystem) Remove(name string) error {
	resolvedPath, err := fs.resolvePath(name, "remove")
	if err != nil {
		return err
	}
	if err := os.Remove(resolvedPath); err != nil {
		return ros.MassagePathError(fs.base, err)
	}
	return nil
}

func (fs *Filesystem) RemoveAll(path string) error {
	resolvedPath, err := fs.resolvePath(path, "remove")
	if err != nil {
		return err
	}
	if err := os.RemoveAll(resolvedPath); err != nil {
		return ros.MassagePathError(fs.base, err)
	}
	return nil
}

func (fs *Filesystem) Rename(oldpath, newpath string) error {
	resolvedOld, err := fs.resolvePath(oldpath, "rename")
	if err != nil {
		return err
	}
	resolvedNew, err := fs.resolvePath(newpath, "rename")
	if err != nil {
		return err
	}
	if err := os.Rename(resolvedOld, resolvedNew); err != nil {
		return ros.MassagePathError(fs.base, err)
	}
	return nil
}

func (fs *Filesystem) Stat(name string) (ros.FileInfo, error) {
	resolvedPath, err := fs.resolvePath(name, "stat")
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return nil, ros.MassagePathError(fs.base, err)
	}
	return info, nil
}

func (fs *Filesystem) Symlink(oldname, newname string) error {
	resolvedOld, err := fs.resolvePath(oldname, "symlink")
	if err != nil {
		return err
	}
	resolvedNew, err := fs.resolvePath(newname, "symlink")
	if err != nil {
		return err
	}
	if err := os.Symlink(resolvedOld, resolvedNew); err != nil {
		return ros.MassagePathError(fs.base, err)
	}
	return nil
}

func (fs *Filesystem) WriteFile(name string, data []byte, perm ros.FileMode) error {
	resolvedPath, err := fs.resolvePath(name, "write")
	if err != nil {
		return err
	}
	if err := os.WriteFile(resolvedPath, data, perm); err != nil {
		return ros.MassagePathError(fs.base, err)
	}
	return nil
}

func (fs *Filesystem) ReadDir(name string) ([]ros.DirEntry, error) {
	resolvedPath, err := fs.resolvePath(name, "read")
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(resolvedPath)
	if err != nil {
		return nil, ros.MassagePathError(fs.base, err)
	}
	results := make([]ros.DirEntry, 0, len(entries))
	for _, entry := range entries {
		results = append(results, &ros.DirEntryWrapper{DirEntry: entry})
	}
	return results, nil
}

func (fs *Filesystem) WalkDir(root string, fn ros.WalkDirFunc) error {
	resolvedRoot, err := fs.resolvePath(root, "read")
	if err != nil {
		return err
	}
	return filepath.WalkDir(resolvedRoot, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return fn(path, nil, err)
		}
		return fn(path, &ros.DirEntryWrapper{DirEntry: info}, nil)
	})
}
