// Copyright (c) 2016, Gareth Watts
// All rights reserved.

package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
)

func addSysinfoToZip(zf *zip.Writer) error {
	info := fmt.Sprintf(`OS: %s
Arch: %s
CPU Count: %d
`, runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	return addStringToZip(zf, "sysinfo.txt", info)
}

var captureFilenames = []string{restrictionsPlistName, "Status.plist"}

// addBackupInfoToZip retrieves information about the supplied backup
// and adds some information about it to the zip file including:
// * some human readable text information such as pathname, parsed pin information, etc
// * A list of all the on-disk files in the backup (but not the contents or the unhashed filenames)
// * The contents of the Status.plist and the restrictions information plist files.
// No other information is included.
func addBackupInfoToZip(zf *zip.Writer, b *backup) error {
	fn := filepath.Base(b.Path)
	if err := addStringToZip(zf, path.Join("backups", fn, "info.txt"), b.debugInfo()); err != nil {
		return err
	}

	// Enumerate the files the backup contains
	var filelist bytes.Buffer
	filepath.Walk(b.Path, func(fpath string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			fmt.Fprintf(&filelist, "%-10d %s\n", info.Size(), fpath[len(b.Path)+1:])
			if oneOf(path.Base(fpath), captureFilenames) {
				addFileToZip(zf, fpath, path.Join("backups", fn, fpath[len(b.Path)+1:]))
			}
		}
		return nil
	})

	if err := addStringToZip(zf, path.Join("backups", fn, "filelist.txt"), filelist.String()); err != nil {
		return err
	}

	return nil
}

// addFileToZip copies a single file into the supplied zip using the given filename.
func addFileToZip(zf *zip.Writer, path, fn string) error {
	f, err := os.Open(path)
	if err != nil {
		return addStringToZip(zf, fn, fmt.Sprintf("failed to open file %s: %v", path, err))
	}
	defer f.Close()
	g, err := zf.Create(fn)
	if err != nil {
		return err
	}
	_, err = io.Copy(g, f)
	return err
}

// buildDebug constructs a .zip file containing debugging information in the given target
// directory.  If targetDir is empty then it will use the user's home or desktop directory.
func buildDebug(targetDir string, backupResult string, allBackups backups) (fn string, err error) {
	if targetDir == "" {
		targetDir, err = getDefaultDir()
		if err != nil {
			return "", err
		}
	}

	fn = filepath.Join(targetDir, "pinfinder-debug.zip")
	debugFile, err := os.Create(fn)
	if err != nil {
		return "", fmt.Errorf("Failed to open %s for write: %v", fn, err)
	}
	defer debugFile.Close()

	zf := zip.NewWriter(debugFile)
	defer zf.Close()
	if err := addStringToZip(zf, "output.txt", backupResult); err != nil {
		return "", err
	}

	// Capture system info
	if err := addSysinfoToZip(zf); err != nil {
		return "", err
	}

	for _, backup := range allBackups {
		if err := addBackupInfoToZip(zf, backup); err != nil {
			return "", err
		}
	}

	return fn, nil
}

// addStringToZip adds a string as a new file to a zip file with the given filename
func addStringToZip(zf *zip.Writer, filename, content string) error {
	f, err := zf.Create(filename)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte(content))
	return err
}

// getDefaultDir returns the directory the executable is in,
// or the user's home directory, or the Windows Desktop directory if that
// is not writable
func getDefaultDir() (string, error) {
	dir := filepath.Dir(os.Args[0])
	if dir != "" && isWritable(dir) {
		return dir, nil
	}

	user, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to fetch user information: %v", err)
	}

	switch runtime.GOOS {
	case "windows":
		return filepath.Join(user.HomeDir, "Desktop"), nil
	default:
		return user.HomeDir, nil
	}
}

func isWritable(dir string) bool {
	tf, err := ioutil.TempFile(dir, "pinfinder")
	if err != nil {
		return false
	}
	defer os.Remove(tf.Name())
	tf.Close()
	return true
}

// oneOf returns true if s matches one of choices.
func oneOf(s string, choices []string) bool {
	for _, ch := range choices {
		if s == ch {
			return true
		}
	}
	return false
}
