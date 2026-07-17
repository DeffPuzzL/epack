package epack

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func assertRoundTrip(t *testing.T, src interface{}, dst interface{}) {
	t.Helper()
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal(%T) err=%v", src, err)
	}
	if err := Unmarshal(data, dst); err != nil {
		t.Fatalf("Unmarshal(%T) err=%v data=%v", src, err, data)
	}
}

func TestMarshalUnmarshal_Primitives(t *testing.T) {
	t.Run("bool", func(t *testing.T) {
		var got bool
		assertRoundTrip(t, true, &got)
		if !got {
			t.Fatal(got)
		}
	})
	t.Run("string", func(t *testing.T) {
		var got string
		assertRoundTrip(t, "hello", &got)
		if got != "hello" {
			t.Fatal(got)
		}
	})
	t.Run("empty_string", func(t *testing.T) {
		var got string
		assertRoundTrip(t, "", &got)
		if got != "" {
			t.Fatal(got)
		}
	})

	t.Run("int8", func(t *testing.T) {
		var g int8
		assertRoundTrip(t, int8(-7), &g)
		if g != -7 {
			t.Fatal(g)
		}
	})
	t.Run("int16", func(t *testing.T) {
		var g int16
		assertRoundTrip(t, int16(-700), &g)
		if g != -700 {
			t.Fatal(g)
		}
	})
	t.Run("int32", func(t *testing.T) {
		var g int32
		assertRoundTrip(t, int32(42), &g)
		if g != 42 {
			t.Fatal(g)
		}
	})
	t.Run("int64", func(t *testing.T) {
		var g int64
		assertRoundTrip(t, int64(1<<40), &g)
		if g != 1<<40 {
			t.Fatal(g)
		}
	})
	t.Run("int", func(t *testing.T) {
		var g int
		assertRoundTrip(t, int(99), &g)
		if g != 99 {
			t.Fatal(g)
		}
	})
	t.Run("uint8", func(t *testing.T) {
		var g uint8
		assertRoundTrip(t, uint8(7), &g)
		if g != 7 {
			t.Fatal(g)
		}
	})
	t.Run("uint16", func(t *testing.T) {
		var g uint16
		assertRoundTrip(t, uint16(700), &g)
		if g != 700 {
			t.Fatal(g)
		}
	})
	t.Run("uint32", func(t *testing.T) {
		var g uint32
		assertRoundTrip(t, uint32(42), &g)
		if g != 42 {
			t.Fatal(g)
		}
	})
	t.Run("uint64", func(t *testing.T) {
		var g uint64
		assertRoundTrip(t, uint64(1<<40), &g)
		if g != 1<<40 {
			t.Fatal(g)
		}
	})
	t.Run("uint", func(t *testing.T) {
		var g uint
		assertRoundTrip(t, uint(99), &g)
		if g != 99 {
			t.Fatal(g)
		}
	})
	t.Run("float32", func(t *testing.T) {
		var g float32
		assertRoundTrip(t, float32(3.5), &g)
		if g != 3.5 {
			t.Fatal(g)
		}
	})
	t.Run("float64", func(t *testing.T) {
		var g float64
		assertRoundTrip(t, math.Pi, &g)
		if g != math.Pi {
			t.Fatal(g)
		}
	})
}

func TestMarshalUnmarshal_Containers(t *testing.T) {
	t.Run("slice_int32", func(t *testing.T) {
		src := []int32{1, 2, 3}
		var got []int32
		assertRoundTrip(t, src, &got)
		if !reflect.DeepEqual(src, got) {
			t.Fatalf("%v vs %v", src, got)
		}
	})
	t.Run("slice_empty", func(t *testing.T) {
		src := []string{}
		var got []string
		assertRoundTrip(t, src, &got)
		if len(got) != 0 {
			t.Fatalf("%#v", got)
		}
	})
	t.Run("slice_string", func(t *testing.T) {
		src := []string{"a", "b"}
		var got []string
		assertRoundTrip(t, src, &got)
		if !reflect.DeepEqual(src, got) {
			t.Fatal(got)
		}
	})
	t.Run("array_int32", func(t *testing.T) {
		src := [3]int32{1, 2, 3}
		var got [3]int32
		assertRoundTrip(t, src, &got)
		if src != got {
			t.Fatal(got)
		}
	})
	t.Run("map_string_int32", func(t *testing.T) {
		src := map[string]int32{"a": 1, "b": 2}
		var got map[string]int32
		assertRoundTrip(t, src, &got)
		if !reflect.DeepEqual(src, got) {
			t.Fatal(got)
		}
	})
	t.Run("map_empty", func(t *testing.T) {
		src := map[string]int{}
		var got map[string]int
		assertRoundTrip(t, src, &got)
		if len(got) != 0 {
			t.Fatal(got)
		}
	})
	t.Run("ptr_field_in_struct", func(t *testing.T) {
		type wrap struct {
			P *int32 `epack:"1"`
		}
		v := int32(9)
		src := wrap{P: &v}
		conf.cache.Delete(reflect.TypeOf(wrap{}).String())
		var got wrap
		assertRoundTrip(t, src, &got)
		if got.P == nil || *got.P != 9 {
			t.Fatal(got.P)
		}
	})
	t.Run("nil_ptr_field_in_struct", func(t *testing.T) {
		type wrap struct {
			P *int32 `epack:"1"`
		}
		src := wrap{P: nil}
		conf.cache.Delete(reflect.TypeOf(wrap{}).String())
		var got wrap
		got.P = new(int32)
		*got.P = 1
		assertRoundTrip(t, src, &got)
		if got.P != nil {
			t.Fatalf("want nil ptr field, got %v", *got.P)
		}
	})
}

func TestMarshalUnmarshal_StructVariants(t *testing.T) {
	type nested struct {
		X int32 `epack:"1"`
	}
	type root struct {
		A int32             `epack:"1"`
		B string            `epack:"2"`
		C *nested           `epack:"3"`
		D []nested          `epack:"4"`
		E map[string]nested `epack:"5"`
		F interface{}       `epack:"6"`
		T time.Time         `epack:"7"`
	}

	ts := time.Unix(0, 123456789).UTC()
	src := root{
		A: 1,
		B: "b",
		C: &nested{X: 2},
		D: []nested{{X: 3}, {X: 4}},
		E: map[string]nested{"k": {X: 5}},
		F: int32(6),
		T: ts,
	}
	conf.cache.Delete(reflect.TypeOf(root{}).String())

	var got root
	assertRoundTrip(t, src, &got)
	if got.A != 1 || got.B != "b" || got.C == nil || got.C.X != 2 {
		t.Fatalf("%+v", got)
	}
	if len(got.D) != 2 || got.D[1].X != 4 {
		t.Fatal(got.D)
	}
	if got.E["k"].X != 5 {
		t.Fatal(got.E)
	}
	if got.F != int32(6) {
		t.Fatalf("F=%v", got.F)
	}
	if !got.T.Equal(ts) {
		t.Fatalf("T=%v", got.T)
	}

	// cache hit path
	var got2 root
	assertRoundTrip(t, src, &got2)
	if got2.A != 1 {
		t.Fatal(got2)
	}
}

func TestMarshalUnmarshal_InterfaceSlice(t *testing.T) {
	src := []interface{}{int32(1), "x", true}
	var got []interface{}
	assertRoundTrip(t, src, &got)
	if len(got) != 3 {
		t.Fatal(got)
	}
}

func TestMarshalUnmarshal_LargeString8bHead(t *testing.T) {
	src := string(make([]byte, 2000))
	var got string
	assertRoundTrip(t, src, &got)
	if len(got) != 2000 {
		t.Fatal(len(got))
	}
}
