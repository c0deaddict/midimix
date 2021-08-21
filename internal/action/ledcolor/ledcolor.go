package ledcolor

import (
	"fmt"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"

	"github.com/c0deaddict/midimix/internal/action"
	"github.com/c0deaddict/midimix/internal/midiclient"
)

const (
	FormatRGB = "rgb"
	FormatHSV = "hsv"
)

type Config struct {
	Host     string   `mapstructure:"host"`
	Controls [3]uint8 `mapstructure:"controls"`
	Format   string   `mapstructure:"format"`
}

type LedColor struct {
	*action.Clients
	cfg   Config
	state [3]float32
}

func New(clients *action.Clients, config map[string]interface{}) (action.Action, error) {
	led := LedColor{}
	led.Clients = clients
	if err := mapstructure.Decode(config, &led.cfg); err != nil {
		return nil, err
	}
	return &led, nil
}

func (l *LedColor) String() string {
	return fmt.Sprintf("LedColor host=%s", l.cfg.Host)
}

func (l *LedColor) OnMidiMessage(msg midiclient.MidiMessage) {
	switch msg := msg.(type) {
	case midiclient.MidiControlChange:
		update := false
		for i, key := range l.cfg.Controls {
			if key == msg.Key {
				l.state[i] = msg.Value
				update = true
			}
		}

		if update {
			if err := l.updateColor(); err != nil {
				log.Warn().Err(err).Msg("nats update color failed")
			}
		}
	}
}

func (l *LedColor) color() string {
	switch l.cfg.Format {
	case FormatHSV:
		h := float64(360.0 * l.state[0])
		s := float64(l.state[1])
		v := float64(l.state[2])
		return colorful.Hsv(h, s, v).Hex()[1:]

	// case FormatRGB:
	default:
		r := uint8(255 * l.state[0])
		g := uint8(255 * l.state[1])
		b := uint8(255 * l.state[2])
		return fmt.Sprintf("%02x%02x%02x", r, g, b)
	}
}

func (l *LedColor) updateColor() error {
	subject := fmt.Sprintf("leds.color.%s", l.cfg.Host)
	return l.Nats.Publish(subject, []byte(l.color()))
}
