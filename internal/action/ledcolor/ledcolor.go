package ledcolor

import (
	"github.com/c0deaddict/midimix/internal/midiclient"
	"github.com/mitchellh/mapstructure"
)

const (
	FORMAT_RGB = "rgb"
	FORMAT_HSV = "hsv"
)

type Config struct {
	Host     string   `mapstructure:"host"`
	Controls [3]uint8 `mapstructure:"controls"`
	Format   string   `mapstructure:"format"`
}

type LedColor struct {
	cfg   Config
	state [3]uint8
}

func New(config map[string]interface{}) (*LedColor, error) {
	led := LedColor{}
	if err := mapstructure.Decode(config, &led.cfg); err != nil {
		return nil, err
	}
	return &led, nil
}

func (l *LedColor) OnMidiMessage(msg midiclient.MidiMessage) {
	switch msg := msg.(type) {
	case midiclient.MidiControlChange:
		update := false
		for i, key := range l.cfg.Controls {
			if key == msg.Key {
				l.state[i] = uint8(255 * msg.Value)
				update = true
			}
		}

		if update {
			// TODO update leds
		}
	}
}
