package bindings

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/davecgh/go-spew/spew"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/jmc-audio/kitauth/auth"
	"github.com/jmc-audio/kitauth/consts"
	"github.com/jmc-audio/kitauth/log"
	"github.com/jmc-audio/kitauth/middleware"

	"github.com/gorilla/handlers"

	"github.com/gorilla/mux"
)

type Servicer interface {
	Run(context.Context, interface{}) (interface{}, error)
}

type Request struct {
	params map[string]string
}

type Response struct {
	Status string
}

func (r *Request) PrincipalToken() *string {
	if id, ok := r.params[consts.RequestPrincipalID]; ok {
		return &id
	}
	return nil
}

func decodeRequest(r *http.Request) (response interface{}, err error) {
	var (
		id string
		ok bool
	)
	urlParams := mux.Vars(r)

	if id, ok = urlParams[consts.RequestPrincipalID]; !ok {
		return nil, errors.New("No principal id in request")
	}
	return &Request{map[string]string{consts.RequestPrincipalID: id}}, nil
}

func encodeResponse(w http.ResponseWriter, i interface{}) error {
	w.Header().Add("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(i.(*Response))
}

type Endpoint struct{}

func NewEndpoint(ctx context.Context) Servicer {
	return &Endpoint{}
}

func (h *Endpoint) Run(ctx context.Context, i interface{}) (interface{}, error) {
	log.Debug(ctx, "ctx", spew.Sdump(ctx))
	return &Response{"OK"}, nil
}

func StartHTTPListener(root context.Context) {
	go func() {
		ctx, cancel := context.WithCancel(root)
		defer cancel()

		errc := ctx.Value(consts.ContextErrorChannel).(chan error)

		sessions := make(map[string]context.Context)
		mtx := &sync.Mutex{}

		ctx = context.WithValue(ctx, "sessions", &sessions)
		ctx = context.WithValue(ctx, "session.mtx", mtx)
		ctx = context.WithValue(ctx, "session.ttl", 5*time.Second)
		ctx = context.WithValue(ctx, "session.refresh", true)

		router := createRouter(ctx, NewEndpoint(ctx))
		errc <- http.ListenAndServe(":6502", handlers.CombinedLoggingHandler(os.Stderr, router))
	}()
}

func createRouter(ctx context.Context, endpoint Servicer) *mux.Router {
	router := mux.NewRouter()

	Authenticated := middleware.NewAuthenticator(
		func(p auth.Principal) bool { return p != nil && *p.PrincipalToken() == "1" },
		func(auth.Principal, []auth.Subject) []auth.Subject {
			return []auth.Subject{}
		}).Authenticated()

	router.Handle(fmt.Sprintf("/principal/{%s}", consts.RequestPrincipalID),
		kithttp.NewServer(
			ctx,
			Authenticated(endpoint.Run),
			decodeRequest,
			encodeResponse,
		)).Methods("GET")
	return router
}