package astibob

import (
	"context"
	"path/filepath"
	"text/template"

	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/template"
	"github.com/asticode/go-astiws"
	"github.com/pkg/errors"
)

// Bob is an object handling a collection of brains.
type Bob struct {
	brains        *brains
	brainsServer  *brainsServer
	cancel        context.CancelFunc
	clientsServer *clientsServer
	ctx           context.Context
	o             Options
}

// Options are Bob options.
type Options struct {
	BrainsServer       ServerOptions
	ClientsServer      ServerOptions
	ResourcesDirectory string
}

// New creates a new Bob.
func New(o Options) (b *Bob, err error) {
	// Create bob
	b = &Bob{
		brains: newBrains(),
		o:      o,
	}

	// Parse templates
	astilog.Debugf("astibob: parsing templates in %s", b.o.ResourcesDirectory)
	var t map[string]*template.Template
	if t, err = astitemplate.ParseDirectoryWithLayouts(filepath.Join(b.o.ResourcesDirectory, "templates", "pages"), filepath.Join(b.o.ResourcesDirectory, "templates", "layouts"), ".html"); err != nil {
		err = errors.Wrapf(err, "astibob: parsing templates in resources directory %s failed", b.o.ResourcesDirectory)
		return
	}

	// Create servers
	b.brainsServer = newBrainsServer(b.brains, o.BrainsServer)
	b.clientsServer = newClientsServer(t, b.brains, b.stop, o)
	return
}

// Close implements the io.Closer interface.
func (b *Bob) Close() (err error) {
	// Close brains server
	astilog.Debug("astibob: closing brains server")
	if err = b.brainsServer.Close(); err != nil {
		astilog.Error(errors.Wrap(err, "astibob: closing brains server failed"))
	}

	// Close clients server
	astilog.Debug("astibob: closing clients server")
	if err = b.clientsServer.Close(); err != nil {
		astilog.Error(errors.Wrap(err, "astibob: closing clients server failed"))
	}
	return
}

// Run runs Bob.
// This is cancellable through the ctx.
func (b *Bob) Run(ctx context.Context) (err error) {
	// Reset ctx
	b.ctx, b.cancel = context.WithCancel(ctx)
	defer b.cancel()

	// Run brains server
	var chanDone = make(chan error)
	go func() {
		if err := b.brainsServer.run(); err != nil {
			chanDone <- err
		}
	}()
	go func() {
		if err := b.clientsServer.run(); err != nil {
			chanDone <- err
		}
	}()

	// Wait for context or chanDone to be done
	select {
	case <-b.ctx.Done():
		if b.ctx.Err() != context.Canceled {
			err = errors.Wrap(err, "astibob: context error")
		}
		return
	case err = <-chanDone:
		if err != nil {
			err = errors.Wrap(err, "astibob: running servers failed")
		}
		return
	}
	return
}

// stop stops Bob
func (b *Bob) stop() {
	b.cancel()
}

// dispatchWsEvent dispatches a websocket event.
func dispatchWsEvent(ws *astiws.Manager, name string, payload interface{}) {
	ws.Loop(func(k interface{}, c *astiws.Client) {
		// Write
		if err := c.Write(name, payload); err != nil {
			astilog.Error(errors.Wrapf(err, "astibob: writing to ws client %v failed", k))
			return
		}
	})
}
