package epack

import (
	"reflect"
	"testing"
	"time"
)

// 更刁钻的组合：多层容器交叉（map/slice/ptr/interface/嵌套结构体）。
type hardLeaf struct {
	ID   int32  `epack:"1"`
	Name string `epack:"2"`
}

type hardMid struct {
	Tag   string      `epack:"1"`
	Leaf  hardLeaf    `epack:"2"`
	PLeaf *hardLeaf   `epack:"3"`
	Any   interface{} `epack:"4"`
}

func TestHardCombo_MapStringSlicePtrStruct(t *testing.T) {
	type root struct {
		M map[string][]*hardLeaf `epack:"1"`
	}
	a, b := &hardLeaf{ID: 1, Name: "a"}, &hardLeaf{ID: 2, Name: "b"}
	src := root{M: map[string][]*hardLeaf{
		"g1": {a, b},
		"g2": {nil, &hardLeaf{ID: 3, Name: "c"}},
	}}
	conf.cache.Delete(reflect.TypeOf(root{}).String())

	var got root
	assertRoundTrip(t, src, &got)
	if len(got.M) != 2 || len(got.M["g1"]) != 2 {
		t.Fatalf("%+v", got.M)
	}
	if got.M["g1"][0] == nil || got.M["g1"][0].ID != 1 || got.M["g1"][1].Name != "b" {
		t.Fatalf("g1=%+v", got.M["g1"])
	}
	if got.M["g2"][0] != nil {
		t.Fatalf("want nil ptr in g2[0], got %+v", got.M["g2"][0])
	}
	if got.M["g2"][1] == nil || got.M["g2"][1].ID != 3 {
		t.Fatalf("g2[1]=%+v", got.M["g2"][1])
	}
}

func TestHardCombo_SliceOfMapStringInterface(t *testing.T) {
	src := []map[string]interface{}{
		{"n": int32(1), "s": "x", "b": true},
		{"n": int32(2), "arr": []int32{7, 8}},
		{},
	}
	var got []map[string]interface{}
	assertRoundTrip(t, src, &got)
	if len(got) != 3 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0]["n"] != int32(1) || got[0]["s"] != "x" || got[0]["b"] != true {
		t.Fatalf("got[0]=%v", got[0])
	}
	// 经 interface{} 解码的数字切片会装箱为 []interface{}
	arr, ok := got[1]["arr"].([]interface{})
	if !ok || len(arr) != 2 || arr[0] != int32(7) || arr[1] != int32(8) {
		t.Fatalf("arr=%T %#v", got[1]["arr"], got[1]["arr"])
	}
	if len(got[2]) != 0 {
		t.Fatalf("empty map got %v", got[2])
	}
}

func TestHardCombo_NestedSliceOfSlice(t *testing.T) {
	type root struct {
		Matrix [][]int32          `epack:"1"`
		Names  [][]string         `epack:"2"`
		Mixed  [][]*hardLeaf      `epack:"3"`
		Maps   []map[string]int32 `epack:"4"`
	}
	src := root{
		Matrix: [][]int32{{1, 2}, {}, {3}},
		Names:  [][]string{{"a"}, {"b", "c"}},
		Mixed: [][]*hardLeaf{
			{&hardLeaf{ID: 9, Name: "n"}},
			{nil},
		},
		Maps: []map[string]int32{
			{"x": 1},
			nil,
			{"y": 2, "z": 3},
		},
	}
	conf.cache.Delete(reflect.TypeOf(root{}).String())

	var got root
	assertRoundTrip(t, src, &got)
	if len(got.Matrix) != 3 || !reflect.DeepEqual(got.Matrix[0], []int32{1, 2}) || len(got.Matrix[1]) != 0 || !reflect.DeepEqual(got.Matrix[2], []int32{3}) {
		t.Fatalf("matrix %+v", got.Matrix)
	}
	if len(got.Names) != 2 || !reflect.DeepEqual(got.Names[0], []string{"a"}) || !reflect.DeepEqual(got.Names[1], []string{"b", "c"}) {
		t.Fatalf("names %+v", got.Names)
	}
	if len(got.Mixed) != 2 || got.Mixed[0][0] == nil || got.Mixed[0][0].ID != 9 {
		t.Fatalf("mixed %+v", got.Mixed)
	}
	if got.Mixed[1][0] != nil {
		t.Fatal("want nil in Mixed[1][0]")
	}
	if got.Maps[0]["x"] != 1 || len(got.Maps[1]) != 0 || got.Maps[2]["z"] != 3 {
		// nil map 经编解码后可能落成空 map
		t.Fatalf("maps %+v", got.Maps)
	}
}

func TestHardCombo_MapOfMidWithInterfaceAndPtr(t *testing.T) {
	type root struct {
		Items map[string]*hardMid `epack:"1"`
	}
	leaf := hardLeaf{ID: 11, Name: "leaf"}
	src := root{Items: map[string]*hardMid{
		"ok": {
			Tag:   "t1",
			Leaf:  leaf,
			PLeaf: &hardLeaf{ID: 12, Name: "p"},
			Any:   []string{"u", "v"},
		},
		"nilval": nil,
		"nilptr": {
			Tag:   "t2",
			PLeaf: nil,
			Any:   int64(99),
		},
	}}
	conf.cache.Delete(reflect.TypeOf(root{}).String())

	var got root
	assertRoundTrip(t, src, &got)
	if got.Items["nilval"] != nil {
		t.Fatal("nil map value")
	}
	ok := got.Items["ok"]
	if ok == nil || ok.Tag != "t1" || ok.Leaf.ID != 11 || ok.PLeaf == nil || ok.PLeaf.ID != 12 {
		t.Fatalf("ok=%+v", ok)
	}
	// interface{} 字段解字符串切片时通常落成 []interface{}
	switch any := ok.Any.(type) {
	case []string:
		if !reflect.DeepEqual(any, []string{"u", "v"}) {
			t.Fatalf("Any=%v", any)
		}
	case []interface{}:
		if len(any) != 2 || any[0] != "u" || any[1] != "v" {
			t.Fatalf("Any=%#v", any)
		}
	default:
		t.Fatalf("Any=%T %#v", ok.Any, ok.Any)
	}
	np := got.Items["nilptr"]
	if np == nil || np.PLeaf != nil || np.Any != int64(99) {
		t.Fatalf("nilptr=%+v", np)
	}
}

func TestHardCombo_ArrayOfStructWithMapSlice(t *testing.T) {
	type cell struct {
		KVs map[string][]int32 `epack:"1"`
		Arr [2]string          `epack:"2"`
	}
	type root struct {
		Cells [2]cell `epack:"1"`
	}
	src := root{Cells: [2]cell{
		{KVs: map[string][]int32{"a": {1, 2}}, Arr: [2]string{"x", "y"}},
		{KVs: map[string][]int32{}, Arr: [2]string{"", "z"}},
	}}
	conf.cache.Delete(reflect.TypeOf(root{}).String())

	var got root
	assertRoundTrip(t, src, &got)
	if !reflect.DeepEqual(got.Cells[0].KVs["a"], []int32{1, 2}) {
		t.Fatalf("%+v", got.Cells[0])
	}
	if got.Cells[0].Arr != [2]string{"x", "y"} || got.Cells[1].Arr[1] != "z" {
		t.Fatalf("arr %+v", got.Cells)
	}
}

func TestHardCombo_InterfaceHoldingMapAndStruct(t *testing.T) {
	type root struct {
		A interface{} `epack:"1"`
		B interface{} `epack:"2"`
		C interface{} `epack:"3"`
	}
	src := root{
		A: map[string]int32{"k": 7},
		B: []interface{}{int32(1), "s", map[string]int32{"n": 2}},
		C: nil,
	}
	conf.cache.Delete(reflect.TypeOf(root{}).String())

	var got root
	assertRoundTrip(t, src, &got)
	switch m := got.A.(type) {
	case map[string]int32:
		if m["k"] != 7 {
			t.Fatalf("A=%v", m)
		}
	case map[string]interface{}:
		if m["k"] != int32(7) && m["k"] != int64(7) {
			t.Fatalf("A=%v", m)
		}
	case map[interface{}]interface{}:
		if m["k"] != int32(7) && m["k"] != int64(7) {
			t.Fatalf("A=%#v", m)
		}
	default:
		t.Fatalf("A=%T %#v", got.A, got.A)
	}
	sl, ok := got.B.([]interface{})
	if !ok || len(sl) != 3 {
		t.Fatalf("B=%T %#v", got.B, got.B)
	}
	if got.C != nil {
		t.Fatalf("C=%v", got.C)
	}
}

func TestHardCombo_PtrToSliceAndPtrToMap(t *testing.T) {
	type root struct {
		PS *[]hardLeaf           `epack:"1"`
		PM *map[string]*hardLeaf `epack:"2"`
		NS *[]hardLeaf           `epack:"3"` // nil
		NM *map[string]*hardLeaf `epack:"4"` // nil
	}
	s := []hardLeaf{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}
	m := map[string]*hardLeaf{"p": {ID: 3, Name: "c"}}
	src := root{PS: &s, PM: &m, NS: nil, NM: nil}
	conf.cache.Delete(reflect.TypeOf(root{}).String())

	var got root
	assertRoundTrip(t, src, &got)
	if got.PS == nil || len(*got.PS) != 2 || (*got.PS)[1].Name != "b" {
		t.Fatalf("PS=%v", got.PS)
	}
	if got.PM == nil || (*got.PM)["p"] == nil || (*got.PM)["p"].ID != 3 {
		t.Fatalf("PM=%v", got.PM)
	}
	if got.NS != nil || got.NM != nil {
		t.Fatalf("nil ptrs NS=%v NM=%v", got.NS, got.NM)
	}
}

func TestHardCombo_DeepKitchenSink(t *testing.T) {
	type node struct {
		Kids map[string][]*node     `epack:"1"`
		Meta map[string]interface{} `epack:"2"`
		When time.Time              `epack:"3"`
	}
	type root struct {
		Tree *node                    `epack:"1"`
		Bag  []map[string][]*hardLeaf `epack:"2"`
	}
	ts := time.Unix(0, 42).UTC()
	child := &node{
		Kids: nil,
		Meta: map[string]interface{}{"v": int32(1)},
		When: ts,
	}
	src := root{
		Tree: &node{
			Kids: map[string][]*node{"c": {child, nil}},
			Meta: map[string]interface{}{"flag": true, "s": "ok"},
			When: ts,
		},
		Bag: []map[string][]*hardLeaf{
			{"x": {&hardLeaf{ID: 5, Name: "e"}}},
		},
	}
	conf.cache.Delete(reflect.TypeOf(root{}).String())
	conf.cache.Delete(reflect.TypeOf(node{}).String())

	var got root
	assertRoundTrip(t, src, &got)
	if got.Tree == nil || !got.Tree.When.Equal(ts) {
		t.Fatalf("tree=%+v", got.Tree)
	}
	kids := got.Tree.Kids["c"]
	if len(kids) != 2 || kids[0] == nil || kids[1] != nil {
		t.Fatalf("kids=%+v", kids)
	}
	if kids[0].Meta["v"] != int32(1) && kids[0].Meta["v"] != int64(1) {
		t.Fatalf("child meta=%v", kids[0].Meta)
	}
	if got.Tree.Meta["flag"] != true || got.Tree.Meta["s"] != "ok" {
		t.Fatalf("meta=%v", got.Tree.Meta)
	}
	if len(got.Bag) != 1 || got.Bag[0]["x"][0] == nil || got.Bag[0]["x"][0].ID != 5 {
		t.Fatalf("bag=%+v", got.Bag)
	}
}
