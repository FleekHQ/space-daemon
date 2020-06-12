package main

import (
	"context"
	"flag"

	"github.com/FleekHQ/space/app"
	"github.com/FleekHQ/space/config"
	"github.com/FleekHQ/space/core/env"
	"github.com/FleekHQ/space/log"
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

	app.Start(ctx, cfg, env)
}
