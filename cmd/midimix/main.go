package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/c0deaddict/midimix/internal/config"
	"github.com/c0deaddict/midimix/internal/midiclient"
	"github.com/c0deaddict/midimix/internal/paclient"
)

var configFile = flag.String("config", "$HOME/.config/midimix/config.yaml", "Config file")

func main() {
	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	cfg, err := config.Read(os.ExpandEnv(*configFile))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read config")
	}

	mc, err := midiclient.Open(cfg.Midi)
	if err != nil {
		log.Fatal().Err(err).Msg("midi error")
	}
	defer mc.Close()

	pa, err := paclient.Open(cfg.PulseAudio, mc)
	if err != nil {
		log.Fatal().Err(err).Msg("pulseaudio error")
	}
	defer pa.Close()

	ch := make(chan midiclient.MidiMessage)
	if err := mc.Listen(ch); err != nil {
		log.Fatal().Err(err).Msg("midi error")
	}

	// nc, err := natsclient.Connect(cfg.Nats)
	// if err != nil {
	// 	log.Fatalln("Nats:", err)
	// }
	// defer nc.Close()

	go func() {
		for msg := range ch {
			log.Info().Msgf("%v", msg)
			// TODO: emit all on nats
			pa.OnMidiMessage(msg)
		}
	}()

	go pa.Listen()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
}
