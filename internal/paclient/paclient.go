package paclient

import (
	"github.com/godbus/dbus"
	"github.com/rs/zerolog/log"
	"github.com/sqp/pulseaudio"

	"github.com/c0deaddict/midimix/internal/config"
	"github.com/c0deaddict/midimix/internal/midiclient"
)

type PulseAudioClient struct {
	client            *pulseaudio.Client
	cfg               config.PulseAudioConfig
	targetsObjectPath []*dbus.ObjectPath
	midi              *midiclient.MidiClient
}

func Open(cfg config.PulseAudioConfig, midi *midiclient.MidiClient) (*PulseAudioClient, error) {
	client, err := pulseaudio.New()
	if err != nil {
		return nil, err
	}

	targetsObjectPath := make([]*dbus.ObjectPath, len(cfg.Targets))
	pa := PulseAudioClient{client, cfg, targetsObjectPath, midi}
	pa.Refresh()
	client.Register(&pa)

	return &pa, nil
}

func (p *PulseAudioClient) Close() error {
	// Clear all leds.
	for _, target := range p.cfg.Targets {
		if target.Mute != nil {
			p.midi.LedOff(*target.Mute)
		}
		if target.Presence != nil {
			p.midi.LedOff(*target.Presence)
		}
	}

	return p.client.Close()
}

func (p *PulseAudioClient) Listen() {
	p.client.Listen()
}

func (p *PulseAudioClient) Refresh() {
	p.refreshTargetType(config.PlaybackStream)
	p.refreshTargetType(config.RecordStream)
	p.refreshTargetType(config.Sink)
	p.refreshTargetType(config.Source)
	p.updateLeds()
}

func (p *PulseAudioClient) updateLeds() {
	for i, path := range p.targetsObjectPath {
		target := p.cfg.Targets[i]
		if target.Presence != nil {
			p.midi.SetLed(*target.Presence, path != nil)
		}

		if target.Mute != nil {
			if path == nil {
				p.midi.LedOff(*target.Mute)
			} else {
				obj := p.getObject(target.Type, *path)
				mute, _ := obj.Bool("Mute")
				p.midi.SetLed(*target.Mute, mute)
			}
		}
	}
}

func (p *PulseAudioClient) getObject(targetType config.PulseAudioTargetType, path dbus.ObjectPath) *pulseaudio.Object {
	switch targetType {
	case config.PlaybackStream, config.RecordStream:
		return p.client.Stream(path)
	case config.Sink, config.Source:
		return p.client.Device(path)
	}

	return nil
}

func (p *PulseAudioClient) refreshTargetType(targetType config.PulseAudioTargetType) {
	var nameProperty string
	switch targetType {
	case config.PlaybackStream, config.RecordStream:
		nameProperty = "application.name"
	case config.Sink, config.Source:
		nameProperty = "device.description"
	}

	dbusType := string(targetType) + "s"
	objs, err := p.client.Core().ListPath(dbusType)
	if err != nil {
		log.Error().Err(err).Msgf("listing %s failed", targetType)
		return
	}

	for _, path := range objs {
		obj := p.getObject(targetType, path)
		name, ok := p.getProperty(obj, nameProperty)
		if !ok {
			log.Warn().Msgf("%s %v doesn't have an '%s' property", targetType, path, nameProperty)
			continue
		}

		for i, target := range p.cfg.Targets {
			if target.Type == targetType && target.Name == name {
				pathRef := path // Need a local copy of path to reference to.
				p.targetsObjectPath[i] = &pathRef
				log.Debug().Msgf("matched targets[%d] (%s) to %v", i, name, path)
				break
			}
		}
	}
}

func (p *PulseAudioClient) getProperty(obj *pulseaudio.Object, property string) (string, bool) {
	props, err := obj.MapString("PropertyList")
	if err != nil {
		log.Warn().Msgf("failed to get property list of %v", obj)
		return "", false
	}

	val, ok := props[property]
	return val, ok
}

func (p *PulseAudioClient) OnMidiMessage(msg midiclient.MidiMessage) {
	switch msg := msg.(type) {
	case midiclient.MidiControlChange:
		for i, target := range p.cfg.Targets {
			if target.Volume != nil && *target.Volume == msg.Key {
				path := p.targetsObjectPath[i]
				if path == nil {
					continue
				}

				obj := p.getObject(target.Type, *path)
				if err := p.setVolume(obj, msg.Value); err != nil {
					log.Error().Err(err).Msg("failed to set volume")
				}
			}
		}

	case midiclient.MidiNoteOff:
		for i, target := range p.cfg.Targets {
			if target.Mute != nil && *target.Mute == msg.Key {
				log.Info().Msgf("toggle mute target %d", i)

				path := p.targetsObjectPath[i]
				if path == nil {
					log.Info().Msg("target not active")
					continue
				}

				obj := p.getObject(target.Type, *path)
				if err := p.toggleMute(obj); err != nil {
					log.Error().Err(err).Msg("failed to toggle mute")
				}
			}
		}
	}
}

func (p *PulseAudioClient) setVolume(obj *pulseaudio.Object, volume float32) error {
	value := uint32(volume * 65535)
	vol := make([]uint32, 0)
	if channels, err := obj.ListUint32("Channels"); err != nil {
		return err
	} else {
		for range channels {
			vol = append(vol, value)
		}
	}

	return obj.Set("Volume", vol)
}

func (p *PulseAudioClient) toggleMute(obj *pulseaudio.Object) error {
	mute, _ := obj.Bool("Mute")
	return obj.Set("Mute", !mute)
}

func (p *PulseAudioClient) NewPlaybackStream(path dbus.ObjectPath) {
	p.Refresh()
}

func (p *PulseAudioClient) PlaybackStreamRemoved(path dbus.ObjectPath) {
	p.Refresh()
}

func (p *PulseAudioClient) DeviceMuteUpdated(path dbus.ObjectPath, mute bool) {
	if target := p.findTargetByPath(path); target != nil {
		if target.Mute != nil {
			p.midi.SetLed(*target.Mute, mute)
		}
	}
}

func (p *PulseAudioClient) StreamMuteUpdated(path dbus.ObjectPath, mute bool) {
	if target := p.findTargetByPath(path); target != nil {
		if target.Mute != nil {
			p.midi.SetLed(*target.Mute, mute)
		}
	}
}

func (p *PulseAudioClient) findTargetByPath(path dbus.ObjectPath) *config.PulseAudioTarget {
	for i, targetPath := range p.targetsObjectPath {
		if targetPath != nil && *targetPath == path {
			return &p.cfg.Targets[i]
		}
	}
	return nil
}
