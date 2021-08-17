package natsclient

import (
	"bufio"
	"log"
	"os"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/c0deaddict/midimix/internal/config"
)

func Connect(cfg config.NatsConfig) (*nats.Conn, error) {
	var opts []nats.Option
	if cfg.Username != "" {
		password, err := readPassword(cfg.PasswordFile)
		if err != nil {
			return nil, err
		}
		opts = append(opts, nats.UserInfo(cfg.Username, *password))
	}

	opts = append(opts, nats.ReconnectWait(2*time.Second))
	opts = append(opts, nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
		log.Printf("Nats got disconnected from %v. Reason: %q\n", nc.ConnectedUrl(), err)
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		log.Printf("Nats got reconnected to %v\n", nc.ConnectedUrl())
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		log.Printf("Nats connection closed. Reason: %q\n", nc.LastError())
	}))

	nc, err := nats.Connect(cfg.Url, opts...)
	if err != nil {
		return nil, err
	}

	return nc, nil
}

func readPassword(passwordFile string) (*string, error) {
	info, err := os.Stat(passwordFile)
	if err != nil {
		return nil, err
	}

	if info.Mode()&0o077 != 0 {
		log.Printf("Warning: permissions are too open on %v\n", passwordFile)
	}

	if password, err := readFirstLine(passwordFile); err != nil {
		return nil, err
	} else {
		return password, nil
	}
}

func readFirstLine(path string) (*string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan()
	line := strings.TrimSpace(scanner.Text())
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &line, nil
}
