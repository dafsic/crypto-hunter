package utils_test

import (
	utils "github.com/dafsic/crypto-hunter/utils"
)

import (
	"testing"
)

func TestSwitch(t *testing.T) {
	var s utils.Switch

	// go func() {
	// 	for i := 0; i < 10; i++ {
	// 		s.On()
	// 		time.Sleep(1)
	// 		s.Off()
	// 		time.Sleep(1)
	// 	}
	// }()

	s.On()
	if s.State() != utils.On {
		t.Error(s.State())
	}

}
