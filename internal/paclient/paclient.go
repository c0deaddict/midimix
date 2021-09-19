package paclient

import (
	"github.com/godbus/dbus"
	"github.com/rs/zerolog/log"
	"github.com/sqp/pulseaudio"

	"github.com/c0deaddict/midimix/internal/config"
	"github.com/c0deaddict/midimix/internal/midiclient"
)

type PulseAudioTarget struct {
	cfg    config.PulseAudioTarget
	paths  []dbus.ObjectPath
	mute   bool
	volume float32
}

type PulseAudioClient struct {
	client  *pulseaudio.Client
	cfg     config.PulseAudioConfig
	targets []PulseAudioTarget
	midi    *midiclient.MidiClient
}

func Open(cfg config.PulseAudioConfig, midi *midiclient.MidiClient) (*PulseAudioClient, error) {
	client, err := pulseaudio.New()
	if err != nil {
		return nil, err
	}

	pa := PulseAudioClient{
		client:  client,
		cfg:     cfg,
		targets: make([]PulseAudioTarget, 0, len(cfg.Targets)),
		midi:    midi,
	}

	for _, targetCfg := range cfg.Targets {
		pa.targets = append(pa.targets, PulseAudioTarget{
			cfg:    targetCfg,
			paths:  make([]dbus.ObjectPath, 0),
			mute:   false,
			volume: 1.0,
		})
	}

	pa.Refresh()
	client.Register(&pa)

	return &pa, nil
}

func (p *PulseAudioClient) Close() error {
	// Clear all leds.
	for _, target := range p.targets {
		if target.cfg.Mute != nil {
			p.midi.LedOff(*target.cfg.Mute)
		}
		if target.cfg.Presence != nil {
			p.midi.LedOff(*target.cfg.Presence)
		}
	}

	return p.client.Close()
}

func (p *PulseAudioClient) Listen() {
	p.client.Listen()
}

func (p *PulseAudioClient) Refresh() {
	for i, _ := range p.targets {
		p.targets[i].paths = make([]dbus.ObjectPath, 0)
	}

	p.refreshTargetType(config.PlaybackStream)
	p.refreshTargetType(config.RecordStream)
	p.refreshTargetType(config.Sink)
	p.refreshTargetType(config.Source)
	p.updateLeds()
}

func (p *PulseAudioClient) updateLeds() {
	for _, target := range p.targets {
		if target.cfg.Presence != nil {
			p.midi.SetLed(*target.cfg.Presence, len(target.paths) != 0)
		}

		if target.cfg.Mute != nil {
			if len(target.paths) == 0 {
				p.midi.LedOff(*target.cfg.Mute)
			} else {
				p.midi.SetLed(*target.cfg.Mute, target.mute)
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

func (p *PulseAudioClient) matchTarget(targetType config.PulseAudioTargetType, obj *pulseaudio.Object) *PulseAudioTarget {
	var nameProperty string
	switch targetType {
	case config.PlaybackStream, config.RecordStream:
		nameProperty = "application.name"
	case config.Sink, config.Source:
		nameProperty = "device.description"
	}

	name, ok := p.getProperty(obj, nameProperty)
	if !ok {
		log.Warn().Msgf("%s %v doesn't have an '%s' property", targetType, obj.Path(), nameProperty)
		return nil
	}

	for i, target := range p.targets {
		if target.cfg.Type == targetType && target.cfg.Name == name {
			log.Info().Msgf("matched target '%s' to %v", name, obj.Path())
			return &p.targets[i]
		}
	}

	return nil
}

func (p *PulseAudioClient) refreshTargetType(targetType config.PulseAudioTargetType) {
	dbusType := string(targetType) + "s"
	objs, err := p.client.Core().ListPath(dbusType)
	if err != nil {
		log.Error().Err(err).Msgf("listing %s failed", targetType)
		return
	}

	for _, path := range objs {
		obj := p.getObject(targetType, path)
		if target := p.matchTarget(targetType, obj); target != nil {
			target.paths = append(target.paths, path)
			mute, _ := obj.Bool("mute")
			target.mute = mute
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
		for i, target := range p.targets {
			if target.cfg.Volume != nil && *target.cfg.Volume == msg.Key {
				volume := msg.Value
				p.targets[i].volume = volume
				for _, path := range target.paths {
					obj := p.getObject(target.cfg.Type, path)
					if err := p.setVolume(obj, volume); err != nil {
						log.Error().Err(err).Msgf("failed to set volume of %v", path)
					}
				}
			}
		}

	case midiclient.MidiNoteOff:
		for i, target := range p.targets {
			if target.cfg.Mute != nil && *target.cfg.Mute == msg.Key {
				mute := !target.mute
				p.targets[i].mute = mute
				for _, path := range target.paths {
					obj := p.getObject(target.cfg.Type, path)
					if err := p.setMute(obj, mute); err != nil {
						log.Error().Err(err).Msgf("failed to set mute on %v", path)
					}
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

func (p *PulseAudioClient) setMute(obj *pulseaudio.Object, mute bool) error {
	return obj.Set("Mute", mute)
}

func (p *PulseAudioClient) NewPlaybackStream(path dbus.ObjectPath) {
	log.Info().Msgf("playback stream added: %v", path)

	targetType := config.PlaybackStream
	obj := p.getObject(targetType, path)
	if target := p.matchTarget(targetType, obj); target != nil {
		log.Info().Msgf("setting mute=%v volume=%v on %v", target.mute, target.volume, path)
		target.paths = append(target.paths, path)
		p.setMute(obj, target.mute)
		if err := p.setVolume(obj, target.volume); err != nil {
			log.Warn().Err(err).Msgf("failed to set volume of %v", path)
		}
		if target.cfg.Presence != nil {
			p.midi.LedOn(*target.cfg.Presence)
		}
	}
}

func (p *PulseAudioClient) PlaybackStreamRemoved(path dbus.ObjectPath) {
	log.Info().Msgf("playback stream removed: %v", path)
	if target, idx := p.findTargetByPath(path); target != nil {
		target.paths = append(target.paths[:idx], target.paths[idx+1:]...)
		if target.cfg.Presence != nil {
			p.midi.LedOff(*target.cfg.Presence)
		}
		if target.cfg.Mute != nil {
			p.midi.LedOff(*target.cfg.Mute)
		}
	}
}

func (p *PulseAudioClient) DeviceMuteUpdated(path dbus.ObjectPath, mute bool) {
	if target, _ := p.findTargetByPath(path); target != nil {
		target.mute = mute
		if target.cfg.Mute != nil {
			p.midi.SetLed(*target.cfg.Mute, mute)
		}
	}
}

func (p *PulseAudioClient) StreamMuteUpdated(path dbus.ObjectPath, mute bool) {
	if target, _ := p.findTargetByPath(path); target != nil {
		target.mute = mute
		if target.cfg.Mute != nil {
			p.midi.SetLed(*target.cfg.Mute, mute)
		}
	}
}

func (p *PulseAudioClient) StreamVolumeUpdated(path dbus.ObjectPath, values []uint32) {
	// Workaround for Firefox bug that sets the volume to 100% when pausing
	// or seeking in an audio stream.
	// https://bugzilla.mozilla.org/show_bug.cgi?id=1422637
	if target, _ := p.findTargetByPath(path); target != nil {
		if target.cfg.Type == config.PlaybackStream && target.cfg.Name == "Firefox" {
			if values[0] == 65536 && target.volume < 1.0 {
				log.Debug().Msgf("fixing firefox volume for %v", path)
				obj := p.getObject(target.cfg.Type, path)
				if obj != nil {
					p.setVolume(obj, target.volume)
				}
			}
		}
	}
}

func (p *PulseAudioClient) findTargetByPath(path dbus.ObjectPath) (*PulseAudioTarget, int) {
	for i, target := range p.targets {
		for idx, targetPath := range target.paths {
			if targetPath == path {
				return &p.targets[i], idx
			}
		}
	}
	return nil, -1
}
