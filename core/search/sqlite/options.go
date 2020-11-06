package sqlite

import "gorm.io/gorm/logger"

func WithDBPath(path string) Option {
	return func(o *sqliteSearchOption) {
		o.dbPath = path
	}
}

func WithLogLevel(level logger.LogLevel) Option {
	return func(o *sqliteSearchOption) {
		o.logLevel = level
	}
}
