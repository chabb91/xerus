package muldivchip

// these are CPU cycles. they are supposed to be 16/8 but i cant get that to work with chrono trigger at least.
const divCycleCount = 16 / 2
const mulCycleCount = 8 / 2

type MulDiv struct {
	Wrmpya, wrmpyb         byte // multiplicands a and b
	Wrdivl, Wrdivh, wrdivb byte // dividend low dividend high divisor
	Rddivl, Rddivh         byte // quotient of divide low and high
	Rdmpyl, Rdmpyh         byte // divide remainder or multiplication product

	divDelay int
	mulDelay int
}

func NewMulDiv() *MulDiv {
	return &MulDiv{
		Wrdivl: 0xFF,
		Wrdivh: 0xFF,
		Wrmpya: 0xFF}
}

func (md *MulDiv) SetMultiplicandB(mulB byte) {
	md.wrmpyb = mulB
	md.mulDelay = mulCycleCount
}

func (md *MulDiv) SetDivisorB(divB byte) {
	md.wrdivb = divB
	md.divDelay = divCycleCount
}

func (md *MulDiv) StepCycle() {
	if md.divDelay > 0 {
		md.divDelay--
		if md.divDelay == 0 {
			if md.wrdivb != 0 {
				divA := createWord(md.Wrdivh, md.Wrdivl)
				md.Rddivh, md.Rddivl = splitWord(divA / uint16(md.wrdivb))
				md.Rdmpyh, md.Rdmpyl = splitWord(divA % uint16(md.wrdivb))
			} else {
				md.Rddivh = 0xFF
				md.Rddivl = 0xFF
				md.Rdmpyh = md.Wrdivh
				md.Rdmpyl = md.Wrdivl
			}
		}
	}
	if md.mulDelay > 0 {
		md.mulDelay--
		if md.mulDelay == 0 {
			md.Rdmpyh, md.Rdmpyl = splitWord(uint16(md.Wrmpya) * uint16(md.wrmpyb))
		}
	}
}

func createWord(high, low byte) uint16 {
	return (uint16(high) << 8) | uint16(low)
}

func splitWord(word uint16) (high, low byte) {
	high = byte(word >> 8)
	low = byte(word & 0xFF)

	return high, low
}
