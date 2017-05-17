package subdirfs

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/src-d/go-billy.v2"
)

// ErrSymlinkNotSupported is returned by Symlink() and Readfile() if the
// underlying filesystem does not support symlinking.
var ErrSymlinkNotSupported = errors.New("symlink not supported")

type subdirFs struct {
	underlying billy.Filesystem
	base       string
}

// New creates a new filesystem wrapping up the given 'fs'.
// The created filesystem has its base in the given subdirectory
// of the underlying filesystem.
//
// This is particularly useful to implement the Dir method for
// other filesystems.
func New(fs billy.Filesystem, base string) billy.Filesystem {
	return &subdirFs{fs, base}
}

func (s *subdirFs) underlyingPath(filename string) string {
	return s.Join(s.Base(), filename)
}

func (s *subdirFs) Create(filename string) (billy.File, error) {
	f, err := s.underlying.Create(s.underlyingPath(filename))
	if err != nil {
		return nil, err
	}

	return newFile(s, f, filename), nil
}

func (s *subdirFs) Open(filename string) (billy.File, error) {
	f, err := s.underlying.Open(s.underlyingPath(filename))
	if err != nil {
		return nil, err
	}

	return newFile(s, f, filename), nil
}

func (s *subdirFs) OpenFile(filename string, flag int, mode os.FileMode) (
	billy.File, error) {

	f, err := s.underlying.OpenFile(s.underlyingPath(filename), flag, mode)
	if err != nil {
		return nil, err
	}

	return newFile(s, f, filename), nil
}

func (s *subdirFs) TempFile(dir, prefix string) (billy.File, error) {
	f, err := s.underlying.TempFile(s.underlyingPath(dir), prefix)
	if err != nil {
		return nil, err
	}

	return newFile(s, f, s.Join(dir, filepath.Base(f.Filename()))), nil
}

func (s *subdirFs) Rename(from, to string) error {
	return s.underlying.Rename(s.underlyingPath(from), s.underlyingPath(to))
}

func (s *subdirFs) Remove(path string) error {
	return s.underlying.Remove(s.underlyingPath(path))
}

func (s *subdirFs) MkdirAll(filename string, perm os.FileMode) error {
	fullpath := s.Join(s.base, filename)
	return s.underlying.MkdirAll(fullpath, perm)
}

func (s *subdirFs) Stat(filename string) (billy.FileInfo, error) {
	fullpath := s.underlyingPath(filename)
	fi, err := s.underlying.Stat(fullpath)
	if err != nil {
		return nil, err
	}

	return newFileInfo(filepath.Base(fullpath), fi), nil
}

func (s *subdirFs) ReadDir(path string) ([]billy.FileInfo, error) {
	prefix := s.underlyingPath(path)
	fis, err := s.underlying.ReadDir(prefix)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(fis); i++ {
		rn := strings.Replace(fis[i].Name(), prefix, "", 1)
		fis[i] = newFileInfo(rn, fis[i])
	}

	return fis, nil
}

func (s *subdirFs) Join(elem ...string) string {
	return s.underlying.Join(elem...)
}

func (s *subdirFs) Dir(path string) billy.Filesystem {
	return New(s.underlying, s.underlyingPath(path))
}

func (s *subdirFs) Base() string {
	return s.base
}

// Symlink creates newname as a symbolic link to oldname.
// All parent directories are created.
func (s *subdirFs) Symlink(oldname, newname string) error {
	fs, ok := s.underlying.(billy.Symlinker)
	if !ok {
		return ErrSymlinkNotSupported
	}

	if filepath.IsAbs(oldname) {
		// only rewrite oldname if it's already absolute
		oldname = string(os.PathSeparator) + s.underlyingPath(oldname)
	}
	newname = s.underlyingPath(newname)
	return fs.Symlink(oldname, newname)
}

// Readlink returns the destination of the named symbolic link.
func (s *subdirFs) Readlink(name string) (string, error) {
	fs, ok := s.underlying.(billy.Symlinker)
	if !ok {
		return "", ErrSymlinkNotSupported
	}

	fullpath := s.underlyingPath(name)
	target, err := fs.Readlink(fullpath)
	if err != nil {
		return "", err
	}

	if !filepath.IsAbs(target) {
		return target, nil
	}

	base := string(os.PathSeparator) + s.base
	target, err = filepath.Rel(base, target)
	if err != nil {
		return "", err
	}

	return string(os.PathSeparator) + target, nil
}
