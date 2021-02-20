package anserpc

type API struct {
	Group   string
	Version string
	Service interface{}
	Public  bool
}

// built-in APIs
var (
	_apiSayHello = &API{
		Version: "1.0",
		Service: &sayHello{},
		Public:  true,
	}
)

// sayhello
type sayHello struct{}
