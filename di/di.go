package di

import (
	"fmt"
	"reflect"
	"sort"
	"sync"
)

type Container struct {
	mu         sync.Mutex
	ctors      map[reflect.Type]any
	singletons map[reflect.Type]reflect.Value
}

func New() *Container {
	return &Container{
		ctors:      map[reflect.Type]any{},
		singletons: map[reflect.Type]reflect.Value{},
	}
}

func (c *Container) Provide(ctor any) error {
	fn := reflect.ValueOf(ctor)
	if fn.Kind() != reflect.Func {
		return fmt.Errorf("di: Provide expects a function, got %T", ctor)
	}
	if fn.Type().NumOut() == 0 {
		return fmt.Errorf("di: constructor %s does not return a value", fn.Type())
	}
	if fn.Type().NumOut() > 2 || (fn.Type().NumOut() == 2 && fn.Type().Out(1) != reflect.TypeOf((*error)(nil)).Elem()) {
		return fmt.Errorf("di: constructor %s: signature func(deps...) (T [, error])", fn.Type())
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ctors[fn.Type().Out(0)] = ctor
	return nil
}

func (c *Container) Resolve(t reflect.Type) (reflect.Value, error) {
	c.mu.Lock()
	if v, ok := c.singletons[t]; ok {
		c.mu.Unlock()
		return v, nil
	}
	ctor, ok := c.ctors[t]
	c.mu.Unlock()
	if !ok {
		return reflect.Value{}, fmt.Errorf("di: no constructor for %s", t)
	}

	fn := reflect.ValueOf(ctor)
	ft := fn.Type()
	in := make([]reflect.Value, ft.NumIn())
	for i := 0; i < ft.NumIn(); i++ {
		dep, err := c.Resolve(ft.In(i))
		if err != nil {
			return reflect.Value{}, fmt.Errorf("di: %s: %w", t, err)
		}
		in[i] = dep
	}

	out := fn.Call(in)
	val := out[0]
	if ft.NumOut() == 2 && !out[1].IsNil() {
		return reflect.Value{}, fmt.Errorf("di: %s: %w", t, out[1].Interface().(error))
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.singletons[t] = val
	return val, nil
}

func (c *Container) All() ([]reflect.Value, error) {
	c.mu.Lock()
	types := make([]reflect.Type, 0, len(c.ctors))
	for t := range c.ctors {
		types = append(types, t)
	}
	c.mu.Unlock()

	sort.Slice(types, func(i, j int) bool { return types[i].String() < types[j].String() })

	values := make([]reflect.Value, 0, len(types))
	for _, t := range types {
		v, err := c.Resolve(t)
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, nil
}
