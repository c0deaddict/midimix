package midiclient

import (
	"fmt"

	"github.com/c0deaddict/midimix/internal/config"
	"github.com/rs/zerolog/log"
	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
)

type MidiClient struct {
	in  drivers.In
	out drivers.Out
	cfg *config.MidiConfig
}

type MidiMessage interface{}

type MidiNoteOn struct {
	Key      uint8
	Velocity float32
}

type MidiNoteOff struct {
	Key uint8
}

type MidiControlChange struct {
	Key   uint8
	Value float32
}

func Open(cfg config.MidiConfig) (*MidiClient, error) {
	in, err := midi.FindInPort(cfg.Input)
	if err != nil {
		return nil, fmt.Errorf("input midi device %s not found: %v", cfg.Input, err)
	}
	log.Info().Msgf("found midi input device: %s", in.String())

	out, err := midi.FindOutPort(cfg.Output)
	if err != nil {
		return nil, fmt.Errorf("output midi device %s not found: %v", cfg.Output, err)
	}
	log.Info().Msgf("found midi output device: %s", out.String())

	if err := out.Open(); err != nil {
		return nil, fmt.Errorf("opening midi out: %v", err)
	}

	return &MidiClient{in, out, &cfg}, nil
}

func (m *MidiClient) Close() {
	midi.CloseDriver()
	log.Info().Msg("midi closed")
}

func (m *MidiClient) Listen(out chan MidiMessage) (func(), error) {
	return midi.ListenTo(m.in, func(msg midi.Message, timestampms int32) {
		var ch, key, vel, con, val uint8
		switch {
		case msg.GetNoteOn(&ch, &key, &vel):
			if ch == m.cfg.Channel {
				out <- MidiNoteOn{
					key,
					float32(vel) / float32(m.cfg.MaxInputValue),
				}
			}

		case msg.GetNoteOff(&ch, &key, &vel):
			if ch == m.cfg.Channel {
				out <- MidiNoteOff{key}
			}

		case msg.GetControlChange(&ch, &con, &val):
			if ch == m.cfg.Channel {
				out <- MidiControlChange{
					con,
					float32(val) / float32(m.cfg.MaxInputValue),
				}
			}
		}
	})
}

func (m *MidiClient) send(msg midi.Message) error {
	err := m.out.Send(msg)
	if err != nil {
		log.Error().Err(err).Msg("send midi message")
	}
	return err
}

func (m *MidiClient) LedOn(key uint8) {
	m.send(midi.NoteOn(m.cfg.Channel, key, 127))
}

func (m *MidiClient) LedOff(key uint8) {
	m.send(midi.NoteOn(m.cfg.Channel, key, 0))
}

func (m *MidiClient) SetLed(key uint8, state bool) {
	if state {
		m.LedOn(key)
	} else {
		m.LedOff(key)
	}
}
