package tool

type Func func(args []string) error

var registry = make(map[string]Func)

func Register(name string, fn Func) {
	registry[name] = fn
}

func All() map[string]Func {
	return registry
}
