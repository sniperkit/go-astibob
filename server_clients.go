package astibob

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"text/template"

	"github.com/asticode/go-astilog"
	"github.com/asticode/go-astitools/http"
	"github.com/asticode/go-astiws"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

// Clients websocket events
const (
	clientsWebsocketEventNamePing = "ping"
)

// clientsServer is a server for the clients
type clientsServer struct {
	*server
	brains   *brains
	stopFunc func()
}

// newClientsServer creates a new clients server.
func newClientsServer(t map[string]*template.Template, brains *brains, stopFunc func(), o Options) (s *clientsServer) {
	// Create server
	s = &clientsServer{
		brains:   brains,
		server:   newServer("clients", o.ClientsServer),
		stopFunc: stopFunc,
	}

	// Init router
	var r = httprouter.New()

	// Static files
	r.ServeFiles("/static/*filepath", http.Dir(filepath.Join(o.ResourcesDirectory, "static")))

	// Web
	r.GET("/", s.handleHomepageGET)
	r.GET("/web/*page", astihttp.ChainRouterMiddlewares(
		s.handleWebGET(t),
		astihttp.RouterMiddlewareTimeout(o.ClientsServer.Timeout),
		astihttp.RouterMiddlewareContentType("text/html; charset=UTF-8"),
	))

	// Websockets
	r.GET("/websocket", s.handleWebsocketGET)

	// API
	r.GET("/api/bob", astihttp.ChainRouterMiddlewares(s.handleAPIBobGET, astihttp.RouterMiddlewareContentType("application/json")))
	r.GET("/api/bob/stop", s.handleAPIBobStopGET)
	r.GET("/api/references", astihttp.ChainRouterMiddlewares(s.handleAPIReferencesGET, astihttp.RouterMiddlewareContentType("application/json")))

	// Abilities
	// TODO
	/*
		b.abilities(func(a *ability) error {
			a.adaptRouter(r)
			return nil
		})
	*/

	// Chain middlewares
	var h = astihttp.ChainMiddlewares(r, astihttp.MiddlewareBasicAuth(o.ClientsServer.Username, o.ClientsServer.Password))

	// Set handler
	s.setHandler(h)
	return
}

// handleHomepageGET handles the homepage.
func (s *clientsServer) handleHomepageGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	http.Redirect(rw, r, "/web/index", http.StatusPermanentRedirect)
}

// handleWebGET handles the Web pages.
func (s *clientsServer) handleWebGET(t map[string]*template.Template) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		// Check if template exists
		var name = p.ByName("page") + ".html"
		if _, ok := t[name]; !ok {
			name = "/errors/404.html"
		}

		// Get data
		var code = http.StatusOK
		var data interface{}
		data = s.templateData(r, p, &name, &code)

		// Write header
		rw.WriteHeader(code)

		// Execute template
		if err := t[name].Execute(rw, data); err != nil {
			astilog.Error(errors.Wrapf(err, "astibob: executing %s template with data %#v failed", name, data))
			return
		}
	}
}

// templateData returns a template data.
func (s *clientsServer) templateData(r *http.Request, p httprouter.Params, name *string, code *int) (data interface{}) {
	// Switch on name
	switch *name {
	case "/errors/404.html":
		*code = http.StatusNotFound
	case "/index.html":
	}
	return
}

// handleWebsocketGET handles the websockets.
func (s *clientsServer) handleWebsocketGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if err := s.ws.ServeHTTP(rw, r, s.adaptWebsocketClient); err != nil {
		astilog.Error(errors.Wrapf(err, "astibob: handling websocket on %s failed", s.s.Addr))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// ClientAdapter returns the client adapter.
func (s *clientsServer) adaptWebsocketClient(c *astiws.Client) {
	// TODO Register on connect with brain's name
	c.AddListener(clientsWebsocketEventNamePing, func(c *astiws.Client, eventName string, payload json.RawMessage) error {
		return c.HandlePing()
	})
}

// APIError represents an API error.
type APIError struct {
	Message string `json:"message"`
}

// APIWriteError writes an API error
func APIWriteError(rw http.ResponseWriter, code int, err error) {
	rw.WriteHeader(code)
	astilog.Error(err)
	if err := json.NewEncoder(rw).Encode(APIError{Message: err.Error()}); err != nil {
		astilog.Error(errors.Wrap(err, "astibob: json encoding failed"))
	}
}

// APIWrite writes API data
func APIWrite(rw http.ResponseWriter, data interface{}) {
	if err := json.NewEncoder(rw).Encode(data); err != nil {
		APIWriteError(rw, http.StatusInternalServerError, errors.Wrap(err, "astibob: json encoding failed"))
		return
	}
}

// APIBob represents Bob.
type APIBob struct {
	Brains map[string]APIBrain `json:"brains,omitempty"`
}

// APIBrain represents a brain
type APIBrain struct {
	Abilities map[string]APIAbility `json:"abilities,omitempty"`
	Name      string                `json:"name"`
}

// APIAbility represents an ability.
type APIAbility struct {
	IsOn bool   `json:"is_on"`
	Name string `json:"name"`
}

// handleAPIBobGET returns Bob's information.
func (s *clientsServer) handleAPIBobGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// Init data
	d := APIBob{Brains: make(map[string]APIBrain)}

	// Loop through brains
	s.brains.brains(func(b *brain) error {
		// Init brain data
		bd := APIBrain{Name: b.name}

		// Loop through abilities
		b.abilities(func(a *ability) error {
			bd.Abilities[a.key] = APIAbility{
				IsOn: a.isOn,
				Name: a.name,
			}
			return nil
		})
		return nil
	})

	// Write
	APIWrite(rw, d)
}

// handleAPIBobStopGET stops Bob.
func (s *clientsServer) handleAPIBobStopGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	s.stopFunc()
}

// APIReferences represents the references.
type APIReferences struct {
	WsURL        string `json:"ws_url"`
	WsPingPeriod int    `json:"ws_ping_period"` // In seconds
}

// handleAPIReferencesGET returns the references.
func (s *clientsServer) handleAPIReferencesGET(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	APIWrite(rw, APIReferences{
		WsURL:        "ws://" + s.o.PublicAddr + "/websocket",
		WsPingPeriod: int(astiws.PingPeriod.Seconds()),
	})
}
