package main

import (
	"bufio"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

// it works just due to some bug in ubuntu and perhaps because of logrotate freq.
func getLogoutsFallback() {
	cu, err := user.Current()
	if err != nil {
		panic(err)
	}
	upstartCache := filepath.Join(
		cu.HomeDir,
		".cache/upstart",
	)
	filez, err := ioutil.ReadDir(upstartCache)
	if err != nil {
		panic(err)
	}
	for _, f := range filez {
		if n := f.Name(); strings.HasPrefix(n, "unity-panel-service-lockscreen.log.") &&
			strings.HasSuffix(n, ".gz") {
			t := f.ModTime()
			end(t)
		}
	}
}

func getLoginsFallback() {
	logz := "/var/log"
	filez, err := ioutil.ReadDir(logz)
	if err != nil {
		panic(err)
	}
	for _, f := range filez {
		if n := f.Name(); strings.HasPrefix(n, "auth.log") &&
			!strings.HasSuffix(n, ".gz") { // now i dont need such old stuff
			stream, err := os.Open(filepath.Join(logz, n))
			if err != nil {
				panic(err)
			}
			defer stream.Close()
			scanner := bufio.NewScanner(stream)
			for scanner.Scan() {
				if !strings.Contains(scanner.Text(), "gkr-pam: unlocked login keyring") {
					continue
				}
				sTime := strings.Join(strings.Fields(scanner.Text())[:3], " ")

				var t time.Time
				loc := time.Local
				parse := func() {
					t, err = time.ParseInLocation(time.Stamp,
						sTime,
						loc,
					)
					if err != nil {
						panic(err)
					}
					t = t.AddDate(f.ModTime().Year(), 0, 0) // timestamp has no year
				}
				parse()

				// if these are different, timestamp needs to be fixed (i love timestamps!)
				stampZone, stampOffset := t.Zone()
				_, localOffset := time.Now().Zone()
				if stampOffset != localOffset {
					// and easiest is reparse
					loc = time.FixedZone(stampZone, stampOffset)
					t = time.Time{}
					parse()
				}

				begin(t)

			}

		}
	}
}

func toolFallback() {
	getLogoutsFallback()
	getLoginsFallback()
}
