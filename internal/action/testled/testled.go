package testled

import (
	"fmt"

	"github.com/mitchellh/mapstructure"

	"github.com/c0deaddict/midimix/internal/action"
	"github.com/c0deaddict/midimix/internal/midiclient"
)

type Config struct {
	Key uint8
}

type TestLed struct {
	*action.Clients
	cfg   Config
	state bool
}

func New(clients *action.Clients, config map[string]interface{}) (action.Action, error) {
	led := TestLed{}
	led.Clients = clients
	if err := mapstructure.Decode(config, &led.cfg); err != nil {
		return nil, err
	}
	return &led, nil
}

func (l *TestLed) String() string {
	return fmt.Sprintf("TestLed key=%d", l.cfg.Key)
}

func (l *TestLed) OnMidiMessage(msg midiclient.MidiMessage) {
	switch msg := msg.(type) {
	case midiclient.MidiNoteOn:
		if l.cfg.Key == msg.Key {
			l.state = !l.state
			l.Midi.SetLed(l.cfg.Key, l.state)
		}
	}
}
