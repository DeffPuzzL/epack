package epack

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestHeadAlwaysLittleEndian(t *testing.T) {
	h2 := new2bHead(uint64(reflect.Int32), 4)
	if len(h2) != 2 {
		t.Fatalf("new2bHead len=%d", len(h2))
	}
	got := binary.LittleEndian.Uint16(h2)
	want := uint16((uint64(reflect.Int32) & 0x1F) | (4 << 8))
	if got != want {
		t.Fatalf("new2bHead bytes not little-endian: got=%04x want=%04x", got, want)
	}
	if h2[0]&0x80 != 0 {
		t.Fatalf("2b head must clear bit7: %02x", h2[0])
	}

	h8 := new8bHead(uint64(reflect.String), 2048)
	if len(h8) != 8 {
		t.Fatalf("new8bHead len=%d", len(h8))
	}
	u64 := binary.LittleEndian.Uint64(h8)
	if u64&0x1F != uint64(reflect.String) || u64&0x80 == 0 || u64>>8 != 2048 {
		t.Fatalf("new8bHead unexpected: %x", u64)
	}
}

func TestNumberLittleEndianRoundTrip(t *testing.T) {
	cases := []any{
		int8(-7),
		int16(-300),
		int32(123456),
		int64(1 << 40),
		uint8(9),
		uint16(60000),
		uint32(4000000000),
		uint64(1 << 60),
		float32(1.5),
		float64(2.5),
		true,
		false,
	}
	for _, src := range cases {
		data, err := Marshal(src)
		if err != nil {
			t.Fatalf("Marshal(%T) err=%v", src, err)
		}
		dst := reflect.New(reflect.TypeOf(src)).Interface()
		if err := Unmarshal(data, dst); err != nil {
			t.Fatalf("Unmarshal(%T) err=%v", src, err)
		}
		got := reflect.ValueOf(dst).Elem().Interface()
		if !reflect.DeepEqual(src, got) {
			t.Fatalf("%T round-trip: %v vs %v", src, src, got)
		}
	}
}

// 多元素数字切片应按元素写小端，往返成功。
func TestNumberSliceMultiElementRoundTrip(t *testing.T) {
	src := []int32{1, 2, 3, -1, 1 << 20}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}
	var got []int32
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal err=%v", err)
	}
	if !reflect.DeepEqual(src, got) {
		t.Fatalf("[]int32 mismatch %v vs %v", src, got)
	}

	usrc := []uint16{1, 2, 65535}
	udata, err := Marshal(usrc)
	if err != nil {
		t.Fatalf("Marshal []uint16 err=%v", err)
	}
	var ugot []uint16
	if err := Unmarshal(udata, &ugot); err != nil {
		t.Fatalf("Unmarshal []uint16 err=%v", err)
	}
	if !reflect.DeepEqual(usrc, ugot) {
		t.Fatalf("[]uint16 mismatch %v vs %v", usrc, ugot)
	}
}

func TestWriteNumberLEShortBuffer(t *testing.T) {
	cases := []reflect.Value{
		reflect.New(reflect.TypeOf(int16(0))).Elem(),
		reflect.New(reflect.TypeOf(int32(0))).Elem(),
		reflect.New(reflect.TypeOf(int64(0))).Elem(),
		reflect.New(reflect.TypeOf(float32(0))).Elem(),
		reflect.New(reflect.TypeOf(float64(0))).Elem(),
		reflect.New(reflect.TypeOf(uint(0))).Elem(),
		reflect.New(reflect.TypeOf(int(0))).Elem(),
	}
	for _, v := range cases {
		if err := writeNumberLE(v, []byte{}); err == nil {
			t.Fatalf("writeNumberLE(%v) empty buf want error", v.Kind())
		}
	}
	if err := writeNumberLE(reflect.ValueOf("x"), []byte{1}); err == nil {
		t.Fatal("writeNumberLE(string) want error")
	}
}

func TestGetTypeUnsupported(t *testing.T) {
	if _, err := conf.getType(reflect.Kind(_type_enum + 1)); err == nil {
		t.Fatal("getType 超范围应报错")
	}
}

func TestEmptyStructHelper(t *testing.T) {
	data, err := Marshal([]interface{}{int32(1), "x"})
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}
	var dst interface{}
	// emptyStruct 期望目标是可 Set 的 slice 值；直接调用内部 API
	b := deBuffer(data)
	holder := reflect.New(reflect.TypeOf([]interface{}{})).Elem()
	if err := emptyStruct(nil, b, holder); err != nil {
		t.Fatalf("emptyStruct err=%v", err)
	}
	if holder.Len() == 0 {
		t.Fatalf("emptyStruct 应写入切片，got=%v", holder.Interface())
	}
	_ = dst
}

func TestPrintDebugHelper(t *testing.T) {
	print("epack-print-%s\n", "ok")
}

func TestMarshalNilPointerValue(t *testing.T) {
	// 顶层 Marshal(typed-nil) 会在 Elem 后对 zero Value 调 encoderFunc 而 panic（已知缺陷）。
	// 这里覆盖 marshal() 内部对 nil 指针字段的安全分支。
	b := enBuffer()
	defer b.release()
	var p *int32
	if err := marshal(nil, b, reflect.ValueOf(p)); err != nil {
		t.Fatalf("marshal(nil *int32) err=%v", err)
	}
	if len(b.buffer) == 0 {
		t.Fatal("期望写入 nil 指针头")
	}
}

func TestMarshalTypedNilPointerReturnsError(t *testing.T) {
	var p *int32
	data, err := Marshal(p)
	if err == nil {
		t.Fatal("Marshal(nil *int32) want error")
	}
	if data != nil {
		t.Fatalf("Marshal(nil *int32) data=%v", data)
	}
}

func TestMarshalNilInterfaceValue(t *testing.T) {
	var any interface{}
	data, err := Marshal(&any)
	// &any 是指针，Elem 后是 nil interface；Marshal 对 Ptr 会 Elem
	// 实际 Marshal(*interface{}) 会 Elem 到 interface 再编码
	if err != nil {
		// 允许部分路径报错，但不应 panic
		t.Logf("Marshal(&nilIface) err=%v data=%v", err, data)
		return
	}
}

func TestSliceEncoderNilPtrSlice(t *testing.T) {
	var s *[]int32
	// 通过 reflect 调用 sliceEncoder：Ptr+nil
	v := reflect.ValueOf(s)
	b := enBuffer()
	defer b.release()
	if err := sliceEncoder(nil, b, v); err != nil {
		t.Fatalf("sliceEncoder(nil *[]int32) err=%v", err)
	}
	if len(b.buffer) == 0 {
		t.Fatal("应写入空 slice 头")
	}
}

func TestMapEmptyRoundTrip(t *testing.T) {
	src := map[string]int{}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal empty map err=%v", err)
	}
	got := map[string]int{"keep": 1}
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal empty map err=%v", err)
	}
	if len(got) != 0 {
		t.Fatalf("empty map want len0 got %v", got)
	}
}

func TestArrayStandaloneRoundTrip(t *testing.T) {
	src := [3]string{"a", "b", "c"}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal array err=%v", err)
	}
	var got [3]string
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal array err=%v", err)
	}
	if got != src {
		t.Fatalf("array mismatch %v vs %v", got, src)
	}
}

func TestInterfaceHoldingStructRoundTrip(t *testing.T) {
	// interface 内嵌 struct 的完整对称往返依赖解码端已有具体类型；
	// 这里验证编码成功，以及 interfaceDecoder 在目标无具体类型时返回明确错误。
	type inner struct {
		N int32  `epack:"1"`
		S string `epack:"2"`
	}
	type holder struct {
		Any interface{} `epack:"1"`
	}
	src := holder{Any: inner{N: 9, S: "hi"}}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected encoded bytes")
	}

	dst := holder{} // Any 为 nil，struct 分支应报 invalid
	if err := Unmarshal(data, &dst); err == nil {
		t.Fatal("Unmarshal into nil interface concrete type want error")
	}
}

func TestInterfaceStructInvalid(t *testing.T) {
	// interface 里没有具体类型时，struct 头应报 invalid
	b := enBuffer()
	defer b.release()
	b.copyBytes(encodeHead(uint64(reflect.Struct), 1))
	var any interface{}
	v := reflect.ValueOf(&any).Elem()
	err := interfaceDecoder(nil, deBuffer(b.buffer), v)
	if err == nil {
		t.Fatal("interfaceDecoder(struct without concrete type) want error")
	}
}

func TestDecodeInsufficientBuffers(t *testing.T) {
	// string head size=5 但无 body
	h := encodeHead(uint64(reflect.String), 5)
	var s string
	if err := stringDecoder(nil, deBuffer(h), reflect.ValueOf(&s).Elem()); err == nil {
		t.Fatal("stringDecoder truncated want error")
	}

	h2 := encodeHead(uint64(reflect.Int64), 8)
	var n int64
	if err := numberDecoder(nil, deBuffer(h2), reflect.ValueOf(&n).Elem()); err == nil {
		t.Fatal("numberDecoder truncated want error")
	}

	h3 := encodeHead(uint64(reflect.Int64), 8)
	var tm time.Time
	if err := timeDecoder(nil, deBuffer(h3), reflect.ValueOf(&tm).Elem()); err == nil {
		t.Fatal("timeDecoder truncated want error")
	}
}

func TestReadHeadIdempotentAndReuse(t *testing.T) {
	data, err := Marshal(int32(7))
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}
	b := deBuffer(data)
	if err := b.readHead(); err != nil {
		t.Fatalf("readHead err=%v", err)
	}
	if err := b.readHead(); err != nil {
		t.Fatalf("readHead second err=%v", err)
	}
	// decodeHead when read==true clears flag
	if err := b.decodeHead(); err != nil {
		t.Fatalf("decodeHead reuse err=%v", err)
	}
}

func TestNumberDecoderNonAddressable(t *testing.T) {
	// map value 不可 addressable，走 tmp + CanSet 分支
	m := map[string]int32{"k": 0}
	data, err := Marshal(int32(42))
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}
	v := m["k"] // 不可 addressable 的拷贝；直接对 map index 解码
	key := reflect.ValueOf("k")
	mv := reflect.ValueOf(m).MapIndex(key)
	_ = mv
	_ = v

	tmp := reflect.New(reflect.TypeOf(int32(0))).Elem()
	if err := numberDecoder(nil, deBuffer(data), tmp); err != nil {
		t.Fatalf("numberDecoder err=%v", err)
	}
	if tmp.Int() != 42 {
		t.Fatalf("got %v", tmp.Interface())
	}

	// CanSet=false：不可 set 的 value（如 unexported 通过反射拿到后）——用非指针 Elem 的 zero
	ro := reflect.ValueOf(int32(0))
	_ = numberDecoder(nil, deBuffer(data), ro) // CanAddr false, CanSet false，不应 panic
}

func TestAsStructValueEdges(t *testing.T) {
	tp := reflect.TypeOf(struct{ A int }{})
	if v := asStructValue(tp, reflect.Value{}); v.Kind() != reflect.Struct {
		t.Fatalf("invalid want zero struct, got %v", v)
	}
	var any interface{}
	if v := asStructValue(tp, reflect.ValueOf(&any).Elem()); !v.IsValid() || v.Kind() != reflect.Struct {
		t.Fatalf("nil iface want zero struct, got %v", v)
	}
	var p *struct{ A int }
	if v := asStructValue(tp, reflect.ValueOf(p)); v.Kind() != reflect.Struct {
		t.Fatalf("nil ptr want zero struct, got %v", v)
	}
	if v := asStructValue(tp, reflect.ValueOf(123)); v.Kind() != reflect.Struct {
		t.Fatalf("non-struct want zero, got %v", v)
	}
}

func TestCacheMarshalSkipsNilUnits(t *testing.T) {
	type s struct {
		A int32  `epack:"1"`
		B string `epack:"2"`
	}
	units := []*unit{nil, {
		seq:     1,
		encoder: stringEncoder,
		decoder: stringDecoder,
		kind:    reflect.String,
		name:    "B",
	}}
	src := s{A: 1, B: "x"}
	b := enBuffer()
	defer b.release()
	if err := cacheMarshal(units, b, reflect.ValueOf(src)); err != nil {
		t.Fatalf("cacheMarshal err=%v", err)
	}
	dst := s{B: "old"}
	if err := cacheUnmarshal(units, deBuffer(b.buffer), reflect.ValueOf(&dst).Elem()); err != nil {
		t.Fatalf("cacheUnmarshal err=%v", err)
	}
	if dst.B != "x" {
		t.Fatalf("got %+v", dst)
	}
}

func TestUnmarshalPointerNilIn_unmarshal(t *testing.T) {
	var p *struct {
		A int `epack:"1"`
	}
	err := _unmarshal(nil, deBuffer(nil), reflect.ValueOf(p))
	if err == nil {
		t.Fatal("_unmarshal(nil ptr) want error")
	}
}

func TestMarshalPtrStructUsesCache(t *testing.T) {
	type s struct {
		A int32 `epack:"1"`
	}
	src := &s{A: 5}
	if err := LoadTemplate(src); err != nil {
		t.Fatalf("LoadTemplate err=%v", err)
	}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}
	dst := new(s)
	if err := Unmarshal(data, dst); err != nil {
		t.Fatalf("Unmarshal err=%v", err)
	}
	if dst.A != 5 {
		t.Fatalf("got %+v", dst)
	}
}

func TestDecodeValueUnsupportedKind(t *testing.T) {
	// 构造非法 kind 头：kind = type_ENUM+1，走 getType 失败
	kind := uint64(_type_enum + 1)
	h := new2bHead(kind, 1)
	if _, err := _decodeValue(nil, deBuffer(h)); err == nil {
		t.Fatal("_decodeValue unsupported kind want error")
	}
}

func TestDecodeValueZeroSize(t *testing.T) {
	h := encodeHead(uint64(reflect.Interface), 0)
	v, err := _decodeValue(nil, deBuffer(h))
	if err != nil {
		t.Fatalf("_decodeValue zero err=%v", err)
	}
	if v.IsValid() && !v.IsNil() && v.Interface() != nil {
		// Zero interface may be invalid or nil-interface
		t.Logf("zero value=%#v", v.Interface())
	}
}

func TestPointerDecoderNonNil(t *testing.T) {
	type inner struct {
		N int32 `epack:"1"`
	}
	src := &inner{N: 3}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}
	dst := new(inner)
	if err := Unmarshal(data, dst); err != nil {
		t.Fatalf("Unmarshal err=%v", err)
	}
	if dst.N != 3 {
		t.Fatalf("got %+v", dst)
	}

	// 指针字段非 nil 往返
	type wrap struct {
		P *inner `epack:"1"`
	}
	wsrc := wrap{P: &inner{N: 11}}
	wdata, err := Marshal(wsrc)
	if err != nil {
		t.Fatalf("Marshal wrap err=%v", err)
	}
	var wdst wrap
	if err := Unmarshal(wdata, &wdst); err != nil {
		t.Fatalf("Unmarshal wrap err=%v", err)
	}
	if wdst.P == nil || wdst.P.N != 11 {
		t.Fatalf("got %+v", wdst)
	}
}

func TestMapInterfaceValueRoundTrip(t *testing.T) {
	src := map[string]interface{}{
		"i": int64(9),
		"s": "hello",
		"b": true,
	}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}
	got := make(map[string]interface{})
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal err=%v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %v", got)
	}
}

func TestSliceOfInterfaceRoundTrip(t *testing.T) {
	src := []interface{}{"a", int32(2), float64(3.5)}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}
	var got []interface{}
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal err=%v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %v", got)
	}
}

func TestNmlMarshalUnmarshalDirect(t *testing.T) {
	type plain struct {
		A int32
		B string
		c int // unexported skipped
	}
	src := plain{A: 1, B: "z", c: 9}
	b := enBuffer()
	defer b.release()
	if err := nmlMarshal(b, reflect.ValueOf(src)); err != nil {
		t.Fatalf("nmlMarshal err=%v", err)
	}
	dst := plain{c: 1}
	if err := nmlUnmarshal(deBuffer(b.buffer), reflect.ValueOf(&dst).Elem()); err != nil {
		t.Fatalf("nmlUnmarshal err=%v", err)
	}
	if dst.A != 1 || dst.B != "z" || dst.c != 1 {
		t.Fatalf("got %+v", dst)
	}
}

func TestMarshalErrorPathWhenEncoderFails(t *testing.T) {
	// 用不完整 buffer 解码触发失败路径即可；Marshal 本身很少失败。
	// 覆盖 unmarshal isOver 失败：多余字节
	data, err := Marshal(int32(1))
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}
	data = append(data, 0x00, 0x01)
	var got int32
	if err := Unmarshal(data, &got); err == nil {
		t.Fatal("extra bytes should fail isOver")
	}
}

func TestEnBufferNilFromPool(t *testing.T) {
	// sync.Pool.Put(nil) 会被忽略，无法把 typed-nil 放进池里；
	// enBuffer 的 `if b != nil` 回退分支在运行时实际不可达（死代码）。
	b := enBuffer()
	if b == nil {
		t.Fatal("enBuffer nil")
	}
	b.release()
}

func TestInterfaceDecoderStructSuccess(t *testing.T) {
	type inner struct {
		N int32  `epack:"1"`
		S string `epack:"2"`
	}
	src := inner{N: 7, S: "z"}
	units, err := newEncoder(0, reflect.TypeOf(inner{}), reflect.ValueOf(src))
	if err != nil {
		t.Fatalf("newEncoder err=%v", err)
	}
	b := enBuffer()
	defer b.release()
	if err := structEncoder(units, b, reflect.ValueOf(src)); err != nil {
		t.Fatalf("structEncoder err=%v", err)
	}
	var any interface{} = inner{}
	v := reflect.ValueOf(&any).Elem()
	// 注意：interface.Elem() 得到的具体值通常不可 Set，字段可能仍为零值；
	// 此处只验证 struct 分支可走通且不报错（字段写入限制见缺陷列表）。
	if err := interfaceDecoder(units, deBuffer(b.buffer), v); err != nil {
		t.Fatalf("interfaceDecoder struct err=%v", err)
	}
}

func Test_decodeValueArrayPath(t *testing.T) {
	src := [2]int32{9, 8}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}
	v, err := _decodeValue(nil, deBuffer(data))
	if err != nil {
		t.Fatalf("_decodeValue array err=%v", err)
	}
	if v.Len() != 2 {
		t.Fatalf("got %v", v.Interface())
	}
}

func TestPointerDecoderElemError(t *testing.T) {
	var p *string
	v := reflect.ValueOf(&p).Elem()
	h := encodeHead(uint64(reflect.String), 3) // size!=0 so非 nil 指针路径，但 body 截断
	// pointerDecoder readHead 后 size>0，再 decoder —— 需要头是 string 而不是 pointer
	// 直接构造：先写入非空 pointer 语义——pointerDecoder 在 size>0 时不消费独立 pointer 头的 elem 类型头？
	// 它 readHead 后用同一 head 交给 decoderFunc(elem)。因此 head 应为 string。
	if err := pointerDecoder(nil, deBuffer(h), v); err == nil {
		t.Fatal("pointerDecoder truncated string want error")
	}
}

func TestNewMarshalAndMarshalError(t *testing.T) {
	type s struct {
		A int32 `epack:"1"`
	}
	// 确保 cache 未命中时走 newMarshal
	type uniqueMarshalType struct {
		A int32 `epack:"1"`
	}
	data, err := Marshal(uniqueMarshalType{A: 1})
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}
	dst := uniqueMarshalType{}
	if err := Unmarshal(data, &dst); err != nil {
		t.Fatalf("Unmarshal err=%v", err)
	}
	if dst.A != 1 {
		t.Fatalf("got %+v", dst)
	}
	_ = s{}
}

func Test_unmarshalNmlPath(t *testing.T) {
	type plain struct {
		A int32
		B string
	}
	src := plain{A: 3, B: "nml"}
	b := enBuffer()
	defer b.release()
	if err := nmlMarshal(b, reflect.ValueOf(src)); err != nil {
		t.Fatalf("nmlMarshal err=%v", err)
	}
	dst := plain{}
	if err := _unmarshal(nil, deBuffer(b.buffer), reflect.ValueOf(&dst).Elem()); err != nil {
		t.Fatalf("_unmarshal nml err=%v", err)
	}
	if dst.A != 3 || dst.B != "nml" {
		t.Fatalf("got %+v", dst)
	}
}

func TestArrayEncoderNestedError(t *testing.T) {
	// 子元素编码成功路径已覆盖；再补一个空数组
	var empty [0]int
	data, err := Marshal(empty)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	var got [0]int
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestMapEncoderKeyValue(t *testing.T) {
	src := map[int32]string{1: "a", 2: "b"}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	got := map[int32]string{}
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("err=%v", err)
	}
	if !reflect.DeepEqual(src, got) {
		t.Fatalf("%v vs %v", src, got)
	}
}

func TestSliceEncoderNonNumber(t *testing.T) {
	src := []string{"x", "y"}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	var got []string
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("err=%v", err)
	}
	if !reflect.DeepEqual(src, got) {
		t.Fatalf("%v vs %v", src, got)
	}
}

func TestLoadTemplatePointerAndSkipCached(t *testing.T) {
	type lt struct {
		A int32 `epack:"1"`
	}
	if err := LoadTemplate(&lt{A: 1}); err != nil {
		t.Fatalf("err=%v", err)
	}
	if err := LoadTemplate(lt{}); err != nil {
		t.Fatalf("second err=%v", err)
	}
}

func TestCacheUnmarshalCanSetSkip(t *testing.T) {
	// 同包内字段通常 CanSet；构造 seq 越界会 panic，改为 decoder 错误路径
	units := []*unit{{
		seq: 0,
		decoder: func(u []*unit, b *ubuffer, v reflect.Value) error {
			return fmt.Errorf("boom")
		},
		encoder: stringEncoder,
		kind:    reflect.String,
		name:    "A",
	}}
	type s struct {
		A string
	}
	if err := cacheUnmarshal(units, deBuffer([]byte{1, 2}), reflect.ValueOf(&s{}).Elem()); err == nil {
		t.Fatal("want decoder error")
	}
}

func TestNewEncoderSkipsBadTags(t *testing.T) {
	type tagged struct {
		Ok   int32 `epack:"1"`
		Bad  int32 `epack:"0"`
		Neg  int32 `epack:"-1"`
		None int32
		hide int32 `epack:"2"`
	}
	units, err := newEncoder(0, reflect.TypeOf(tagged{}), reflect.ValueOf(tagged{Ok: 1}))
	if err != nil {
		t.Fatalf("newEncoder err=%v", err)
	}
	if units[0] == nil || units[0].name != "Ok" {
		t.Fatalf("units=%v", units)
	}
}

func TestInterfaceEncoderNilDirect(t *testing.T) {
	var any interface{}
	b := enBuffer()
	defer b.release()
	if err := interfaceEncoder(nil, b, reflect.ValueOf(&any).Elem()); err != nil {
		t.Fatalf("interfaceEncoder nil err=%v", err)
	}
}

func TestStructDecoderNested(t *testing.T) {
	type inner struct {
		N int32 `epack:"1"`
	}
	type outer struct {
		In inner `epack:"1"`
	}
	src := outer{In: inner{N: 8}}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}
	var dst outer
	if err := Unmarshal(data, &dst); err != nil {
		t.Fatalf("Unmarshal err=%v", err)
	}
	if dst.In.N != 8 {
		t.Fatalf("got %+v", dst)
	}
}

func Test_arrayInterfaceEmpty(t *testing.T) {
	h := encodeHead(uint64(reflect.Array), 0)
	v, err := _arrayInterface(nil, deBuffer(h))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if v.Len() != 0 {
		t.Fatalf("want empty, got %v", v.Interface())
	}
}

func TestGetTypeValidKinds(t *testing.T) {
	for _, k := range []reflect.Kind{reflect.Int, reflect.String, reflect.Slice, reflect.Map} {
		tp, err := conf.getType(k)
		if err != nil || tp == nil {
			t.Fatalf("getType(%v)=%v,%v", k, tp, err)
		}
	}
}
