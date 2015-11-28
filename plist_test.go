package main

import (
	"encoding/xml"
	"reflect"
	"testing"
)

const plistTest = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">  
<plist version="1.0">
<dict>
  <key>Key One</key>
  <string>String One</string>
  <key>Key Two</key>
  <data>Data Two</data>
</dict>
</plist>
`

func TestPlist(t *testing.T) {
	var a struct {
		D plistDict `xml:"dict"`
	}

	if err := xml.Unmarshal([]byte(plistTest), &a); err != nil {
		t.Fatal("Unmarshal failed", err)
	}

	expected := plistDict{
		"Key One": plistval{Type: "string", Value: "String One"},
		"Key Two": plistval{Type: "data", Value: "Data Two"},
	}
	if !reflect.DeepEqual(a.D, expected) {
		t.Fatal("Unexpected result ", a.D)
	}
}
