package timer

import (
	"testing"
	"time"
)

func Test_Timer(t *testing.T) {
	t.Run("timer1", func(t *testing.T) {
		timer1()
	})
	t.Run("timer2_123", func(t *testing.T) {
		timer2(time.Second*1, time.Second*2, time.Second*3)
	})
	t.Run("timer2_321", func(t *testing.T) {
		timer2(time.Second*3, time.Second*2, time.Second*1)
	})

}
