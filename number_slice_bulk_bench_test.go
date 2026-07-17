package epack

import (
	"reflect"
	"testing"
)

func encodeNumSliceLoop(b *ubuffer, v reflect.Value) {
	n := v.Len()
	ek := v.Type().Elem().Kind()
	b.buffer = append(b.buffer, simpleNumber, byte(ek))
	for i := 0; i < n; i++ {
		appendNumberPayloadLE(b, v.Index(i))
	}
}

func encodeNumSliceBulk(b *ubuffer, v reflect.Value) {
	n := v.Len()
	ek := v.Type().Elem().Kind()
	b.buffer = append(b.buffer, simpleNumber, byte(ek))
	if canBulkNumberCopy(ek) {
		if raw, ok := numberSliceBytes(v); ok {
			b.buffer = append(b.buffer, raw...)
			return
		}
	}
	for i := 0; i < n; i++ {
		appendNumberPayloadLE(b, v.Index(i))
	}
}

// ---- decode：从 simpleNumber 头之后解析 ----

func decodeNumSliceLoop(buf []byte, n int, ek reflect.Kind) (reflect.Value, error) {
	b := deBuffer(buf)
	et, err := conf.getType(ek)
	if err != nil {
		return reflect.Value{}, err
	}
	es := uint64(et.Size())
	need := es * uint64(n)
	if b.remain < need {
		return reflect.Value{}, errShortNumSlice
	}
	ns := reflect.MakeSlice(reflect.SliceOf(et), n, n)
	for i := 0; i < n; i++ {
		payload := b.buffer[b.pos : b.pos+es]
		if err := setNumberFromLE(ns.Index(i), payload); err != nil {
			return reflect.Value{}, err
		}
		b.walk(es)
	}
	return ns, nil
}

func decodeNumSliceBulk(buf []byte, n int, ek reflect.Kind) (reflect.Value, error) {
	b := deBuffer(buf)
	et, err := conf.getType(ek)
	if err != nil {
		return reflect.Value{}, err
	}
	es := uint64(et.Size())
	need := es * uint64(n)
	if b.remain < need {
		return reflect.Value{}, errShortNumSlice
	}
	ns := reflect.MakeSlice(reflect.SliceOf(et), n, n)
	if n > 0 && canBulkNumberCopy(ek) {
		if dst, ok := numberSliceBytes(ns); ok {
			copy(dst, b.buffer[b.pos:b.pos+need])
			return ns, nil
		}
	}
	return decodeNumSliceLoop(buf, n, ek)
}

func TestNumberSliceBulk_MatchesLoop(t *testing.T) {
	if !localEndian {
		t.Skip("host is not little-endian")
	}
	cases := []any{
		[]int32{1, -2, 3, 1 << 20},
		[]int64{1, 2, 3, 1 << 40},
		[]float64{1.5, -2.5, 3.25},
		[]uint16{1, 2, 65535},
		[]int{1, 2, 3},
	}
	for _, src := range cases {
		v := reflect.ValueOf(src)
		n := v.Len()
		ek := v.Type().Elem().Kind()

		bl := &ubuffer{buffer: make([]byte, 0, 256)}
		bb := &ubuffer{buffer: make([]byte, 0, 256)}
		encodeNumSliceLoop(bl, v)
		encodeNumSliceBulk(bb, v)
		if !reflect.DeepEqual(bl.buffer, bb.buffer) {
			t.Fatalf("%v encode mismatch\nloop=%v\nbulk=%v", ek, bl.buffer, bb.buffer)
		}

		payload := bl.buffer[2:]
		gotL, err := decodeNumSliceLoop(payload, n, ek)
		if err != nil {
			t.Fatal(err)
		}
		gotB, err := decodeNumSliceBulk(append([]byte(nil), payload...), n, ek)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(gotL.Interface(), gotB.Interface()) || !reflect.DeepEqual(gotL.Interface(), src) {
			t.Fatalf("%v decode mismatch src=%v loop=%v bulk=%v", ek, src, gotL.Interface(), gotB.Interface())
		}
	}
}

func TestNumberSliceBulk_ProdMarshalRoundTrip(t *testing.T) {
	cases := []any{
		[]int64{1, -2, 1 << 40},
		[]int32{1, 2, 3},
		[]bool{true, false, true}, // bool 走逐元素
		[]float32{1.25, -3.5},
	}
	for _, src := range cases {
		data, err := Marshal(src)
		if err != nil {
			t.Fatalf("Marshal %T: %v", src, err)
		}
		dst := reflect.New(reflect.TypeOf(src)).Interface()
		if err := Unmarshal(data, dst); err != nil {
			t.Fatalf("Unmarshal %T: %v", src, err)
		}
		got := reflect.ValueOf(dst).Elem().Interface()
		if !reflect.DeepEqual(src, got) {
			t.Fatalf("%T: %v vs %v", src, src, got)
		}
	}
}

func makeInt64Slice(n int) []int64 {
	s := make([]int64, n)
	for i := range s {
		s[i] = int64(i) * 17
	}
	return s
}

func makeInt32Slice(n int) []int32 {
	s := make([]int32, n)
	for i := range s {
		s[i] = int32(i) * 17
	}
	return s
}

func BenchmarkNumberSliceEncode(b *testing.B) {
	if !localEndian {
		b.Skip("host is not little-endian")
	}
	for _, n := range []int{16, 256, 4096, 65536} {
		src := makeInt64Slice(n)
		v := reflect.ValueOf(src)
		buf := &ubuffer{buffer: make([]byte, 0, n*8+16)}

		b.Run("loop/int64/"+itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(n * 8))
			for i := 0; i < b.N; i++ {
				buf.buffer = buf.buffer[:0]
				encodeNumSliceLoop(buf, v)
			}
		})
		b.Run("bulk/int64/"+itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(n * 8))
			for i := 0; i < b.N; i++ {
				buf.buffer = buf.buffer[:0]
				encodeNumSliceBulk(buf, v)
			}
		})
	}
	for _, n := range []int{256, 4096} {
		src := makeInt32Slice(n)
		v := reflect.ValueOf(src)
		buf := &ubuffer{buffer: make([]byte, 0, n*4+16)}
		b.Run("loop/int32/"+itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(n * 4))
			for i := 0; i < b.N; i++ {
				buf.buffer = buf.buffer[:0]
				encodeNumSliceLoop(buf, v)
			}
		})
		b.Run("bulk/int32/"+itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(n * 4))
			for i := 0; i < b.N; i++ {
				buf.buffer = buf.buffer[:0]
				encodeNumSliceBulk(buf, v)
			}
		})
	}
}

func BenchmarkNumberSliceDecode(b *testing.B) {
	if !localEndian {
		b.Skip("host is not little-endian")
	}
	for _, n := range []int{16, 256, 4096, 65536} {
		src := makeInt64Slice(n)
		v := reflect.ValueOf(src)
		enc := &ubuffer{buffer: make([]byte, 0, n*8+16)}
		encodeNumSliceLoop(enc, v)
		payload := enc.buffer[2:]

		b.Run("loop/int64/"+itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(n * 8))
			for i := 0; i < b.N; i++ {
				if _, err := decodeNumSliceLoop(payload, n, reflect.Int64); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run("bulk/int64/"+itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(n * 8))
			for i := 0; i < b.N; i++ {
				if _, err := decodeNumSliceBulk(payload, n, reflect.Int64); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkNumberSliceMarshalE2E(b *testing.B) {
	if !localEndian {
		b.Skip("host is not little-endian")
	}
	for _, n := range []int{256, 4096, 65536} {
		src := makeInt64Slice(n)
		b.Run("prod_Marshal/int64/"+itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(n * 8))
			for i := 0; i < b.N; i++ {
				if _, err := Marshal(src); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var a [16]byte
	i := len(a)
	for n > 0 {
		i--
		a[i] = byte('0' + n%10)
		n /= 10
	}
	return string(a[i:])
}
