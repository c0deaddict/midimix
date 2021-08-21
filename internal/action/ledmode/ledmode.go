package ledmode

import (
	"fmt"

	"github.com/mitchellh/mapstructure"

	"github.com/c0deaddict/midimix/internal/action"
	"github.com/c0deaddict/midimix/internal/midiclient"
)

type Config struct {
	Key   uint8  `mapstructure:"key"`
	Group string `mapstructure:"group"`
}

type LedMode struct {
	*action.Clients
	cfg   Config
	state bool
}

func New(clients *action.Clients, config map[string]interface{}) (action.Action, error) {
	led := LedMode{}
	led.Clients = clients
	if err := mapstructure.Decode(config, &led.cfg); err != nil {
		return nil, err
	}
	return &led, nil
}

func (l *LedMode) String() string {
	return fmt.Sprintf("LedMode group=%s", l.cfg.Group)
}

func (l *LedMode) OnMidiMessage(msg midiclient.MidiMessage) {
	switch msg := msg.(type) {
	case midiclient.MidiNoteOn:
		if msg.Key == l.cfg.Key {
			l.state = !l.state
			l.updateMode()
			l.Midi.SetLed(l.cfg.Key, l.state)
		}
	}
}

func (l *LedMode) mode() string {
	if l.state {
		return "on"
	} else {
		return "off"
	}
}

func (l *LedMode) updateMode() error {
	subject := fmt.Sprintf("leds.mode.%s", l.cfg.Group)
	return l.Nats.Publish(subject, []byte(l.mode()))
}
