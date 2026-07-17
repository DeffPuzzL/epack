package epack

import (
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

const (
	TYPE_ENUM   = 32
	SIZE_MALLOC = 8192 // 8K
)

const SIMPLE_NUMBER byte = 0xFF

type Config struct {
	malloc int
	cache  sync.Map
	cTH    [][]byte
	rType  []reflect.Type
	eFunc  []coderFunc
	dFunc  []coderFunc
}

var (
	errObjectNil          = errors.New("epack: got nil")
	errElementNil         = errors.New("epack: Marshal(nil pointer)")
	errObjectNotStruct    = errors.New("epack: LoadTemplate(non-struct)")
	errObjectMustPointer  = errors.New("epack: Unmarshal(non-pointer)")
	errPointerNil         = errors.New("epack: Unmarshal(nil pointer)")
	errBufferNotFinished  = errors.New("epack: invalid trailing data after top-level value")
	errUnsupportedType    = errors.New("epack: unsupported type")
	errShortHead          = errors.New("epack: unexpected end of epack input while reading head")
	errShortString        = errors.New("epack: unexpected end of epack input while reading string")
	errShortTime          = errors.New("epack: unexpected end of epack input while reading time")
	errShortNumber        = errors.New("epack: unexpected end of epack input while reading number")
	errShortNumSliceHdr   = errors.New("epack: unexpected end of epack input while reading number-slice header")
	errShortNumSlice      = errors.New("epack: unexpected end of epack input while reading number-slice body")
	errShortNumPayload    = errors.New("epack: unexpected end of epack input while reading number payload")
	errBadNumSliceElem    = errors.New("epack: unsupported type: number slice element")
	errBadNumberKind      = errors.New("epack: unsupported type: number")
	errStructInvalid      = errors.New("epack: cannot decode into invalid struct value")
	errInvalidNumberValue = errors.New("epack: unsupported value: number")
	errBadArrayLen        = errors.New("epack: cannot unmarshal array: length mismatch")
	errBadTimeSize        = errors.New("epack: cannot unmarshal time: expected 8-byte payload")
	errBadTagIndex        = errors.New("epack: invalid epack tag index")
)

var (
	conf  *Config
	gPool *sync.Pool
)

type coderFunc func(u []*Unit, b *ubuffer, v reflect.Value) error

func init() {
	// 线格式固定小端，不再探测/配置本机端序。
	conf = &Config{
		malloc: SIZE_MALLOC,
		cache:  sync.Map{},
		rType:  make([]reflect.Type, TYPE_ENUM),
		cTH:    make([][]byte, TYPE_ENUM),
		eFunc:  make([]coderFunc, TYPE_ENUM),
		dFunc:  make([]coderFunc, TYPE_ENUM),
	}

	gPool = &sync.Pool{
		New: func() interface{} {
			return &ubuffer{
				buffer: make([]byte, 0, conf.malloc),
			}
		},
	}

	conf.initialize()
}

func (c *Config) initialize() {
	for i := reflect.Invalid; i < TYPE_ENUM; i++ {
		switch i {
		case reflect.String:
			conf.eFunc[i] = stringEncoder
			conf.dFunc[i] = stringDecoder
			conf.rType[i] = reflect.TypeOf("")
			conf.cTH[i] = encodeHead(uint64(i), 0)
		case reflect.Int:
			conf.eFunc[i] = numberEncoder
			conf.dFunc[i] = numberDecoder
			conf.rType[i] = reflect.TypeOf(int(0))
			conf.cTH[i] = encodeHead(uint64(i), uint64(unsafe.Sizeof(int(0))))
		case reflect.Int8:
			conf.eFunc[i] = numberEncoder
			conf.dFunc[i] = numberDecoder
			conf.rType[i] = reflect.TypeOf(int8(0))
			conf.cTH[i] = encodeHead(uint64(i), uint64(unsafe.Sizeof(int8(0))))
		case reflect.Int16:
			conf.eFunc[i] = numberEncoder
			conf.dFunc[i] = numberDecoder
			conf.rType[i] = reflect.TypeOf(int16(0))
			conf.cTH[i] = encodeHead(uint64(i), uint64(unsafe.Sizeof(int16(0))))
		case reflect.Int32:
			conf.eFunc[i] = numberEncoder
			conf.dFunc[i] = numberDecoder
			conf.rType[i] = reflect.TypeOf(int32(0))
			conf.cTH[i] = encodeHead(uint64(i), uint64(unsafe.Sizeof(int32(0))))
		case reflect.Int64:
			conf.eFunc[i] = numberEncoder
			conf.dFunc[i] = numberDecoder
			conf.rType[i] = reflect.TypeOf(int64(0))
			conf.cTH[i] = encodeHead(uint64(i), uint64(unsafe.Sizeof(int64(0))))
		case reflect.Uint:
			conf.eFunc[i] = numberEncoder
			conf.dFunc[i] = numberDecoder
			conf.rType[i] = reflect.TypeOf(uint(0))
			conf.cTH[i] = encodeHead(uint64(i), uint64(unsafe.Sizeof(uint(0))))
		case reflect.Uint8:
			conf.eFunc[i] = numberEncoder
			conf.dFunc[i] = numberDecoder
			conf.rType[i] = reflect.TypeOf(uint8(0))
			conf.cTH[i] = encodeHead(uint64(i), uint64(unsafe.Sizeof(uint8(0))))
		case reflect.Uint16:
			conf.eFunc[i] = numberEncoder
			conf.dFunc[i] = numberDecoder
			conf.rType[i] = reflect.TypeOf(uint16(0))
			conf.cTH[i] = encodeHead(uint64(i), uint64(unsafe.Sizeof(uint16(0))))
		case reflect.Uint32:
			conf.eFunc[i] = numberEncoder
			conf.dFunc[i] = numberDecoder
			conf.rType[i] = reflect.TypeOf(uint32(0))
			conf.cTH[i] = encodeHead(uint64(i), uint64(unsafe.Sizeof(uint32(0))))
		case reflect.Uint64:
			conf.eFunc[i] = numberEncoder
			conf.dFunc[i] = numberDecoder
			conf.rType[i] = reflect.TypeOf(uint64(0))
			conf.cTH[i] = encodeHead(uint64(i), uint64(unsafe.Sizeof(uint64(0))))
		case reflect.Float32:
			conf.eFunc[i] = numberEncoder
			conf.dFunc[i] = numberDecoder
			conf.rType[i] = reflect.TypeOf(float32(0))
			conf.cTH[i] = encodeHead(uint64(i), uint64(unsafe.Sizeof(float32(0))))
		case reflect.Float64:
			conf.eFunc[i] = numberEncoder
			conf.dFunc[i] = numberDecoder
			conf.rType[i] = reflect.TypeOf(float64(0))
			conf.cTH[i] = encodeHead(uint64(i), uint64(unsafe.Sizeof(float64(0))))
		case reflect.Slice:
			conf.eFunc[i] = sliceEncoder
			conf.dFunc[i] = sliceDecoder
			conf.rType[i] = reflect.TypeOf([]interface{}{})
			conf.cTH[i] = encodeHead(uint64(i), 0)
		case reflect.Array:
			conf.eFunc[i] = arrayEncoder
			conf.dFunc[i] = arrayDecoder
			conf.rType[i] = reflect.TypeOf([]interface{}{})
			conf.cTH[i] = encodeHead(uint64(i), 0)
		case reflect.Map:
			conf.eFunc[i] = mapEncoder
			conf.dFunc[i] = mapDecoder
			conf.rType[i] = reflect.TypeOf(map[interface{}]interface{}{})
			conf.cTH[i] = encodeHead(uint64(i), 0)
		case reflect.Interface:
			conf.eFunc[i] = interfaceEncoder
			conf.dFunc[i] = interfaceDecoder
			conf.rType[i] = reflect.TypeOf((*interface{})(nil)).Elem()
			conf.cTH[i] = encodeHead(uint64(i), 0)
		case reflect.Ptr:
			conf.eFunc[i] = pointerEncoder
			conf.dFunc[i] = pointerDecoder
			conf.rType[i] = reflect.TypeOf((*interface{})(nil))
			conf.cTH[i] = encodeHead(uint64(i), 0)
		case reflect.Struct:
			conf.eFunc[i] = structEncoder
			conf.dFunc[i] = structDecoder
			conf.rType[i] = reflect.TypeOf(struct{}{})
			conf.cTH[i] = encodeHead(uint64(i), 0)
		case reflect.Bool:
			conf.eFunc[i] = numberEncoder
			conf.dFunc[i] = numberDecoder
			conf.rType[i] = reflect.TypeOf(false)
			conf.cTH[i] = encodeHead(uint64(i), uint64(unsafe.Sizeof(bool(false))))
		default:
			conf.eFunc[i] = nil
			conf.dFunc[i] = nil
			conf.cTH[i] = nil
			conf.rType[i] = reflect.TypeOf(nil)
		}
	}
}

func (c *Config) getType(ek reflect.Kind) (reflect.Type, error) {
	if ek >= TYPE_ENUM {
		return nil, errUnsupportedType
	}

	return conf.rType[ek], nil
}

// new8bHead 编码 8 字节小端头：byte0 = 0x80|type，其后 7 字节为 size（小端，56 位）。
func new8bHead(t, s uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, (s<<8)|(0x80|(t&0x1F)))
	return buf
}

// new2bHead 编码 2 字节小端头：byte0 低 5 位 type、bit5-6 为 size 高 2 位（bit7=0），byte1 为 size 低 8 位。
func new2bHead(t, s uint64) []byte {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, uint16((t&0x1F)|(((s>>8)&0x3)<<5)|((s&0xFF)<<8)))
	return buf
}

func cacheTH(t reflect.Kind) []byte {
	return conf.cTH[t]
}

type Unit struct {
	seq     int
	level   int
	kind    reflect.Kind
	name    string
	encoder coderFunc
	decoder coderFunc
	child   []*Unit
}

type EPack struct {
	units []*Unit
}

type ubuffer struct {
	buffer []byte
	remain uint64
	pos    uint64
	read   bool
	kind   reflect.Kind
	size   uint64
}

func isSliceStruct(t reflect.Type) (reflect.Type, bool) {
	if t.Kind() != reflect.Slice {
		return nil, false
	}

	e := t.Elem()
	if e.Kind() == reflect.Ptr {
		e = e.Elem()
	}

	return e, e.Kind() == reflect.Struct
}

func enBuffer() *ubuffer {
	return gPool.Get().(*ubuffer)
}

func (b *ubuffer) release() {
	b.buffer = b.buffer[:0]
	b.pos = 0
	b.size = 0
	b.remain = 0
	b.read = false

	gPool.Put(b)

}

func (b *ubuffer) copyBytes(data []byte) {
	b.buffer = append(b.buffer, data...)
}

func (b *ubuffer) grow(n int) []byte {
	off := len(b.buffer)
	if cap(b.buffer)-off < n {
		nb := make([]byte, off, off+n+conf.malloc)
		copy(nb, b.buffer)
		b.buffer = nb
	}

	b.buffer = b.buffer[:off+n]
	return b.buffer[off : off+n]
}

func deBuffer(buffer []byte) *ubuffer {
	return &ubuffer{
		buffer: buffer,
		remain: uint64(len(buffer)),
	}
}

func (b *ubuffer) walk(size uint64) {
	b.pos += size
	b.remain -= size
}

func (b *ubuffer) isExceeded() bool {
	return b.remain < b.size
}

func (b *ubuffer) isOver() error {
	if b.remain == 0 {
		return nil
	}

	return errBufferNotFinished
}

func IntsToBytes(ints []int) []byte {
	if len(ints) == 0 {
		return nil
	}

	intSize := int(unsafe.Sizeof(int(0)))
	totalSize := len(ints) * intSize

	intPtr := unsafe.Pointer(&ints[0])

	return *(*[]byte)(unsafe.Pointer(&struct {
		addr uintptr
		len  int
		cap  int
	}{
		addr: uintptr(intPtr),
		len:  totalSize,
		cap:  totalSize,
	}))
}

func BytesToInts(bytes []byte) []int {
	if len(bytes) == 0 {
		return nil
	}

	intSize := int(unsafe.Sizeof(int(0)))
	if len(bytes)%intSize != 0 {
		panic(fmt.Sprintf("byte length must be multiple of %d", intSize))
	}

	count := len(bytes) / intSize
	bytePtr := unsafe.Pointer(&bytes[0])

	return *(*[]int)(unsafe.Pointer(&struct {
		addr uintptr
		len  int
		cap  int
	}{
		addr: uintptr(bytePtr),
		len:  count,
		cap:  count,
	}))
}
