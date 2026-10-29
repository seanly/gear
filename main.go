package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/seanly/gear/internal/gear"
	"github.com/urfave/cli/v3"
)

// VERSION is set at build time via -ldflags "-X main.VERSION=...".
var VERSION = "0.0.0-dev"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cmd := &cli.Command{
		Name:                  "gear",
		Version:               VERSION,
		EnableShellCompletion: true,
		Usage:                 "Export a Dockerfile stage with docker buildx",
		Commands: []*cli.Command{
			{
				Name:      "build",
				Usage:     "Build a stage and export it to a directory",
				ArgsUsage: "[context]",
				Flags:     append(sharedFlags(), outputFlag()),
				Action: func(_ context.Context, c *cli.Command) error {
					return gear.Build(mustRunner(), request(c, false))
				},
			},
			{
				Name:      "debug",
				Usage:     "Build a stage and open a shell in it",
				ArgsUsage: "[command...]",
				Flags:     sharedFlags(),
				Action: func(_ context.Context, c *cli.Command) error {
					return gear.Debug(mustRunner(), request(c, true))
				},
			},
			{
				Name:  "show",
				Usage: "Show Dockerfile information",
				Commands: []*cli.Command{
					{
						Name:        "targets",
						Usage:       "List stages and stage dependencies",
						Flags:       []cli.Flag{fileFlag()},
						Description: "Examples:\n  gear show targets\n  gear show targets --file examples/Dockerfile",
						Action: func(_ context.Context, c *cli.Command) error {
							out, err := gear.ShowTargetsFile(c.String("file"))
							if err != nil {
								return err
							}
							fmt.Fprint(os.Stdout, out)
							return nil
						},
					},
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func fileFlag() *cli.StringFlag {
	return &cli.StringFlag{
		Name:    "file",
		Aliases: []string{"f"},
		Value:   gear.DefaultFile,
		Usage:   "Dockerfile to build",
	}
}

func sharedFlags() []cli.Flag {
	return []cli.Flag{
		fileFlag(),
		&cli.StringFlag{
			Name:  "target",
			Usage: "Dockerfile stage to build",
		},
		&cli.StringSliceFlag{
			Name:  "build-arg",
			Usage: "Build-time variable, repeatable (KEY=VALUE)",
		},
		&cli.StringFlag{
			Name:  "secret",
			Usage: "Secret to expose to the build (id=mysecret[,src=path])",
		},
		&cli.StringFlag{
			Name:  "cache-from",
			Usage: "Cache source passed to buildx",
		},
		&cli.StringFlag{
			Name:  "cache-to",
			Usage: "Cache destination passed to buildx",
		},
		&cli.StringFlag{
			Name:  "platform",
			Usage: "Target platform, for example linux/amd64",
		},
		&cli.StringFlag{
			Name:    "registry-mirror",
			Sources: cli.EnvVars("GEAR_REGISTRY_MIRROR"),
			Usage:   "Docker Hub mirror host used when the daemon has none, for example mirror.example.com",
		},
	}
}

func outputFlag() cli.Flag {
	return &cli.StringFlag{
		Name:    "output",
		Aliases: []string{"o"},
		Value:   gear.DefaultOutput,
		Usage:   "Directory that receives the exported stage",
	}
}

func request(c *cli.Command, debug bool) gear.Request {
	req := gear.Request{
		File:           c.String("file"),
		Target:         c.String("target"),
		BuildArgs:      c.StringSlice("build-arg"),
		Secret:         c.String("secret"),
		CacheFrom:      c.String("cache-from"),
		CacheTo:        c.String("cache-to"),
		Platform:       c.String("platform"),
		RegistryMirror: c.String("registry-mirror"),
		TTY:            isatty.IsTerminal(os.Stdout.Fd()),
	}
	if debug {
		req.Command = c.Args().Slice()
		return req
	}
	req.Output = c.String("output")
	if c.Args().Len() > 0 {
		req.Context = c.Args().First()
	}
	return req
}

func mustRunner() gear.Runner {
	r, err := gear.NewDockerRunner()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return r
}
