package midimix

import (
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/c0deaddict/midimix/internal/action"
	"github.com/c0deaddict/midimix/internal/action/ledcolor"
	"github.com/c0deaddict/midimix/internal/action/ledmode"
	"github.com/c0deaddict/midimix/internal/action/testled"
	"github.com/c0deaddict/midimix/internal/config"
	"github.com/c0deaddict/midimix/internal/midiclient"
	"github.com/c0deaddict/midimix/internal/natsclient"
	"github.com/c0deaddict/midimix/internal/paclient"
)

var actions = map[string]action.NewAction{
	"LedColor": ledcolor.New,
	"LedMode":  ledmode.New,
	"TestLed":  testled.New,
}

type Midimix struct {
	action.Clients
	actions []action.Action
	ch      chan midiclient.MidiMessage
}

func Open(cfg *config.Config) (*Midimix, error) {
	m := &Midimix{}
	var err error

	m.Nats, err = natsclient.Connect(cfg.Nats)
	if err != nil {
		return nil, fmt.Errorf("nats: %v", err)
	}

	m.Midi, err = midiclient.Open(cfg.Midi)
	if err != nil {
		m.Nats.Close()
		return nil, fmt.Errorf("midi: %v", err)
	}

	m.ch = make(chan midiclient.MidiMessage)
	if err := m.Midi.Listen(m.ch); err != nil {
		m.Midi.Close()
		m.Nats.Close()
		log.Fatal().Err(err).Msg("midi listen failed")
	}

	m.Pulse, err = paclient.Open(cfg.PulseAudio, m.Midi)
	if err != nil {
		m.Midi.Close()
		m.Nats.Close()
		return nil, fmt.Errorf("pulseaudio: %v", err)
	}

	m.actions = make([]action.Action, 0, len(cfg.Actions))
	for _, actionCfg := range cfg.Actions {
		newAction, ok := actions[actionCfg.Type]
		if !ok {
			log.Error().Msgf("unknown action type: %v", actionCfg.Type)
			continue
		}

		action, err := newAction(&m.Clients, actionCfg.Config)
		if err != nil {
			log.Error().Err(err).Msgf("instantiate action %s failed", actionCfg.Type)
			continue
		}

		log.Info().Msgf("instantiated action %v", action)
		m.actions = append(m.actions, action)
	}

	return m, nil
}

func (m *Midimix) Run() {
	go func() {
		for msg := range m.ch {
			log.Info().Msgf("%v", msg)
			m.Pulse.OnMidiMessage(msg)
			for _, action := range m.actions {
				action.OnMidiMessage(msg)
			}
			// TODO: emit all on nats
		}
	}()

	m.Pulse.Listen()
}

func (m *Midimix) Close() {
	if m.ch != nil {
		close(m.ch)
	}
	if m.Pulse != nil {
		m.Pulse.Close()
	}
	if m.Midi != nil {
		m.Midi.Close()
	}
	m.Nats.Close()
}
