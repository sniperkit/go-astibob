package astibrain

import (
	"context"
	"io"
	"time"

	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

// Brain is an object handling one or more abilities
type Brain struct {
	abilities *abilities
	cancel    context.CancelFunc
	ctx       context.Context
	o         Options
	ws        *webSocket
}

// Options are brain's options
type Options struct {
	Name      string           `toml:"name"`
	WebSocket WebSocketOptions `toml:"websocket"`
}

// New creates a new brain
func New(o Options) (b *Brain) {
	// Create brain
	b = &Brain{
		abilities: newAbilities(),
		o:         o,
	}

	// Add websocket
	b.ws = newWebSocket(b.abilities, o.WebSocket)
	return
}

// Close implements the io.Closer interface
func (b *Brain) Close() (err error) {
	// Close abilities
	b.abilities.abilities(func(a *ability) error {
		// Log
		astilog.Debugf("astibrain: closing ability %s", a.name)

		// Switch the ability off
		a.off()

		// Wait for the toggle to be switched off indicating that the ability is really switched off
		for {
			if !a.t.isOn() {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		// Close
		if v, ok := a.r.(io.Closer); ok {
			if err := v.Close(); err != nil {
				astilog.Error(errors.Wrapf(err, "astibrain: closing ability %s failed", a.name))
			}
		}
		return nil
	})

	// Close ws
	astilog.Debug("astibrain: closing websocket")
	if err = b.ws.Close(); err != nil {
		err = errors.Wrap(err, "astibrain: closing websocket failed")
		return
	}
	return
}

// Learn allows the brain to learn a new ability.
func (b *Brain) Learn(name string, r Runner, o AbilityOptions) {
	b.abilities.set(newAbility(name, r, b.ws, o))
}

// Run runs the brain
func (b *Brain) Run(ctx context.Context) (err error) {
	// Reset context
	b.ctx, b.cancel = context.WithCancel(ctx)
	defer b.cancel()

	// Dial
	go b.ws.dial(b.ctx, b.o.Name)

	// Loop through abilities
	if err = b.abilities.abilities(func(a *ability) (err error) {
		// Initialize
		if v, ok := a.r.(Initializer); ok {
			astilog.Debugf("astibrain: initializing %s", a.name)
			if err = v.Init(); err != nil {
				err = errors.Wrapf(err, "astibrain: initializing %s failed", a.name)
				return
			}

		}

		// Auto start
		if a.o.AutoStart {
			a.on()
		}
		return
	}); err != nil {
		err = errors.Wrap(err, "astibrain: initializing abilities failed")
		return
	}

	// Wait for context to be done
	<-b.ctx.Done()
	return
}
