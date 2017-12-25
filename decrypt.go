// +build !nodecrypt

package main

import (
	"bytes"

	plist "github.com/DHowett/go-plist"
	iosbackup "github.com/gwatts/ios/backup"
)

var (
	decryptEnabled = true
)

func decrypt(backupDir string, b *backup) {
	pw := getpw()
	if pw == "" {
		b.Status = msgIsEncrypted
		return
	}
	encbw, err := iosbackup.Open(backupDir)
	if err != nil {
		b.Status = "Failed to open backup: " + err.Error()
		return
	}
	if err := encbw.SetPassword(pw); err != nil {
		b.Status = msgIncorrectPassword
		return
	}
	if err := encbw.Load(); err != nil {
		b.Status = err.Error()
		return
	}
	rec := encbw.RecordById(restrictionsPlistName)
	if rec == nil {
		b.Status = msgNoPassword
		return
	}
	data, err := encbw.ReadFile(*rec)
	if err != nil {
		b.Status = msgIncorrectPassword
		return
	}
	buf := bytes.NewReader(data)
	if err := plist.NewDecoder(buf).Decode(&b.Restrictions); err != nil {
		b.Status = msgIncorrectPassword
		return
	}

}
