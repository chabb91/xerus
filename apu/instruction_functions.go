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
