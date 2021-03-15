package anserpc

import (
	"encoding/json"

	"github.com/rcrowley/go-metrics"
)

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

func (s builtInService) Metrics() (string, error) {
	data, err := json.Marshal(metrics.DefaultRegistry.GetAll())
	if err != nil {
		_xlog.Debug("metrics error", "err", err)
		return "", _errJSONContent
	}

	return string(data), nil
}
