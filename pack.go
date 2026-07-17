package epack

import (
	"reflect"
)

func nmlMarshal(b *ubuffer, val reflect.Value) error {
	var err error
	t := val.Type()
	for i := 0; i < val.NumField(); i++ {
		if len(t.Field(i).PkgPath) > 0 {
			continue
		}

		field := val.Field(i)
		if err = encoderFunc(field)(nil, b, field); err != nil {
			return err
		}
	}

	return nil
}

func cacheMarshal(u []*Unit, b *ubuffer, val reflect.Value) error {
	var err error
	for i := 0; i < len(u); i++ {
		if u[i] == nil {
			continue
		}

		unit := u[i]
		field := val.Field(unit.seq)

		if err = unit.encoder(unit.child, b, field); err != nil {
			return err
		}
	}

	return nil
}

func marshal(u []*Unit, b *ubuffer, val reflect.Value) error {
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			b.buffer = append(b.buffer, encodeHead(uint64(reflect.Pointer), 0)...)
			return nil
		}

		val = val.Elem()
	}

	if len(u) == 0 && val.NumField() > 0 {
		return nmlMarshal(b, val)
	}

	return cacheMarshal(u, b, val)
}

func newMarshal(val reflect.Value, b *ubuffer) error {
	var err error
	e := new(EPack)
	e.units, err = newEncoder(0, val.Type(), val)
	if err != nil {
		return err
	}

	conf.cache.Store(val.Type().String(), e)

	return marshal(e.units, b, val)
}

func Marshal(obj interface{}) ([]byte, error) {
	if obj == nil {
		return nil, errObjectNil
	}

	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if !val.IsValid() {
		return nil, errElementNil
	}

	b := enBuffer()
	defer b.release()

	var err error
	if val.Kind() != reflect.Struct {
		err = encoderFunc(val)(nil, b, val)
	} else {
		v, exist := conf.cache.Load(val.Type().String())
		if !exist {
			err = newMarshal(val, b)
		} else {
			e := v.(*EPack)
			err = marshal(e.units, b, val)
		}
	}
	if err != nil {
		return nil, err
	}

	out := make([]byte, len(b.buffer))
	copy(out, b.buffer)

	return out, nil
}
