package apu

type Timer struct {
	stage1Counter byte // Stage 1: Prescaler (128 for T0/T1, 16 for T2)
	stage1Rate    byte // 128 or 16

	stage2Counter byte // Stage 2: Divisor (0-255, wraps)
	target        byte // TnTARGET register value

	output byte // Stage 3: 4-bit output counter (TnOUT)

	enabled bool
}

func NewTimer(rate byte) *Timer {
	return &Timer{
		stage1Rate: rate,
		output:     0xF,
	}
}

func (t *Timer) Tick() {
	t.stage1Counter++
	if t.stage1Counter >= t.stage1Rate {
		t.stage1Counter = 0

		if t.enabled {
			t.stage2Counter++

			if t.stage2Counter == t.target {
				t.output = (t.output + 1) & 0xF
				t.stage2Counter = 0
			}
		}
	}
}

func (t *Timer) ReadOutput() byte {
	val := t.output
	t.output = 0
	return val
}

func (t *Timer) SetControl() {
	t.stage2Counter = 0
	t.output = 0
}
