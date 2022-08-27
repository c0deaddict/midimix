package paclient

import (
	"time"

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
	cfg       config.PulseAudioTarget
	ids       []targetId
	mute      bool
	volume    float32
	channels  int
	isDefault bool
}

type PulseAudioClient struct {
	client  *pulseaudio.Client
	cfg     config.PulseAudioConfig
	targets []PulseAudioTarget
	midi    *midiclient.MidiClient
	updates <-chan pulseaudio.SubscriptionEvent
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

	// Refresh now and after 10 seconds. Midimix sometimes starts before the
	// PulseAudio API sees devices.
	pa.refreshAll()
	time.AfterFunc(10*time.Second, pa.refreshAll)

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
		if target.cfg.Default != nil {
			p.midi.LedOff(*target.cfg.Default)
		}
	}

	p.client.Close()
}

func (p *PulseAudioClient) Listen() {
	for event := range p.updates {
		var targetType config.PulseAudioTargetType
		switch event.Event & pulseaudio.EventFacilityMask {
		case pulseaudio.EventSink:
			targetType = config.Sink
		case pulseaudio.EventSource:
			targetType = config.Source
		case pulseaudio.EventSinkInput:
			targetType = config.PlaybackStream
		case pulseaudio.EventSourceOutput:
			targetType = config.RecordStream
		default:
			continue
		}

		// TODO: lock a mutex for concurrent targets access?
		switch event.Event & pulseaudio.EventTypeMask {
		case pulseaudio.EventTypeChange:
			p.refreshByIndex(event.Index, targetType)
		case pulseaudio.EventTypeRemove:
			target := p.removeTargetByIndex(event.Index, targetType)
			if target != nil {
				p.updateLedsForTarget(target)
			}
		case pulseaudio.EventTypeNew:
			obj := p.getInfo(event.Index, targetType)
			if target := p.lookup(obj); target != nil {
				log.Info().Msgf("new target: (%s) %s", target.cfg.Type, target.cfg.Name)
				target.refresh(obj)
				p.updateLedsForTarget(target)
			}
		}
	}
}

func (p *PulseAudioClient) refreshAll() {
	sinks, err := p.client.Sinks()
	if err != nil {
		log.Error().Err(err).Msg("list sinks")
	} else {
		for _, sink := range sinks {
			p.lookupAndRefresh(sink)
		}
	}

	sources, err := p.client.Sources()
	if err != nil {
		log.Error().Err(err).Msg("list sources")
	} else {
		for _, source := range sources {
			p.lookupAndRefresh(source)
		}
	}

	sinkInputs, err := p.client.SinkInputs()
	if err != nil {
		log.Error().Err(err).Msg("list sink inputs")
	} else {
		for _, sinkInput := range sinkInputs {
			p.lookupAndRefresh(sinkInput)
		}
	}

	sourceOutputs, err := p.client.SourceOutputs()
	if err != nil {
		log.Error().Err(err).Msg("list source outputs")
	} else {
		for _, sourceOutput := range sourceOutputs {
			p.lookupAndRefresh(sourceOutput)
		}
	}

	server, err := p.client.ServerInfo()
	if err != nil {
		log.Error().Err(err).Msg("get server info")
	} else {
		for i, target := range p.targets {
			if target.cfg.Type == config.Sink {
				p.targets[i].isDefault = target.matchName(server.DefaultSink)
			} else if target.cfg.Type == config.Source {
				p.targets[i].isDefault = target.matchName(server.DefaultSource)
			}
		}
	}

	for i := range p.targets {
		p.updateLedsForTarget(&p.targets[i])
	}
}

func (p *PulseAudioClient) lookup(object interface{}) *PulseAudioTarget {
	var desc string
	var targetType config.PulseAudioTargetType

	switch obj := object.(type) {
	case pulseaudio.Sink:
		targetType = config.Sink
		desc = obj.Name
		if value, ok := obj.PropList["device.description"]; ok {
			desc = value
		}
	case pulseaudio.Source:
		targetType = config.Source
		desc = obj.Name
		if value, ok := obj.PropList["device.description"]; ok {
			desc = value
		}
	case pulseaudio.SinkInput:
		targetType = config.PlaybackStream
		desc = obj.Name
		if value, ok := obj.PropList["application.name"]; ok {
			desc = value
		}
	case pulseaudio.SourceOutput:
		targetType = config.RecordStream
		desc = obj.Name
		if value, ok := obj.PropList["application.name"]; ok {
			desc = value
		}
	}

	return p.findTarget(desc, targetType)
}

func (p *PulseAudioClient) lookupAndRefresh(obj interface{}) {
	if t := p.lookup(obj); t != nil {
		t.refresh(obj)
	}
}

func (p *PulseAudioClient) getInfo(index uint32, targetType config.PulseAudioTargetType) interface{} {
	switch targetType {
	case config.Sink:
		sink, err := p.client.GetSinkInfo(index)
		if err != nil {
			log.Error().Err(err).Msg("refresh sink")
			return nil
		}
		return *sink

	case config.Source:
		source, err := p.client.GetSourceInfo(index)
		if err != nil {
			log.Error().Err(err).Msg("refresh source")
			return nil
		}
		return *source

	case config.PlaybackStream:
		sinkInput, err := p.client.GetSinkInputInfo(index)
		if err != nil {
			log.Error().Err(err).Msg("refresh sink input")
			return nil
		}
		return *sinkInput
	case config.RecordStream:
		sourceOutput, err := p.client.GetSourceOutputInfo(index)
		if err != nil {
			log.Error().Err(err).Msg("refresh source output")
			return nil
		}
		return *sourceOutput
	default:
		return nil
	}
}

func (p *PulseAudioClient) refreshByIndex(index uint32, targetType config.PulseAudioTargetType) *PulseAudioTarget {
	target := p.findTargetByIndex(index, targetType)
	if target == nil {
		return nil
	}

	obj := p.getInfo(index, targetType)
	target.refresh(obj)

	p.updateLedsForTarget(target)
	return target
}

func (p *PulseAudioClient) updateLedsForTarget(target *PulseAudioTarget) {
	if target.cfg.Default != nil {
		p.midi.SetLed(*target.cfg.Default, target.isDefault)
	}

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

func (p *PulseAudioClient) findTarget(description string, targetType config.PulseAudioTargetType) *PulseAudioTarget {
	for i, target := range p.targets {
		if target.cfg.Type == targetType && target.cfg.Name == description {
			return &p.targets[i]
		}
	}

	return nil
}

func (p *PulseAudioClient) findTargetByIndex(index uint32, targetType config.PulseAudioTargetType) *PulseAudioTarget {
	for i, target := range p.targets {
		if target.cfg.Type != targetType {
			continue
		}

		for _, id := range target.ids {
			if id.index == index {
				return &p.targets[i]
			}
		}
	}

	return nil
}

func (p *PulseAudioClient) removeTargetByIndex(index uint32, targetType config.PulseAudioTargetType) *PulseAudioTarget {
	for i, target := range p.targets {
		if target.cfg.Type != targetType {
			continue
		}

		t := &p.targets[i]
		for j, id := range t.ids {
			if id.index == index {
				t.ids = append(t.ids[:j], t.ids[j+1:]...)
				return t
			}
		}
	}

	return nil
}

func (p *PulseAudioClient) setDefault(target *PulseAudioTarget) {
	for i, other := range p.targets {
		if &other == target {
			continue
		}

		if other.cfg.Type == target.cfg.Type && p.targets[i].isDefault {
			p.targets[i].isDefault = false
			if other.cfg.Default != nil {
				p.midi.LedOff(*other.cfg.Default)
			}
		}
	}

	target.isDefault = true
	if target.cfg.Default != nil {
		p.midi.LedOn(*target.cfg.Default)
	}

	if target.cfg.Type == config.Sink {
		err := p.client.SetDefaultSink(target.name())
		if err != nil {
			log.Error().Err(err).Msgf("set default sink")
		}
	} else if target.cfg.Type == config.Source {
		err := p.client.SetDefaultSource(target.name())
		if err != nil {
			log.Error().Err(err).Msgf("set default source")
		}
	}
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

			if target.cfg.Default != nil && *target.cfg.Default == msg.Key {
				p.setDefault(&p.targets[i])
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

func (t *PulseAudioTarget) addId(index uint32, name string) {
	for _, id := range t.ids {
		if id.index == index {
			return
		}
	}

	t.ids = append(t.ids, targetId{index, name})
}

func (t *PulseAudioTarget) matchName(name string) bool {
	for _, id := range t.ids {
		if id.name == name {
			return true
		}
	}
	return false
}

func (t *PulseAudioTarget) name() string {
	if len(t.ids) == 0 {
		return ""
	} else {
		return t.ids[0].name
	}
}

func (t *PulseAudioTarget) refresh(object interface{}) {
	switch obj := object.(type) {
	case pulseaudio.Sink:
		t.addId(obj.Index, obj.Name)
		t.mute = obj.Muted
		t.channels = len(obj.ChannelMap)
		// t.volume = obj.Cvolume[0]
	case pulseaudio.Source:
		if obj.MonitorSourceName == "" {
			t.addId(obj.Index, obj.Name)
			t.mute = obj.Muted
			t.channels = len(obj.ChannelMap)
			// t.volume = obj.Cvolume[0]
		} else {
			log.Info().Msgf("ignoring monitor for source %s", t.cfg.Name)
		}
	case pulseaudio.SinkInput:
		t.addId(obj.Index, obj.Name)
		t.mute = obj.Muted
		t.channels = len(obj.ChannelMap)
		// t.volume = sinkInput.Cvolume[0]
	case pulseaudio.SourceOutput:
		t.addId(obj.Index, obj.Name)
		t.mute = obj.Muted
		t.channels = len(obj.ChannelMap)
		// t.volume = sourceOutput.Cvolume[0]
	}
}
