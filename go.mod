module github.com/c0deaddict/midimix

go 1.21

require (
	github.com/lawl/pulseaudio v0.0.0-00010101000000-000000000000
	github.com/lucasb-eyer/go-colorful v1.2.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/nats-io/nats.go v1.32.0
	github.com/rs/zerolog v1.31.0
	gitlab.com/gomidi/midi/v2 v2.0.30
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/klauspost/compress v1.17.2 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/nats-io/nkeys v0.4.7 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.18.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
)

// to update: replace version with "master" and run "go mod download"
replace github.com/lawl/pulseaudio => github.com/c0deaddict/pulseaudio v0.0.0-20220826212152-fdaa260adcbf

// replace github.com/lawl/pulseaudio => ../pulseaudio
