package core

// Component represents core application components. Modules should implement this interface to allow for proper
// dependency checks and shutdown
type Component interface {
	Shutdown() error
}

// AsyncComponent represents components that have some async initialization
// and therefore must provide a ready channel to listen to
type AsyncComponent interface {
	Component
	WaitForReady() chan bool
}
