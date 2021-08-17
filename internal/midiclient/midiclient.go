package midiclient

import (
	"fmt"
	"log"

	"github.com/c0deaddict/midimix/internal/config"
	"gitlab.com/gomidi/midi"
	"gitlab.com/gomidi/midi/reader"
	"gitlab.com/gomidi/midi/writer"
	"gitlab.com/gomidi/portmididrv"
)

type MidiClient struct {
	drv midi.Driver
	in  midi.In
	out midi.Out
	wr  *writer.Writer
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
		log.Printf("Found input midi device: %s", port.String())
		if port.String() == cfg.Input {
			in = port
			break
		}
	}

	for _, port := range outs {
		log.Printf("Found output midi device: %s", port.String())
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

	return &MidiClient{drv, in, out, wr}, nil
}

func (m *MidiClient) Close() {
	m.in.Close()
	m.out.Close()
	m.drv.Close()
	log.Println("Midi: closed")
}

func (m *MidiClient) Listen(ch chan midi.Message) error {
	rd := reader.New(
		reader.NoLogger(),
		reader.Each(func(pos *reader.Position, msg midi.Message) {
			ch <- msg
		}),
	)

	return rd.ListenTo(m.in)
}
