package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/c0deaddict/midimix/internal/action"
	"github.com/c0deaddict/midimix/internal/action/ledcolor"
	"github.com/c0deaddict/midimix/internal/config"
	"github.com/c0deaddict/midimix/internal/midiclient"
	"github.com/c0deaddict/midimix/internal/natsclient"
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

	nc, err := natsclient.Connect(cfg.Nats)
	if err != nil {
		log.Fatal().Err(err).Msg("nats error")
	}
	defer nc.Close()

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

	actions := make([]action.Action, 0, len(cfg.Actions))
	for _, acfg := range cfg.Actions {
		a, err := NewAction(acfg, nc)
		if err != nil {
			log.Warn().Err(err).Msg("instantiate action failed")
		} else {
			log.Info().Msgf("instantiated action %v", a)
			actions = append(actions, a)
		}
	}

	ch := make(chan midiclient.MidiMessage)
	if err := mc.Listen(ch); err != nil {
		log.Fatal().Err(err).Msg("midi error")
	}

	go func() {
		for msg := range ch {
			log.Info().Msgf("%v", msg)
			pa.OnMidiMessage(msg)
			for _, a := range actions {
				a.OnMidiMessage(msg)
			}
			// TODO: emit all on nats
		}
	}()

	go pa.Listen()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
}

func NewAction(action config.Action, nc *nats.Conn) (action.Action, error) {
	switch action.Type {
	case "LedColor":
		return ledcolor.New(action.Config, nc)
	default:
		return nil, fmt.Errorf("unknown action type: %s", action.Type)
	}
}
