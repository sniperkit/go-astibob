package astibob

import (
	"encoding/json"
	"net/http"

	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/http"
	"github.com/asticode/go-astiws"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

// brainsServer is a server for the brains
type brainsServer struct {
	*server
	brains *brains
}

// newBrainsServer creates a new brains server.
func newBrainsServer(brains *brains, o ServerOptions) (s *brainsServer) {
	// Create server
	s = &brainsServer{
		brains: brains,
		server: newServer("brains", o),
	}

	// Init router
	var r = httprouter.New()

	// Websocket
	r.GET("/websocket", s.handleWebsocketGET)

	// Chain middlewares
	var h = astihttp.ChainMiddlewares(r, astihttp.MiddlewareBasicAuth(o.Username, o.Password))

	// Set handler
	s.setHandler(h)
	return
}

// handleWebsocketGET handles the websockets.
func (s *brainsServer) handleWebsocketGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if err := s.ws.ServeHTTP(rw, r, s.adaptWebsocketClient); err != nil {
		astilog.Error(errors.Wrapf(err, "astibob: handling webrainsocket on %s failed", s.s.Addr))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// ClientAdapter returns the client adapter.
func (s *brainsServer) adaptWebsocketClient(c *astiws.Client) {
	// TODO Register on connect with brain's name
	c.AddListener(clientsWebsocketEventNamePing, func(c *astiws.Client, eventName string, payload json.RawMessage) error {
		return c.HandlePing()
	})
}
