package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/asticode/go-astibob/brain"
	"github.com/asticode/go-astibob/hearing"
	"github.com/asticode/go-astibob/portaudio"
	"github.com/asticode/go-astibob/speaking"
	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/config"
	"github.com/pkg/errors"
)

// Flags
var (
	ctx, cancel = context.WithCancel(context.Background())
	config      = flag.String("c", "", "the config path")
)

func main() {
	// Parse flags
	flag.Parse()
	astilog.FlagInit()

	// Init configuration
	c := newConfiguration()

	// Handle signals
	handleSignals()

	// Init portaudio
	p, err := astiportaudio.New()
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "astibrain: creating portaudio failed"))
	}
	defer p.Close()

	// Init portaudio stream
	s, err := p.NewDefaultStream(make([]int32, 192), c.PortAudio)
	if err != nil {
		astilog.Fatal(errors.Wrap(err, "astibrain: creating portaudio default stream failed"))
	}
	defer s.Close()

	// Init hearing
	hearing := astihearing.New(s, c.Hearing)

	// Init speaking
	speaking := astispeaking.New(c.Speaking)

	// Init brain
	brain := astibrain.New(c.Brain)
	defer brain.Close()

	// Learn abilities
	brain.Learn("Hearing", hearing, astibrain.AbilityOptions{AutoStart: c.AutoStart})
	brain.Learn("Speaking", speaking, astibrain.AbilityOptions{AutoStart: c.AutoStart})

	// Run the brain
	if err = brain.Run(ctx); err != nil {
		astilog.Fatal(errors.Wrap(err, "astibrain: running brain failed"))
	}
}

// Configuration represents a configuration
type Configuration struct {
	AutoStart bool                        `toml:"auto_start"`
	Brain     astibrain.Options           `toml:"brain"`
	Hearing   astihearing.Options         `toml:"hearing"`
	PortAudio astiportaudio.StreamOptions `toml:"portaudio"`
	Speaking  astispeaking.Options        `toml:"speaking"`
}

// newConfiguration creates a new configuration
func newConfiguration() *Configuration {
	// Global config
	gc := &Configuration{
		Brain: astibrain.Options{
			WebSocket: astibrain.WebSocketOptions{
				URL: "ws://127.0.0.1:6970/websocket",
			},
		},
		Hearing: astihearing.Options{
			WorkingDirectory: filepath.Join(os.TempDir(), "bob", "hearing"),
		},
		PortAudio: astiportaudio.StreamOptions{
			NumInputChannels: 1,
			SampleRate:       16000,
		},
		Speaking: astispeaking.Options{
			BinaryPath: "espeak",
		},
	}

	// TODO Flag config
	fc := &Configuration{}

	// Build configuration
	c, err := asticonfig.New(gc, *config, fc)
	if err != nil {
		astilog.Fatal(err)
	}
	return c.(*Configuration)
}

func handleSignals() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch)
	go func() {
		for s := range ch {
			astilog.Debugf("astibrain: received signal %s", s)
			if s == syscall.SIGABRT || s == syscall.SIGKILL || s == syscall.SIGINT || s == syscall.SIGQUIT || s == syscall.SIGTERM {
				cancel()
			}
		}
	}()
}
