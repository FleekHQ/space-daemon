package rlimit

// Rlimit not supported on windows
func SetRLimit() {
	return
}
