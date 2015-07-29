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
	"path"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

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

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage:", path.Base(os.Args[0]), " <path to latest itunes backup directory>")
		os.Exit(101)
	}

	path := os.Args[1]

	pl, err := findRestrictions(path)
	if err != nil {
		fmt.Println("Failed to find/load restrictions plist file: ", err)
		os.Exit(102)
	}

	pw, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(pl.Data[0]))
	salt, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(pl.Data[1]))

	for i := 0; i < 10000; i++ {
		guess := fmt.Sprintf("%04d", i)
		fmt.Println("Trying ", guess)
		k := pbkdf2.Key([]byte(guess), salt, 1000, len(pw), sha1.New)
		if bytes.Equal(k, pw) {
			fmt.Println("Matched PIN", guess)
			os.Exit(0)
		}
	}
	fmt.Println("No match found")
	os.Exit(103)
}
