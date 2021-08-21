package midiclient

import (
	"fmt"

	"github.com/c0deaddict/midimix/internal/config"
	"github.com/rs/zerolog/log"
	"gitlab.com/gomidi/midi"
	"gitlab.com/gomidi/midi/midimessage/channel"
	"gitlab.com/gomidi/midi/reader"
	"gitlab.com/gomidi/midi/writer"
	"gitlab.com/gomidi/portmididrv"
)

type MidiClient struct {
	drv midi.Driver
	in  midi.In
	out midi.Out
	wr  *writer.Writer
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
	drv, err := portmididrv.New()
	if err != nil {
		return nil, err
	}

	ins, err := drv.Ins()
	if err != nil {
		drv.Close()
		return nil, err
	}

	outs, err := drv.Outs()
	if err != nil {
		drv.Close()
		return nil, err
	}

	var in midi.In
	var out midi.Out

	for _, port := range ins {
		log.Info().Msgf("found midi input device: %s", port.String())
		if port.String() == cfg.Input {
			in = port
			break
		}
	}

	for _, port := range outs {
		log.Info().Msgf("found midi output device: %s", port.String())
		if port.String() == cfg.Output {
			out = port
			break
		}
	}

	if in == nil {
		return nil, fmt.Errorf("input midi device %s not found", cfg.Input)
	}

	if out == nil {
		return nil, fmt.Errorf("output midi device %s not found", cfg.Output)
	}

	if err := in.Open(); err != nil {
		drv.Close()
		return nil, err
	}

	if err := out.Open(); err != nil {
		in.Close()
		drv.Close()
		return nil, err
	}

	wr := writer.New(out)

	return &MidiClient{drv, in, out, wr, &cfg}, nil
}

func (m *MidiClient) Close() {
	m.in.Close()
	m.out.Close()
	m.drv.Close()
	log.Info().Msg("midi closed")
}

func (m *MidiClient) Listen(ch chan MidiMessage) error {
	rd := reader.New(
		reader.NoLogger(),
		reader.IgnoreMIDIClock(),
		reader.Each(func(pos *reader.Position, msg midi.Message) {
			switch msg := msg.(type) {
			case channel.NoteOn:
				if msg.Channel() == m.cfg.Channel {
					ch <- MidiNoteOn{
						msg.Key(),
						float32(msg.Velocity()) / float32(m.cfg.MaxInputValue),
					}
				}

			case channel.NoteOff:
				if msg.Channel() == m.cfg.Channel {
					ch <- MidiNoteOff{msg.Key()}
				}

			case channel.ControlChange:
				if msg.Channel() == m.cfg.Channel {
					ch <- MidiControlChange{
						msg.Controller(),
						float32(msg.Value()) / float32(m.cfg.MaxInputValue),
					}
				}
			}
		}),
	)

	return rd.ListenTo(m.in)
}

func (m *MidiClient) LedOn(key uint8) {
	writer.NoteOn(m.wr, key, 127)
}

func (m *MidiClient) LedOff(key uint8) {
	// NOTE: without the NoteOn the LED doesn't go off most of the time..?
	writer.NoteOn(m.wr, key, 127)
	writer.NoteOff(m.wr, key)
}

func (m *MidiClient) SetLed(key uint8, state bool) {
	if state {
		m.LedOn(key)
	} else {
		m.LedOff(key)
	}
}
