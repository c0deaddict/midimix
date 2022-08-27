module github.com/c0deaddict/midimix

go 1.18

require (
	github.com/lawl/pulseaudio v0.0.0-20210928141934-ed754c0c6618
	github.com/lucasb-eyer/go-colorful v1.2.0
	github.com/mitchellh/mapstructure v1.4.1
	github.com/nats-io/nats.go v1.11.1-0.20210623165838-4b75fc59ae30
	github.com/rs/zerolog v1.23.0
	gitlab.com/gomidi/midi/v2 v2.0.21
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/nats-io/nats-server/v2 v2.3.4 // indirect
	github.com/nats-io/nkeys v0.3.0 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)

// to update: replace version with "master" and run "go mod download"
replace github.com/lawl/pulseaudio => github.com/c0deaddict/pulseaudio v0.0.0-20220826212152-fdaa260adcbf

// replace github.com/lawl/pulseaudio => ../pulseaudio
