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

	"github.com/FleekHQ/space-daemon/log"

	"github.com/FleekHQ/space-daemon/app"
	"github.com/FleekHQ/space-daemon/config"
	"github.com/FleekHQ/space-daemon/core/env"
)

var (
	cpuprofile           = flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile           = flag.String("memprofile", "", "write memory profile to `file`")
	debugMode            = flag.Bool("debug", true, "run daemon with debug mode for profiling")
	devMode              = flag.Bool("dev", false, "run daemon in dev mode to use .env file")
	ipfsaddr             = flag.String("ipfsaddr", "/ip4/127.0.0.1/tcp/5001", "IPFS multiaddress to connect to (defaults to local node)")
	ipfsnode             = flag.Bool("ipfsnode", true, "run IPFS embedded into the daemon (defaults to true)")
	ipfsnodeaddr         string
	ipfsnodepath         string
	powergate            = flag.Bool("filecoin", false, "run a devnet powergate daemon on port 9995")
	lotusaddr            = flag.String("lotusaddr", "/ip4/127.0.0.1/tcp/7777", "multi address pointing to lotus server (defaults to /ip4/127.0.0.1/tcp/7777)")
	powdgrpchostaddr     = flag.String("powdgrpchostaddr", "/ip4/0.0.0.0/tcp/5005", "powergate grpc host multi address")
	powdgrpcwebproxyaddr = flag.String("powdgrpcwebproxyaddr", "0.0.0.0:6005", "powergate grpc web proxy address (defaults to 0.0.0.0:6005)")
	powdgatewayhostaddr  = flag.String("powdgatewayhostaddr", "0.0.0.0:7001", "powergate gateway host address (defaults to 0.0.0.0:7001)")
	mongousr             string
	mongopw              string
	mongohost            string
	mongorepset          string
	spaceapi             string
	vaultapi             string
	vaultsaltsecret      string
	spacehubauth         string
	textilehub           string
	textilehubma         string
	textilethreads       string
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
		Ipfsaddr:                *ipfsaddr,
		Ipfsnode:                *ipfsnode == true,
		Ipfsnodeaddr:            ipfsnodeaddr,
		Ipfsnodepath:            ipfsnodepath,
		Mongousr:                mongousr,
		Mongopw:                 mongopw,
		Mongohost:               mongohost,
		Mongorepset:             mongorepset,
		ServicesAPIURL:          spaceapi,
		VaultAPIURL:             vaultapi,
		VaultSaltSecret:         vaultsaltsecret,
		ServicesHubAuthURL:      spacehubauth,
		DevMode:                 *devMode == true,
		TextileHubTarget:        textilehub,
		TextileHubMa:            textilehubma,
		TextileThreadsTarget:    textilethreads,
		TextileUserKey:          textileuserkey,
		TextileUserSecret:       textileusersecret,
		StartPowergate:          *powergate,
		LotusAddress:            *lotusaddr,
		PowdGrpcHostAddress:     *powdgrpchostaddr,
		PowdGrpcWebProxyAddress: *powdgrpcwebproxyaddr,
		PowdGatewayHostAddress:  *powdgatewayhostaddr,
	}

	// CPU profiling
	if *debugMode == true {
		log.Debug("Running daemon with profiler. Visit http://localhost:6060/debug/pprof")
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
		log.Error("Application startup failed", err)
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
