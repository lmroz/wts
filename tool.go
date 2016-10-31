package main

import (
	"bufio"
	"fmt"
	"os"
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

func begin(t time.Time) {
	t = t.Truncate(time.Second)
	Y, M, D := t.Date()
	key := Day{Y, int(M), D}
	val := detected[key]
	if !val.Start.IsZero() && val.Start.Before(t) {
		return
	}
	val.Start = t
	detected[key] = val

}
func end(t time.Time) {
	t = t.Truncate(time.Second)
	Y, M, D := t.Date()
	key := Day{Y, int(M), D}
	val := detected[key]
	if !val.Stop.IsZero() && val.Stop.After(t) {
		return
	}
	val.Stop = t
	detected[key] = val
}

type ByStart []Period

func (a ByStart) Len() int           { return len(a) }
func (a ByStart) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByStart) Less(i, j int) bool { return a[i].Start.Before(a[j].Start) }

const (
	workDay8  = time.Hour * 8
	workDay64 = (time.Hour * 64) / 10
)

func parseEventFile() {
	stream, err := os.Open(logfileEE)
	if err != nil {
		panic(err)
	}
	defer stream.Close()
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		flds := strings.SplitN(scanner.Text(), " ", 2)
		useFun := end
		switch flds[0] {
		case "LOCKED":
			useFun = begin
			fallthrough
		case "UNLOCKED":
			t, err := time.Parse(timeFormat, flds[1])
			if err != nil {
				panic(err)
			}
			useFun(t)
		default:
			panic("unexpeced input: " + scanner.Text())
		}
	}

}

func toolMode(fallback bool) {
	end(time.Now())

	if fallback {
		getLoginsFallback()
		getLogoutsFallback()
	} else {
		parseEventFile()
	}

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
		fmt.Printf("%v - %v%15v%15v%15v%15v\n",
			period.Start.Format(timeFormat),
			period.Stop.Format(timeFormat),
			duration,
			time.Now().Truncate(time.Minute).Sub(period.Start),
			balance8,
			balance64,
		)
	}
}
