package cpu

type CPU struct {
	r registers

	E bool
}

func (c *CPU) Reset() {
	// set emulation flag
	c.E = true

	c.r.PB = 0x00
	c.r.DB = 0x00
	c.r.D = 0x0000

	// the default stack head in emulation mode
	c.r.S = 0x01FF

	// set M X and I to 1
	c.r.P = 0x34

	//TODO
	// 3. Read the 16-bit Reset Vector from the bus
	/*
		resetVectorLow := uint16(c.bus.ReadByte(0x00FFFC))
		resetVectorHigh := uint16(c.bus.ReadByte(0x00FFFD))
		c.PC = (resetVectorHigh << 8) | resetVectorLow
	*/
}
