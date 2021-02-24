package anserpc

import (
	"bytes"
	"context"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var (
	_contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	_errorType   = reflect.TypeOf((*error)(nil)).Elem()
)

type serviceRegistry struct {
	mu     sync.Mutex
	groups map[string]*group
}

func (s *serviceRegistry) modules() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	builder := strings.Builder{}

	i := 0
	names := make([]string, len(s.groups))
	for name := range s.groups {
		names[i] = name
		i++
	}

	sort.Strings(names)
	for _, name := range names {
		if name != "" {
			builder.WriteString("[Group=" + name + "]\n")
		} else {
			builder.WriteString("[Group]\n")
		}

		if s.groups[name] == nil || len(s.groups[name].services) == 0 {
			continue
		}

		for _, service := range s.groups[name].services {
			builder.WriteString(" service=")
			builder.WriteString(string(service.fingerprint()))
			builder.WriteString(" public=")
			builder.WriteString(strconv.FormatBool(service.public))
			builder.WriteString(" methods=")
			builder.WriteString(strings.Join(service.methods(), " "))
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

func newServiceRegistry() *serviceRegistry {
	sr := &serviceRegistry{
		groups: make(map[string]*group),
	}

	for _, api := range _builtInAPIs {
		sr.register(api)
	}

	return sr
}

func (s *serviceRegistry) register(api *API) {
	s.mu.Lock()
	defer s.mu.Unlock()

	srv, err := makeService(api.Service, api.Version,
		api.Public, reflect.ValueOf(api.Receiver))
	if err != nil {
		_xlog.Warn("Failed to register service", "group", api.Group,
			"service", api.Service, "service version", api.Version,
			"err", err)
		return
	}

	if _, ok := s.groups[api.Group]; !ok {
		s.groups[api.Group] = newGroup()
	}

	s.groups[api.Group].add(srv)
}

type group struct {
	services []*service
}

func newGroup() *group {
	return &group{
		services: make([]*service, 0),
	}
}

func (g *group) load(s *service) *service {
	i := sort.Search(len(g.services), func(i int) bool {
		return bytes.Compare(g.services[i].fingerprint(), s.fingerprint()) >= 0
	})

	return g.services[i]
}

func (g *group) add(s *service) {
	srvs := make([]*service, len(g.services))
	copy(srvs, g.services)
	i := sort.Search(len(srvs), func(i int) bool {
		return bytes.Compare(srvs[i].fingerprint(), s.fingerprint()) >= 0
	})

	if i < len(srvs) && bytes.Compare(srvs[i].fingerprint(), s.fingerprint()) == 0 {
		srvs[i] = s
	} else if i < len(g.services) {
		srvs = append(srvs, &service{})
		copy(srvs[i+1:], srvs[i:])
		srvs[i] = s
	} else {
		srvs = append(srvs, s)
	}

	g.services = srvs
}

type service struct {
	name      string
	version   string
	callbacks map[string]*callback
	public    bool
}

func (s service) fingerprint() []byte {
	return []byte(s.name + "_" + s.version)
}

func (s *service) methods() []string {
	i := 0
	names := make([]string, len(s.callbacks))
	for name := range s.callbacks {
		names[i] = name
		i++
	}

	return names
}

func makeService(name, version string, public bool, rcvr reflect.Value) (*service, error) {
	if name == "" || version == "" {
		return nil, _errServiceNameOrVersion
	}

	cbs, err := makeCallbacks(rcvr)
	if err != nil {
		return nil, err
	}

	return &service{
		name:      name,
		version:   version,
		callbacks: cbs,
		public:    public,
	}, nil
}

type callback struct {
	// function of method and receiver
	fn   reflect.Value
	rcvr reflect.Value

	//  args-in of method
	argTypes []reflect.Type

	// the first arg of method is ctx or not
	hasCtx bool

	// -1: no return value, 0: only error return, 1: result and error return
	returnType int
}

func makeCallbacks(rcvr reflect.Value) (map[string]*callback, error) {
	rType := rcvr.Type()
	numOfMethod := rType.NumMethod()

	cbs := make(map[string]*callback)
	for n := 0; n < numOfMethod; n++ {
		method := rType.Method(n)
		if method.PkgPath != "" { // not exported
			continue
		}

		cb, err := makeCallback(rcvr, method.Func)
		if err != nil {
			return nil, err
		}

		cbs[method.Name] = cb
	}

	if len(cbs) == 0 {
		return nil, _errMethodNotFound
	}

	return cbs, nil
}

func makeCallback(rcvr, fn reflect.Value) (*callback, error) {
	fnType := fn.Type()
	numOfIn := fnType.NumIn()

	start := 0
	if rcvr.IsValid() {
		start++
	}

	hasCtx := false
	if numOfIn > start && fnType.In(start) == _contextType {
		start++
		hasCtx = true
	}

	cb := &callback{
		rcvr:       rcvr,
		fn:         fn,
		argTypes:   make([]reflect.Type, numOfIn-start),
		hasCtx:     hasCtx,
		returnType: -1,
	}

	for n := start; n < numOfIn; n++ {
		cb.argTypes[n-start] = fnType.In(n)
	}

	numOfOut := fnType.NumOut()
	if numOfOut == 1 {
		if fnType.Out(0) != _errorType {
			return nil, _errResultErrorNotFound
		}

		cb.returnType = 0
	} else if numOfOut == 2 {
		if fnType.Out(1) != _errorType {
			return nil, _errResultErrorNotFound
		}

		cb.returnType = 1
	} else if numOfOut > 2 {
		return nil, _errNumOfResult
	}

	return cb, nil
}
