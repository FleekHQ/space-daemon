package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/FleekHQ/space-poc/log"

	"github.com/FleekHQ/space-poc/app"
	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/env"
)

var (
	cpuprofile   = flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile   = flag.String("memprofile", "", "write memory profile to `file`")
	debugMode    = flag.Bool("debug", true, "run daemon with debug mode for profiling")
	devMode      = flag.Bool("dev", false, "run daemon in dev mode to use .env file")
	ipfsaddr     string
	mongousr     string
	mongopw      string
	mongohost    string
	spaceapi     string
	spacehubauth string
)

func main() {
	// this defer code here ensures all profile defer call work properly
	returnCode := 0
	defer func() { os.Exit(returnCode) }()

	// flags
	flag.Parse()

	log.Printf("INFO: dev mode %v", *devMode)

	cf := &config.Flags{
		Ipfsaddr:           ipfsaddr,
		Mongousr:           mongousr,
		Mongopw:            mongopw,
		Mongohost:          mongohost,
		ServicesAPIURL:     spaceapi,
		ServicesHubAuthURL: spacehubauth,
		DevMode:            *devMode == true,
	}

	// CPU profiling
	if *debugMode == true {
		fmt.Println("DEBUG: running daemon with profiler on localhost:6060..")
		go func() {
			fmt.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	if *cpuprofile != "" {
		cleanupCpuProfile := runCpuProfiler(*cpuprofile)
		defer cleanupCpuProfile()
	}

	// env
	env := env.New()

	// load configs
	cfg := config.NewMap(env, cf)

	// setup context
	ctx := context.Background()

	spaceApp := app.New(cfg, env)
	// this blocks and returns on exist
	err := spaceApp.Start(ctx)

	if *memprofile != "" {
		cleanupMemProfile := runMemProfiler(*memprofile)
		defer cleanupMemProfile()
	}

	if err != nil {
		returnCode = 1
	}
}

func runCpuProfiler(outputFilePath string) func() {
	f, err := os.Create(outputFilePath)
	if err != nil {
		log.Error("Could not create CPU profile", err)
		return func() {}
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		log.Error("Could not start CPU profile", err)
	}

	// return cleanup function
	return func() {
		if f != nil {
			_ = f.Close() // error is ignored
		}
		pprof.StopCPUProfile()
	}
}

func runMemProfiler(outputFilePath string) func() {
	f, err := os.Create(outputFilePath)
	if err != nil {
		log.Error("could not create memory profile", err)
		return func() {}
	}

	runtime.GC() // get up-to-date statistics
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Error("could not write memory profile", err)
	}

	// return cleanup function
	return func() {
		if f != nil {
			_ = f.Close()
		}
	}
}
