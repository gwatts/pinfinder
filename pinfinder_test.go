package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

const pinData = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>RestrictionsPasswordKey</key>
	<data>
	ioN63+yl6OFZ4/C7xl9VejMLDi0=
	</data>
	<key>RestrictionsPasswordSalt</key>
	<data>
	iNciDA==
	</data>
</dict>
</plist>
`

var (
	// extracted values from above plist (pin is 1234)
	dataKey  = []byte{0x8a, 0x83, 0x7a, 0xdf, 0xec, 0xa5, 0xe8, 0xe1, 0x59, 0xe3, 0xf0, 0xbb, 0xc6, 0x5f, 0x55, 0x7a, 0x33, 0xb, 0xe, 0x2d}
	dataSalt = []byte{0x88, 0xd7, 0x22, 0xc}
	dataPIN  = "1234"
)

func setupDataDir() string {
	tmp, err := ioutil.TempDir("", "pinfinder")
	if err != nil {
		panic("Could not create test directory: " + err.Error())
	}
	ioutil.WriteFile(
		filepath.Join(tmp, "398bc9c2aeeab4cb0c12ada0f52eea12cf14f40b"),
		[]byte(pinData),
		0644)
	ioutil.WriteFile(
		filepath.Join(tmp, "398bc9c2aeeab4cb0c12ada0f52eea12cf14f40c"),
		[]byte("not a plist"),
		0644)
	return tmp
}

func TestFindRestrictions(t *testing.T) {
	tmpDir := setupDataDir()
	defer os.RemoveAll(tmpDir)

	pl, err := findRestrictions(tmpDir)
	if err != nil {
		t.Fatal("Unexpected error", err)
	}
	if pl.Keys[0] != "RestrictionsPasswordKey" {
		t.Error("Incorrect plist")
	}
}

func TestParseRestriction(t *testing.T) {
	pl := &plist{
		Keys: []string{"RestrictionsPasswordKey", "RestrictionsPasswordSalt"},
		Data: []string{"ioN63+yl6OFZ4/C7xl9VejMLDi0=", "iNciDA=="},
	}
	key, salt := parseRestrictions(pl)
	if !bytes.Equal(key, dataKey) {
		t.Error("key doesn't match")
	}
	if !bytes.Equal(salt, dataSalt) {
		t.Error("salt doesn't match")
	}
	fmt.Printf("key: %#v\nsalt: %#v\n", key, salt)
}

func TestFindPINOK(t *testing.T) {
	pin, err := findPIN(dataKey, dataSalt)
	if err != nil {
		t.Error("Unexpected error", err)
	}
	if pin != dataPIN {
		t.Error("Did not get expected PIN.  Received", pin)
	}
}

func TestFindPINFail(t *testing.T) {
	_, err := findPIN(dataKey, []byte{0x88, 0xd7, 0x22, 0xc0}) // change last byte of salt
	if err == nil {
		t.Error("Did not receive expected error")
	}
}
