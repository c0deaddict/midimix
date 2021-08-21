package ledanimation

import (
	"fmt"
	"math"

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"

	"github.com/c0deaddict/midimix/internal/action"
	"github.com/c0deaddict/midimix/internal/midiclient"
)

type Config struct {
	Key        uint8    `mapstructure:"key"`
	Host       string   `mapstructure:"host"`
	Animations []string `mapstructure:"animations"`
}

type LedAnimation struct {
	*action.Clients
	cfg       Config
	animation int
}

func New(clients *action.Clients, config map[string]interface{}) (action.Action, error) {
	led := LedAnimation{}
	led.Clients = clients
	if err := mapstructure.Decode(config, &led.cfg); err != nil {
		return nil, err
	}
	if len(led.cfg.Animations) == 0 {
		return nil, fmt.Errorf("no animations configured")
	}
	return &led, nil
}

func (l *LedAnimation) String() string {
	return fmt.Sprintf("LedAnimation host=%s", l.cfg.Host)
}

func (l *LedAnimation) OnMidiMessage(msg midiclient.MidiMessage) {
	switch msg := msg.(type) {
	case midiclient.MidiControlChange:
		if msg.Key == l.cfg.Key {
			animation := int(math.Round(float64(msg.Value) * float64(len(l.cfg.Animations)-1)))
			if l.animation != animation {
				l.animation = animation
				l.update()
			}
		}
	}
}

func (l *LedAnimation) update() error {
	animation := l.cfg.Animations[l.animation]
	log.Info().Msgf("setting animation of %s to %s", l.cfg.Host, animation)
	subject := fmt.Sprintf("leds.animation.%s", l.cfg.Host)
	return l.Nats.Publish(subject, []byte(animation))
}
