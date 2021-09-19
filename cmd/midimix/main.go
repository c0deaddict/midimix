package main

import (
	"flag"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/c0deaddict/midimix/internal/config"
	"github.com/c0deaddict/midimix/internal/midimix"
)

var configFile = flag.String("config", "$HOME/.config/midimix/config.yaml", "Config file")
var cpuProfile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()

	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to start cpu profile")
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	cfg, err := config.Read(os.ExpandEnv(*configFile))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read config")
	}

	midimix, err := midimix.Open(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start midimix")
	}
	defer midimix.Close()

	go midimix.Run()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
}
