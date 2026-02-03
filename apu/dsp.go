package apu

type DSP struct {
	psram                        SPCMemory
	state                        int
	samples                      [16]int16
	brrBlock                     []byte
	brrBlockEnd                  bool
	hist1, hist2                 int32
	samplePointer                byte
	brrChanStart, brrChanRestart uint16
	brrBlockPointer              *uint16
}

func NewDsp() *DSP {
	return &DSP{
		brrBlock: make([]byte, 9),
	}
}

func (dsp *DSP) Step() {
	if dsp.state < 31 {
		dsp.state++
		return
	}
	if dsp.samplePointer&0xF == 0 {
		dsp.state = 0
		if dsp.brrBlockPointer == nil {
			brrAddr := uint16(dsp.psram.dspRegs[0x5D])<<8 | uint16(dsp.psram.dspRegs[0x04])
			dsp.brrChanStart = uint16(dsp.psram.ram[brrAddr+1])<<8 | uint16(dsp.psram.ram[brrAddr])
			dsp.brrChanRestart = uint16(dsp.psram.ram[brrAddr+3])<<8 | uint16(dsp.psram.ram[brrAddr+2])
			dsp.brrBlockPointer = &dsp.brrChanStart
			copy(dsp.brrBlock, dsp.psram.ram[*dsp.brrBlockPointer:])
			dsp.brrBlockEnd = dsp.decodeBRRBlock()
			*dsp.brrBlockPointer += 9
		}
	}
	if dsp.samplePointer == 0xF {
		if dsp.brrBlockEnd {
			dsp.psram.dspRegs[0x7C] |= 1
			dsp.brrBlockPointer = &dsp.brrChanRestart
			copy(dsp.brrBlock, dsp.psram.ram[*dsp.brrBlockPointer:])
			dsp.brrBlockEnd = dsp.decodeBRRBlock()
			*dsp.brrBlockPointer += 9
		} else {
			dsp.brrBlockPointer = nil
		}
	}

	//dsp.samples[dsp.samplePointer] //send this to oto
	dsp.samplePointer = (dsp.samplePointer + 1) & 0xF
}

func (dsp *DSP) decodeBRRBlock() bool {
	header := dsp.brrBlock[0]
	shift := header >> 4
	filter := (header >> 2) & 0x03
	end := (header & 0x01) != 0

	data := dsp.brrBlock[1:]

	for i := range 16 {
		var nibble byte
		if i&1 == 0 {
			nibble = data[i>>1] >> 4
		} else {
			nibble = data[i>>1] & 0x0F
		}

		sample := signExtend4(nibble) << shift

		switch filter {
		case 0:
			// no filter
		case 1:
			sample += dsp.hist1 * 15 / 16
		case 2:
			sample += dsp.hist1*61/32 - dsp.hist2*15/16
		case 3:
			sample += dsp.hist1*115/64 - dsp.hist2*13/16
		}

		if sample > 32767 {
			sample = 32767
		} else if sample < -32768 {
			sample = -32768
		}

		dsp.samples[i] = int16(sample)
		dsp.hist2 = dsp.hist1
		dsp.hist1 = sample
	}

	return end
}
