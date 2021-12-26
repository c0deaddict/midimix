module github.com/c0deaddict/midimix

go 1.16

require (
	github.com/godbus/dbus v4.1.0+incompatible
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/lawl/pulseaudio v0.0.0-20210928141934-ed754c0c6618 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.4.1
	github.com/nats-io/nats-server/v2 v2.3.4 // indirect
	github.com/nats-io/nats.go v1.11.1-0.20210623165838-4b75fc59ae30
	github.com/rs/zerolog v1.23.0
	github.com/sqp/pulseaudio v0.0.0-20180916175200-29ac6bfa231c
	gitlab.com/gomidi/midi/v2 v2.0.21
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/lawl/pulseaudio => github.com/c0deaddict/pulseaudio v0.0.0-20211226174250-3b3e39eac5c0

// replace github.com/lawl/pulseaudio => ../pulseaudio
