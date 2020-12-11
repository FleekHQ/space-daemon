package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"

	"github.com/FleekHQ/space-daemon/tracing"

	"github.com/opentracing/opentracing-go"

	"github.com/FleekHQ/space-daemon/log"

	"github.com/FleekHQ/space-daemon/app"
	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/env"
	"github.com/FleekHQ/space-daemon/core/util/rlimit"
)

var (
	cpuprofile           = flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile           = flag.String("memprofile", "", "write memory profile to `file`")
	debugMode            = flag.Bool("debug", true, "run daemon with debug mode for profiling")
	enableTracing        = flag.Bool("trace", false, "run tracing on daemon rpc")
	devMode              = flag.Bool("dev", false, "run daemon in dev mode to use .env file")
	ipfsnode             = flag.Bool("ipfsnode", true, "run IPFS embedded into the daemon (defaults to true)")
	ipfsaddr             string
	ipfsnodeaddr         string
	ipfsnodepath         string
	spaceapi             string
	spacestoragesiteurl  string
	vaultapi             string
	vaultsaltsecret      string
	spacehubauth         string
	textilehub           string
	textilehubma         string
	textilethreads       string
	textilehubgatewayurl string
	textileuserkey       string
	textileusersecret    string
)

func main() {
	// this defer code here ensures all profile defer call work properly
	returnCode := 0
	defer func() { os.Exit(returnCode) }()

	// flags
	flag.Parse()

	log.Debug("Running mode", fmt.Sprintf("DevMode:%v", *devMode))

	cf := &config.Flags{
		Ipfsaddr:             ipfsaddr,
		Ipfsnode:             *ipfsnode == true,
		Ipfsnodeaddr:         ipfsnodeaddr,
		Ipfsnodepath:         ipfsnodepath,
		ServicesAPIURL:       spaceapi,
		SpaceStorageSiteUrl:  spacestoragesiteurl,
		VaultAPIURL:          vaultapi,
		VaultSaltSecret:      vaultsaltsecret,
		ServicesHubAuthURL:   spacehubauth,
		DevMode:              *devMode == true,
		TextileHubTarget:     textilehub,
		TextileHubMa:         textilehubma,
		TextileThreadsTarget: textilethreads,
		TextileHubGatewayUrl: textilehubgatewayurl,
		TextileUserKey:       textileuserkey,
		TextileUserSecret:    textileusersecret,
	}

	// CPU profiling
	if *debugMode == true {
		log.Debug("Running daemon with profiler. Visit http://localhost:6060/debug/pprof")
		go func() {
			fmt.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	// initialize tracing
	if *enableTracing {
		log.Debug("Enabling Tracing on the Daemon")
		tracer, closer := tracing.MustInit("space-daemon")
		defer closer.Close()
		opentracing.SetGlobalTracer(tracer)
	}

	if *cpuprofile != "" {
		cleanupCpuProfile := runCpuProfiler(*cpuprofile)
		defer cleanupCpuProfile()
	}

	// env
	env := env.New()

	// load configs
	cfg := config.NewMap(cf)

	rlimit.SetRLimit()

	spaceApp := app.New(cfg, env)

	err := spaceApp.Start()
	if err != nil {
		log.Error("Application startup failed", err)
		returnCode = 1
	}

	// setup to detect interruption
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	<-interrupt // wait for interrupt and then shutdown app
	err = spaceApp.Shutdown()

	if *memprofile != "" {
		cleanupMemProfile := runMemProfiler(*memprofile)
		defer cleanupMemProfile()
	}

	if err != nil {
		log.Error("Application shutdown failed", err)
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
		pprof.StopCPUProfile()
		if f != nil {
			_ = f.Close() // error is ignored
		}
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
