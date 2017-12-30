package astibrain

import (
	"context"

	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

// Runner represents an object capable of running.
type Runner interface {
	Run(ctx context.Context) error
}

// Initializer represents an object capable of initializing itself
type Initializer interface {
	Init() error
}

// AbilityOptions represents ability options
type AbilityOptions struct {
	AutoStart bool
}

// ability represents an ability.
type ability struct {
	name string
	o    AbilityOptions
	r    Runner
	t    *toggle
	ws   *webSocket
}

// newAbility creates a new ability.
func newAbility(name string, r Runner, ws *webSocket, o AbilityOptions) *ability {
	return &ability{
		name: name,
		o:    o,
		r:    r,
		t:    newToggle(r.Run),
		ws:   ws,
	}
}

// on switches the ability on.
func (a *ability) on() {
	// Ability is already on
	if a.t.isOn() {
		return
	}

	// Switch on
	astilog.Debugf("astibrain: switching %s on", a.name)
	a.t.on()

	// Wait for the end of execution in a go routine
	go func() {
		// Wait
		if err := a.t.wait(); err != nil && err != context.Canceled {
			// Log
			astilog.Error(errors.Wrapf(err, "astibrain: %s crashed", a.name))

			// Dispatch websocket event
			a.ws.send(WebsocketEventNameAbilityCrashed, a.name)
		} else {
			// Log
			astilog.Infof("astibrain: %s have been switched off", a.name)

			// Dispatch websocket event
			a.ws.send(WebsocketEventNameAbilityStopped, a.name)
		}
	}()

	// Log
	astilog.Infof("astibrain: %s have been switched on", a.name)

	// Dispatch websocket event
	a.ws.send(WebsocketEventNameAbilityStarted, a.name)
}

// off switches the ability off.
func (a *ability) off() {
	// Ability is already off
	if !a.t.isOn() {
		return
	}

	// Switch off
	astilog.Debugf("astibrain: switching %s off", a.name)
	a.t.off()

	// The rest is handled through the wait function
}
