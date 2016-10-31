package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexflint/go-filemutex"
	"github.com/godbus/dbus"
)

const (
	lockfile = "/tmp/lmroz_wts.lock"
	logfile  = "$HOME/locklog"
)

var logfileEE = os.ExpandEnv(logfile)

const DBUS_ENV = "DBUS_SESSION_BUS_ADDRESS"

var timeFormat = time.RFC1123Z

func writeEvent(a ...interface{}) {
	f, err := os.OpenFile(logfileEE, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error opening event file:", err)
		return
	}
	_, err = fmt.Fprintln(f, a...)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error writing event file:", err)
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error closing event file:", err)
			return
		}
	}()

}

// this function requires DBUS_SESSION_BUS_ADDRESS to be set in env, sometimes
// its not set, and this is really hard to get why events are not collected
func service() {
	if _, ok := os.LookupEnv(DBUS_ENV); !ok {
		panic(DBUS_ENV + " env var not set")
	}

	conn, err := dbus.SessionBus()
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',path='/com/canonical/Unity/Session',interface='com.canonical.Unity.Session'")
	c := make(chan *dbus.Signal, 10)
	conn.Signal(c)
	for v := range c {
		nowStr := time.Now().Format(timeFormat)

		switch v.Name {
		case "com.canonical.Unity.Session.Locked":
			writeEvent("LOCKED", nowStr)
		case "com.canonical.Unity.Session.Unlocked":
			writeEvent("UNLOCKED", nowStr)
		}
	}
}

func lock() func() {
	m, err := filemutex.New(lockfile)
	if err != nil {
		panic(err)
	}
	fmt.Fprintln(os.Stderr, "Waiting for ", lockfile)
	m.Lock()
	fmt.Fprintln(os.Stderr, "Lock acquired:", lockfile)

	unlock := func() {
		m.Unlock()
		fmt.Fprintln(os.Stderr, "Lock released:", lockfile)
	}

	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGPIPE, syscall.SIGKILL)

	go func() {
		<-sigs
		unlock()
		os.Exit(1)
	}()

	return unlock

}

var flagTool = flag.Bool("tool", false, "True runs tool mode (shows log).")
var flagFallback = flag.Bool("fallback", false, "True runs tool mode using fallback technique.")

func main() {
	flag.Parse()

	if *flagTool || *flagFallback {
		toolMode(*flagFallback)
		return
	}

	unlock := lock()
	defer unlock()

	service()
}
