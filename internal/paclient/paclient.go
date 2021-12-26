package paclient

import (
	"github.com/lawl/pulseaudio"
	"github.com/rs/zerolog/log"

	"github.com/c0deaddict/midimix/internal/config"
	"github.com/c0deaddict/midimix/internal/midiclient"
)

type targetId struct {
	index uint32
	name  string
}

type PulseAudioTarget struct {
	cfg      config.PulseAudioTarget
	ids      []targetId
	mute     bool
	volume   float32
	channels int
}

type PulseAudioClient struct {
	client  *pulseaudio.Client
	cfg     config.PulseAudioConfig
	targets []PulseAudioTarget
	midi    *midiclient.MidiClient
	updates <-chan pulseaudio.Event
}

func Open(cfg config.PulseAudioConfig, midi *midiclient.MidiClient) (*PulseAudioClient, error) {
	client, err := pulseaudio.NewClient()
	if err != nil {
		return nil, err
	}

	updates, err := client.Updates()
	if err != nil {
		return nil, err
	}

	pa := PulseAudioClient{
		client:  client,
		cfg:     cfg,
		targets: make([]PulseAudioTarget, 0, len(cfg.Targets)),
		midi:    midi,
		updates: updates,
	}

	for _, targetCfg := range cfg.Targets {
		pa.targets = append(pa.targets, PulseAudioTarget{
			cfg:    targetCfg,
			ids:    make([]targetId, 0),
			mute:   false,
			volume: 1.0,
		})
	}

	pa.Refresh()

	return &pa, nil
}

func (p *PulseAudioClient) Close() {
	// Clear all leds.
	for _, target := range p.targets {
		if target.cfg.Mute != nil {
			p.midi.LedOff(*target.cfg.Mute)
		}
		if target.cfg.Presence != nil {
			p.midi.LedOff(*target.cfg.Presence)
		}
	}

	p.client.Close()
}

func (p *PulseAudioClient) Listen() {
	for event := range p.updates {
		facility := event & pulseaudio.EventFacilityMask
		if facility != pulseaudio.EventSink && facility != pulseaudio.EventSource && facility != pulseaudio.EventSinkInput &&
			facility != pulseaudio.EventSourceOutput {
			continue
		}

		eventType := event & pulseaudio.EventTypeMask
		if eventType == pulseaudio.EventTypeChange {
			log.Info().Msgf("change %d", facility)
			// TODO: debounce a refresh to get mute status from external changes?
		} else if eventType == pulseaudio.EventTypeNew || eventType == pulseaudio.EventTypeRemove {
			// TODO: could debounce this?
			log.Info().Msg("Pulseaudio state changed, refreshing")
			p.Refresh()
		}
	}
}

func (p *PulseAudioClient) Refresh() {
	for i, _ := range p.targets {
		p.targets[i].ids = make([]targetId, 0)
	}

	p.refreshSinks()
	p.refreshSources()
	p.refreshPlaybackStreams()
	p.refreshRecordStreams()
	p.updateLeds()
}

func (p *PulseAudioClient) refreshSinks() {
	sinks, err := p.client.Sinks()
	if err != nil {
		log.Error().Err(err).Msg("listing sinks failed")
		return
	}

	for _, sink := range sinks {
		desc := sink.Name
		if value, ok := sink.PropList["device.description"]; ok {
			desc = value
		}

		if target := p.findTarget(desc, config.Sink); target != nil {
			target.ids = append(target.ids, targetId{sink.Index, sink.Name})
			target.mute = sink.Muted
			target.channels = len(sink.ChannelMap)
			// target.volume = sink.Cvolume[0]
		}
	}
}

func (p *PulseAudioClient) refreshSources() {
	sources, err := p.client.Sources()
	if err != nil {
		log.Error().Err(err).Msg("listing sources failed")
		return
	}

	for _, source := range sources {
		desc := source.Name
		if value, ok := source.PropList["device.description"]; ok {
			desc = value
		}

		if target := p.findTarget(desc, config.Source); target != nil {
			target.ids = append(target.ids, targetId{source.Index, source.Name})
			target.mute = source.Muted
			target.channels = len(source.ChannelMap)
			// target.volume = sink.Cvolume[0]
		}
	}
}

func (p *PulseAudioClient) refreshPlaybackStreams() {
	sinkInputs, err := p.client.SinkInputs()
	if err != nil {
		log.Error().Err(err).Msg("listing playback streams failed")
		return
	}

	for _, sinkInput := range sinkInputs {
		desc := sinkInput.Name
		if value, ok := sinkInput.PropList["application.name"]; ok {
			desc = value
		}

		if target := p.findTarget(desc, config.PlaybackStream); target != nil {
			target.ids = append(target.ids, targetId{sinkInput.Index, sinkInput.Name})
			target.mute = sinkInput.Muted
			target.channels = len(sinkInput.ChannelMap)
			// target.volume = sink.Cvolume[0]
		}
	}
}

func (p *PulseAudioClient) refreshRecordStreams() {
	recordStreams, err := p.client.SourceOutputs()
	if err != nil {
		log.Error().Err(err).Msg("listing record streams")
		return
	}

	for _, recordStream := range recordStreams {
		desc := recordStream.Name
		if value, ok := recordStream.PropList["application.name"]; ok {
			desc = value
		}

		if target := p.findTarget(desc, config.RecordStream); target != nil {
			target.ids = append(target.ids, targetId{recordStream.Index, recordStream.Name})
			target.mute = recordStream.Muted
			target.channels = len(recordStream.ChannelMap)
			// target.volume = sink.Cvolume[0]
		}
	}
}

func (p *PulseAudioClient) updateLeds() {
	for _, target := range p.targets {
		if target.cfg.Presence != nil {
			p.midi.SetLed(*target.cfg.Presence, len(target.ids) != 0)
		}

		if target.cfg.Mute != nil {
			if len(target.ids) == 0 {
				p.midi.LedOff(*target.cfg.Mute)
			} else {
				p.midi.SetLed(*target.cfg.Mute, target.mute)
			}
		}
	}
}

func (p *PulseAudioClient) findTarget(description string, targetType config.PulseAudioTargetType) *PulseAudioTarget {
	for i, target := range p.targets {
		if target.cfg.Type == targetType && target.cfg.Name == description {
			return &p.targets[i]
		}
	}

	return nil
}

func (p *PulseAudioClient) OnMidiMessage(msg midiclient.MidiMessage) {
	switch msg := msg.(type) {
	case midiclient.MidiControlChange:
		for i, target := range p.targets {
			if target.cfg.Volume != nil && *target.cfg.Volume == msg.Key {
				volume := msg.Value
				p.targets[i].volume = volume
				for _, id := range target.ids {
					if err := p.setVolume(&target, id, volume); err != nil {
						log.Error().Err(err).Msgf("failed to set volume of %s %s", target.cfg.Type, id.name)
					}
				}
			}
		}

	case midiclient.MidiNoteOff:
		for i, target := range p.targets {
			if target.cfg.Mute != nil && *target.cfg.Mute == msg.Key {
				mute := !target.mute
				p.targets[i].mute = mute
				for _, id := range target.ids {
					if err := p.setMute(&target, id, mute); err != nil {
						log.Error().Err(err).Msgf("failed to set mute on %s %s", target.cfg.Type, id.name)
					} else {
						target.mute = mute
						if target.cfg.Mute != nil {
							p.midi.SetLed(*target.cfg.Mute, mute)
						}
					}
				}
			}
		}
	}
}

func (p *PulseAudioClient) setVolume(target *PulseAudioTarget, id targetId, volume float32) error {
	switch target.cfg.Type {
	case config.Sink:
		return p.client.SetSinkVolume(id.name, volume)
	case config.Source:
		return p.client.SetSourceVolume(id.name, volume)
	case config.PlaybackStream:
		return p.client.SetSinkInputVolume(id.index, volume)
	case config.RecordStream:
		return p.client.SetSourceOutputVolume(id.index, volume)
	default:
		return nil
	}
}

func (p *PulseAudioClient) setMute(target *PulseAudioTarget, id targetId, mute bool) error {
	switch target.cfg.Type {
	case config.Sink:
		return p.client.SetSinkMute(id.name, mute)
	case config.Source:
		return p.client.SetSourceMute(id.name, mute)
	case config.PlaybackStream:
		return p.client.SetSinkInputMute(id.index, mute)
	case config.RecordStream:
		return p.client.SetSourceOutputMute(id.index, mute)
	default:
		return nil
	}
}

// func (p *PulseAudioClient) NewPlaybackStream(path dbus.ObjectPath) {
// 	log.Info().Msgf("playback stream added: %v", path)
//
// 	targetType := config.PlaybackStream
// 	obj := p.getObject(targetType, path)
// 	if target := p.matchTarget(targetType, obj); target != nil {
// 		log.Info().Msgf("setting mute=%v volume=%v on %v", target.mute, target.volume, path)
// 		target.paths = append(target.paths, path)
// 		p.setMute(obj, target.mute)
// 		if err := p.setVolume(obj, target.volume); err != nil {
// 			log.Warn().Err(err).Msgf("failed to set volume of %v", path)
// 		}
// 		if target.cfg.Presence != nil {
// 			p.midi.LedOn(*target.cfg.Presence)
// 		}
// 	}
// }
//
// func (p *PulseAudioClient) StreamVolumeUpdated(path dbus.ObjectPath, values []uint32) {
// 	// Workaround for Firefox bug that sets the volume to 100% when pausing
// 	// or seeking in an audio stream.
// 	// https://bugzilla.mozilla.org/show_bug.cgi?id=1422637
// 	if target, _ := p.findTargetByPath(path); target != nil {
// 		if target.cfg.Type == config.PlaybackStream && target.cfg.Name == "Firefox" {
// 			if values[0] == 65536 && values[1] == 65536 && target.volume < 1.0 {
// 				log.Debug().Msgf("fixing firefox volume for %v", path)
// 				obj := p.getObject(target.cfg.Type, path)
// 				if obj != nil {
// 					p.setVolume(obj, target.volume)
// 				}
// 			}
// 		}
// 	}
// }
