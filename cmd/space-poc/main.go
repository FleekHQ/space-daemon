package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/FleekHQ/space-poc/app"
	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/env"
	spacelog "github.com/FleekHQ/space-poc/log"

	_ "net/http/pprof"
)

var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile = flag.String("memprofile", "", "write memory profile to `file`")
	debugMode  = flag.Bool("debug", true, "run daemon with debug mode for profiling")
	devMode    = flag.Bool("dev", false, "run daemon in dev mode to use .env file")
	ipfsaddr   string
	mongousr   string
	mongopw    string
	mongohost  string
)

func main() {
	// flags
	flag.Parse()

	log.Printf("INFO: dev mode %v", *devMode)

	cf := &config.Flags{
		Ipfsaddr:  ipfsaddr,
		Mongousr:  mongousr,
		Mongopw:   mongopw,
		Mongohost: mongohost,
		DevMode: *devMode == true,
	}

	// CPU profiling
	if *debugMode == true {
		fmt.Println("DEBUG: running daemon with profiler on localhost:6060..")
		go func() {
			fmt.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	// env
	env := env.New()

	// load configs
	cfg := config.NewMap(env, cf)

	// setup logger
	spacelog.New(env)
	// setup context
	ctx := context.Background()

	app.Start(ctx, cfg, env)

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
}
