// Copyright (c) 2016, Gareth Watts
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//     * Redistributions of source code must retain the above copyright
//       notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above copyright
//       notice, this list of conditions and the following disclaimer in the
//       documentation and/or other materials provided with the distribution.
//     * Neither the name of the <organization> nor the
//       names of its contributors may be used to endorse or promote products
//       derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL <COPYRIGHT HOLDER> BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// iOS Restrictions Passcode Finder
//
// This program will examine an iTunes backup folder for an iOS device and attempt
// to find the PIN used for restricting permissions on the device (NOT the unlock PIN)

// To regenerate licenses.go:
// 1) go get github.com/gwatts/emebdfiles
// 2) go generate

//go:generate embedfiles -filename licenses.go -var licenses LICENSE*

package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/DHowett/go-plist"

	"golang.org/x/crypto/pbkdf2"
)

const (
	maxPIN                = 10000
	version               = "1.5.0"
	restrictionsPlistName = "398bc9c2aeeab4cb0c12ada0f52eea12cf14f40b"

	msgIsEncrypted = "backup is encrypted"
	msgNoPasscode  = "no passcode stored"
)

var (
	noPause     = flag.Bool("nopause", false, "Set to true to prevent the program pausing for input on completion")
	showLicense = flag.Bool("license", false, "Display license information")
)

func isDir(p string) bool {
	s, err := os.Stat(p)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func dumpFile(fn string) {
	if f, err := os.Open(fn); err != nil {
		fmt.Printf("Failed to open %s: %s\n", fn, err)
	} else {
		defer f.Close()
		io.Copy(os.Stdout, f)
	}
}

// figure out where iTunes keeps its backups on the current OS
func findSyncDir() (string, error) {

	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get information about current user: %s", err)
	}

	var dir string
	switch runtime.GOOS {
	case "darwin":
		dir = filepath.Join(usr.HomeDir, "Library", "Application Support", "MobileSync", "Backup")

	case "windows":
		// this seesm to be correct for all versions of Windows.. Tested on XP and Windows 8
		dir = filepath.Join(os.Getenv("APPDATA"), "Apple Computer", "MobileSync", "Backup")

	default:
		return "", errors.New("could not detect backup directory for this operating system; pass explicitly")
	}
	if !isDir(dir) {
		return "", fmt.Errorf("directory %s does not exist", dir)
	}
	return dir, nil
}

func parsePlist(fn string, target interface{}) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}

	return plist.NewDecoder(f).Decode(target)
}

type backup struct {
	path             string
	status           string
	restrictionsPath string
	info             struct {
		LastBackup     time.Time `plist:"Last Backup Date"`
		DisplayName    string    `plist:"Display Name"`
		ProductName    string    `plist:"Product Name"`
		ProductType    string    `plist:"Product Type"`
		ProductVersion string    `plist:"Product Version"`
	}
	manifest struct {
		IsEncrypted interface{} `plist:"IsEncrypted"`
	}
	restrictions struct {
		Key  []byte `plist:"RestrictionsPasswordKey"`
		Salt []byte `plist:"RestrictionsPasswordSalt"`
	}
}

func (b *backup) isEncrypted() bool {
	switch v := b.manifest.IsEncrypted.(type) {
	case int:
		return v != 0
	case uint64:
		return v != 0
	case bool:
		return v
	case nil:
		return false
	default:
		return false
	}
}

type backups []*backup

func (b backups) Len() int { return len(b) }
func (b backups) Less(i, j int) bool {
	return b[i].info.LastBackup.Before(b[j].info.LastBackup)
}
func (b backups) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

func loadBackups(syncDir string) (backups backups, err error) {
	// loop over all directories and see whether they contain an Info.plist
	d, err := os.Open(syncDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open directory %q: %s", syncDir, err)
	}
	defer d.Close()
	fl, err := d.Readdir(-1)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %q: %s", syncDir, err)
	}
	for _, fi := range fl {
		if !fi.Mode().IsDir() {
			continue
		}
		path := filepath.Join(syncDir, fi.Name())
		if b := loadBackup(path); b != nil {
			backups = append(backups, b)
		}
	}
	sort.Sort(sort.Reverse(backups))
	return backups, nil
}

func loadBackup(backupDir string) *backup {
	var b backup

	if err := parsePlist(filepath.Join(backupDir, "Info.plist"), &b.info); err != nil {
		return nil // no Info.plist == invalid backup dir
	}

	if err := parsePlist(filepath.Join(backupDir, "Manifest.plist"), &b.manifest); err != nil {
		return nil // no Manifest.plist == invaild backup dir
	}

	b.restrictionsPath = filepath.Join(backupDir, restrictionsPlistName)
	if _, err := os.Stat(b.restrictionsPath); err != nil {
		// iOS 10 moved backup files into sub-folders beginning with
		// the first 2 letters of the filename.
		b.restrictionsPath = filepath.Join(backupDir, restrictionsPlistName[:2], restrictionsPlistName)
	}

	if err := parsePlist(b.restrictionsPath, &b.restrictions); os.IsNotExist(err) {
		b.status = msgNoPasscode

	} else if err != nil {
		if b.isEncrypted() {
			b.status = msgIsEncrypted
		} else {
			b.status = err.Error()
		}
	}

	b.path = backupDir
	return &b
}

type swg struct{ sync.WaitGroup }

func (wg *swg) WaitChan() chan struct{} {
	c := make(chan struct{}, 1)
	go func() {
		wg.Wait()
		c <- struct{}{}
	}()
	return c
}

// use all available cores to brute force the PIN
func findPIN(key, salt []byte) (string, error) {
	found := make(chan string)
	var wg swg
	var start, end int

	perCPU := maxPIN / runtime.NumCPU()

	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		if i == runtime.NumCPU()-1 {
			end = maxPIN
		} else {
			end += perCPU
		}

		go func(start, end int) {
			for j := start; j < end; j++ {
				guess := fmt.Sprintf("%04d", j)
				k := pbkdf2.Key([]byte(guess), salt, 1000, len(key), sha1.New)
				if bytes.Equal(k, key) {
					found <- guess
					return
				}
			}
			wg.Done()
		}(start, end)

		start += perCPU
	}

	select {
	case <-wg.WaitChan():
		return "", errors.New("failed to calculate PIN number")
	case pin := <-found:
		return pin, nil
	}
}

func exit(status int, addUsage bool, errfmt string, a ...interface{}) {
	if errfmt != "" {
		fmt.Fprintf(os.Stderr, errfmt+"\n", a...)
	}
	if addUsage {
		usage()
	}
	if !*noPause {
		fmt.Printf("Press Enter to exit")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
	}
	os.Exit(status)
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:", path.Base(os.Args[0]), " [flags] [<path to latest iTunes backup directory>]")
	flag.PrintDefaults()
}

func init() {
	flag.Usage = usage
}

func displayLicense() {
	fmt.Println("LICENSE INFORMATION")
	fmt.Println("-------------------")
	fmt.Println()
	for _, fn := range licenses.Filenames() {
		fmt.Println(fn)
		fmt.Println()
		f, _ := licenses.Open(fn)
		io.Copy(os.Stdout, f)
		fmt.Println()
		fmt.Println()
	}
	fmt.Println()
}

func main() {
	var syncDir string
	var err error
	var allBackups backups

	fmt.Println("PIN Finder", version)
	fmt.Println("http://github.com/gwatts/pinfinder")

	flag.Parse()

	if *showLicense {
		displayLicense()
		return
	}

	args := flag.Args()
	switch len(args) {
	case 0:
		syncDir, err = findSyncDir()
		if err != nil {
			exit(101, true, err.Error())
		}
		allBackups, err = loadBackups(syncDir)
		if err != nil {
			exit(101, true, err.Error())
		}
		fmt.Println("Sync Directory:", syncDir)

	case 1:
		b := loadBackup(args[0])
		if b == nil {
			exit(101, true, "Invalid backup directory")
		}
		allBackups = backups{b}

	default:
		exit(102, true, "Too many arguments")
	}

	fmt.Println()
	fmt.Printf("%-40.40s  %-25s  %s\n", "IOS DEVICE", "BACKUP TIME", "RESTRICTIONS PASSCODE")
	failed := make(backups, 0)

	foundEncrypted := false
	for _, b := range allBackups {
		info := b.info
		fmt.Printf("%-40.40s  %s  ",
			info.DisplayName,
			info.LastBackup.In(time.Local).Format("Jan _2, 2006 03:04 PM MST"))

		if len(b.restrictions.Key) > 0 {
			pin, err := findPIN(b.restrictions.Key, b.restrictions.Salt)
			if err != nil {
				fmt.Println("Failed to find passcode")
				failed = append(failed, b)
			} else {
				fmt.Println(pin)
			}
		} else {
			fmt.Println(b.status)
			if b.status == msgIsEncrypted {
				foundEncrypted = true
			}
		}
	}

	if foundEncrypted {
		fmt.Println("")
		fmt.Println("NOTE: Some backups had passcodes that are encrypted.")
		fmt.Println("Select the device in iTunes, uncheck the \"Encrypt iPhone backup\" checkbox, ")
		fmt.Println("re-sync and then run pinfinder again.")
	}

	fmt.Println()
	for _, b := range failed {
		fmt.Printf("Failed to find PIN for backup %s\nPlease file a bug report at https://github.com/gwatts/pinfinder/issues\n", b.path)
		fmt.Printf("%-20s: %s\n", "Product Name", b.info.ProductName)
		fmt.Printf("%-20s: %s\n", "Product Type", b.info.ProductType)
		fmt.Printf("%-20s: %s\n", "Product Version", b.info.ProductVersion)

		dumpFile(b.restrictionsPath)
		fmt.Println()
	}
	exit(0, false, "")
}
