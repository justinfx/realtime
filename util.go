package main

import (
	"os"
	"fmt"
	"path/filepath"
	"bufio"
	"bytes"
	"http"
	"strings"
	"crypto/sha1"

	"github.com/kless/goconfig/config"
)

var (
	PADDING = []byte("Rk8ohYJQBXopu82XmVTFsAgG3r4f")
)

const (
	LOCALHOST = `localhost`
)

func init() {
	root, _ := filepath.Split(os.Args[0])
	ROOT, _ = filepath.Abs(root)
}

// Looks for a realtime.conf file in either the current
// directory, an etc/ subdir, or an etc/ directory one up
// from the current directory
// Returns a new Config object
func getConf() (*config.Config, os.Error) {
	p1 := filepath.Join(ROOT, CONF_NAME)
	
	parent, _ := filepath.Split(ROOT)
	p2 := filepath.Join(parent, "etc", CONF_NAME)
	
	p3 := filepath.Join(ROOT, "etc", CONF_NAME)
	
	for _, p := range []string{p1, p2, p3} {
		if fileExists(p) {
			if c, err := config.ReadDefault(p); err != nil {
				return nil, os.NewError(fmt.Sprintf("Error reading config: %v", p))
			} else {
				return c, nil
			}
		}
	}

	return nil, os.NewError("No config file found")

}

type License []string

// Looks for a license.txt file in either the current
// directory, an etc subdir, or an etc directory one up
// from the current directory
// Returns a new License object, populated with the parsed
// license keys
func NewLicense() (license License, err os.Error) {
	lic := "license.txt"
	
	p1 := filepath.Join(ROOT, lic)
	
	parent, _ := filepath.Split(ROOT)
	p2 := filepath.Join(parent, "etc", lic)
	
	p3 := filepath.Join(ROOT, "etc", lic)

	var (
		reader *bufio.Reader
		line   []byte
		prefix bool
		fh     *os.File
		buffer bytes.Buffer
	)

	for _, p := range []string{p1, p2, p3} {
		if fileExists(p) {
			fh, err = os.Open(p)
			if err != nil {
				continue
			}

			reader, err = bufio.NewReaderSize(fh, 50)
			if err != nil {
				continue
			}

			buffer.Reset()

			for {
				line, prefix, err = reader.ReadLine()
				if err != nil {
					err = nil
					break
				}
				if len(line) == 0 || bytes.IndexAny(line, "#/;") > -1 { 
					continue 
				}
				
				buffer.WriteString(string(line))
				if prefix {
					continue
				}

				license = append(license, string(buffer.Bytes()))
				buffer.Reset()
			}
		}
	}
	if len(license) == 0 {
		err = os.NewError("Unable to find/read any valid licenses")
	}
	return license, err

}

// Matches a given license string against the licenses
// in the current configuration, and return true if its valid.
func (l License) IsValid(lic string) bool {
	for _, val := range l {
		if val == lic {
			return true
		}
	}
	return false
}

func (l License) CheckHttpRequest(req *http.Request) bool {
	var (
		url *http.URL
		err os.Error
		origin, host string
	)
	
	origin = req.Header.Get("Origin")
	
	if origin != "" {
		url, err = http.ParseURL(origin)
		if err == nil && url.Host != "" {
			origin = strings.SplitN(url.Host, ":", 2)[0]
		}
	}

	host = strings.SplitN(req.Host, ":", 2)[0]

	// localhost connections to a local server are always allowed
	if host == LOCALHOST && (origin == "" || origin == LOCALHOST) {
		return true
	} else if origin == "" {
		origin = host
	}
	
	hash := sha1.New()
	hash.Write([]byte(origin))
	hash.Write(PADDING)
	return l.IsValid(fmt.Sprintf("%x", hash.Sum()))
	
}
