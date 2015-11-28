// Copyright (c) 2015, Gareth Watts
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

package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/base64"
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
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

const (
	maxPIN                = 10000
	version               = "1.3.0"
	restrictionsPlistName = "398bc9c2aeeab4cb0c12ada0f52eea12cf14f40b"
)

var (
	noPause = flag.Bool("nopause", false, "Set to true to prevent the program pausing for input on completion")
)

type plist struct {
	Dict plistDict `xml:"dict"`
	path string
}

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
		// vista & newer
		dir = filepath.Join(usr.HomeDir, "AppData", "Roaming", "Apple Computer", "MobileSync", "Backup")
		if !isDir(dir) {
			// XP; untested.  This should really query the registry to find the right documents directory.
			dir = filepath.Join("C:\\Documents and Settings", usr.Username, "Application Data", "Apple Computer", "MobileSync", "Backup")
		}
	default:
		return "", errors.New("could not detect backup directory for this operating system; pass explicitly")
	}
	if !isDir(dir) {
		return "", fmt.Errorf("directory %s does not exist", dir)
	}
	return dir, nil
}

type backup struct {
	path         string
	info         plist
	restrictions plist
}

type backups []*backup

func (b backups) Len() int { return len(b) }
func (b backups) Less(i, j int) bool {
	di, dj := b[i].info.Dict["Last Backup Date"].Value, b[j].info.Dict["Last Backup Date"].Value
	return di < dj
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
		if b, err := loadBackup(path); err == nil {
			backups = append(backups, b)
		}
	}
	sort.Sort(sort.Reverse(backups))
	return backups, nil
}

func loadBackup(backupDir string) (*backup, error) {
	var b backup
	if err := loadXML(filepath.Join(backupDir, "Info.plist"), &b.info); err != nil {
		return nil, fmt.Errorf("%s is not a backup directory", backupDir)
	}
	b.info.path = filepath.Join(backupDir, "Info.plist")
	if err := loadXML(filepath.Join(backupDir, restrictionsPlistName), &b.restrictions); err == nil {
		b.restrictions.path = filepath.Join(backupDir, restrictionsPlistName)
	}
	b.path = backupDir
	return &b, nil
}

func (b *backup) parseRestrictions() (pw, salt []byte) {
	pw, _ = base64.StdEncoding.DecodeString(strings.TrimSpace(b.restrictions.Dict["RestrictionsPasswordKey"].Value))
	salt, _ = base64.StdEncoding.DecodeString(strings.TrimSpace(b.restrictions.Dict["RestrictionsPasswordSalt"].Value))
	return pw, salt
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

func main() {
	var syncDir string
	var err error
	var allBackups backups

	fmt.Println("PIN Finder", version)
	fmt.Println("http://github.com/gwatts/pinfinder")

	flag.Parse()

	args := flag.Args()
	switch len(args) {
	case 0:
		syncDir, err = findSyncDir()
		if err != nil {
			fmt.Println(err.Error)
			usage()
		}
		allBackups, err = loadBackups(syncDir)
		if err != nil {
			exit(101, true, err.Error())
		}

	case 1:
		b, err := loadBackup(args[0])
		if err != nil {
			exit(101, true, err.Error())
		}
		allBackups = backups{b}

	default:
		exit(102, true, "Too many arguments")
	}

	fmt.Println()
	fmt.Printf("%-40.40s  %-25s  %s\n", "IOS DEVICE", "BACKUP TIME", "RESTRICTIONS PASSCODE")
	failed := make(backups, 0)
	for _, b := range allBackups {
		info := b.info.Dict
		var backupTime string
		if t, err := time.Parse(time.RFC3339, info["Last Backup Date"].Value); err != nil {
			backupTime = info["Last Backup Date"].Value
		} else {
			backupTime = t.In(time.Local).Format("Jan _2, 2006 03:04 PM MST")
		}
		fmt.Printf("%-40.40s  %s  ", info["Display Name"].Value, backupTime)
		if b.restrictions.Dict != nil {
			key, salt := b.parseRestrictions()
			pin, err := findPIN(key, salt)
			if err != nil {
				fmt.Println("Failed to find passcode")
				failed = append(failed, b)
			} else {
				fmt.Println(pin)
			}
		} else {
			fmt.Println("No passcode found")
		}
	}

	fmt.Println()
	for _, b := range failed {
		fmt.Printf("Failed to find PIN for backup %s\nPlease file a bug report at https://github.com/gwatts/pinfinder/issues\n", b.path)
		for _, key := range []string{"Product Name", "Product Type", "Product Version"} {
			fmt.Printf("%-20s: %s\n", key, b.info.Dict[key].Value)
		}
		dumpFile(b.restrictions.path)
		fmt.Println()
	}
	exit(0, false, "")
}
