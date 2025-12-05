package apu

type InstructionFunc8 func(*CPU, byte, uint16) byte

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
