module github.com/c0deaddict/midimix

go 1.16

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/lawl/pulseaudio v0.0.0-20210928141934-ed754c0c6618
	github.com/lucasb-eyer/go-colorful v1.2.0
	github.com/mitchellh/mapstructure v1.4.1
	github.com/nats-io/nats-server/v2 v2.3.4 // indirect
	github.com/nats-io/nats.go v1.11.1-0.20210623165838-4b75fc59ae30
	github.com/rs/zerolog v1.23.0
	gitlab.com/gomidi/midi/v2 v2.0.21
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/lawl/pulseaudio => github.com/c0deaddict/pulseaudio v0.0.0-20220728194319-dcd5883e0316

// replace github.com/lawl/pulseaudio => ../pulseaudio
