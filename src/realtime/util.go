package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"github.com/kless/goconfig/config"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var (
	// License checking is sha1("domain.com"+PADDING)
	PADDING = []byte("Rk8ohYJQBXopu82XmVTFsAgG3r4f")
)

const (
	LOCALHOST = `localhost`
)

// Looks for a realtime.conf file in either the current
// directory, an etc/ subdir, or an etc/ directory one up
// from the current directory
// Returns a new Config object
func getConf() (*config.Config, error) {
	p1 := filepath.Join(ROOT, CONF_NAME)

	parent, _ := filepath.Split(ROOT)
	p2 := filepath.Join(parent, "etc", CONF_NAME)

	p3 := filepath.Join(ROOT, "etc", CONF_NAME)

	for _, p := range []string{p1, p2, p3} {
		if fileExists(p) {
			if c, err := config.ReadDefault(p); err != nil {
				return nil, errors.New(fmt.Sprintf("Error reading config: %v", p))
			} else {
				return c, nil
			}
		}
	}

	return nil, errors.New("No config file found")

}

type License []string

// Looks for a license.txt file in either the current
// directory, an etc subdir, or an etc directory one up
// from the current directory
// Returns a new License object, populated with the parsed
// license keys
func NewLicense() (license License, err error) {
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

			reader = bufio.NewReaderSize(fh, 50)

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
		err = errors.New("Unable to find/read any valid licenses")
	}
	return license, err

}

// Matches a given license string against the licenses
// in the current configuration, and return true if its valid.
func (l License) IsValid(lic string) bool {
	buf := new(bytes.Buffer)
	for _, val := range l {
		buf.WriteString("License test failures: " + val + " != " + lic + "\n")
		if val == lic {
			return true
		}
	}
	Debugln(buf.String())
	return false
}

func (l License) CheckHttpRequest(req *http.Request) bool {
	var (
		url_         *url.URL
		err          error
		origin, host string
	)

	origin = req.Header.Get("Origin")
	if origin == "" {
		origin = req.Header.Get("Sec-Websocket-Origin")
	}

	if origin != "" {
		url_, err = url.Parse(origin)

		if err == nil && url_.Host != "" {
			origin = strings.SplitN(url_.Host, ":", 2)[0]

			if strings.Count(origin, ".") > 1 {
				if ok := net.ParseIP(origin); ok == nil {
					parts := strings.Split(origin, ".")
					origin = strings.Join(parts[len(parts)-2:], ".")
				}
			}
		}
	}

	host = strings.SplitN(req.Host, ":", 2)[0]

	// localhost connections to a local server are always allowed
	if host == LOCALHOST && (origin == "" || origin == LOCALHOST) {
		return true
	} else if origin == "" {
		origin = host
	}

	if len(l) == 0 {
		return false
	}

	hash := sha1.New()
	hash.Write([]byte(origin))
	hash.Write(PADDING)
	passed := l.IsValid(fmt.Sprintf("%x", hash.Sum(nil)))
	if !passed {
		Debugf("host: %v, origin: %v\n", host, origin)
	}
	return passed

}
