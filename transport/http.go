package transport

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/net/context"

	schemaregistry "github.com/chop-dbhi/schema-registry"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/julienschmidt/httprouter"
)

// Default request decoder is a no-op.
func defaultRequestDecoder(cxt context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

// Default response encoder encodes the value to JSON if not nil.
func defaultResponseEncoder(cxt context.Context, w http.ResponseWriter, v interface{}) error {
	var (
		status int
		body   interface{}
	)

	if x, ok := v.(*httpSuccess); ok {
		status = x.Status
		body = x.Body
	} else {
		status = http.StatusOK
		body = v
	}

	if body != nil {
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(status)
		return json.NewEncoder(w).Encode(body)
	}

	w.WriteHeader(status)
	return nil
}

type httpSuccess struct {
	Status int
	Body   interface{}
}

// An http error type that is used in the request/response cycle.
// Use this if an error can be mapped to an http error. If a body
// if present, it will be encoded as JSON.
type httpError struct {
	Status int
	Body   interface{}
}

func (e *httpError) Error() string {
	return fmt.Sprintf("%d: %s", e.Status, e.Body)
}

// Customer error encoder to map http errors.
func errorEncoder(cxt context.Context, err error, w http.ResponseWriter) {
	enc := json.NewEncoder(w)

	var (
		status int
		body   interface{}
	)

	// Unpack wrapped error.
	if e, ok := err.(httptransport.Error); ok {
		err = e.Err
	}

	if e, ok := err.(*httpError); ok {
		status = e.Status
		body = e.Body
	} else {
		switch err {
		case io.EOF:
			status = http.StatusBadRequest

		case schemaregistry.ErrSchemaDoesNotExist:
			status = http.StatusNotFound

		case schemaregistry.ErrSchemaExists:
			status = http.StatusConflict

		case schemaregistry.ErrSchemaIdRequired, schemaregistry.ErrSchemaTypeRequired, schemaregistry.ErrSchemaTypeUnknown:
			status = http.StatusUnprocessableEntity
			body = err

		default:
			status = http.StatusInternalServerError
			body = err
		}
	}

	if body != nil {
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(status)

		if err, ok := body.(error); ok {
			enc.Encode(map[string]string{
				"error": err.Error(),
			})

		} else {
			enc.Encode(body)
		}

		return
	}

	w.WriteHeader(status)
}

type createRequest struct {
	ID   string                 `json:"id"`
	Type string                 `json:"type"`
	Def  map[string]interface{} `json:"def"`
}

func decodeCreateRequest(cxt context.Context, r *http.Request) (interface{}, error) {
	var v createRequest
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}

func makeCreateEndpoint(s schemaregistry.Service) endpoint.Endpoint {
	return func(cxt context.Context, request interface{}) (interface{}, error) {
		req := request.(*createRequest)

		s, err := s.Create(req.ID, req.Type, req.Def)
		if err != nil {
			return nil, err
		}

		return s, nil
	}
}

func decodeUpdateRequest(cxt context.Context, r *http.Request) (interface{}, error) {
	p := r.Context().Value("params").(httprouter.Params)

	v := createRequest{
		ID: p.ByName("id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&v.Def); err != nil {
		return nil, err
	}

	return &v, nil
}

func makeUpdateEndpoint(s schemaregistry.Service) endpoint.Endpoint {
	return func(cxt context.Context, request interface{}) (interface{}, error) {
		req := request.(*createRequest)

		s, err := s.Update(req.ID, req.Def)
		if err != nil {
			return nil, err
		}

		return s, nil
	}
}

func encodeListResponse(cxt context.Context, w http.ResponseWriter, v interface{}) error {
	// Required to encode an empty array.
	if a, ok := v.([]string); ok {
		if len(a) == 0 {
			v = []string{}
		}
	}

	return defaultResponseEncoder(cxt, w, v)
}

func makeListEndpoint(s schemaregistry.Service) endpoint.Endpoint {
	return func(cxt context.Context, request interface{}) (interface{}, error) {
		return s.List()
	}
}

type getRequest struct {
	ID string
}

func decodeGetRequest(cxt context.Context, r *http.Request) (interface{}, error) {
	p := r.Context().Value("params").(httprouter.Params)
	return &getRequest{p.ByName("id")}, nil
}

func makeGetEndpoint(s schemaregistry.Service) endpoint.Endpoint {
	return func(cxt context.Context, request interface{}) (interface{}, error) {
		req := request.(*getRequest)
		return s.Get(req.ID)
	}
}

type validateRequest struct {
	ID   string
	Body io.ReadCloser
}

func decodeValidateRequest(cxt context.Context, r *http.Request) (interface{}, error) {
	p := r.Context().Value("params").(httprouter.Params)

	return &validateRequest{
		ID:   p.ByName("id"),
		Body: r.Body,
	}, nil
}

func encodeValidateResponse(cxt context.Context, w http.ResponseWriter, v interface{}) error {
	var a []interface{}

	if errs, ok := v.([]error); ok {
		for _, err := range errs {
			if _, ok := err.(json.Marshaler); ok {
				a = append(a, err)
			} else {
				a = append(a, err.Error())
			}
		}
	}

	if len(a) == 0 {
		a = []interface{}{}
	}

	return defaultResponseEncoder(cxt, w, a)
}

func makeValidateEndpoint(s schemaregistry.Service) endpoint.Endpoint {
	return func(cxt context.Context, request interface{}) (interface{}, error) {
		req := request.(*validateRequest)
		defer req.Body.Close()
		return s.Validate(req.ID, req.Body)
	}
}

func decodeDeleteRequest(cxt context.Context, r *http.Request) (interface{}, error) {
	p := r.Context().Value("params").(httprouter.Params)

	return &getRequest{
		ID: p.ByName("id"),
	}, nil
}

func makeDeleteEndpoint(s schemaregistry.Service) endpoint.Endpoint {
	return func(cxt context.Context, request interface{}) (interface{}, error) {
		req := request.(*getRequest)
		return nil, s.Delete(req.ID)
	}
}

// Wraps an standard handler with an httprouter handler
// passing in the params as a context value.
func routerWrapHandler(h http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		r = r.WithContext(context.WithValue(r.Context(), "params", p))
		h.ServeHTTP(w, r)
	}
}

func NewHTTP(cxt context.Context, svc schemaregistry.Service, logger log.Logger) http.Handler {
	m := httprouter.New()
	logger = log.NewContext(logger).With("transport", "http")

	m.POST("/schema", routerWrapHandler(httptransport.NewServer(
		cxt,
		endpointLoggingMiddleware(log.NewContext(logger).With("method", "create"))(makeCreateEndpoint(svc)),
		decodeCreateRequest,
		encodeListResponse,
		httptransport.ServerErrorEncoder(errorEncoder),
	)))

	m.GET("/schema", routerWrapHandler(httptransport.NewServer(
		cxt,
		endpointLoggingMiddleware(log.NewContext(logger).With("method", "list"))(makeListEndpoint(svc)),
		defaultRequestDecoder,
		encodeListResponse,
		httptransport.ServerErrorEncoder(errorEncoder),
	)))

	m.GET("/schema/:id", routerWrapHandler(httptransport.NewServer(
		cxt,
		endpointLoggingMiddleware(log.NewContext(logger).With("method", "get"))(makeGetEndpoint(svc)),
		decodeGetRequest,
		defaultResponseEncoder,
		httptransport.ServerErrorEncoder(errorEncoder),
	)))

	m.PUT("/schema/:id", routerWrapHandler(httptransport.NewServer(
		cxt,
		endpointLoggingMiddleware(log.NewContext(logger).With("method", "update"))(makeUpdateEndpoint(svc)),
		decodeUpdateRequest,
		defaultResponseEncoder,
		httptransport.ServerErrorEncoder(errorEncoder),
	)))

	m.DELETE("/schema/:id", routerWrapHandler(httptransport.NewServer(
		cxt,
		endpointLoggingMiddleware(log.NewContext(logger).With("method", "delete"))(makeDeleteEndpoint(svc)),
		decodeDeleteRequest,
		defaultResponseEncoder,
		httptransport.ServerErrorEncoder(errorEncoder),
	)))

	m.POST("/schema/:id", routerWrapHandler(httptransport.NewServer(
		cxt,
		endpointLoggingMiddleware(log.NewContext(logger).With("method", "validate"))(makeValidateEndpoint(svc)),
		decodeValidateRequest,
		encodeValidateResponse,
		httptransport.ServerErrorEncoder(errorEncoder),
	)))

	return m
}
