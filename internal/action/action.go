package action

import "github.com/c0deaddict/midimix/internal/midiclient"

type Action interface {
	OnMidiMessage(msg midiclient.MidiMessage)
}
