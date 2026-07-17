package epack

import (
	"reflect"
	"strconv"
	"time"
)

func encodeHead(t, s uint64) []byte {
	if s < 0x0400 { // 1024
		return new2bHead(t, s)
	}

	return new8bHead(t, s)
}

func _numberSlice(t reflect.Kind) bool {
	switch t {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool, reflect.Int, reflect.Uint:
		return true
	default:
		return false
	}
}

func numberEncoder(_ []*Unit, b *ubuffer, t reflect.Value) error {
	appendNumberHeadAndPayload(b, t)
	return nil
}

func stringEncoder(u []*Unit, b *ubuffer, t reflect.Value) error {
	s := t.String()

	// fmt.Println("stringEncoder", s, len(s), b.pos, encodeHead(uint64(reflect.String), uint64(len(s))))
	b.copyBytes(encodeHead(uint64(reflect.String), uint64(len(s))))
	b.copyBytes([]byte(s))
	return nil
}

// timeEncoder encode time.Time type
func timeEncoder(_ []*Unit, b *ubuffer, t reflect.Value) error {
	timestamp := t.Interface().(time.Time).UnixNano()
	appendNumberHeadAndPayload(b, reflect.ValueOf(timestamp))
	return nil
}

func interfaceEncoder(u []*Unit, b *ubuffer, t reflect.Value) error {
	if t.IsNil() {
		b.copyBytes(cacheTH(reflect.Interface))
		return nil
	}

	// fmt.Println("ENCODE :interfaceEncoder", t.Type().String(), len(u))

	e := t.Elem()
	return encoderFunc(e)(u, b, e)
}

func mapEncoder(u []*Unit, b *ubuffer, t reflect.Value) error {
	var err error

	keys := t.MapKeys()
	b.copyBytes(encodeHead(uint64(reflect.Map), uint64(len(keys))))
	if len(keys) == 0 {
		return nil
	}

	mt := t.Type()
	keyEnc := encoderFunc(reflect.Zero(mt.Key()))
	valEnc := encoderFunc(reflect.Zero(mt.Elem()))
	for _, key := range keys {
		val := t.MapIndex(key)
		if err = keyEnc(u, b, key); err != nil {
			return err
		}

		if err = valEnc(u, b, val); err != nil {
			return err
		}
	}

	return nil
}

func structEncoder(u []*Unit, b *ubuffer, t reflect.Value) error {
	// fmt.Println("structEncoder", t.Type().String(), len(u), uint64(t.NumField()))
	b.copyBytes(encodeHead(uint64(reflect.Struct), uint64(t.NumField())))
	return marshal(u, b, t)
}

func sliceEncoder(u []*Unit, b *ubuffer, t reflect.Value) error {
	if t.Kind() == reflect.Ptr && t.IsNil() {
		b.copyBytes(cacheTH(reflect.Slice))
		return nil
	}

	b.copyBytes(encodeHead(uint64(reflect.Slice), uint64(t.Len())))
	n := t.Len()
	if n == 0 {
		return nil
	}

	et := t.Type().Elem()
	ek := et.Kind()

	if _numberSlice(ek) {
		b.buffer = append(b.buffer, SIMPLE_NUMBER, byte(ek))
		// 本机 LE 且定宽数字：整段 copy；否则逐元素写小端载荷
		if canBulkNumberCopy(ek) {
			if raw, ok := numberSliceBytes(t); ok {
				b.buffer = append(b.buffer, raw...)
				return nil
			}
		}
		for i := 0; i < n; i++ {
			appendNumberPayloadLE(b, t.Index(i))
		}
		return nil
	}

	var err error
	for i := 0; i < n; i++ {
		v := t.Index(i)
		if err = encoderFunc(v)(u, b, v); err != nil {
			return err
		}
	}

	return nil
}

func arrayEncoder(u []*Unit, b *ubuffer, t reflect.Value) error {
	var err error
	b.copyBytes(encodeHead(uint64(reflect.Array), uint64(t.Len())))
	for i := 0; i < t.Len(); i++ {
		v := t.Index(i)
		if err = encoderFunc(v)(u, b, v); err != nil {
			return err
		}
	}

	return nil
}

func pointerEncoder(u []*Unit, b *ubuffer, t reflect.Value) error {
	if t.IsNil() {
		b.copyBytes(cacheTH(reflect.Pointer))
		return nil
	}

	return encoderFunc(t.Elem())(u, b, t.Elem())
}

func encoderFunc(t reflect.Value) coderFunc {
	// if t.Kind() == reflect.String && t.Type() != conf.rType[reflect.String] {
	// 	return stringEncoder
	// }

	if t.Type() == reflect.TypeOf(time.Time{}) {
		return timeEncoder
	}

	return conf.eFunc[t.Kind()]
}

// asStructValue 保证返回与 tp（struct）匹配的 struct Value。
// 对 []*T / *T 场景，Index(0)/Field 得到的是指针，直接 Field 会 panic。
func asStructValue(tp reflect.Type, vl reflect.Value) reflect.Value {
	for vl.IsValid() && vl.Kind() == reflect.Interface {
		if vl.IsNil() {
			return reflect.Zero(tp)
		}
		vl = vl.Elem()
	}
	for vl.IsValid() && vl.Kind() == reflect.Ptr {
		if vl.IsNil() {
			return reflect.Zero(tp)
		}
		vl = vl.Elem()
	}
	if !vl.IsValid() || vl.Kind() != reflect.Struct {
		return reflect.Zero(tp)
	}
	return vl
}

func newEncoder(level int, tp reflect.Type, vl reflect.Value) ([]*Unit, error) {
	vl = asStructValue(tp, vl)
	units := make([]*Unit, tp.NumField())
	for i := 0; i < tp.NumField(); i++ {
		field := tp.Field(i)
		idx, err := strconv.Atoi(field.Tag.Get("epack"))
		if err != nil || idx <= 0 {
			continue
		}

		if len(field.PkgPath) > 0 { // 不能导出的字段
			continue
		}

		u := &Unit{
			seq:     i,
			level:   level,
			name:    field.Name,
			kind:    field.Type.Kind(),
			encoder: encoderFunc(reflect.Zero(field.Type)),
			decoder: decoderFunc(reflect.Zero(field.Type)),
		}

		if u.encoder == nil || u.decoder == nil {
			continue
		}

		if u.kind == reflect.Struct {
			u.child, err = newEncoder(level+1, field.Type, vl.Field(i))
		} else if u.kind == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct {
			u.child, err = newEncoder(level+1, field.Type.Elem(), vl.Field(i))
		} else if e, ok := isSliceStruct(field.Type); ok {
			sliceVal := vl.Field(i)
			elemVal := reflect.Zero(e) // 默认零值
			if sliceVal.IsValid() && sliceVal.Len() > 0 {
				elemVal = sliceVal.Index(0) // 如果有元素，覆盖为第一个元素（可能是 *T）
			}
			u.child, err = newEncoder(level+1, e, elemVal)
		} else if u.kind == reflect.Interface && !vl.Field(i).IsNil() && vl.Field(i).Elem().Kind() == reflect.Struct {
			u.child, err = newEncoder(level+1, vl.Field(i).Elem().Type(), vl.Field(i).Elem())
		}

		if err != nil {
			return nil, err
		}

		if idx > tp.NumField() {
			return nil, errBadTagIndex
		}
		units[idx-1] = u
	}

	return units, nil
}

func LoadTemplate(obj ...interface{}) (err error) {
	for i := range obj {
		o := obj[i]
		if o == nil {
			return errObjectNil
		}

		tp := reflect.TypeOf(o)
		vl := reflect.ValueOf(o)
		if tp.Kind() == reflect.Ptr {
			tp = tp.Elem()
			vl = vl.Elem()
		}

		if tp.Kind() != reflect.Struct {
			return errObjectNotStruct
		}

		if _, exist := conf.cache.Load(tp.String()); exist {
			continue
		}

		e := new(EPack)
		if e.units, err = newEncoder(0, tp, vl); err != nil {
			return err
		}

		conf.cache.Store(tp.String(), e)
	}

	return nil
}
