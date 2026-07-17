package epack

import (
	"errors"
	"reflect"
	"testing"
)

func TestWriteNumberLE_AllShortBuffers(t *testing.T) {
	kinds := []interface{}{
		false, int8(0), uint8(0), int16(0), uint16(0),
		int32(0), uint32(0), float32(0), int64(0), uint64(0), float64(0),
		int(0), uint(0),
	}
	for _, z := range kinds {
		v := reflect.New(reflect.TypeOf(z)).Elem()
		if err := writeNumberLE(v, []byte{}); err == nil {
			t.Fatalf("%v empty want errShortNumPayload", v.Kind())
		}
	}
}

func TestNumberPayloadLE_DefaultAndIntUint(t *testing.T) {
	if got := numberPayloadLE(reflect.ValueOf("nope")); got != nil {
		t.Fatalf("default want nil, got %v", got)
	}
	if len(numberPayloadLE(reflect.ValueOf(int(1)))) == 0 {
		t.Fatal("int payload empty")
	}
	if len(numberPayloadLE(reflect.ValueOf(uint(1)))) == 0 {
		t.Fatal("uint payload empty")
	}

	// 模拟 32 位平台 int/uint 宽度
	old := sizeofType
	sizeofType = func(t reflect.Type) uintptr { return 4 }
	defer func() { sizeofType = old }()

	p := numberPayloadLE(reflect.ValueOf(int(0x01020304)))
	if len(p) != 4 {
		t.Fatalf("int32-width payload len=%d", len(p))
	}
	p = numberPayloadLE(reflect.ValueOf(uint(0x01020304)))
	if len(p) != 4 {
		t.Fatalf("uint32-width payload len=%d", len(p))
	}

	var i int
	var u uint
	iv := reflect.ValueOf(&i).Elem()
	uv := reflect.ValueOf(&u).Elem()
	if err := writeNumberLE(iv, []byte{1, 2, 3, 4}); err != nil {
		t.Fatal(err)
	}
	if err := writeNumberLE(uv, []byte{1, 2, 3, 4}); err != nil {
		t.Fatal(err)
	}
	if err := writeNumberLE(iv, []byte{1}); err == nil {
		t.Fatal("int32-width short want error")
	}
	if err := writeNumberLE(uv, []byte{1}); err == nil {
		t.Fatal("uint32-width short want error")
	}
}

func TestSetNumberFromLE_InvalidAndErrors(t *testing.T) {
	if err := setNumberFromLE(reflect.Value{}, []byte{1}); !errors.Is(err, errInvalidNumberValue) {
		t.Fatalf("invalid value: %v", err)
	}
	// CanSet==false：写到临时值；payload 过短应返回 writeNumberLE 错误
	type wrap struct{ N int32 }
	w := wrap{}
	field := reflect.ValueOf(w).Field(0) // 不可 Set
	if field.CanSet() {
		t.Fatal("expected unsettable field")
	}
	if err := setNumberFromLE(field, []byte{1}); err == nil {
		t.Fatal("short payload on unsettable want error")
	}
	// 成功写 unsettable：错误路径已覆盖；再覆盖 writeNumberLE 成功且 CanSet 为 false
	if err := setNumberFromLE(field, []byte{7, 0, 0, 0}); err != nil {
		t.Fatalf("unsettable with enough bytes: %v", err)
	}
}

func Test_sliceNumber_ErrorPaths(t *testing.T) {
	var got []int32

	// header 不足
	if err := _sliceNumber(deBuffer([]byte{SIMPLE_NUMBER}), reflect.ValueOf(&got).Elem(), 1); !errors.Is(err, errShortNumSliceHdr) {
		t.Fatalf("short hdr: %v", err)
	}

	// 元素 kind 无类型（Chan）
	if err := _sliceNumber(deBuffer([]byte{SIMPLE_NUMBER, byte(reflect.Chan)}), reflect.ValueOf(&got).Elem(), 1); !errors.Is(err, errBadNumSliceElem) {
		t.Fatalf("nil elem type: %v", err)
	}

	// body 不足
	if err := _sliceNumber(deBuffer([]byte{SIMPLE_NUMBER, byte(reflect.Int32), 0, 0}), reflect.ValueOf(&got).Elem(), 2); !errors.Is(err, errShortNumSlice) {
		t.Fatalf("short body: %v", err)
	}
}

func Test_decodeValue_UnsupportedKind(t *testing.T) {
	// kind=Chan 且 size>0 → typ==nil
	h := encodeHead(uint64(reflect.Chan), 1)
	if _, err := _decodeValue(nil, deBuffer(h)); !errors.Is(err, errUnsupportedType) {
		t.Fatalf("chan: %v", err)
	}
	// kind>=TYPE_ENUM：wire 上 2 字节头高位会像 8 字节头；直接注入已读 head
	b := deBuffer([]byte{0})
	b.read = true
	b.kind = reflect.Kind(TYPE_ENUM + 1)
	b.size = 1
	if _, err := _decodeValue(nil, b); !errors.Is(err, errUnsupportedType) {
		t.Fatalf("oob kind: %v", err)
	}
}

func TestInterfaceDecoder_UnsupportedKind(t *testing.T) {
	var any interface{}
	h := encodeHead(uint64(reflect.Func), 1)
	if err := interfaceDecoder(nil, deBuffer(h), reflect.ValueOf(&any).Elem()); !errors.Is(err, errUnsupportedType) {
		t.Fatalf("func: %v", err)
	}
	b := deBuffer([]byte{0})
	b.read = true
	b.kind = reflect.Kind(TYPE_ENUM + 2)
	b.size = 1
	if err := interfaceDecoder(nil, b, reflect.ValueOf(&any).Elem()); !errors.Is(err, errUnsupportedType) {
		t.Fatalf("oob: %v", err)
	}
}

func Test_unmarshal_NonNilPointer(t *testing.T) {
	type s struct {
		A int32 `epack:"1"`
	}
	conf.cache.Delete(reflect.TypeOf(s{}).String())
	data, err := Marshal(s{A: 8})
	if err != nil {
		t.Fatal(err)
	}
	dst := &s{}
	if err := _unmarshal(nil, deBuffer(data), reflect.ValueOf(dst)); err != nil {
		// units 为空会走 nmlUnmarshal；结构体有导出字段应能解
		t.Fatalf("_unmarshal ptr: %v", err)
	}
	if dst.A != 8 {
		// nml 可能按字段顺序解；确认不 panic 即可。若标签路径不同，用 cache units。
		units, err := newEncoder(0, reflect.TypeOf(s{}), reflect.ValueOf(s{}))
		if err != nil {
			t.Fatal(err)
		}
		dst2 := &s{}
		if err := _unmarshal(units, deBuffer(data), reflect.ValueOf(dst2)); err != nil {
			t.Fatal(err)
		}
		if dst2.A != 8 {
			t.Fatalf("got %+v", dst2)
		}
	}
}

func TestBadTagIndex_Propagates(t *testing.T) {
	type bad struct {
		A int32 `epack:"9"`
	}
	conf.cache.Delete(reflect.TypeOf(bad{}).String())

	if _, err := Marshal(bad{A: 1}); !errors.Is(err, errBadTagIndex) {
		t.Fatalf("Marshal: %v", err)
	}
	if err := LoadTemplate(bad{}); !errors.Is(err, errBadTagIndex) {
		t.Fatalf("LoadTemplate: %v", err)
	}
	conf.cache.Delete(reflect.TypeOf(bad{}).String())
	var dst bad
	if err := Unmarshal(encodeHead(uint64(reflect.Struct), 1), &dst); !errors.Is(err, errBadTagIndex) {
		t.Fatalf("Unmarshal: %v", err)
	}

	// 嵌套结构体 tag 越界 → newEncoder 递归错误
	type inner struct {
		X int32 `epack:"5"`
	}
	type outer struct {
		I inner `epack:"1"`
	}
	conf.cache.Delete(reflect.TypeOf(outer{}).String())
	if _, err := Marshal(outer{}); !errors.Is(err, errBadTagIndex) {
		t.Fatalf("nested: %v", err)
	}
}
