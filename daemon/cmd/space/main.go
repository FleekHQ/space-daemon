package main

import (
	"context"
	"flag"

	"github.com/FleekHQ/space/daemon/app"
	"github.com/FleekHQ/space/daemon/config"
	"github.com/FleekHQ/space/daemon/core/env"
	"github.com/FleekHQ/space/daemon/log"
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
