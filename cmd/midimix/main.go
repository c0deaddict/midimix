package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/c0deaddict/midimix/internal/config"
	"github.com/c0deaddict/midimix/internal/midiclient"
	"gitlab.com/gomidi/midi"
)

var configFile = flag.String("config", "$HOME/.config/midimix/config.yaml", "Config file")

func main() {
	flag.Parse()

	cfg, err := config.Read(os.ExpandEnv(*configFile))
	if err != nil {
		log.Fatalln("Failed to read config:", err)
	}

	mc, err := midiclient.Open(cfg.Midi)
	if err != nil {
		log.Fatalln("Midi:", err)
	}
	defer mc.Close()
	defer log.Println("test")

	ch := make(chan midi.Message)
	if err := mc.Listen(ch); err != nil {
		log.Fatalln("Midi:", err)
	}

	// nc, err := natsclient.Connect(cfg.Nats)
	// if err != nil {
	// 	log.Fatalln("Nats:", err)
	// }
	// defer nc.Close()

	go func() {
		for msg := range ch {
			log.Println(msg)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
}
