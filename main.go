package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type (
	Day struct {
		Y, M, D int
	}

	Period struct {
		Start, Stop time.Time
	}
)

var detected = map[Day]Period{}

// it works just due to some bug in ubuntu and perhaps because of logrotate freq.
func getLogouts() {
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
			Y, M, D := t.Date()
			key := Day{Y, int(M), D}
			val := detected[key]
			val.Stop = t
			detected[key] = val
		}
	}
}

func getLogins() {
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
				t, err := time.ParseInLocation(time.Stamp,
					strings.Join(strings.Fields(scanner.Text())[:3], " "),
					time.Local,
				)

				if err != nil {
					panic(err)
				}

				t = t.AddDate(f.ModTime().Year(), 0, 0) // timestamp has no year
				Y, M, D := t.Date()
				key := Day{Y, int(M), D}
				val := detected[key]
				if !val.Start.IsZero() && val.Start.Before(t) {
					continue
				}
				val.Start = t
				detected[key] = val

			}

		}
	}
}

type ByStart []Period

func (a ByStart) Len() int           { return len(a) }
func (a ByStart) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByStart) Less(i, j int) bool { return a[i].Start.Before(a[j].Start) }

const (
	workDay8  = time.Hour * 8
	workDay64 = (time.Hour * 64) / 10
)

func main() {
	getLogouts()
	getLogins()

	withBoth := ByStart{}
	for _, period := range detected {
		if !period.Start.IsZero() && !period.Stop.IsZero() {
			withBoth = append(withBoth, period)
		}
	}

	sort.Sort(withBoth)
	var balance8, balance64 time.Duration
	for _, period := range withBoth {
		duration := period.Stop.Sub(period.Start)
		balance8 += duration - workDay8
		balance64 += duration - workDay64
		fmt.Printf("%v - %v%15v%15v%15v\n",
			period.Start.Format(time.ANSIC),
			period.Stop.Format(time.ANSIC),
			duration,
			balance8,
			balance64,
		)
	}

}
