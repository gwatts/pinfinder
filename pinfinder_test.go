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

func mkInfo(tm, devname string) []byte {
	return []byte(fmt.Sprintf(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"> 
<dict>
	<key>Last Backup Date</key>
	<date>%s</date>
	<key>Device Name</key>
	<string>%s</string>
</dict>
</plist>
`, tm, devname))
}

func setupDataDir() string {
	tmp, err := ioutil.TempDir("", "pinfinder")
	if err != nil {
		panic("Could not create test directory: " + err.Error())
	}
	b1path := filepath.Join(tmp, "backup1")
	b2path := filepath.Join(tmp, "backup2")
	b3path := filepath.Join(tmp, "nobackup")
	os.Mkdir(b1path, 0777)
	os.Mkdir(b2path, 0777)

	ioutil.WriteFile(
		filepath.Join(b1path, "398bc9c2aeeab4cb0c12ada0f52eea12cf14f40b"),
		[]byte(pinData),
		0644)
	ioutil.WriteFile(
		filepath.Join(b1path, "398bc9c2aeeab4cb0c12ada0f52eea12cf14f40c"),
		[]byte("not a plist"),
		0644)
	ioutil.WriteFile(
		filepath.Join(b1path, "Info.plist"),
		mkInfo("2014-11-25T21:39:29Z", "device one"),
		0644)

	// no passcode for b2
	ioutil.WriteFile(
		filepath.Join(b2path, "398bc9c2aeeab4cb0c12ada0f52eea12cf14f40c"),
		[]byte("not a plist"),
		0644)
	ioutil.WriteFile(
		filepath.Join(b2path, "Info.plist"),
		mkInfo("2015-11-25T21:39:29Z", "device two"),
		0644)

	// b3 doesn't contain a backup at all
	ioutil.WriteFile(
		filepath.Join(b3path, "random file"),
		[]byte("not a plist"),
		0644)

	return tmp
}

func TestLoadBackup(t *testing.T) {
	tmpDir := setupDataDir()
	defer os.RemoveAll(tmpDir)

	path := filepath.Join(tmpDir, "backup1")
	backup, err := loadBackup(path)
	if err != nil {
		t.Fatal("loadBackup failed", err)
	}
	if backup.path != path {
		t.Errorf("Path incorrect expected=%q actual=%q", path, backup.path)
	}

	if backup.info.Dict["Device Name"].Value != "device one" {
		t.Errorf("Incorrect device name: %v", backup.info.Dict)
	}
}

func TestLoadBackups(t *testing.T) {
	tmpDir := setupDataDir()
	defer os.RemoveAll(tmpDir)

	b, err := loadBackups(tmpDir)
	if err != nil {
		t.Fatal("loadBackups failed", err)
	}
	if len(b) != 2 {
		t.Fatal("Incorrect backup count", len(b))
	}
	// Should of been sorted into reverse time order
	if devname := b[0].info.Dict["Device Name"].Value; devname != "device two" {
		t.Errorf("First entry is not device two, got %q", devname)
	}
	if devname := b[1].info.Dict["Device Name"].Value; devname != "device one" {
		t.Errorf("Second entry is not device one, got %q", devname)
	}
}

func TestParseRestriction(t *testing.T) {
	tmpDir := setupDataDir()
	defer os.RemoveAll(tmpDir)

	path := filepath.Join(tmpDir, "backup1")
	b, err := loadBackup(path)
	if err != nil {
		t.Fatal("Failed to load backup", err)
	}

	key, salt := b.parseRestrictions()

	/*
		pl := &plist{
			Keys: []string{"RestrictionsPasswordKey", "RestrictionsPasswordSalt"},
			Data: []string{"ioN63+yl6OFZ4/C7xl9VejMLDi0=", "iNciDA=="},
		}
		key, salt := parseRestrictions(pl)
	*/
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
