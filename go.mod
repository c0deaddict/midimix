module github.com/c0deaddict/midimix

go 1.18

require (
	github.com/lawl/pulseaudio v0.0.0-20210928141934-ed754c0c6618
	github.com/lucasb-eyer/go-colorful v1.2.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/nats-io/nats.go v1.26.0
	github.com/rs/zerolog v1.29.1
	gitlab.com/gomidi/midi/v2 v2.0.30
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/klauspost/compress v1.16.5 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/nats-io/nats-server/v2 v2.9.17 // indirect
	github.com/nats-io/nkeys v0.4.4 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.8.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

// to update: replace version with "master" and run "go mod download"
replace github.com/lawl/pulseaudio => github.com/c0deaddict/pulseaudio v0.0.0-20220826212152-fdaa260adcbf

// replace github.com/lawl/pulseaudio => ../pulseaudio
