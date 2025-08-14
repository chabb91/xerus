package cpu

func trb(val uint16, width int, cpu *CPU) (result uint16) {
	result = val
	for v := range width {
		if (cpu.r.A>>v)&1 == 1 {
			result &= ^(1 << v)
		}
	}
	cpu.r.setFlag(FlagZ, (val&cpu.r.A) != 0)
	return result
}

func tsb(val uint16, width int, cpu *CPU) (result uint16) {
	result = val
	for v := range width {
		if (cpu.r.A>>v)&1 == 1 {
			result |= 1 << v
		}
	}
	cpu.r.setFlag(FlagZ, (val&cpu.r.A) != 0)
	return result
}
