package action

import (
	"github.com/c0deaddict/midimix/internal/midiclient"
)

type Action interface {
	String() string
	OnMidiMessage(msg midiclient.MidiMessage)
}
