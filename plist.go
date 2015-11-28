package main

import (
	"encoding/xml"
	"io"
	"os"
)

type plistval struct {
	Type  string
	Value string
}

// simple helper to load plist dicts
type plistDict map[string]plistval

func (p *plistDict) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	*p = make(plistDict)

	var sval struct {
		XMLName xml.Name
		Value   string `xml:",chardata"`
	}

	var key string
	for {
		t, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch t1 := t.(type) {
		case xml.StartElement:
			if err := d.DecodeElement(&sval, &t1); err != nil {
				return err
			}
			if sval.XMLName.Local == "key" {
				key = sval.Value
			} else {
				(*p)[key] = plistval{Type: sval.XMLName.Local, Value: sval.Value}
			}
		}
	}
}

func loadXML(path string, v interface{}) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return xml.NewDecoder(f).Decode(v)
}
