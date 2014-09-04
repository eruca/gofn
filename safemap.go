package main

import (
	"errors"
	"sync"
	"unsafe"
)

//safemap---------------------------------
type safemap struct {
	smap map[unsafe.Pointer]unsafe.Pointer
	*sync.RWMutex
}

func new_safemap() *safemap {
	return &safemap{
		make(map[unsafe.Pointer]unsafe.Pointer),
		new(sync.RWMutex),
	}
}

func (this *safemap) query(key unsafe.Pointer) unsafe.Pointer {
	if this == nil || this.smap == nil {
		panic("the safemap is nil")
	}

	this.RLock()
	defer this.RUnlock()
	if v, ok := this.smap[key]; ok {
		return v
	}
	return nil
}

func (this *safemap) insert(key, value unsafe.Pointer) (isUpdate bool, err error) {
	if this == nil || this.smap == nil {
		return false, errors.New("the safemap is nil")
	}

	this.Lock()
	defer this.Unlock()

	if _, ok := this.smap[key]; ok {
		isUpdate = true
	}

	this.smap[key] = value
	return isUpdate, nil
}
