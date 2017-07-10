package stats

import (
	"math"
	"sync/atomic"
	"unsafe"
)

type f64 uint64

func (a *f64) ptr() *uint64 {
	return (*uint64)(unsafe.Pointer(a))
}

func (a *f64) float() float64 {
	return math.Float64frombits(atomic.LoadUint64(a.ptr()))
}

func (a *f64) add(f float64) float64 {
	for {
		p := a.ptr()
		v := atomic.LoadUint64(p)
		x := math.Float64frombits(v) + f

		if atomic.CompareAndSwapUint64(p, v, math.Float64bits(x)) {
			return x
		}
	}
}

func (a *f64) set(f float64) (value float64, delta float64) {
	for {
		p := a.ptr()
		v := atomic.LoadUint64(p)
		x := math.Float64frombits(v)

		if atomic.CompareAndSwapUint64(p, v, math.Float64bits(f)) {
			return f, f - x
		}
	}
}
