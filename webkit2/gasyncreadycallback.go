package webkit2

// #include <stdlib.h>
// #include <gio/gio.h>
// #include "gasyncreadycallback.go.h"
import "C"
import (
	"errors"
	"reflect"
	"unsafe"
	"sync"
)

type garCallback struct {
	f reflect.Value
}

var (
	// CGo no longer allows pointers with pointers to Go structures to be passed to C.
	// Instead, we use an arbitrary pointer and a lookup table.
	// The arbitrary pointer is allocated to ensure uniqueness, but not actually used.
	callbackMap map[C.gpointer]*garCallback
	// Locking to ensure thread-safety.
	callbackMapMutex sync.RWMutex
)

func init() {
	callbackMap = make(map[C.gpointer]*garCallback)
}

//export _go_gasyncreadycallback_call
func _go_gasyncreadycallback_call(cbinfoRaw C.gpointer, cresult unsafe.Pointer) {
	result := (*C.GAsyncResult)(cresult)
	callbackMapMutex.Lock()
	cbinfo, exists := callbackMap[cbinfoRaw]
	delete(callbackMap, cbinfoRaw)
	callbackMapMutex.Unlock()
	// If a value existed in our lookup table, free the allocated pointer, and call the required callback.
	if exists {
		C.free(unsafe.Pointer(cbinfoRaw))
		cbinfo.f.Call([]reflect.Value{reflect.ValueOf(result)})
	}
}

func newGAsyncReadyCallback(f interface{}) (cCallback C.GAsyncReadyCallback, userData C.gpointer, err error) {
	rf := reflect.ValueOf(f)
	if rf.Kind() != reflect.Func {
		return nil, nil, errors.New("f is not a function")
	}
	cbinfo := &garCallback{rf}
	// Allocate some memory to assure a unique pointer
	ptr := C.gpointer(C.malloc(4))
	callbackMapMutex.Lock()
	callbackMap[ptr] = cbinfo
	callbackMapMutex.Unlock()
	return C.GAsyncReadyCallback(C._gasyncreadycallback_call), ptr, nil
}
