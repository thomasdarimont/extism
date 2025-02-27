package extism

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime/cgo"
	"unsafe"
)

/*
#cgo CFLAGS: -I/usr/local/include
#cgo LDFLAGS: -L/usr/local/lib -lextism
#include <extism.h>
#include <stdlib.h>

int64_t extism_val_i64(ExtismValUnion* x){
	return x->i64;
}

int32_t extism_val_i32(ExtismValUnion* x){
	return x->i32;
}

float extism_val_f32(ExtismValUnion* x){
	return x->f32;
}

double extism_val_f64(ExtismValUnion* x){
	return x->f64;
}


void extism_val_set_i64(ExtismValUnion* x, int64_t i){
	x->i64 = i;
}


void extism_val_set_i32(ExtismValUnion* x, int32_t i){
	x->i32 = i;
}

void extism_val_set_f32(ExtismValUnion* x, float f){
	x->f32 = f;
}

void extism_val_set_f64(ExtismValUnion* x, double f){
	x->f64 = f;
}

*/
import "C"

type ValType = C.ExtismValType

type Val = C.ExtismVal

type Size = C.ExtismSize

var (
	I32       ValType = C.I32
	I64       ValType = C.I64
	F32       ValType = C.F32
	F64       ValType = C.F64
	V128      ValType = C.V128
	FuncRef   ValType = C.FuncRef
	ExternRef ValType = C.ExternRef
)

// Function is used to define host functions
type Function struct {
	pointer  *C.ExtismFunction
	userData cgo.Handle
}

// Free a function
func (f *Function) Free() {
	if f.pointer != nil {
		C.extism_function_free(f.pointer)
		f.pointer = nil
		f.userData.Delete()
	}
}

// NewFunction creates a new host function with the given name, input/outputs and optional user data, which can be an
// arbitrary `interface{}`
func NewFunction(name string, inputs []ValType, outputs []ValType, f unsafe.Pointer, userData interface{}) Function {
	var function Function
	function.userData = cgo.NewHandle(userData)
	cname := C.CString(name)
	ptr := unsafe.Pointer(function.userData)
	var inputsPtr *C.ExtismValType = nil
	if len(inputs) > 0 {
		inputsPtr = (*C.ExtismValType)(&inputs[0])
	}
	var outputsPtr *C.ExtismValType = nil
	if len(outputs) > 0 {
		outputsPtr = (*C.ExtismValType)(&outputs[0])
	}
	function.pointer = C.extism_function_new(
		cname,
		inputsPtr,
		C.uint64_t(len(inputs)),
		outputsPtr,
		C.uint64_t(len(outputs)),
		(*[0]byte)(f),
		ptr,
		nil,
	)
	C.free(unsafe.Pointer(cname))
	return function
}

func (f *Function) SetNamespace(s string) {
	cstr := C.CString(s)
	defer C.free(unsafe.Pointer(cstr))
	C.extism_function_set_namespace(f.pointer, cstr)
}

func (f Function) WithNamespace(s string) Function {
	f.SetNamespace(s)
	return f
}

type CurrentPlugin struct {
	pointer *C.ExtismCurrentPlugin
}

func GetCurrentPlugin(ptr unsafe.Pointer) CurrentPlugin {
	return CurrentPlugin{
		pointer: (*C.ExtismCurrentPlugin)(ptr),
	}
}

type MemoryHandle = uint

func (p *CurrentPlugin) Memory(offs MemoryHandle) []byte {
	length := C.extism_current_plugin_memory_length(p.pointer, C.uint64_t(offs))
	data := unsafe.Pointer(C.extism_current_plugin_memory(p.pointer))
	return unsafe.Slice((*byte)(unsafe.Add(data, offs)), C.int(length))
}

// Alloc a new memory block of the given length, returning its offset
func (p *CurrentPlugin) Alloc(n uint) MemoryHandle {
	return uint(C.extism_current_plugin_memory_alloc(p.pointer, C.uint64_t(n)))
}

// Free the memory block specified by the given offset
func (p *CurrentPlugin) Free(offs MemoryHandle) {
	C.extism_current_plugin_memory_free(p.pointer, C.uint64_t(offs))
}

// Length returns the number of bytes allocated at the specified offset
func (p *CurrentPlugin) Length(offs MemoryHandle) int {
	return int(C.extism_current_plugin_memory_length(p.pointer, C.uint64_t(offs)))
}

// Plugin is used to call WASM functions
type Plugin struct {
	ptr       *C.ExtismPlugin
	functions []Function
}

type WasmData struct {
	Data []byte `json:"data"`
	Hash string `json:"hash,omitempty"`
	Name string `json:"name,omitempty"`
}

type WasmFile struct {
	Path string `json:"path"`
	Hash string `json:"hash,omitempty"`
	Name string `json:"name,omitempty"`
}

type WasmUrl struct {
	Url     string            `json:"url"`
	Hash    string            `json:"hash,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Name    string            `json:"name,omitempty"`
	Method  string            `json:"method,omitempty"`
}

type Wasm interface{}

type Manifest struct {
	Wasm   []Wasm `json:"wasm"`
	Memory struct {
		MaxPages uint32 `json:"max_pages,omitempty"`
	} `json:"memory,omitempty"`
	Config       map[string]string `json:"config,omitempty"`
	AllowedHosts []string          `json:"allowed_hosts,omitempty"`
	AllowedPaths map[string]string `json:"allowed_paths,omitempty"`
	Timeout      uint              `json:"timeout_ms,omitempty"`
}

func makePointer(data []byte) unsafe.Pointer {
	var ptr unsafe.Pointer = nil
	if len(data) > 0 {
		ptr = unsafe.Pointer(&data[0])
	}
	return ptr
}

// SetLogFile sets the log file and level, this is a global setting
func SetLogFile(filename string, level string) bool {
	name := C.CString(filename)
	l := C.CString(level)
	r := C.extism_log_file(name, l)
	C.free(unsafe.Pointer(name))
	C.free(unsafe.Pointer(l))
	return bool(r)
}

// ExtismVersion gets the Extism version string
func ExtismVersion() string {
	return C.GoString(C.extism_version())
}

func register(data []byte, functions []Function, wasi bool) (Plugin, error) {
	ptr := makePointer(data)
	functionPointers := []*C.ExtismFunction{}
	for _, f := range functions {
		functionPointers = append(functionPointers, f.pointer)
	}

	var plugin *C.ExtismPlugin
	errmsg := (*C.char)(nil)
	if len(functions) == 0 {
		plugin = C.extism_plugin_new(
			(*C.uchar)(ptr),
			C.uint64_t(len(data)),
			nil,
			0,
			C._Bool(wasi),
			&errmsg)
	} else {
		plugin = C.extism_plugin_new(
			(*C.uchar)(ptr),
			C.uint64_t(len(data)),
			&functionPointers[0],
			C.uint64_t(len(functions)),
			C._Bool(wasi),
			&errmsg,
		)
	}

	if plugin == nil {
		msg := C.GoString(errmsg)
		C.extism_plugin_new_error_free(errmsg)
		return Plugin{}, errors.New(
			fmt.Sprintf("Unable to load plugin: %s", msg),
		)
	}

	return Plugin{ptr: plugin, functions: functions}, nil
}

// NewPlugin creates a plugin
func NewPlugin(module io.Reader, functions []Function, wasi bool) (Plugin, error) {
	wasm, err := io.ReadAll(module)
	if err != nil {
		return Plugin{}, err
	}

	return register(wasm, functions, wasi)
}

// NewPlugin creates a plugin from a manifest
func NewPluginFromManifest(manifest Manifest, functions []Function, wasi bool) (Plugin, error) {
	data, err := json.Marshal(manifest)
	if err != nil {
		return Plugin{}, err
	}

	return register(data, functions, wasi)
}

// Set configuration values
func (plugin Plugin) SetConfig(data map[string][]byte) error {
	if plugin.ptr == nil {
		return errors.New("Cannot set config, Plugin already freed")
	}
	s, err := json.Marshal(data)
	if err != nil {
		return err
	}
	ptr := makePointer(s)
	C.extism_plugin_config(plugin.ptr, (*C.uchar)(ptr), C.uint64_t(len(s)))
	return nil
}

// FunctionExists returns true when the named function is present in the plugin
func (plugin Plugin) FunctionExists(functionName string) bool {
	if plugin.ptr == nil {
		return false
	}
	name := C.CString(functionName)
	b := C.extism_plugin_function_exists(plugin.ptr, name)
	C.free(unsafe.Pointer(name))
	return bool(b)
}

// Call a function by name with the given input, returning the output
func (plugin Plugin) Call(functionName string, input []byte) ([]byte, error) {
	if plugin.ptr == nil {
		return []byte{}, errors.New("Plugin has already been freed")
	}
	ptr := makePointer(input)
	name := C.CString(functionName)
	rc := C.extism_plugin_call(
		plugin.ptr,
		name,
		(*C.uchar)(ptr),
		C.uint64_t(len(input)),
	)
	C.free(unsafe.Pointer(name))

	if rc != 0 {
		err := C.extism_plugin_error(plugin.ptr)
		msg := "<unset by plugin>"
		if err != nil {
			msg = C.GoString(err)
		}

		return nil, errors.New(
			fmt.Sprintf("Plugin error: %s, code: %d", msg, rc),
		)
	}

	length := C.extism_plugin_output_length(plugin.ptr)

	if length > 0 {
		x := C.extism_plugin_output_data(plugin.ptr)
		return unsafe.Slice((*byte)(x), C.int(length)), nil
	}

	return []byte{}, nil
}

// Free a plugin
func (plugin *Plugin) Free() {
	if plugin.ptr == nil {
		return
	}
	C.extism_plugin_free(plugin.ptr)
	plugin.ptr = nil
}

// ValGetI64 returns an I64 from an ExtismVal, it accepts a pointer to a C.ExtismVal
func ValGetI64(v unsafe.Pointer) int64 {
	return int64(C.extism_val_i64(&(*Val)(v).v))
}

// ValGetUInt returns a uint from an ExtismVal, it accepts a pointer to a C.ExtismVal
func ValGetUInt(v unsafe.Pointer) uint {
	return uint(C.extism_val_i64(&(*Val)(v).v))
}

// ValGetI32 returns an int32 from an ExtismVal, it accepts a pointer to a C.ExtismVal
func ValGetI32(v unsafe.Pointer) int32 {
	return int32(C.extism_val_i32(&(*Val)(v).v))
}

// ValGetF32 returns a float32 from an ExtismVal, it accepts a pointer to a C.ExtismVal
func ValGetF32(v unsafe.Pointer) float32 {
	return float32(C.extism_val_f32(&(*Val)(v).v))
}

// ValGetF32 returns a float64 from an ExtismVal, it accepts a pointer to a C.ExtismVal
func ValGetF64(v unsafe.Pointer) float64 {
	return float64(C.extism_val_i64(&(*Val)(v).v))
}

// ValSetI64 stores an int64 in an ExtismVal, it accepts a pointer to a C.ExtismVal and the new value
func ValSetI64(v unsafe.Pointer, i int64) {
	C.extism_val_set_i64(&(*Val)(v).v, C.int64_t(i))
}

// ValSetI32 stores an int32 in an ExtismVal, it accepts a pointer to a C.ExtismVal and the new value
func ValSetI32(v unsafe.Pointer, i int32) {
	C.extism_val_set_i32(&(*Val)(v).v, C.int32_t(i))
}

// ValSetF32 stores a float32 in an ExtismVal, it accepts a pointer to a C.ExtismVal and the new value
func ValSetF32(v unsafe.Pointer, i float32) {
	C.extism_val_set_f32(&(*Val)(v).v, C.float(i))
}

// ValSetF64 stores a float64 in an ExtismVal, it accepts a pointer to a C.ExtismVal and the new value
func ValSetF64(v unsafe.Pointer, f float64) {
	C.extism_val_set_f64(&(*Val)(v).v, C.double(f))
}

func (p *CurrentPlugin) ReturnBytes(v unsafe.Pointer, b []byte) {
	mem := p.Alloc(uint(len(b)))
	ptr := p.Memory(mem)
	copy(ptr, b)
	ValSetI64(v, int64(mem))
}

func (p *CurrentPlugin) ReturnString(v unsafe.Pointer, s string) {
	p.ReturnBytes(v, []byte(s))
}

func (p *CurrentPlugin) InputBytes(v unsafe.Pointer) []byte {
	return p.Memory(ValGetUInt(v))
}

func (p *CurrentPlugin) InputString(v unsafe.Pointer) string {
	return string(p.InputBytes(v))
}

type CancelHandle struct {
	pointer *C.ExtismCancelHandle
}

func (p *Plugin) CancelHandle() CancelHandle {
	pointer := C.extism_plugin_cancel_handle(p.ptr)
	return CancelHandle{pointer}
}

func (c *CancelHandle) Cancel() bool {
	return bool(C.extism_plugin_cancel(c.pointer))
}
