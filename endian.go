package epack

import (
	"encoding/binary"
	"math"
	"reflect"
	"unsafe"
)

var (
	localEndian bool
	sizeofType  = func(t reflect.Type) uintptr { return t.Size() }
)

func init() {
	var x uint16 = 0x0102
	localEndian = *(*byte)(unsafe.Pointer(&x)) == 0x02
}

// canBulkNumberCopy 本机 LE 且元素为定宽数字（不含 bool）时可整段拷贝。
func canBulkNumberCopy(ek reflect.Kind) bool {
	if !localEndian {
		return false
	}

	switch ek {
	case reflect.Int8, reflect.Uint8,
		reflect.Int16, reflect.Uint16,
		reflect.Int32, reflect.Uint32, reflect.Float32,
		reflect.Int64, reflect.Uint64, reflect.Float64,
		reflect.Int, reflect.Uint:
		return true
	default:
		return false
	}
}

// numberSliceBytes 将连续数字切片底层视为 []byte；不可寻址时 ok=false。
func numberSliceBytes(v reflect.Value) (raw []byte, ok bool) {
	n := v.Len()
	if n == 0 {
		return nil, true
	}
	elem := v.Index(0)
	if !elem.CanAddr() {
		return nil, false
	}
	es := int(v.Type().Elem().Size())
	return unsafe.Slice((*byte)(elem.Addr().UnsafePointer()), n*es), true
}

// appendNumberHeadAndPayload 写入类型头 + 小端数值载荷。
func appendNumberHeadAndPayload(b *ubuffer, v reflect.Value) {
	b.copyBytes(conf.cTH[v.Kind()])
	appendNumberPayloadLE(b, v)
}

// appendNumberPayloadLE 将数值小端载荷直接写入 ubuffer，避免中间 make。
func appendNumberPayloadLE(b *ubuffer, v reflect.Value) {
	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			b.buffer = append(b.buffer, 1)
		} else {
			b.buffer = append(b.buffer, 0)
		}
	case reflect.Int8:
		b.buffer = append(b.buffer, byte(int8(v.Int())))
	case reflect.Uint8:
		b.buffer = append(b.buffer, byte(v.Uint()))
	case reflect.Int16:
		binary.LittleEndian.PutUint16(b.grow(2), uint16(v.Int()))
	case reflect.Uint16:
		binary.LittleEndian.PutUint16(b.grow(2), uint16(v.Uint()))
	case reflect.Int32:
		binary.LittleEndian.PutUint32(b.grow(4), uint32(v.Int()))
	case reflect.Uint32:
		binary.LittleEndian.PutUint32(b.grow(4), uint32(v.Uint()))
	case reflect.Float32:
		binary.LittleEndian.PutUint32(b.grow(4), math.Float32bits(float32(v.Float())))
	case reflect.Int64:
		binary.LittleEndian.PutUint64(b.grow(8), uint64(v.Int()))
	case reflect.Uint64:
		binary.LittleEndian.PutUint64(b.grow(8), v.Uint())
	case reflect.Float64:
		binary.LittleEndian.PutUint64(b.grow(8), math.Float64bits(v.Float()))
	case reflect.Int:
		if sizeofType(v.Type()) == 8 {
			binary.LittleEndian.PutUint64(b.grow(8), uint64(v.Int()))
		} else {
			binary.LittleEndian.PutUint32(b.grow(4), uint32(v.Int()))
		}
	case reflect.Uint:
		if sizeofType(v.Type()) == 8 {
			binary.LittleEndian.PutUint64(b.grow(8), v.Uint())
		} else {
			binary.LittleEndian.PutUint32(b.grow(4), uint32(v.Uint()))
		}
	}
}

// numberPayloadLE 将本机数值编码为固定小端字节（不含类型头）。
// 仅用于测试/诊断；热路径请用 appendNumberPayloadLE。
func numberPayloadLE(v reflect.Value) []byte {
	b := &ubuffer{buffer: make([]byte, 0, 8)}
	appendNumberPayloadLE(b, v)
	if len(b.buffer) == 0 {
		return nil
	}
	return b.buffer
}

// setNumberFromLE 从小端载荷写回本机数值。
func setNumberFromLE(t reflect.Value, buf []byte) error {
	if !t.IsValid() {
		return errInvalidNumberValue
	}

	dst := t
	if !t.CanSet() {
		// 不可导出/不可寻址字段：写入临时值后丢弃（与历史行为一致，避免 panic）。
		dst = reflect.New(t.Type()).Elem()
	}

	return writeNumberLE(dst, buf)
}

func writeNumberLE(t reflect.Value, buf []byte) error {
	switch t.Kind() {
	case reflect.Bool:
		if len(buf) < 1 {
			return errShortNumPayload
		}
		t.SetBool(buf[0] != 0)
	case reflect.Int8:
		if len(buf) < 1 {
			return errShortNumPayload
		}
		t.SetInt(int64(int8(buf[0])))
	case reflect.Uint8:
		if len(buf) < 1 {
			return errShortNumPayload
		}
		t.SetUint(uint64(buf[0]))
	case reflect.Int16:
		if len(buf) < 2 {
			return errShortNumPayload
		}
		t.SetInt(int64(int16(binary.LittleEndian.Uint16(buf))))
	case reflect.Uint16:
		if len(buf) < 2 {
			return errShortNumPayload
		}
		t.SetUint(uint64(binary.LittleEndian.Uint16(buf)))
	case reflect.Int32:
		if len(buf) < 4 {
			return errShortNumPayload
		}
		t.SetInt(int64(int32(binary.LittleEndian.Uint32(buf))))
	case reflect.Uint32:
		if len(buf) < 4 {
			return errShortNumPayload
		}
		t.SetUint(uint64(binary.LittleEndian.Uint32(buf)))
	case reflect.Float32:
		if len(buf) < 4 {
			return errShortNumPayload
		}
		t.SetFloat(float64(math.Float32frombits(binary.LittleEndian.Uint32(buf))))
	case reflect.Int64:
		if len(buf) < 8 {
			return errShortNumPayload
		}
		t.SetInt(int64(binary.LittleEndian.Uint64(buf)))
	case reflect.Uint64:
		if len(buf) < 8 {
			return errShortNumPayload
		}
		t.SetUint(binary.LittleEndian.Uint64(buf))
	case reflect.Float64:
		if len(buf) < 8 {
			return errShortNumPayload
		}
		t.SetFloat(math.Float64frombits(binary.LittleEndian.Uint64(buf)))
	case reflect.Int:
		if sizeofType(t.Type()) == 8 {
			if len(buf) < 8 {
				return errShortNumPayload
			}
			t.SetInt(int64(binary.LittleEndian.Uint64(buf)))
		} else {
			if len(buf) < 4 {
				return errShortNumPayload
			}
			t.SetInt(int64(int32(binary.LittleEndian.Uint32(buf))))
		}
	case reflect.Uint:
		if sizeofType(t.Type()) == 8 {
			if len(buf) < 8 {
				return errShortNumPayload
			}
			t.SetUint(binary.LittleEndian.Uint64(buf))
		} else {
			if len(buf) < 4 {
				return errShortNumPayload
			}
			t.SetUint(uint64(binary.LittleEndian.Uint32(buf)))
		}
	default:
		return errBadNumberKind
	}
	return nil
}
