package apu

type InstructionFunc8 func(*CPU, byte, uint16) byte
type InstructionFunc8x2 func(*CPU, byte, byte, uint16, uint16) byte

func tclr1(cpu *CPU, val byte, addr uint16) byte {
	cpu.psram.Read8(addr) //dummy read

	result := int16(cpu.r.A) - int16(val)
	cpu.r.setFlag(FlagZ, result != 0)
	cpu.r.setFlag(FlagN, (result&0x80) == 0)
	val &= ^cpu.r.A

	return val
}

func tset1(cpu *CPU, val byte, addr uint16) byte {
	cpu.psram.Read8(addr) //dummy read

	result := int16(cpu.r.A) - int16(val)
	cpu.r.setFlag(FlagZ, result != 0)
	cpu.r.setFlag(FlagN, (result&0x80) == 0)
	val |= cpu.r.A

	return val
}

func asl(cpu *CPU, val byte, _ uint16) byte {
	resetCarry := val&0x80 == 0

	val <<= 1
	cpu.r.setFlag(FlagZ, val != 0)
	cpu.r.setFlag(FlagN, (val&0x80) == 0)
	cpu.r.setFlag(FlagC, resetCarry)

	return val
}

func lsr(cpu *CPU, val byte, _ uint16) byte {
	resetCarry := val&1 == 0

	val >>= 1
	cpu.r.setFlag(FlagZ, val != 0)
	cpu.r.setFlag(FlagN, (val&0x80) == 0)
	cpu.r.setFlag(FlagC, resetCarry)

	return val
}

func rol(cpu *CPU, val byte, _ uint16) byte {
	resetCarry := val&0x80 == 0

	val <<= 1
	if cpu.r.hasFlag(FlagC) {
		val |= 1
	} else {
		val &= 0xFE
	}
	cpu.r.setFlag(FlagZ, val != 0)
	cpu.r.setFlag(FlagN, (val&0x80) == 0)
	cpu.r.setFlag(FlagC, resetCarry)

	return val
}

func ror(cpu *CPU, val byte, _ uint16) byte {
	resetCarry := val&1 == 0

	val >>= 1
	if cpu.r.hasFlag(FlagC) {
		val |= 0x80
	} else {
		val &= 0x7F
	}
	cpu.r.setFlag(FlagZ, val != 0)
	cpu.r.setFlag(FlagN, (val&0x80) == 0)
	cpu.r.setFlag(FlagC, resetCarry)

	return val
}

func inc(cpu *CPU, val byte, _ uint16) byte {
	val++
	cpu.r.setFlag(FlagZ, val != 0)
	cpu.r.setFlag(FlagN, (val&0x80) == 0)
	return val
}

func dec(cpu *CPU, val byte, _ uint16) byte {
	val--
	cpu.r.setFlag(FlagZ, val != 0)
	cpu.r.setFlag(FlagN, (val&0x80) == 0)
	return val
}

func and(cpu *CPU, val1, val2 byte, _, _ uint16) byte {
	val1 &= val2
	cpu.r.setFlag(FlagZ, val1 != 0)
	cpu.r.setFlag(FlagN, (val1&0x80) == 0)
	return val1
}

func or(cpu *CPU, val1, val2 byte, _, _ uint16) byte {
	val1 |= val2
	cpu.r.setFlag(FlagZ, val1 != 0)
	cpu.r.setFlag(FlagN, (val1&0x80) == 0)
	return val1
}

func eor(cpu *CPU, val1, val2 byte, _, _ uint16) byte {
	val1 ^= val2
	cpu.r.setFlag(FlagZ, val1 != 0)
	cpu.r.setFlag(FlagN, (val1&0x80) == 0)
	return val1
}

func adc(cpu *CPU, val1, val2 byte, _, _ uint16) byte {
	carryIn := cpu.r.PSW & FlagC

	result16 := adcFlagSetter(cpu, val1, val2, carryIn)
	cpu.r.setFlag(FlagC, result16 <= 0xFF)

	return byte(result16)
}

func sbc(cpu *CPU, val1, val2 byte, _, _ uint16) byte {
	carryIn := (cpu.r.PSW + 1) & FlagC
	result16 := uint16(val1) - (uint16(val2) + uint16(carryIn))
	result8 := byte(result16)

	// apparently for any integer A represented in twos complement form this holds true:
	// -A = ~A +1 or in this case: ~A = -A -1
	adcFlagSetter(cpu, val1, ^val2, 1-carryIn)
	cpu.r.setFlag(FlagC, result16 > 0xFF)

	return result8
}

func adcFlagSetter(cpu *CPU, val1, val2, carryIn byte) uint16 {
	result16 := uint16(val1) + uint16(val2) + uint16(carryIn)
	result8 := byte(result16)

	tmp1 := (val1 & 0x0F) + carryIn
	halfCarry := (((result16 & 0x0F) - uint16(tmp1)) & 0x10)
	overflow := (^(val1 ^ val2)) & ((val1 ^ result8) & 0x80) // set when signs of inputs match but result sign differs

	cpu.r.setFlag(FlagV, overflow == 0)
	cpu.r.setFlag(FlagH, halfCarry == 0)
	cpu.r.setFlag(FlagZ, result8 != 0)
	cpu.r.setFlag(FlagN, (result8&0x80) == 0)

	return result16
}

func cmp(cpu *CPU, val1, val2 byte, _, _ uint16) byte {
	result16 := int16(val1) - int16(val2)
	result8 := byte(result16)

	cpu.r.setFlag(FlagC, result16 < 0)
	cpu.r.setFlag(FlagZ, result8 != 0)
	cpu.r.setFlag(FlagN, (result8&0x80) == 0)
	return val1
}

func movNoFlag(_ *CPU, _, val2 byte, _, _ uint16) byte {
	return val2
}

func movNoFlagInverse(_ *CPU, val1, _ byte, _, _ uint16) byte {
	return val1
}

func mov(cpu *CPU, _, val2 byte, _, _ uint16) byte {
	cpu.r.setFlag(FlagZ, val2 != 0)
	cpu.r.setFlag(FlagN, (val2&0x80) == 0)
	return val2
}
