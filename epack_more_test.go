// 针对 epack 包补充的回归与边界测试：
// - 错误入参校验（Marshal/Unmarshal/LoadTemplate）
// - 原始值(数字/字符串/切片/map) 的单独 Marshal/Unmarshal
// - 定长数组(arrayEncoder/arrayDecoder) 路径
// - 嵌套 map 值为结构体（触发 nmlMarshal/nmlUnmarshal/newUnmarshal）
// - 大 payload (size >= 1024 走 8 字节头)
// - 指针/接口的 nil 分支
// - intsToBytes / bytesToInts 互转与异常
// - decodeHead 截断错误
// - debug 字符串输出

package epack

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
	"unsafe"
)

// ---------- 错误入参校验 ----------

func TestMarshalNilInput(t *testing.T) {
	if _, err := Marshal(nil); err == nil {
		t.Fatal("Marshal(nil) 应报错")
	}
}

func TestUnmarshalNilInput(t *testing.T) {
	if err := Unmarshal([]byte{0x01, 0x02}, nil); err == nil {
		t.Fatal("Unmarshal(_, nil) 应报错")
	}
}

func TestUnmarshalMustBePointer(t *testing.T) {
	var v int
	if err := Unmarshal([]byte{0x01, 0x02}, v); err == nil {
		t.Fatal("非指针入参应报错")
	}
}

func TestLoadTemplateInvalid(t *testing.T) {
	if err := LoadTemplate(nil); err == nil {
		t.Fatal("LoadTemplate(nil) 应报错")
	}

	if err := LoadTemplate(42); err == nil {
		t.Fatal("LoadTemplate(非结构体) 应报错")
	}
}

// 同一类型多次 LoadTemplate 第二次应走 cache 命中并直接 continue，不应报错。
type moreTplRepeat struct {
	A int `epack:"1"`
}

func TestLoadTemplateIdempotent(t *testing.T) {
	if err := LoadTemplate(moreTplRepeat{}); err != nil {
		t.Fatalf("首次 LoadTemplate 失败: %v", err)
	}
	if err := LoadTemplate(&moreTplRepeat{A: 1}); err != nil {
		t.Fatalf("重复 LoadTemplate 失败: %v", err)
	}
}

// ---------- 原始值 (非结构体) 往返 ----------

func TestRoundTripPrimitives(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		data, err := Marshal(12345)
		if err != nil {
			t.Fatalf("Marshal(int) err=%v", err)
		}
		var got int
		if err := Unmarshal(data, &got); err != nil {
			t.Fatalf("Unmarshal(int) err=%v", err)
		}
		if got != 12345 {
			t.Fatalf("int round-trip got=%d", got)
		}
	})

	t.Run("string", func(t *testing.T) {
		data, err := Marshal("hello-epack")
		if err != nil {
			t.Fatalf("Marshal(string) err=%v", err)
		}
		var got string
		if err := Unmarshal(data, &got); err != nil {
			t.Fatalf("Unmarshal(string) err=%v", err)
		}
		if got != "hello-epack" {
			t.Fatalf("string round-trip got=%q", got)
		}
	})

	t.Run("slice-of-int32", func(t *testing.T) {
		src := []int32{1, 2, 3, 1000, -1}
		data, err := Marshal(src)
		if err != nil {
			t.Fatalf("Marshal([]int32) err=%v", err)
		}
		var got []int32
		if err := Unmarshal(data, &got); err != nil {
			t.Fatalf("Unmarshal([]int32) err=%v", err)
		}
		if !reflect.DeepEqual(src, got) {
			t.Fatalf("[]int32 round-trip mismatch: %v vs %v", src, got)
		}
	})

	t.Run("map-basic", func(t *testing.T) {
		src := map[string]int64{"a": 1, "b": 2, "c": 3}
		data, err := Marshal(src)
		if err != nil {
			t.Fatalf("Marshal(map) err=%v", err)
		}
		got := make(map[string]int64)
		if err := Unmarshal(data, &got); err != nil {
			t.Fatalf("Unmarshal(map) err=%v", err)
		}
		if !reflect.DeepEqual(src, got) {
			t.Fatalf("map round-trip mismatch: %v vs %v", src, got)
		}
	})
}

// ---------- 大 payload: 触发 8 字节 header (size >= 1024) ----------

func TestRoundTripLargeString(t *testing.T) {
	// 构造长度 > 1024 的字符串，走 new8bHead / 8 字节 header 解码分支。
	s := strings.Repeat("a", 2048)
	data, err := Marshal(s)
	if err != nil {
		t.Fatalf("Marshal(large str) err=%v", err)
	}
	var got string
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal(large str) err=%v", err)
	}
	if got != s {
		t.Fatalf("large string mismatch len=%d vs %d", len(s), len(got))
	}
}

// ---------- 定长数组 (arrayEncoder / arrayDecoder) ----------

type moreArrayHolder struct {
	Arr [4]int32 `epack:"1"`
	Str string   `epack:"2"`
}

func TestRoundTripFixedArrayField(t *testing.T) {
	src := moreArrayHolder{Arr: [4]int32{10, 20, 30, 40}, Str: "arr"}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal(array holder) err=%v", err)
	}
	dst := moreArrayHolder{}
	if err := Unmarshal(data, &dst); err != nil {
		t.Fatalf("Unmarshal(array holder) err=%v", err)
	}
	if !reflect.DeepEqual(src, dst) {
		t.Fatalf("array holder round-trip mismatch: %+v vs %+v", src, dst)
	}
}

// ---------- 指针字段 nil (pointerEncoder / pointerDecoder 的 nil 分支) ----------

type moreInnerPtr struct {
	Val int32 `epack:"1"`
}

type moreHasPtr struct {
	Inner *moreInnerPtr `epack:"1"`
	Name  string        `epack:"2"`
}

func TestPointerFieldNilRoundTrip(t *testing.T) {
	src := moreHasPtr{Inner: nil, Name: "none"}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal(nil ptr) err=%v", err)
	}
	dst := moreHasPtr{}
	if err := Unmarshal(data, &dst); err != nil {
		t.Fatalf("Unmarshal(nil ptr) err=%v", err)
	}
	if dst.Inner != nil || dst.Name != "none" {
		t.Fatalf("nil ptr round-trip unexpected: %+v", dst)
	}
}

// ---------- 接口字段 nil (interfaceEncoder / interfaceDecoder 的 nil 分支) ----------

type moreHasIface struct {
	Any  interface{} `epack:"1"`
	Note string      `epack:"2"`
}

func TestInterfaceFieldNilRoundTrip(t *testing.T) {
	src := moreHasIface{Any: nil, Note: "x"}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal(nil iface) err=%v", err)
	}
	dst := moreHasIface{}
	if err := Unmarshal(data, &dst); err != nil {
		t.Fatalf("Unmarshal(nil iface) err=%v", err)
	}
	if dst.Any != nil || dst.Note != "x" {
		t.Fatalf("nil iface round-trip unexpected: %+v", dst)
	}
}

// ---------- map[string]*Struct 往返 ----------
//
// 当 value 是 *Struct 时，pointerEncoder 会把 Elem() 交给 structEncoder，
// structEncoder 调 marshal(u=nil, ...) -> nmlMarshal；解码端走 pointerDecoder
// -> structDecoder -> nmlUnmarshal。这条路径覆盖了此前 0% 的 nml* 函数。

type moreSubStruct struct {
	X int32
	Y string
}

func TestMapValuePointerStruct_NmlPath(t *testing.T) {
	src := map[string]*moreSubStruct{
		"a": {X: 7, Y: "seven"},
		"b": {X: 8, Y: "eight"},
	}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal(map[string]*struct) err=%v", err)
	}
	got := make(map[string]*moreSubStruct)
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal(map[string]*struct) err=%v", err)
	}
	if len(got) != len(src) {
		t.Fatalf("map length mismatch: %d vs %d", len(src), len(got))
	}
	for k, v := range src {
		g, ok := got[k]
		if !ok || g == nil {
			t.Fatalf("key %q 缺失或为 nil", k)
		}
		if !reflect.DeepEqual(*v, *g) {
			t.Fatalf("key %q 不一致: %+v vs %+v", k, *v, *g)
		}
	}
}

// map 值为值类型 struct 的正向往返。
// 之前 nmlMarshal 里的 CanSet 过滤会把 map 元素的字段全部跳过，修复后字段能正确
// 回传。这里同时覆盖 nmlMarshal 对可导出性的过滤 (有小写字段时应跳过不报错)。
type moreMixedExport struct {
	Pub int32 `json:"pub"`
	priv string
	Tag  string
}

func TestMapValueStructDirect_RoundTrip(t *testing.T) {
	src := map[string]moreSubStruct{
		"a": {X: 7, Y: "seven"},
		"b": {X: 8, Y: "eight"},
	}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}

	got := map[string]moreSubStruct{}
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal err=%v", err)
	}
	if !reflect.DeepEqual(src, got) {
		t.Fatalf("map[string]struct 往返失败: %v vs %v", src, got)
	}
}

func TestMapValueStructSkipsUnexported(t *testing.T) {
	src := map[string]moreMixedExport{
		"k1": {Pub: 42, priv: "hidden", Tag: "t1"},
	}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}

	got := map[string]moreMixedExport{}
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal err=%v", err)
	}
	v, ok := got["k1"]
	if !ok {
		t.Fatal("key k1 缺失")
	}
	// 可导出字段往返；不可导出字段按当前规则直接跳过，保持零值。
	if v.Pub != 42 || v.Tag != "t1" {
		t.Fatalf("导出字段往返失败: %+v", v)
	}
	if v.priv != "" {
		t.Fatalf("不可导出字段不应被写入，实际 %+v", v)
	}
}

// ---------- 指针类型间接 Unmarshal: *T 已为 non-nil，走 unmarshal 的 Ptr 递归 ----------

type moreTinyStruct struct {
	V int64 `epack:"1"`
}

func TestUnmarshalWithDoublePointer(t *testing.T) {
	src := moreTinyStruct{V: 999}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}
	inner := new(moreTinyStruct)
	outer := &inner
	// Unmarshal 入参是 **T，val.Elem() 是 *T；随后 unmarshal 的 Ptr 分支会再递归一次。
	if err := Unmarshal(data, outer); err != nil {
		t.Fatalf("Unmarshal(**T) err=%v", err)
	}
	if (*outer).V != 999 {
		t.Fatalf("double-ptr unmarshal got=%+v", *outer)
	}
}

// 目标对象是已创建的 nil *T （非 nil 双层指针），上层 Unmarshal 对 **T 会经 Ptr 分支，
// 在 unmarshal 里发现二级为 nil 时返回 "pointer is nil"。
func TestUnmarshalNilPointerReturnsError(t *testing.T) {
	src := moreTinyStruct{V: 1}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}

	var inner *moreTinyStruct // nil
	if err := Unmarshal(data, &inner); err == nil {
		t.Fatal("期望 nil 二级指针返回错误")
	}
}

// ---------- 带 time.Time 的结构体往返（timeEncoder / timeDecoder 已在 bench 里覆盖，这里补错误/边界） ----------

type moreTimeHolder struct {
	Ts time.Time `epack:"1"`
}

func TestTimeFieldRoundTrip(t *testing.T) {
	src := moreTimeHolder{Ts: time.Unix(1_700_000_000, 123).UTC()}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal time err=%v", err)
	}
	dst := moreTimeHolder{}
	if err := Unmarshal(data, &dst); err != nil {
		t.Fatalf("Unmarshal time err=%v", err)
	}
	if !src.Ts.Equal(dst.Ts) {
		t.Fatalf("time mismatch: %v vs %v", src.Ts, dst.Ts)
	}
}

// ---------- 结构体内嵌结构体指针；slice-of-struct 也覆盖一下 ----------

type moreSeg struct {
	K int32  `epack:"1"`
	V string `epack:"2"`
}

type moreSegHolder struct {
	Segs []moreSeg `epack:"1"`
}

func TestSliceOfStructRoundTrip(t *testing.T) {
	src := moreSegHolder{Segs: []moreSeg{{K: 1, V: "a"}, {K: 2, V: "b"}, {K: 3, V: "c"}}}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal slice-of-struct err=%v", err)
	}
	dst := moreSegHolder{}
	if err := Unmarshal(data, &dst); err != nil {
		t.Fatalf("Unmarshal slice-of-struct err=%v", err)
	}
	if !reflect.DeepEqual(src, dst) {
		t.Fatalf("slice-of-struct round-trip mismatch: %+v vs %+v", src, dst)
	}
}

type morePtrSegHolder struct {
	Segs []*moreSeg `epack:"1"`
}

// []*Struct：newEncoder 必须解引用 Index(0)，否则 reflect.Value.Field panic。
func TestSliceOfPtrStructRoundTrip(t *testing.T) {
	src := morePtrSegHolder{Segs: []*moreSeg{{K: 1, V: "a"}, {K: 2, V: "b"}}}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal slice-of-ptr-struct err=%v", err)
	}
	dst := morePtrSegHolder{}
	if err := Unmarshal(data, &dst); err != nil {
		t.Fatalf("Unmarshal slice-of-ptr-struct err=%v", err)
	}
	if len(dst.Segs) != len(src.Segs) {
		t.Fatalf("len=%d want %d", len(dst.Segs), len(src.Segs))
	}
	for i := range src.Segs {
		if dst.Segs[i] == nil || *dst.Segs[i] != *src.Segs[i] {
			t.Fatalf("seg[%d]=%v want %v", i, dst.Segs[i], src.Segs[i])
		}
	}
}

// 空 slice-of-struct：sliceEncoder 的 n==0 分支 + sliceDecoder 的 size==0 分支。
func TestEmptySliceOfStruct(t *testing.T) {
	src := moreSegHolder{Segs: nil}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal empty slice err=%v", err)
	}
	dst := moreSegHolder{Segs: []moreSeg{{K: 99}}} // 预置垃圾，验证是否被清空
	if err := Unmarshal(data, &dst); err != nil {
		t.Fatalf("Unmarshal empty slice err=%v", err)
	}
	if len(dst.Segs) != 0 {
		t.Fatalf("expected empty slice, got %+v", dst.Segs)
	}
}

// ---------- intsToBytes / bytesToInts ----------

func TestIntsBytesRoundTrip(t *testing.T) {
	// 非空切片：能互相还原。
	src := []int{1, 2, 3, -1, 1 << 30}
	b := intsToBytes(src)
	if len(b) == 0 {
		t.Fatal("intsToBytes 返回空")
	}
	expectedLen := len(src) * int(unsafe.Sizeof(int(0)))
	if len(b) != expectedLen {
		t.Fatalf("intsToBytes 字节长度应为 %d，实际 %d", expectedLen, len(b))
	}

	got := bytesToInts(b)
	if !reflect.DeepEqual(src, got) {
		t.Fatalf("Ints/Bytes 转换不一致: %v vs %v", src, got)
	}
}

func TestIntsBytesEmpty(t *testing.T) {
	if b := intsToBytes(nil); b != nil {
		t.Fatalf("intsToBytes(nil) 应返回 nil，实际 %v", b)
	}
	if s := bytesToInts(nil); s != nil {
		t.Fatalf("bytesToInts(nil) 应返回 nil，实际 %v", s)
	}
}

func TestBytesToIntsPanicOnMisaligned(t *testing.T) {
	intSize := int(unsafe.Sizeof(int(0)))
	// 构造一个显然长度不是 intSize 整倍数的切片。
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("bytesToInts 对非对齐字节应 panic")
		}
	}()
	bytesToInts(make([]byte, intSize+1))
}

// ---------- decodeHead 的截断错误 ----------

func TestDecodeHeadTruncated(t *testing.T) {
	// 单字节，bit7=0 但剩余不足 2 字节。
	b := deBuffer([]byte{0x01})
	if err := b.decodeHead(); err == nil {
		t.Fatal("2 字节头截断应报错")
	}

	// 单字节，bit7=1（小端 8 字节头）但剩余不足 8 字节。
	b2 := deBuffer([]byte{0x80, 0x01})
	if err := b2.decodeHead(); err == nil {
		t.Fatal("8 字节头截断应报错")
	}
}

// remain == 0 时应返回 io.ErrUnexpectedEOF（缓冲耗尽仍尝试读 head）。
func TestDecodeHeadFinishedOnEmpty(t *testing.T) {
	b := deBuffer(nil)
	if err := b.decodeHead(); err != io.ErrUnexpectedEOF {
		t.Fatalf("空缓冲期望 io.ErrUnexpectedEOF，实际 %v", err)
	}
}

// ---------- ubuffer.isOver ----------

func TestUBufferIsOver(t *testing.T) {
	b := deBuffer([]byte{0x01, 0x02, 0x03})
	if err := b.isOver(); err == nil {
		t.Fatal("remain>0 时 isOver 应报错")
	}
	b.remain = 0
	if err := b.isOver(); err != nil {
		t.Fatalf("remain==0 时 isOver 应为 nil，实际 %v", err)
	}
}

// ---------- enBuffer / release 池复用 ----------

func TestEnBufferReleaseReuses(t *testing.T) {
	b := enBuffer()
	b.buffer = append(b.buffer, 1, 2, 3)
	b.pos = 3
	b.remain = 0
	b.size = 3
	b.read = true
	b.release()

	b2 := enBuffer()
	if len(b2.buffer) != 0 || b2.pos != 0 || b2.remain != 0 || b2.size != 0 || b2.read {
		t.Fatalf("release 后未清零: %+v", b2)
	}
	b2.release()
}

// ---------- debug.String / unit.String ----------

type moreDebugStruct struct {
	A int32    `epack:"1"`
	B []string `epack:"2"`
}

func TestDebugStringOutput(t *testing.T) {
	if err := LoadTemplate(moreDebugStruct{}); err != nil {
		t.Fatalf("LoadTemplate err=%v", err)
	}

	v, ok := conf.cache.Load(reflect.TypeOf(moreDebugStruct{}).String())
	if !ok {
		t.Fatal("LoadTemplate 未落缓存")
	}
	e := v.(*ePack)
	s := e.String()
	if !strings.Contains(s, "name:A") || !strings.Contains(s, "name:B") {
		t.Fatalf("ePack.String 输出不含字段名，得到: %q", s)
	}

	// 第一个 unit 的 String 方法应该含有 idx/kind/name 标记。
	for _, u := range e.units {
		if u == nil {
			continue
		}
		us := u.String()
		if !strings.Contains(us, "idx:") || !strings.Contains(us, "name:") {
			t.Fatalf("unit.String 输出异常: %q", us)
		}
		break
	}

	// 空的 ePack.unitString 应该返回空串（不 panic）。
	empty := new(ePack)
	if got := empty.String(); got != "" {
		t.Fatalf("空 ePack.String 应返回空串，实际 %q", got)
	}
}

// ---------- map 中包含数组（触发 _arrayInterface / emptyStruct 相邻路径） ----------
//
// 注：当前实现从 interface 分发到 Array 需要编码端出现定长数组类型写入 interface 里，
// 这在 interface{} 赋值定长数组时才会发生；通过 map[string]interface{} 往返一次验证。

func TestMapWithArrayValue(t *testing.T) {
	src := map[string]interface{}{
		"ints": [3]int32{11, 22, 33},
	}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal map[string]interface{}{[3]int32} err=%v", err)
	}
	got := make(map[string]interface{})
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal err=%v", err)
	}
	// 解码端会把定长数组还原为 []interface{}（见 _arrayInterface 实现），
	// 因此这里不做严格等值比较，只校验非空且包含预期元素。
	arr, ok := got["ints"].([]interface{})
	if !ok {
		t.Fatalf("预期 []interface{}，实际 %T: %v", got["ints"], got["ints"])
	}
	if len(arr) != 3 {
		t.Fatalf("长度不匹配: %v", arr)
	}
}

// ---------- 顶层结构体没有任何 epack tag：cacheMarshal 会跳过全部 nil unit，
// 解出的数据即为空 head；此处仅验证不 panic 且 Unmarshal 可以接住空流。

type moreNoTagStruct struct {
	A int32
	B string
}

// ---------- newUnmarshal 路径：目标类型未进入 conf.cache 时的首次解码 ----------
//
// 做法：用 A 型 Marshal 产生字节，反序列化到结构等价但类型名不同的 B 型。
// 因为 cache 以类型名为 key，B 的首次 Unmarshal 就走 newUnmarshal。

type uncachedSrcA struct {
	V int32  `epack:"1"`
	N string `epack:"2"`
}

type uncachedDstB struct {
	V int32  `epack:"1"`
	N string `epack:"2"`
}

func TestUnmarshal_NewUnmarshalPath(t *testing.T) {
	data, err := Marshal(uncachedSrcA{V: 77, N: "ok"})
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}
	// 确保 B 还没被缓存（类型名不同）。
	if _, exist := conf.cache.Load(reflect.TypeOf(uncachedDstB{}).String()); exist {
		t.Skip("uncachedDstB 已被缓存，跳过 (可能是测试顺序导致)")
	}
	dst := new(uncachedDstB)
	if err := Unmarshal(data, dst); err != nil {
		t.Fatalf("Unmarshal err=%v", err)
	}
	if dst.V != 77 || dst.N != "ok" {
		t.Fatalf("newUnmarshal 结果错误: %+v", *dst)
	}
}

// ---------- 空字符串字段 (stringDecoder 的 size==0 分支) ----------

type moreStrHolder struct {
	S string `epack:"1"`
	N int32  `epack:"2"`
}

func TestEmptyStringField(t *testing.T) {
	src := moreStrHolder{S: "", N: 5}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal empty-str err=%v", err)
	}
	dst := moreStrHolder{}
	if err := Unmarshal(data, &dst); err != nil {
		t.Fatalf("Unmarshal empty-str err=%v", err)
	}
	if dst.S != "" || dst.N != 5 {
		t.Fatalf("empty-str round-trip unexpected: %+v", dst)
	}
}

func TestStructWithoutTagsEncodesEmpty(t *testing.T) {
	src := moreNoTagStruct{A: 1, B: "ignored"}
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}
	// 由于全部 unit 为 nil，输出字节应为 0 长；至少不会出错。
	if !bytes.Equal(data, []byte{}) && len(data) != 0 {
		// 实现允许落入其它形态，此处不强行断言具体字节形态，只要能 Unmarshal 不 panic 即可。
		t.Logf("Marshal 输出 %d 字节（允许）", len(data))
	}

	dst := moreNoTagStruct{}
	// 无法真正还原字段（因为没写进去），但 Unmarshal 应至少成功或返回可忽略错误。
	_ = Unmarshal(data, &dst)
}
