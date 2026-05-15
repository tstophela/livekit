package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/livekit/livekit-server/pkg/config"
	"github.com/livekit/livekit-server/pkg/logger"
	"github.com/livekit/livekit-server/pkg/service"
	"github.com/urfave/cli/v2"
)

func main() {
	// Seed random for various uses throughout the server
	rand.Seed(time.Now().UnixNano())

	app := &cli.App{
		Name:  "livekit-server",
		Usage: "High performance WebRTC server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Usage:   "path to LiveKit config file",
				EnvVars: []string{"LIVEKIT_CONFIG_FILE"},
			},
			&cli.StringFlag{
				Name:    "config-body",
				Usage:   "LiveKit config in YAML, read from stdin or env",
				EnvVars: []string{"LIVEKIT_CONFIG"},
			},
			&cli.StringFlag{
				Name:    "bind",
				Usage:   "address to bind to (default: 0.0.0.0)",
				EnvVars: []string{"LIVEKIT_BIND"},
			},
			&cli.StringFlag{
				Name:    "key-file",
				Usage:   "path to file that contains API keys/secrets",
				EnvVars: []string{"LIVEKIT_KEY_FILE"},
			},
			&cli.StringSliceFlag{
				Name:    "keys",
				Usage:   "api keys (key: secret)",
				EnvVars: []string{"LIVEKIT_KEYS"},
			},
			&cli.StringFlag{
				Name:    "node-ip",
				Usage:   "IP address of the node, used to advertise to other nodes",
				EnvVars: []string{"LIVEKIT_NODE_IP"},
			},
			&cli.StringFlag{
				Name:    "redis",
				Usage:   "redis address for distributed mode (host:port)",
				EnvVars: []string{"LIVEKIT_REDIS_HOST"},
			},
			&cli.BoolFlag{
				Name:  "dev",
				Usage: "run in development mode (insecure, disable TLS)",
			},
		},
		Action: startServer,
		Version: service.Version,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func startServer(c *cli.Context) error {
	// Initialize logger early so we can log configuration issues
	logger.InitFromConfig(nil, "info")

	conf, err := config.NewConfig(c.String("config"), c.String("config-body"), c)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Re-initialize logger with loaded config
	logger.InitFromConfig(&conf.Logging, conf.Development)

	server, err := service.InitializeServer(conf)
	if err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-sigChan
		logger.Infow("received signal, shutting down", "signal", sig)
		server.Stop(false)
	}()

	return server.Start()
}
