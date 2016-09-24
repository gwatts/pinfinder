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
	<key>Display Name</key>
	<string>%s</string>
</dict>
</plist>
`, tm, devname))
}

func mkManifest(isEncrypted bool) []byte {
	b := "<false />"
	if isEncrypted {
		b = "<true />"
	}
	return []byte(fmt.Sprintf(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"> 
<dict>
	<key>IsEncrypted</key>
	%s
</dict>
</plist>
`, b))
}

func setupDataDir() string {
	tmp, err := ioutil.TempDir("", "pinfinder")
	if err != nil {
		panic("Could not create test directory: " + err.Error())
	}
	b1path := filepath.Join(tmp, "backup1")
	b2path := filepath.Join(tmp, "backup2")
	b3path := filepath.Join(tmp, "nobackup")
	b4path := filepath.Join(tmp, "encbackup")
	b5path := filepath.Join(tmp, "encnopcbackup")
	b6path := filepath.Join(tmp, "ios10backup")
	os.Mkdir(b1path, 0777)
	os.Mkdir(b2path, 0777)
	os.Mkdir(b3path, 0777)
	os.Mkdir(b4path, 0777)
	os.Mkdir(b5path, 0777)
	os.Mkdir(b6path, 0777)
	os.Mkdir(filepath.Join(b6path, "39"), 0777)

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
	ioutil.WriteFile(
		filepath.Join(b1path, "Manifest.plist"),
		mkManifest(false),
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

	ioutil.WriteFile(
		filepath.Join(b2path, "Manifest.plist"),
		mkManifest(false),
		0644)

	// b3 doesn't contain a backup at all
	ioutil.WriteFile(
		filepath.Join(b3path, "random file"),
		[]byte("not a plist"),
		0644)

	// b4 is marked as encrypted
	ioutil.WriteFile(
		filepath.Join(b4path, "398bc9c2aeeab4cb0c12ada0f52eea12cf14f40b"),
		[]byte("this would be an encrypted plist"),
		0644)
	ioutil.WriteFile(
		filepath.Join(b4path, "Info.plist"),
		mkInfo("2014-11-24T20:39:29Z", "device three"),
		0644)
	ioutil.WriteFile(
		filepath.Join(b4path, "Manifest.plist"),
		mkManifest(true),
		0644)

	// b5 is encrypted, but has no passcode file
	ioutil.WriteFile(
		filepath.Join(b5path, "Info.plist"),
		mkInfo("2014-11-24T19:39:29Z", "device four"),
		0644)
	ioutil.WriteFile(
		filepath.Join(b5path, "Manifest.plist"),
		mkManifest(true),
		0644)

	// b6 contains a passcode with iOS 10 file layout
	ioutil.WriteFile(
		filepath.Join(b6path, "39", "398bc9c2aeeab4cb0c12ada0f52eea12cf14f40b"),
		[]byte(pinData),
		0644)
	ioutil.WriteFile(
		filepath.Join(b6path, "Info.plist"),
		mkInfo("2016-09-23T21:39:29Z", "ios10 device"),
		0644)
	ioutil.WriteFile(
		filepath.Join(b6path, "Manifest.plist"),
		mkManifest(false),
		0644)

	return tmp
}

func TestLoadBackup(t *testing.T) {
	tmpDir := setupDataDir()
	defer os.RemoveAll(tmpDir)

	path := filepath.Join(tmpDir, "backup1")
	backup := loadBackup(path)
	if backup == nil {
		t.Fatal("loadBackup failed")
	}
	if backup.path != path {
		t.Errorf("Path incorrect expected=%q actual=%q", path, backup.path)
	}

	if backup.info.DisplayName != "device one" {
		t.Errorf("Incorrect device name: %v", backup.info)
	}
}

func TestLoadBackups(t *testing.T) {
	tmpDir := setupDataDir()
	defer os.RemoveAll(tmpDir)

	b, err := loadBackups(tmpDir)
	if err != nil {
		t.Fatal("loadBackups failed", err)
	}
	if len(b) != 5 {
		t.Fatal("Incorrect backup count", len(b))
	}

	// Should of been sorted into reverse time order
	if devname := b[0].info.DisplayName; devname != "ios10 device" {
		t.Errorf("First entry is not ios10 device got %q", devname)
	}
	if devname := b[1].info.DisplayName; devname != "device two" {
		t.Errorf("Second entry is not device two, got %q", devname)
	}
	if devname := b[2].info.DisplayName; devname != "device one" {
		t.Errorf("Third entry is not device one, got %q", devname)
	}
	if devname := b[3].info.DisplayName; devname != "device three" {
		t.Errorf("Fourth entry is not device wthree, got %q", devname)
	}
	if !b[3].isEncrypted() {
		t.Error("device three not marked as encrypted")
	}

	if status := b[3].status; status != msgIsEncrypted {
		t.Error("device three does not have correct status: ", status)
	}

	if status := b[4].status; status != msgNoPasscode {
		t.Error("device four does not have correct status", status)
	}
}

func TestParseRestriction(t *testing.T) {
	tmpDir := setupDataDir()
	defer os.RemoveAll(tmpDir)

	for _, base := range []string{"backup1", "ios10backup"} {
		path := filepath.Join(tmpDir, base)
		b := loadBackup(path)
		if b == nil {
			t.Fatal("Failed to load backup")
		}

		key := b.restrictions.Key
		salt := b.restrictions.Salt

		if !bytes.Equal(key, dataKey) {
			t.Error("key doesn't match")
		}
		if !bytes.Equal(salt, dataSalt) {
			t.Error("salt doesn't match")
		}
		fmt.Printf("key: %#v\nsalt: %#v\n", key, salt)
	}
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
