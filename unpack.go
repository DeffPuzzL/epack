package epack

import (
	"reflect"
)

func nmlUnmarshal(b *ubuffer, val reflect.Value) error {
	var err error
	t := val.Type()
	for i := 0; i < val.NumField() && b.remain > 0; i++ {
		if len(t.Field(i).PkgPath) > 0 {
			continue
		}

		field := val.Field(i)
		if err = decoderFunc(field)(nil, b, field); err != nil {
			return err
		}
	}

	return nil
}

func cacheUnmarshal(u []*unit, b *ubuffer, val reflect.Value) error {
	var err error
	for i := 0; i < len(u) && b.remain > 0; i++ {
		if u[i] == nil {
			continue
		}

		unit := u[i]
		field := val.Field(unit.seq)

		if !field.CanSet() {
			continue
		}

		if field.Kind() == reflect.Ptr && field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}

		// print("unmarshal, %v, %v, %v, %v, %v\n", field.Type().String(), field.Kind(), unit.name, unit.kind, len(unit.child))
		if err = unit.decoder(unit.child, b, field); err != nil {
			return err
		}
	}

	return nil
}

func _unmarshal(u []*unit, b *ubuffer, val reflect.Value) error {
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return errPointerNil
		}

		val = val.Elem()
	}

	if len(u) == 0 && val.NumField() > 0 {
		return nmlUnmarshal(b, val)
	}

	return cacheUnmarshal(u, b, val)
}

func newUnmarshal(b *ubuffer, val reflect.Value) error {
	var err error
	e := new(ePack)
	e.units, err = newEncoder(0, val.Type(), val)
	if err != nil {
		return err
	}

	conf.cache.Store(val.Type().String(), e)
	return _unmarshal(e.units, b, val)
}

func Unmarshal(buffer []byte, obj interface{}) error {
	if obj == nil {
		return errObjectNil
	}

	b := deBuffer(buffer)
	val := reflect.ValueOf(obj)
	if val.Kind() != reflect.Ptr {
		return errObjectMustPointer
	}

	val = val.Elem()
	return unmarshal(b, val)
}

func unmarshal(b *ubuffer, val reflect.Value) error {
	var err error
	if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return errPointerNil
		}

		val = val.Elem()
		return unmarshal(b, val)
	} else if val.Kind() != reflect.Struct {
		if err = decoderFunc(val)(nil, b, val); err != nil {
			return err
		}

		return b.isOver()
	}

	v, exist := conf.cache.Load(val.Type().String())
	if !exist {
		err = newUnmarshal(b, val)
	} else {
		err = _unmarshal(v.(*ePack).units, b, val)
	}

	if err != nil {
		return err
	}

	return b.isOver()
}
