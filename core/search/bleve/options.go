package bleve

func WithDBPath(path string) Option {
	return func(o *bleveSearchOption) {
		o.dbPath = path
	}
}
