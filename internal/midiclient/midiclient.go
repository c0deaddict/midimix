package midiclient

import (
	"fmt"

	"github.com/c0deaddict/midimix/internal/config"
	"github.com/rs/zerolog/log"
	"gitlab.com/gomidi/midi"
	"gitlab.com/gomidi/midi/midimessage/channel"
	"gitlab.com/gomidi/midi/midimessage/sysex"
	"gitlab.com/gomidi/midi/reader"
	"gitlab.com/gomidi/midi/writer"
	"gitlab.com/gomidi/rtmididrv"
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
	drv, err := rtmididrv.New()
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

			case sysex.SysEx:
				cfg := &ConfigurationResponse{}
				if err := cfg.ReadFrom(msg.Raw()); err != nil {
					log.Error().Err(err).Msg("failed to parse configuration response message")
				} else {
					log.Info().Msgf("cfg response: %v", cfg)
				}
			default:
				log.Debug().Msgf("unknown message: %v", msg)
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

type RequestExistingConfiguration struct{}

func (r RequestExistingConfiguration) String() string {
	return "RequestExistingConfiguration"
}

// Request Existing Configuration (0x66)
// https://docs.google.com/document/d/1zeRPklp_Mo_XzJZUKu2i-p1VgfBoUWF0JKZ5CzX8aB0/view
func (r RequestExistingConfiguration) Raw() []byte {
	return []byte{0xF0, 0x47, 0x00, 0x31, 0x66, 0x00, 0x01, 0xF7}
}

func (m *MidiClient) RequestAll() error {
	return m.wr.Write(RequestExistingConfiguration{})
}

type ConfigurationResponse struct {
	Dials [24]struct {
		Channel byte // 0-15 representing channel 1-16
		Value   byte // 0-127
	}
	Sliders [9]struct {
		Channel byte // 0-15 representing channel 1-16
		Value   byte // 0-127
	}
	MuteButtons [8]struct {
		Channel    byte // 0-15 representing channel 1-16
		ButtonMode byte // 0 = note, 1 = CC (continuous control)
		Value      byte
	}
	RecArmButtons [8]struct {
		Channel    byte // 0-15 representing channel 1-16
		ButtonMode byte // 0 = note, 1 = CC (continuous control)
		Value      byte
	}
	MuteSoloButtons [8]struct {
		Channel    byte // 0-15 representing channel 1-16
		ButtonMode byte // 0 = note, 1 = CC (continuous control)
		Value      byte
	}
}

func (c *ConfigurationResponse) ReadFrom(msg []byte) error {
	if msg[0] != 0xF0 {
		return fmt.Errorf("not a SysEx message")
	}

	if len(msg) != 146 {
		return fmt.Errorf("length must be 146 bytes")
	}

	if msg[1] != 0x47 || msg[3] != 0x31 {
		return fmt.Errorf("manufacturer ID (%#x) and/or model ID (%#x) do not match", msg[1], msg[3])
	}

	if msg[4] != 0x67 {
		return fmt.Errorf("not a configuration response message")
	}

	idx := 7

	for i := 0; i < 24; i++ {
		c.Dials[i].Channel = msg[idx]
		c.Dials[i].Value = msg[idx+1]
		idx += 2
	}

	for i := 0; i < 9; i++ {
		c.Sliders[i].Channel = msg[idx]
		c.Sliders[i].Value = msg[idx+1]
		idx += 2
	}

	for i := 0; i < 8; i++ {
		c.MuteButtons[i].Channel = msg[idx]
		c.MuteButtons[i].ButtonMode = msg[idx+1]
		c.MuteButtons[i].Value = msg[idx+2]
		idx += 3
	}

	for i := 0; i < 8; i++ {
		c.RecArmButtons[i].Channel = msg[idx]
		c.RecArmButtons[i].ButtonMode = msg[idx+1]
		c.RecArmButtons[i].Value = msg[idx+2]
		idx += 3
	}

	for i := 0; i < 8; i++ {
		c.MuteSoloButtons[i].Channel = msg[idx]
		c.MuteSoloButtons[i].ButtonMode = msg[idx+1]
		c.MuteSoloButtons[i].Value = msg[idx+2]
		idx += 3
	}

	return nil
}
