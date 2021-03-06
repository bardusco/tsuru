// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package fs/testing provides fake implementations of the fs package.
//
// These implementations can be used to mock out the file system in tests.
package testing

import (
	"fmt"
	"github.com/globocom/tsuru/fs"
	"os"
	"strings"
	"syscall"
)

// FakeFile representss a fake instance of the File interface.
//
// Methods from FakeFile act like methods in os.File, but instead of working in
// a real file, them work in an internal string.
//
// An instance of FakeFile is returned by RecordingFs.Open method.
type FakeFile struct {
	content string
	current int64
	r       *strings.Reader
}

func (f *FakeFile) reader() *strings.Reader {
	if f.r == nil {
		f.r = strings.NewReader(f.content)
	}
	return f.r
}

func (f *FakeFile) Close() error {
	f.current = 0
	return nil
}

func (f *FakeFile) Read(p []byte) (n int, err error) {
	n, err = f.reader().Read(p)
	f.current += int64(n)
	return
}

func (f *FakeFile) ReadAt(p []byte, off int64) (n int, err error) {
	n, err = f.reader().ReadAt(p, off)
	f.current += off + int64(n)
	return
}

func (f *FakeFile) Seek(offset int64, whence int) (int64, error) {
	var err error
	f.current, err = f.reader().Seek(offset, whence)
	return f.current, err
}

func (f *FakeFile) Stat() (fi os.FileInfo, err error) {
	return
}

func (f *FakeFile) Write(p []byte) (n int, err error) {
	n = len(p)
	f.content = f.content[:f.current] + string(p)
	return
}

func (f *FakeFile) WriteString(s string) (ret int, err error) {
	ret = len(s)
	f.content = s
	return
}

func (f *FakeFile) Truncate(size int64) error {
	f.content = f.content[:size]
	return nil
}

// RecordingFs implements the Fs interface providing a "recording" file system.
//
// A recording file system is a file system that does not execute any action,
// just record them.
//
// All methods from RecordingFs never return errors.
type RecordingFs struct {
	actions []string
	files   map[string]*FakeFile

	// FileContent is used to provide content for files opened using
	// RecordingFs.
	FileContent string
}

// HasAction checks if a given action was executed in the filesystem.
//
// For example, when you call the Open method with the "/tmp/file.txt"
// argument, RecordingFs will store locally the action "open /tmp/file.txt" and
// you can check it calling HasAction:
//
//     rfs.Open("/tmp/file.txt")
//     rfs.HasAction("open /tmp/file.txt") // true
func (r *RecordingFs) HasAction(action string) bool {
	for _, a := range r.actions {
		if action == a {
			return true
		}
	}
	return false
}

func (r *RecordingFs) open(name string, read bool) (fs.File, error) {
	if r.files == nil {
		r.files = make(map[string]*FakeFile)
		if r.FileContent == "" && read {
			return nil, syscall.ENOENT
		}
	} else if f, ok := r.files[name]; ok {
		f.r = nil
		return f, nil
	} else if r.FileContent == "" && read {
		return nil, syscall.ENOENT
	}
	fil := &FakeFile{content: r.FileContent}
	r.files[name] = fil
	return fil, nil
}

// Create records the action "create <name>" and returns an instance of
// FakeFile and nil error.
func (r *RecordingFs) Create(name string) (fs.File, error) {
	r.actions = append(r.actions, "create "+name)
	return r.open(name, false)
}

// Mkdir records the action "mkdir <name> with mode <perm>" and returns nil.
func (r *RecordingFs) Mkdir(name string, perm os.FileMode) error {
	r.actions = append(r.actions, fmt.Sprintf("mkdir %s with mode %#o", name, perm))
	return nil
}

// MkdirAll records the action "mkdirall <path> with mode <perm>" and returns
// nil.
func (r *RecordingFs) MkdirAll(path string, perm os.FileMode) error {
	r.actions = append(r.actions, fmt.Sprintf("mkdirall %s with mode %#o", path, perm))
	return nil
}

// Open records the action "open <name>" and returns an instance of FakeFile
// and nil error.
func (r *RecordingFs) Open(name string) (fs.File, error) {
	r.actions = append(r.actions, "open "+name)
	return r.open(name, true)
}

// OpenFile records the action "openfile <name> with mode <perm>" and returns
// an instance of FakeFile and nil error.
func (r *RecordingFs) OpenFile(name string, flag int, perm os.FileMode) (fs.File, error) {
	r.actions = append(r.actions, fmt.Sprintf("openfile %s with mode %#o", name, perm))
	read := flag&syscall.O_CREAT != syscall.O_CREAT &&
		flag&syscall.O_APPEND != syscall.O_APPEND &&
		flag&syscall.O_RDWR != syscall.O_RDWR &&
		flag&syscall.O_TRUNC != syscall.O_TRUNC &&
		flag&syscall.O_WRONLY != syscall.O_WRONLY
	f, err := r.open(name, read)
	if flag&syscall.O_TRUNC == syscall.O_TRUNC {
		f.Truncate(0)
	}
	return f, err
}

func (r *RecordingFs) deleteFile(name string) {
	if r.files != nil {
		delete(r.files, name)
	}
}

// Remove records the action "remove <name>" and returns nil.
func (r *RecordingFs) Remove(name string) error {
	r.actions = append(r.actions, "remove "+name)
	r.deleteFile(name)
	return nil
}

// RemoveAll records the action "removeall <path>" and returns nil.
func (r *RecordingFs) RemoveAll(path string) error {
	r.actions = append(r.actions, "removeall "+path)
	r.deleteFile(path)
	return nil
}

// Stat records the action "stat <name>" and returns nil, nil.
func (r *RecordingFs) Stat(name string) (os.FileInfo, error) {
	r.actions = append(r.actions, "stat "+name)
	return nil, nil
}

// FailureFs is like RecordingFs, except that it returns ENOENT on Open,
// OpenFile and Remove.
type FailureFs struct {
	RecordingFs
}

// Open is used to simulate ENOENT.
func (r *FailureFs) Open(name string) (fs.File, error) {
	r.RecordingFs.Open(name)
	err := os.PathError{Err: syscall.ENOENT, Path: name}
	return nil, &err
}

func (r *FailureFs) Remove(name string) error {
	r.RecordingFs.Remove(name)
	return &os.PathError{Err: syscall.ENOENT, Path: name}
}

func (r *FailureFs) RemoveAll(path string) error {
	r.RecordingFs.RemoveAll(path)
	return &os.PathError{Err: syscall.ENOENT, Path: path}
}

func (r *FailureFs) OpenFile(name string, flag int, perm os.FileMode) (fs.File, error) {
	r.RecordingFs.OpenFile(name, flag, perm)
	return r.Open(name)
}
