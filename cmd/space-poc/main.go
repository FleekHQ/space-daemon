package main

import (
	"context"
	"flag"
	"github.com/FleekHQ/space-poc/app"
	"github.com/FleekHQ/space-poc/config"
	"github.com/FleekHQ/space-poc/core/env"
	"github.com/FleekHQ/space-poc/log"
)

func main() {
	// flags
	flag.Parse()

	// env
	env := env.New()

	// load configs
	cfg := config.New(env)

	// setup logger
	log.New(env)
	// setup context
	ctx := context.Background()

	app.Start(ctx, cfg)
}
