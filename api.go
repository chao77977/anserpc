package anserpc

type API struct {
	Group    string
	Service  string
	Version  string
	Receiver interface{}
	Public   bool
}

// built-in APIs
var _builtInAPIs = []*API{
	&API{
		Service:  "built-in",
		Version:  "1.0",
		Receiver: &builtInService{},
		Public:   true,
	},
}

type builtInService struct{}

func (s builtInService) Hello() (string, error) { return "olleh", nil }
