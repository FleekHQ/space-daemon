package watcher

type watcherOptions struct {
	paths []string
}

// Option configuration for the FileWatcher
// Use exported Option factory functions
type Option func(option *watcherOptions)

// WithPaths configures the list of paths the file watcher would be watching recursively.
// For best results do not include two paths withing the same directory
func WithPaths(path ...string) Option {
	return func(option *watcherOptions) {
		for _, p := range path {
			option.paths = append(option.paths, p)
		}
	}
}
