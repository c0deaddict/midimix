package action

import (
	"github.com/c0deaddict/midimix/internal/midiclient"
	"github.com/c0deaddict/midimix/internal/paclient"
	"github.com/nats-io/nats.go"
)

type NewAction = func(clients *Clients, config map[string]interface{}) (Action, error)

type Action interface {
	String() string
	OnMidiMessage(msg midiclient.MidiMessage)
}

type Clients struct {
	Nats  *nats.Conn
	Midi  *midiclient.MidiClient
	Pulse *paclient.PulseAudioClient
}
