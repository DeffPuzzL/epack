package epack

import (
	"encoding/binary"
	"io"
	"reflect"
	"time"
)

func (b *ubuffer) readHead() error {
	if b.read {
		return nil
	}

	if err := b.decodeHead(); err != nil {
		return err
	}

	b.read = true
	return nil
}

func (b *ubuffer) decodeHead() error {
	if b.read {
		// print("byte:%v\n", b.buffer[b.pos-2:b.pos])
		b.read = false
		return nil
	}

	if b.remain <= 0 {
		return io.ErrUnexpectedEOF
	}

	if b.buffer[b.pos]&0x80 == 0x80 { // 小端：首字节 bit7=1 表示 8 字节头
		if b.remain < 8 {
			return errShortHead
		}

		head := binary.LittleEndian.Uint64(b.buffer[b.pos : b.pos+8])
		b.walk(8)
		b.kind = reflect.Kind(head & 0x1F)
		b.size = head >> 8
		return nil
	}

	if b.remain < 2 {
		return errShortHead
	}

	head := binary.LittleEndian.Uint16(b.buffer[b.pos : b.pos+2])
	b.walk(2)

	b.kind = reflect.Kind(head & 0x1F)
	b.size = ((uint64(head>>5) & 0x3) << 8) | uint64(head>>8)
	return nil
}

func _arrayInterface(u []*Unit, b *ubuffer) (reflect.Value, error) {
	if err := b.decodeHead(); err != nil {
		return reflect.Value{}, err
	}

	if b.size == 0 {
		return reflect.Zero(reflect.TypeOf([]interface{}{})), nil
	}

	ks := int(b.size)
	st := reflect.TypeOf([]interface{}{})
	ns := reflect.MakeSlice(st, ks, ks)

	for i := 0; i < ks; i++ {
		elem, err := _decodeValue(u, b)
		if err != nil {
			return reflect.Value{}, err
		}

		ns.Index(i).Set(elem)
		// ns = reflect.Append(ns, elem)
	}

	return ns, nil
}

func emptyStruct(u []*Unit, b *ubuffer, t reflect.Value) error {
	// fmt.Println("emptyStruct", t.Type().String(), t.Elem(), t.Elem().Kind())

	st := reflect.TypeOf([]interface{}{})
	aVal := reflect.New(st).Elem()
	if err := sliceDecoder(u, b, aVal); err != nil {
		return err
	}

	t.Set(aVal)
	return nil
}

func stringDecoder(u []*Unit, b *ubuffer, t reflect.Value) error {
	// print("stringDecoder")
	if err := b.decodeHead(); err != nil {
		return err
	}

	// fmt.Println("stringEncoder", b.size)
	if b.size == 0 {
		return nil
	}

	if b.isExceeded() {
		return errShortString
	}

	str := string(b.buffer[b.pos : b.pos+b.size])
	b.walk(b.size)
	// print("\tstringDecoder, size:%d, Value:'%v'\n", b.size, str)

	t.SetString(str)
	return nil
}

// timeDecoder 专门处理 time.Time 类型
func timeDecoder(u []*Unit, b *ubuffer, t reflect.Value) error {
	if err := b.decodeHead(); err != nil {
		return err
	}

	if b.size != 8 {
		return errBadTimeSize
	}

	if b.isExceeded() {
		return errShortTime
	}

	timestamp := int64(binary.LittleEndian.Uint64(b.buffer[b.pos : b.pos+8]))
	b.walk(8)

	t.Set(reflect.ValueOf(time.Unix(0, timestamp)))
	return nil
}

func numberDecoder(_ []*Unit, b *ubuffer, t reflect.Value) error {
	if err := b.decodeHead(); err != nil {
		return err
	}

	if b.isExceeded() {
		return errShortNumber
	}

	buffer := b.buffer[b.pos : b.pos+b.size]
	b.walk(b.size)
	return setNumberFromLE(t, buffer)
}

func _sliceNumber(b *ubuffer, t reflect.Value, n int) error {
	if b.remain < 2 {
		return errShortNumSliceHdr
	}
	ek := reflect.Kind(b.buffer[b.pos+1])
	b.walk(2)

	et, err := conf.getType(ek)
	if err != nil {
		return err
	}
	if et == nil {
		return errBadNumSliceElem
	}

	es := uint64(et.Size())
	need := es * uint64(n)
	if b.remain < need {
		return errShortNumSlice
	}

	st := reflect.SliceOf(et)
	ns := reflect.MakeSlice(st, n, n)
	if n > 0 && canBulkNumberCopy(ek) {
		if dst, ok := numberSliceBytes(ns); ok {
			copy(dst, b.buffer[b.pos:b.pos+need])
			b.walk(need)
			t.Set(ns)
			return nil
		}
	}
	for i := 0; i < n; i++ {
		payload := b.buffer[b.pos : b.pos+es]
		if err := setNumberFromLE(ns.Index(i), payload); err != nil {
			return err
		}
		b.walk(es)
	}

	t.Set(ns)
	return nil
}

// decodeNumberSlice 解码 SIMPLE_NUMBER 数字切片。
// 目标元素为 interface{} 时先解到具体切片再装箱，避免 Set 具体类型 panic。
func decodeNumberSlice(b *ubuffer, t reflect.Value, n int) error {
	if t.Type().Elem().Kind() != reflect.Interface {
		return _sliceNumber(b, t, n)
	}

	return _sliceNumberToInterface(b, t, n)
}

func _sliceNumberToInterface(b *ubuffer, t reflect.Value, n int) error {
	if b.remain < 2 {
		return errShortNumSliceHdr
	}

	ek := reflect.Kind(b.buffer[b.pos+1])
	et, err := conf.getType(ek)
	if err != nil {
		return err
	}
	if et == nil {
		return errBadNumSliceElem
	}

	tmp := reflect.New(reflect.SliceOf(et)).Elem()
	if err := _sliceNumber(b, tmp, n); err != nil {
		return err
	}

	out := reflect.MakeSlice(t.Type(), tmp.Len(), tmp.Len())
	for i := 0; i < tmp.Len(); i++ {
		out.Index(i).Set(tmp.Index(i))
	}

	t.Set(out)
	return nil
}

func sliceDecoder(u []*Unit, b *ubuffer, t reflect.Value) error {
	if err := b.decodeHead(); err != nil {
		return err
	}

	if b.size == 0 {
		t.Set(reflect.Zero(t.Type()))
		return nil
	}

	if b.remain > 0 && b.buffer[b.pos] == SIMPLE_NUMBER {
		return decodeNumberSlice(b, t, int(b.size))
	}

	st := t.Type()
	ks := int(b.size)
	ns := reflect.MakeSlice(st, ks, ks)
	for i := 0; i < ks; i++ {
		e := ns.Index(i)
		if err := _decoder_func(e.Kind())(u, b, e); err != nil {
			return err
		}
	}

	t.Set(ns)
	return nil
}

func arrayDecoder(u []*Unit, b *ubuffer, t reflect.Value) error {
	if err := b.decodeHead(); err != nil {
		return err
	}

	n := int(b.size)
	if n != t.Len() {
		return errBadArrayLen
	}

	for i := 0; i < n; i++ {
		elem := t.Index(i)
		if err := decoderFunc(elem)(u, b, elem); err != nil {
			return err
		}
	}

	return nil
}

func mapDecoder(u []*Unit, b *ubuffer, t reflect.Value) error {
	if err := b.decodeHead(); err != nil {
		return err
	}

	var err error
	mapType := t.Type()
	keyType := mapType.Key()
	elemType := mapType.Elem()
	newMap := reflect.MakeMap(mapType)

	for i, j := 0, int(b.size); i < j; i++ {
		key := reflect.New(keyType).Elem()
		if err := decoderFunc(key)(u, b, key); err != nil {
			return err
		}

		var value reflect.Value
		if elemType.Kind() == reflect.Interface {
			if value, err = _decodeValue(u, b); err != nil {
				return err
			}
		} else {
			value = reflect.New(elemType).Elem()
			if err = decoderFunc(value)(u, b, value); err != nil {
				return err
			}
		}

		newMap.SetMapIndex(key, value)
	}

	t.Set(newMap)
	return nil
}

func _decodeValue(u []*Unit, b *ubuffer) (reflect.Value, error) {
	if err := b.readHead(); err != nil {
		return reflect.Value{}, err
	}

	if b.size == 0 {
		b.read = false
		return reflect.Zero(reflect.TypeOf((*interface{})(nil)).Elem()), nil
	}

	kind := reflect.Kind(b.kind)
	if kind == reflect.Array {
		return _arrayInterface(u, b)
	}

	var err error
	var typ reflect.Type
	if typ, err = conf.getType(kind); err != nil {
		return reflect.Value{}, err
	}
	if typ == nil {
		return reflect.Value{}, errUnsupportedType
	}

	decoder := _decoder_func(kind)
	if decoder == nil {
		return reflect.Value{}, errUnsupportedType
	}
	aVal := reflect.New(typ).Elem()

	if err := decoder(u, b, aVal); err != nil {
		return reflect.Value{}, err
	}

	return aVal, nil
}

func pointerDecoder(u []*Unit, b *ubuffer, t reflect.Value) error {
	if err := b.readHead(); err != nil {
		return err
	}

	if b.size == 0 {
		b.read = false
		t.Set(reflect.Zero(t.Type()))
		return nil
	}

	elemType := t.Type().Elem()
	elem := reflect.New(elemType).Elem()
	// fmt.Println("\tpointerDecoder", elem.String(), elem.Kind(), len(u))

	if err := decoderFunc(elem)(u, b, elem); err != nil {
		return err
	}

	t.Set(elem.Addr())
	return nil
}

func structDecoder(u []*Unit, b *ubuffer, t reflect.Value) error {
	if err := b.decodeHead(); err != nil {
		return err
	}

	return _unmarshal(u, b, t)
}

func interfaceDecoder(u []*Unit, b *ubuffer, t reflect.Value) error {
	if err := b.readHead(); err != nil {
		return err
	}

	if b.size == 0 {
		b.read = false
		t.Set(reflect.Zero(t.Type()))
		return nil
	}

	var err error
	var typ reflect.Type
	var aVal reflect.Value
	kind := reflect.Kind(b.kind)
	if kind == reflect.Array {
		if aVal, err = _arrayInterface(u, b); err != nil {
			return err
		}

		t.Set(aVal)
		return nil
	} else if kind == reflect.Struct {
		e := t.Elem()
		if e.Kind() == reflect.Invalid {
			return errStructInvalid
		}

		return structDecoder(u, b, e)
	}

	if typ, err = conf.getType(kind); err != nil {
		return err
	}
	if typ == nil {
		return errUnsupportedType
	}

	aVal = reflect.New(typ).Elem()
	if err := _decoder_func(kind)(u, b, aVal); err != nil {
		return err
	}

	t.Set(aVal)
	return nil
}

func decoderFunc(t reflect.Value) coderFunc {
	if t.Type() == reflect.TypeOf(time.Time{}) {
		return timeDecoder
	}

	return _decoder_func(t.Kind())
}

func _decoder_func(k reflect.Kind) coderFunc {
	return conf.dFunc[k]
}
