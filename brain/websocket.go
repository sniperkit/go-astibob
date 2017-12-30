package astibrain

import (
	"context"
	"time"

	"encoding/json"
	"fmt"

	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

// Websocket event names
const (
	WebsocketEventNameAbilityCrashed = "ability.crashed"
	WebsocketEventNameAbilityStart   = "ability.start"
	WebsocketEventNameAbilityStarted = "ability.started"
	WebsocketEventNameAbilityStop    = "ability.stop"
	WebsocketEventNameAbilityStopped = "ability.stopped"
	WebsocketEventNameRegister       = "register"
)

// webSocket represents a websocket wrapper
type webSocket struct {
	abilities *abilities
	c         *astiws.Client
	o         WebSocketOptions
}

// WebSocketOptions are websocket options
type WebSocketOptions struct {
	URL string `toml:"url"`
}

// newWebSocket creates a new websocket wrapper
func newWebSocket(abilities *abilities, o WebSocketOptions) (ws *webSocket) {
	// Create websocket
	ws = &webSocket{
		abilities: abilities,
		c:         astiws.NewClient(4096),
		o:         o,
	}

	// TODO Add listeners
	return
}

// Close implements the io.Closer interface
func (ws *webSocket) Close() (err error) {
	// Close client
	astilog.Debug("astibrain: closing websocket client")
	if err = ws.c.Close(); err != nil {
		err = errors.Wrap(err, "astibrain: closing websocket client failed")
		return
	}
	return
}

// dial dials the websocket
func (ws *webSocket) dial(ctx context.Context, name string) {
	// Infinite loop to handle reconnect
	const sleepError = 5 * time.Second
	for {
		// Check context error
		if ctx.Err() != nil {
			return
		}

		// Dial
		if err := ws.c.Dial(ws.o.URL); err != nil {
			astilog.Error(errors.Wrap(err, "astibrain: dialing websocket failed"))
			time.Sleep(sleepError)
			continue
		}

		// Register
		if err := ws.sendRegister(name); err != nil {
			astilog.Error(errors.Wrap(err, "astibrain: sending register websocket event failed"))
			time.Sleep(sleepError)
			continue
		}

		// Read
		if err := ws.c.Read(); err != nil {
			astilog.Error(errors.Wrap(err, "astibrain: reading websocket failed"))
			time.Sleep(sleepError)
			continue
		}
	}
}

// WebSocketRegister is a websocket register payload
type WebSocketRegister struct {
	Abilities map[string]WebSocketAbility `json:"abilities"`
	Name      string                      `json:"name"`
}

// WebSocketAbility is a websocket ability
type WebSocketAbility struct {
	IsOn bool   `json:"is_on"`
	Name string `json:"name"`
}

// sendRegister sends a register event
func (ws *webSocket) sendRegister(name string) (err error) {
	// Create payload
	p := WebSocketRegister{
		Abilities: make(map[string]WebSocketAbility),
		Name:      name,
	}

	// Loop through abilities
	ws.abilities.abilities(func(a *ability) error {
		p.Abilities[a.name] = WebSocketAbility{
			IsOn: a.t.isOn(),
			Name: a.name,
		}
		return nil
	})

	// Write
	if err = ws.c.Write(WebsocketEventNameRegister, p); err != nil {
		err = errors.Wrapf(err, "astibrain: sending register event with payload %#v failed", p)
		return
	}
	return
}

// send sends an event and mutes the error (which is still logged)
func (ws *webSocket) send(eventName string, payload interface{}) {
	if err := ws.c.Write(eventName, payload); err != nil {
		astilog.Error(errors.Wrapf(err, "astibrain: sending %s websocket event with payload %#v failed", eventName, payload))
	}
}

// handleAbilityStart handles the websocket ability.start event
func (ws *webSocket) handleAbilityStart(c *astiws.Client, eventName string, payload json.RawMessage) (err error) {
	// Decode payload
	var name string
	if err = json.Unmarshal(payload, &name); err != nil {
		err = errors.Wrapf(err, "astibrain: json unmarshaling ability.start payload %#v failed", payload)
		return
	}

	// Retrieve ability
	a, ok := ws.abilities.ability(name)
	if !ok {
		err = fmt.Errorf("astibrain: unknown ability %s", name)
	}

	// Start ability
	a.on()
	return nil
}

// handleAbilityStop handles the websocket ability.stop event
func (ws *webSocket) handleAbilityStop(c *astiws.Client, eventName string, payload json.RawMessage) (err error) {
	// Decode payload
	var name string
	if err = json.Unmarshal(payload, &name); err != nil {
		err = errors.Wrapf(err, "astibrain: json unmarshaling ability.stop payload %#v failed", payload)
		return
	}

	// Retrieve ability
	a, ok := ws.abilities.ability(name)
	if !ok {
		err = fmt.Errorf("astibrain: unknown ability %s", name)
	}

	// Stop ability
	a.off()
	return nil
}
