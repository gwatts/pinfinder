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

// iOS Restrictions PIN Finder
//
// This program will examine an iTunes backup folder for an iOS device and attempt
// to find the PIN used for restricting permissions on the device (NOT the unlock PIN)

package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

const maxPIN = 10000

func isDir(p string) bool {
	s, err := os.Stat(p)
	if err != nil {
		return false
	}
	return s.IsDir()
}

// figure out where iTunes keeps its backups on the current OS
func findSyncDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	var dir string
	switch runtime.GOOS {
	case "darwin":
		dir = filepath.Join(usr.HomeDir, "Library", "Application Support", "MobileSync", "Backup")
	case "windows":
		// vista & newer
		dir = filepath.Join(usr.HomeDir, "AppData", "Roaming", "Apple Computer", "MobileSync", "Backup")
		if !isDir(dir) {
			// XP; untested.
			dir = filepath.Join("Documents and Settings", usr.Username, "Application Data", "Apple Computer", "MobileSync", "Backup")
		}
	default:
		return "", errors.New("Could not detect backup directory for this operating system; pass explicitly")
	}
	if !isDir(dir) {
		return "", fmt.Errorf("Directory %s does not exist", dir)
	}
	return dir, nil
}

// Fidn the latest backup folder
func findLatestBackup(backupDir string) (string, error) {
	d, err := os.Open(backupDir)
	if err != nil {
		return "", err
	}
	files, err := d.Readdir(10000)
	if err != nil {
		return "", err
	}
	var newest string
	var lastMT time.Time

	for _, fi := range files {
		if mt := fi.ModTime(); mt.After(lastMT) {
			lastMT = mt
			newest = fi.Name()
		}
	}
	if newest != "" {
		return filepath.Join(backupDir, newest), nil
	}
	return "", errors.New("No backup directories found in " + backupDir)
}

type Plist struct {
	Keys []string `xml:"dict>key"`
	Data []string `xml:"dict>data"`
}

func loadPlist(fn string) (*Plist, error) {
	var p Plist
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err := xml.NewDecoder(f).Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func findRestrictions(fpath string) (*Plist, error) {
	d, err := os.Open(fpath)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	fl, err := d.Readdir(-1)
	if err != nil {
		return nil, err
	}
	c := 0
	for _, fi := range fl {
		if !fi.Mode().IsRegular() {
			continue
		}
		if size := fi.Size(); size < 300 || size > 500 {
			continue
		}
		if pl, err := loadPlist(path.Join(fpath, fi.Name())); err == nil {
			c++
			if len(pl.Keys) == 2 && len(pl.Data) == 2 && pl.Keys[0] == "RestrictionsPasswordKey" {
				return pl, nil
			}
		}
	}
	if c == 0 {
		return nil, errors.New("No plist files; are you sure you have the right directory?")
	}
	return nil, errors.New("No matching plist file - Are parental restrictions turned on?")
}

func parseRestrictions(pl *Plist) (pw, salt []byte) {
	pw, _ = base64.StdEncoding.DecodeString(strings.TrimSpace(pl.Data[0]))
	salt, _ = base64.StdEncoding.DecodeString(strings.TrimSpace(pl.Data[1]))
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
		return "", errors.New("failed to calculate PIN number.")
	case pin := <-found:
		return pin, nil
	}
}

func usage() {
	fmt.Println("Usage:", path.Base(os.Args[0]), " [<path to latest itunes backup directory>]")
	os.Exit(101)
}

func main() {
	var backupDir, syncDir string
	var err error

	switch len(os.Args) {
	case 1:
		syncDir, err = findSyncDir()
		if err != nil {
			fmt.Println(err.Error)
			usage()
		}
		backupDir, err = findLatestBackup(syncDir)
		if err != nil {
			fmt.Println(err.Error())
			usage()
		}
	case 2:
		backupDir = os.Args[1]
	default:
		usage()
	}

	if !isDir(backupDir) {
		fmt.Println("Directory not found: ", backupDir)
		usage()
	}

	fmt.Println("Searching backup at", backupDir)
	pl, err := findRestrictions(backupDir)
	if err != nil {
		fmt.Println("Failed to find/load restrictions plist file: ", err)
		os.Exit(102)
	}

	key, salt := parseRestrictions(pl)

	fmt.Print("Finding PIN...")
	startTime := time.Now()
	pin, err := findPIN(key, salt)
	if err != nil {
		fmt.Println(err)
		os.Exit(103)
	}
	fmt.Printf(" FOUND!\nPIN number is: %s (found in %s)\n", pin, time.Since(startTime))
}
