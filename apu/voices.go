package apu

type envelopeState int

const (
	IDLE envelopeState = iota
	ATTACK
	DECAY
	SUSTAIN
	RELEASE
)

type Voice struct {
	id     int
	idMask byte
	state  envelopeState
	regs   *[DSP_REG_SIZE]byte
	ram    *[PSRAM_SIZE]byte

	envLevel int // 0-2047

	buffer                       [16]int16
	bufferIndex                  int
	hist1, hist2                 int32
	brrBlockPointer              uint16
	brrStartAddr, brrRestartAddr uint16
}

func newVoice(id int, dspRegs *[DSP_REG_SIZE]byte, psram *[PSRAM_SIZE]byte) *Voice {
	return &Voice{
		id:     id,
		idMask: 1 << id,
		regs:   dspRegs,
		ram:    psram,
	}
}

func (v *Voice) Tick() int16 {
	if v.state == IDLE || v.state == RELEASE {
		return 0
	}

	/*
		v.updateEnvelope()
	*/
	if v.bufferIndex >= 16 {
		v.decodeNextBRRBlock()
	}

	sample := v.buffer[v.bufferIndex]
	v.bufferIndex++

	//return int16((int32(sample) * int32(v.envLevel)) >> 11)
	return sample
}

func signExtend4(n byte) int32 {
	v := int32(n & 0xF)
	if v&0x8 != 0 {
		v |= ^int32(0xF)
	}
	return v
}

func (v *Voice) keyOn() {
	v.state = ATTACK

	v.regs[0x7C] &= ^v.idMask

	brrAddr := uint16(v.regs[0x5D])<<8 | uint16(v.regs[v.id<<4|0x04])
	v.brrStartAddr = uint16(v.ram[brrAddr+1])<<8 | uint16(v.ram[brrAddr])
	v.brrRestartAddr = uint16(v.ram[brrAddr+3])<<8 | uint16(v.ram[brrAddr+2])

	v.brrBlockPointer = v.brrStartAddr
	//fmt.Println("VOICE ", v.index, " POINTER: ", v.brrBlockPointer)
	v.decodeNextBRRBlock()
}

func (v *Voice) keyOff() {
	v.state = RELEASE
}

func (v *Voice) decodeNextBRRBlock() {
	v.bufferIndex = 0
	brrBlock := v.ram[v.brrBlockPointer : v.brrBlockPointer+9]
	header := brrBlock[0]
	shift := header >> 4
	filter := (header >> 2) & 0x03
	end := (header & 0x01) != 0
	loop := (header & 0x02) != 0

	for i := range 16 {
		nibble := brrBlock[(i>>1)+1]
		if i&1 == 0 {
			nibble >>= 4
		} else {
			nibble &= 0x0F
		}

		sample := signExtend4(nibble) << shift

		switch filter {
		case 1:
			sample += v.hist1 + (-v.hist1 >> 4)
		case 2:
			sample += v.hist1<<1 + (-(v.hist1<<1 + v.hist1) >> 5) - v.hist2 + (v.hist2 >> 4)
		case 3:
			sample += v.hist1<<1 + (-(v.hist1 + v.hist1<<2 + v.hist1<<3) >> 6) - v.hist2 + ((v.hist2<<1 + v.hist2) >> 4)
		}

		if sample > 32767 {
			sample = 32767
		}
		if sample < -32768 {
			sample = -32768
		}

		v.buffer[i] = int16(sample)
		v.hist2 = v.hist1
		v.hist1 = sample
	}

	if end {
		if loop {
			v.brrBlockPointer = v.brrRestartAddr
		} else {
			v.state = IDLE
			v.regs[0x7C] |= v.idMask
		}
	} else {
		v.brrBlockPointer += 9
	}
}
