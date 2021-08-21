package ledsetting

import (
	"encoding/json"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"

	"github.com/c0deaddict/midimix/internal/action"
	"github.com/c0deaddict/midimix/internal/midiclient"
)

type Config struct {
	Key      uint8   `mapstructure:"key"`
	Host     string  `mapstructure:"host"`
	Setting  string  `mapstructure:"setting"`
	MinValue float32 `mapstructure:"minValue"`
	MaxValue float32 `mapstructure:"maxValue"`
}

type LedSetting struct {
	*action.Clients
	cfg Config
}

func New(clients *action.Clients, config map[string]interface{}) (action.Action, error) {
	led := LedSetting{}
	led.Clients = clients
	if err := mapstructure.Decode(config, &led.cfg); err != nil {
		return nil, err
	}
	if led.cfg.Setting == "" {
		return nil, fmt.Errorf("no setting configured")
	}
	if led.cfg.MinValue >= led.cfg.MaxValue {
		return nil, fmt.Errorf("minValue >= maxValue")
	}
	return &led, nil
}

func (l *LedSetting) String() string {
	return fmt.Sprintf("LedSetting host=%s setting=%s", l.cfg.Host, l.cfg.Setting)
}

func (l *LedSetting) OnMidiMessage(msg midiclient.MidiMessage) {
	switch msg := msg.(type) {
	case midiclient.MidiControlChange:
		if msg.Key == l.cfg.Key {
			value := l.cfg.MinValue + (msg.Value * (l.cfg.MaxValue - l.cfg.MinValue))
			l.update(value)
		}
	}
}

func (l *LedSetting) update(value float32) error {
	log.Info().Msgf("host %s setting %s to %f", l.cfg.Host, l.cfg.Setting, value)
	data := make(map[string]float32)
	data[l.cfg.Setting] = value
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("esp.settings.patch.%s", l.cfg.Host)
	return l.Nats.Publish(subject, []byte(payload))
}
