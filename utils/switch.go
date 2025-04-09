package utils

import "sync/atomic"

type Switch int32

const (
	Off Switch = 0
	On  Switch = 1
)

func (s *Switch) On() {
	atomic.CompareAndSwapInt32((*int32)(s), 0, 1)
}

func (s *Switch) Off() {
	atomic.CompareAndSwapInt32((*int32)(s), 1, 0)
}

func (s *Switch) State() Switch {
	return Switch(atomic.LoadInt32((*int32)(s)))
}
