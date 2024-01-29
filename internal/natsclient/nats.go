package natsclient

import (
	"bufio"
	"os"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/c0deaddict/midimix/internal/config"
)

func Connect(clientName string, cfg config.NatsConfig) (*nats.Conn, error) {
	var opts []nats.Option
	if cfg.Username != "" {
		password, err := readPassword(cfg.PasswordFile)
		if err != nil {
			return nil, err
		}
		opts = append(opts, nats.UserInfo(cfg.Username, *password))
	}

	// Set the client name.
	opts = append(opts, nats.Name(clientName))

	// Try to reconnect every 2 seconds, forever.
	opts = append(opts, nats.MaxReconnects(-1))
	opts = append(opts, nats.ReconnectWait(2*time.Second))

	opts = append(opts, nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
		log.Info().Err(err).Msgf("nats got disconnected from %v", nc.ConnectedUrl())
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		log.Info().Msgf("nats got reconnected to %v", nc.ConnectedUrl())
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		log.Info().Err(nc.LastError()).Msg("nats connection closed")
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
		log.Warn().Msgf("permissions are too open on %s", passwordFile)
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
