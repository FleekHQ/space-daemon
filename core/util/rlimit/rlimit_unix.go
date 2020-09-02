// +build aix darwin dragonfly freebsd js,wasm linux nacl netbsd openbsd solaris

package rlimit

import (
	"fmt"
	"math"
	"syscall"

	"github.com/FleekHQ/space-daemon/log"
)

// Sets rlimit to the maximum allowed value in UNIX systems
func SetRLimit() {
	var rLimit syscall.Rlimit

	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		log.Error("Error Getting Rlimit. Please run `ulimit -n 1000` from a privileged user to avoid issues when running the space daemon.", err)
		return
	}
	log.Debug(fmt.Sprintf("Got Rlimit: Cur: %d, Max: %d", rLimit.Cur, rLimit.Max))

	// Max allowed value is 10240 even when rLimit.Max can go beyond that
	rLimit.Cur = uint64(math.Min(10240, float64(rLimit.Max)))

	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		log.Error("Error setting Rlimit. Please run `ulimit -n 1000` from a privileged user to avoid issues when running the space daemon.", err)
		return
	}

	log.Debug(fmt.Sprintf("Set Rlimit: Cur: %d, Max: %d", rLimit.Cur, rLimit.Max))
}
