package apu

import (
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
)

// Number of samples per counter event
var counter_rates = [32]int{
	0, 2048, 1536,
	1280, 1024, 768,
	640, 512, 384,
	320, 256, 192,
	160, 128, 96,
	80, 64, 48,
	40, 32, 24,
	20, 16, 12,
	10, 8, 6,
	5, 4, 3,
	2,
	1,
}

// Counter offset from zero (i.e. not all counters are aligned at zero for all rates)
var counter_offsets = [32]int{
	0, 0, 1040,
	536, 0, 1040,
	536, 0, 1040,
	536, 0, 1040,
	536, 0, 1040,
	536, 0, 1040,
	536, 0, 1040,
	536, 0, 1040,
	536, 0, 1040,
	536, 0, 1040,
	0,
	0,
}

const DSP_REG_SIZE = 0x80

type dspReg byte

const (
	//the 10 per voice register masks
	VxVolL   dspReg = 0x00 //Left volume for Vx
	VxVolR   dspReg = 0x01 //Right volume for Vx
	VxPitchL dspReg = 0x02 //Pitch scaler low byte for Vx
	VxPitchH dspReg = 0x03 //Pitch scaler high byte for Vx
	VxScrn   dspReg = 0x04 //Source number for Vx
	VxAdsr1  dspReg = 0x05 //ADSR part 1 for Vx
	VxAdsr2  dspReg = 0x06 //ADSR part 2 for Vx
	VxGain   dspReg = 0x07 //GAIN for Vx
	VxEnvX   dspReg = 0x08 //The current envelope value for Vx
	VxOutX   dspReg = 0x09 //The current sample value for Vx
	//general purpose registers
	MVolL dspReg = 0x0C //Left channel master volume
	MVolR dspReg = 0x1C //Right channel master volume
	EVolL dspReg = 0x2C //Left channel echo volume
	EVolR dspReg = 0x3C //Right channel echo volume
	KOn   dspReg = 0x4C //Key on for all voices
	KOff  dspReg = 0x5C //Key off for all voices
	FLG   dspReg = 0x6C //Reset, Mute, Echo-Write flags and Noise Clock
	EndX  dspReg = 0x7C //Voice end flags
	EFB   dspReg = 0x0D //Echo feedback volume
	PMOn  dspReg = 0x2D //Pitch modulation enable
	NOn   dspReg = 0x3D //Noise enable
	EOn   dspReg = 0x4D //Echo enable
	DIR   dspReg = 0x5D //Sample table address
	ESA   dspReg = 0x6D //Echo ring buffer address
	EDL   dspReg = 0x7D //Echo delay (ring buffer size)
	FFCx  dspReg = 0x0F //FIR filter coefficient for Vx
)

type DSPInterface interface {
	ReadRegister(reg byte) byte
	WriteRegister(reg byte, val byte)
}

type DspRegisters struct {
	registers [DSP_REG_SIZE]byte

	newKOn  byte //the internal KON latch (when the register is written)
	KOn     byte //the currently tracked internal KON value
	KOff    byte
	poll16k bool

	NON, EON, DIR byte
	ESA           byte
}

type DSP struct {
	state int

	counter int

	noiseRate      byte
	noiseSampleRaw uint16
	noiseSample    int16

	echoBufferIdx, echoBufferAddress uint16
	echoBufferSize, echoPointer      uint16
	fir                              fir
	ecenx                            bool // flg bit 5

	echoBufferL, echoBufferR int16

	outL, outR   int32
	echoL, echoR int32

	ram *[PSRAM_SIZE]byte

	DspRegisters

	Buffer

	Voices [8]*Voice
}

func NewDsp(psram *SPCMemory) *DSP {
	dsp := &DSP{
		noiseSampleRaw: 0x4000,
		//Buffer:         newRingBuffer(11),
		Buffer: newAtomicBuffer(11),
		ram:    &psram.ram,
	}
	for i := range len(dsp.Voices) {
		dsp.Voices[i] = newVoice(i, &dsp.DspRegisters, &psram.ram)
		dsp.Voices[i].envelope.advanceEnvelope = dsp.rateEvent
		dsp.Voices[i].currentNoiseSample = &dsp.noiseSample

		if i > 0 {
			dsp.Voices[i].prevVoiceOut = &dsp.Voices[i-1].voiceOut
		}
	}

	//TODO this is not the right way to do this but
	//echo samples cant be generated during the boot sequence no matter what
	dsp.registers[FLG] = 0x20

	return dsp
}

func (dsp *DSP) Step() {
	switch dsp.state {
	case 0:
		dsp.Voices[0].S5()
		dsp.Voices[1].S2()
	case 1:
		dsp.Voices[0].S6()
		dsp.Voices[1].S3()
	case 2:
		dsp.Voices[0].S7()
		dsp.Voices[1].S4()
		dsp.Voices[3].S1()
	case 3:
		dsp.Voices[0].S8()
		dsp.Voices[1].S5()
		dsp.Voices[2].S2()
	case 4:
		dsp.Voices[0].S9()
		dsp.Voices[1].S6()
		dsp.Voices[2].S3()
	case 5:
		dsp.Voices[1].S7()
		dsp.Voices[2].S4()
		dsp.Voices[4].S1()
	case 6:
		dsp.Voices[1].S8()
		dsp.Voices[2].S5()
		dsp.Voices[3].S2()
	case 7:
		dsp.Voices[1].S9()
		dsp.Voices[2].S6()
		dsp.Voices[3].S3()
	case 8:
		dsp.Voices[2].S7()
		dsp.Voices[3].S4()
		dsp.Voices[5].S1()
	case 9:
		dsp.Voices[2].S8()
		dsp.Voices[3].S5()
		dsp.Voices[4].S2()
	case 10:
		dsp.Voices[2].S9()
		dsp.Voices[3].S6()
		dsp.Voices[4].S3()
	case 11:
		dsp.Voices[3].S7()
		dsp.Voices[4].S4()
		dsp.Voices[6].S1()
	case 12:
		dsp.Voices[3].S8()
		dsp.Voices[4].S5()
		dsp.Voices[5].S2()
	case 13:
		dsp.Voices[3].S9()
		dsp.Voices[4].S6()
		dsp.Voices[5].S3()
	case 14:
		dsp.Voices[4].S7()
		dsp.Voices[5].S4()
		dsp.Voices[7].S1()
	case 15:
		dsp.Voices[4].S8()
		dsp.Voices[5].S5()
		dsp.Voices[6].S2()
	case 16:
		dsp.Voices[4].S9()
		dsp.Voices[5].S6()
		dsp.Voices[6].S3()
	case 17:
		dsp.Voices[0].S1()
		dsp.Voices[5].S7()
		dsp.Voices[6].S4()
	case 18:
		dsp.Voices[5].S8()
		dsp.Voices[6].S5()
		dsp.Voices[7].S2()
	case 19:
		dsp.Voices[5].S9()
		dsp.Voices[6].S6()
		dsp.Voices[7].S3()
	case 20:
		dsp.Voices[1].S1()
		dsp.Voices[6].S7()
		dsp.Voices[7].S4()
	case 21:
		dsp.Voices[0].S2()
		dsp.Voices[6].S8()
		dsp.Voices[7].S5()
	case 22:
		dsp.Voices[0].S3a()
		dsp.Voices[6].S9()
		dsp.Voices[7].S6()

		//TODO unnecessay var
		dsp.echoBufferAddress = uint16(dsp.registers[ESA]) << 8
		dsp.echoPointer = dsp.echoBufferAddress + dsp.echoBufferIdx<<2

		idx := dsp.echoPointer
		dsp.echoBufferL = int16(uint16(dsp.ram[idx+0])|uint16(dsp.ram[idx+1])<<8) >> 1 // <<1

		dsp.latchFirCoeffAt(0)
	case 23:
		dsp.Voices[7].S7()

		idx := dsp.echoPointer
		dsp.echoBufferR = int16(uint16(dsp.ram[idx+2])|uint16(dsp.ram[idx+3])<<8) >> 1 // <<1

		dsp.latchFirCoeffAt(1)
		dsp.latchFirCoeffAt(2)
	case 24:
		dsp.Voices[7].S8()

		dsp.latchFirCoeffAt(3)
		dsp.latchFirCoeffAt(4)
		dsp.latchFirCoeffAt(5)
	case 25:
		dsp.Voices[0].S3b()
		dsp.Voices[7].S9()

		dsp.latchFirCoeffAt(6)
		dsp.latchFirCoeffAt(7)
	case 26:
		dsp.outL, dsp.outR = 0, 0
		dsp.echoL, dsp.echoR = 0, 0

		for _, v := range dsp.Voices {
			dsp.outL = clamp16(dsp.outL + v.sampleL)
			dsp.outR = clamp16(dsp.outR + v.sampleR)
			if dsp.EON&v.idMask != 0 {
				dsp.echoL = clamp16(dsp.echoL + v.sampleL)
				dsp.echoR = clamp16(dsp.echoR + v.sampleR)
			}
		}

		dsp.fir.getSample(dsp)
		efb := int32(int8(dsp.registers[EFB]))
		dsp.echoBufferL = int16(clamp16(((dsp.fir.left*efb)>>7)+dsp.echoL)) &^ 1
		dsp.echoBufferR = int16(clamp16(((dsp.fir.right*efb)>>7)+dsp.echoR)) &^ 1

		dsp.echoL = (int32(dsp.fir.left) * int32(int8(dsp.registers[EVolL])) >> 7)
		dsp.echoR = (int32(dsp.fir.right) * int32(int8(dsp.registers[EVolR])) >> 7)

		dsp.outL = (dsp.outL * int32(int8(dsp.registers[MVolL])) >> 7)
		dsp.outR = (dsp.outR * int32(int8(dsp.registers[MVolR])) >> 7)
	case 27:
		if dsp.registers[FLG]&0x40 == 0 {
			dsp.Buffer.Write(int16(clamp16(dsp.outL+dsp.echoL)), int16(clamp16(dsp.outR+dsp.echoR)))
		} else {
			dsp.Buffer.Write(0, 0)
		}

		//TODO not like this
		pmon := dsp.registers[PMOn]
		for _, v := range dsp.Voices {
			v.pmon = pmon&v.idMask > 1
		}
	case 28:
		dsp.NON = dsp.registers[NOn]
		dsp.EON = dsp.registers[EOn]
		dsp.DIR = dsp.registers[DIR]
		for i := range 8 {
			//TODO useNoiseSample is not necessary, this check should be done by each voice
			dsp.Voices[i].useNoiseSample = dsp.NON&dsp.Voices[i].idMask != 0
		}
		dsp.updateEcenx()
	case 29:
		//dsp.counter = (dsp.counter - 1) & 0x77FF
		dsp.counter--
		if dsp.counter < 0 {
			dsp.counter = 0x77FF
		}

		if !dsp.ecenx {
			idx := dsp.echoPointer
			dsp.ram[idx+0], dsp.ram[idx+1] = byte(uint16(dsp.echoBufferL)), byte(uint16(dsp.echoBufferL)>>8)
		}

		if dsp.echoBufferIdx == 0 {
			dsp.echoBufferSize = uint16(dsp.registers[EDL]&0xF) << 9
		}
		dsp.ESA = dsp.registers[ESA]
		dsp.updateEcenx()
		if dsp.poll16k {
			dsp.newKOn &^= dsp.KOn
		}
	case 30:
		dsp.poll16k = dsp.poll16k != true

		dsp.Voices[0].S3c()

		if !dsp.ecenx {
			idx := dsp.echoPointer
			dsp.ram[idx+2], dsp.ram[idx+3] = byte(uint16(dsp.echoBufferR)), byte(uint16(dsp.echoBufferR)>>8)
		}

		dsp.echoBufferIdx++
		if dsp.echoBufferIdx >= dsp.echoBufferSize {
			dsp.echoBufferIdx = 0
		}
		dsp.noiseRate = dsp.registers[FLG] & 0x1F
		if dsp.rateEvent(dsp.noiseRate) {
			N := uint(dsp.noiseSampleRaw) //just storing the generated noise bits as unsigned
			dsp.noiseSampleRaw = uint16((N>>1)|(((N<<14)^(N<<13))&0x4000)) & 0x7FFF
			dsp.noiseSample = int16(dsp.noiseSampleRaw << 1)
		}
		if dsp.poll16k {
			dsp.KOn = dsp.newKOn
			dsp.KOff = dsp.registers[KOff]
		}
	case 31:
		dsp.Voices[0].S4()
		dsp.Voices[2].S1()

		dsp.state = 0
		return
	}
	dsp.state++
	/*

		if dsp.state <= 31 {
			if dsp.state == 1 {
				dsp.poll16k = dsp.poll16k != true
			}
			if dsp.state == 29 {
				//dsp.counter = (dsp.counter - 1) & 0x77FF
				dsp.counter--
				if dsp.counter < 0 {
					dsp.counter = 0x77FF
				}
				if dsp.poll16k {
					dsp.newKOn &^= dsp.KOn
				}
			}
			if dsp.state == 30 {
				dsp.noiseRate = dsp.registers[FLG] & 0x1F
				if dsp.poll16k {
					dsp.KOn = dsp.newKOn
					dsp.KOff = dsp.registers[KOff]
				}
			}
			if dsp.state == 28 {
				dsp.NON = dsp.registers[NOn]
				dsp.EON = dsp.registers[EOn]
				dsp.DIR = dsp.registers[DIR]
				for i := range 8 {
					dsp.Voices[i].useNoiseSample = dsp.NON&dsp.Voices[i].idMask != 0
				}
				if dsp.rateEvent(dsp.noiseRate) {
					N := uint(dsp.noiseSampleRaw) //just storing the generated noise bits as unsigned
					dsp.noiseSampleRaw = uint16((N>>1)|(((N<<14)^(N<<13))&0x4000)) & 0x7FFF
					dsp.noiseSample = int16(dsp.noiseSampleRaw << 1)
				}
			}
				if dsp.state == 14 {
					for _, v := range dsp.Voices {
						id := v.idReg
						v.envelope.setAdsr1(dsp.registers[id|VxAdsr1])
						v.envelope.setAdsr2(dsp.registers[id|VxAdsr2])
						v.envelope.setGain(dsp.registers[id|VxGain])
					}
				}

			if dsp.state == 23 {
				for _, v := range dsp.Voices {
					id := v.idReg
					dsp.registers[id|VxEnvX] = byte(v.envelope.level >> 4)
				}
			}
			if dsp.state == 24 {
				//dsp.echoBufferSize = uint16(dsp.registers[EDL]&0xF) << 9
				dsp.echoBufferAddress = uint16(dsp.registers[ESA]) << 8
				//fmt.Println(dsp.echoBufferAddress, "      ", dsp.registers[ESA])
			}
			if dsp.state == 27 {
				pmon := dsp.registers[PMOn]
				for _, v := range dsp.Voices {
					v.pmon = pmon&v.idMask > 1
				}
			}
			return
		}

		var outL, outR, echoL, echoR int32
		for _, v := range dsp.Voices {
			v.S1()
			v.S2()
			v.S3a()
			v.S3b()
			v.S3c()
			v.S4()
			v.S5()
			v.S6()
			v.S7()
			v.S8()
			v.S9()

			//	out := int32(v.Tick()) //>> 1
			//	leftChan := (out * int32(int8(dsp.registers[v.idReg|VxVolL]))) >> 7
			//	rightChan := (out * int32(int8(dsp.registers[v.idReg|VxVolR]))) >> 7
			outL = clamp16(outL + v.sampleL)
			outR = clamp16(outR + v.sampleR)
			if dsp.registers[EOn]&v.idMask != 0 {
				echoL = clamp16(echoL + v.sampleL)
				echoR = clamp16(echoR + v.sampleR)
			}
		}

		firL, firR := dsp.fir.getSample(dsp)

		efb := int32(int8(dsp.registers[EFB]))
		dsp.writeEchoBuffer(int16(clamp16(((firL*efb)>>7)+echoL))&^1,
			int16(clamp16(((firR*efb)>>7)+echoR))&^1)

		firL = (int32(firL) * int32(int8(dsp.registers[EVolL])) >> 7)
		firR = (int32(firR) * int32(int8(dsp.registers[EVolR])) >> 7)

		outL = (outL * int32(int8(dsp.registers[MVolL])) >> 7)
		outR = (outR * int32(int8(dsp.registers[MVolR])) >> 7)

		if dsp.registers[FLG]&0x40 == 0 {
			dsp.Buffer.Write(int16(clamp16(outL+firL)), int16(clamp16(outR+firR)))
		} else {
			dsp.Buffer.Write(0, 0)
		}
		//fmt.Printf("masterL: %v, masterR: %v\n", mainL, mainR)
		//fmt.Println(dsp.registers[MVolL])

		dsp.state = 0
	*/
}

func (d *DSP) rateEvent(rate byte) bool {
	rate &= 0x1F
	if rate == 0 {
		return false
	} else {
		return (d.counter+counter_offsets[rate])%counter_rates[rate] == 0
	}
}
func (d *DSP) updateEcenx() {
	d.ecenx = d.registers[FLG]&0x20 != 0
}

// TODO figure out how the real snes writes these bytes
func (d *DSP) writeEchoBuffer(echoL, echoR int16) {
	if d.registers[FLG]&0x20 == 0 {
		idx := d.echoBufferAddress + d.echoBufferIdx<<2
		d.ram[idx+0], d.ram[idx+1] = byte(uint16(echoL)), byte(uint16(echoL)>>8)
		d.ram[idx+2], d.ram[idx+3] = byte(uint16(echoR)), byte(uint16(echoR)>>8)
	} else {
		//fmt.Println("echo locked")
	}
	if d.echoBufferIdx == 0 {
		d.echoBufferSize = uint16(d.registers[EDL]&0xF) << 9
	}
	d.echoBufferIdx++
	if d.echoBufferIdx >= d.echoBufferSize {
		d.echoBufferIdx = 0
	}
}

func (d *DSP) readEchoBuffer() (echoL, echoR int16) {
	idx := d.echoBufferAddress + d.echoBufferIdx<<2
	echoL = int16(uint16(d.ram[idx+0])|uint16(d.ram[idx+1])<<8) >> 1 // <<1
	echoR = int16(uint16(d.ram[idx+2])|uint16(d.ram[idx+3])<<8) >> 1 // <<1

	return
}

type fir struct {
	leftSampleBuffer  [8]int16
	rightSampleBuffer [8]int16

	coeffs [8]int32

	left, right int32

	idx int
}

func (d *DSP) latchFirCoeffAt(index int) {
	d.fir.coeffs[index] = int32(int8(d.registers[index<<4|int(FFCx)]))
}

func (f *fir) getSample(d *DSP) (echoL, echoR int32) {
	//f.leftSampleBuffer[f.idx], f.rightSampleBuffer[f.idx] = d.readEchoBuffer()
	f.leftSampleBuffer[f.idx], f.rightSampleBuffer[f.idx] = d.echoBufferL, d.echoBufferR
	f.idx = (f.idx + 1) & 7

	for i := range 8 {
		idx := (f.idx + i) & 7
		//firCoeff := int32(int8(d.registers[i<<4|int(FFCx)]))
		firCoeff := f.coeffs[i]
		echoL += (int32(f.leftSampleBuffer[idx]) * firCoeff) >> 6
		echoR += (int32(f.rightSampleBuffer[idx]) * firCoeff) >> 6

		if i == 6 {
			echoL = int32(int16(echoL))
			echoR = int32(int16(echoR))
		}
	}
	//i==7 aka newest sample
	echoL = clamp16(echoL) &^ 1
	echoR = clamp16(echoR) &^ 1

	//fmt.Println(echoL)

	f.left = echoL
	f.right = echoR

	return
}

func (d *DSP) ReadRegister(reg byte) byte {
	//fmt.Printf("READING ADDRESS: %x\n", reg)
	if dspReg(reg) == VxOutX {
		fmt.Println("READING OUTX ")
	}
	//if dspReg(reg) == VxEnvX {
	//	fmt.Println("READING EVNELOPE FOR VOICE ", reg>>4)
	//}
	return d.registers[reg&0x7F]
}

func (d *DSP) WriteRegister(reg byte, val byte) {
	if reg <= 0x7F {
		d.registers[reg] = val

		reg := dspReg(reg)

		if reg == KOn {
			d.newKOn = val
			//fmt.Println("KEYON: ", val)
			/*
				for i := range 8 {
					if val&(1<<i) != 0 {
						d.Voices[i].keyOn()
					}
				}
			*/
		}
		/*
			if reg == KOff {
				//fmt.Println("KEYOFF: ", val)
				for i := range 8 {
					if val&(1<<i) != 0 {
						//d.Voices[i].keyOff()
					}
				}
			}
			if reg&0x0F == VxPitchL {
				pitch := &d.Voices[reg>>4].pitchValue
				*pitch = (*pitch & 0x3F00) | uint16(val)
			}
			if reg&0x0F == VxPitchH {
				pitch := &d.Voices[reg>>4].pitchValue
				*pitch = (*pitch & 0xFF) | uint16(val&0x3F)<<8
			}
			if reg == PMOn {
				//fmt.Println("PMON: ", val)
			}
		*/
		if reg == FLG {
			//fmt.Println("FLG: ", val)
			/*
				if val >= 0x80 {
					for i := range 8 {
						//TODO voices are supposed to stay this state till the bit is reset
						d.Voices[i].keyOff()
						d.Voices[i].envelope.reset()
					}
				}
			*/
			//d.noiseRate = val & 0x1F
		}
	}

}

type Buffer interface {
	Write(sampleL, sampleR int16)
	Read(p []byte) (n int, err error)
}

type RingBuffer struct {
	sync.Mutex
	storage []uint32
	head    int
	tail    int
	count   int
	size    int
}

// the buffer size will be (1<<sizeShift)-1 to avoid modulo.
// note that % size == & size-1
func newRingBuffer(sizeShift int) *RingBuffer {
	mask := 1 << sizeShift
	return &RingBuffer{
		size:    mask - 1,
		storage: make([]uint32, mask),
	}
}

func (ab *RingBuffer) Write(sampleL, sampleR int16) {
	ab.Lock()
	defer ab.Unlock()

	//go sign extends uint32(sample) so the inner cast isnt optional
	//oto ioreader in an int16 bit depth stream requests data in this order
	//read back as little endian (so left -> right)
	ab.storage[ab.head] = (uint32(uint16(sampleR)) << 16) | uint32(uint16(sampleL))
	ab.head = (ab.head + 1) & ab.size

	if ab.count <= ab.size {
		ab.count++
	} else {
		//if full the tail moves forward because the oldest sample just got overwritten
		ab.tail = (ab.tail + 1) & ab.size
	}
}

func (ab *RingBuffer) Read(p []byte) (n int, err error) {
	ab.Lock()
	defer ab.Unlock()

	for i := 0; i < len(p); i += 4 {
		if ab.count > 0 {
			binary.LittleEndian.PutUint32(p[i:], ab.storage[ab.tail])
			ab.tail = (ab.tail + 1) & ab.size
			ab.count--
		} else {
			//empty buffer
			//only works because this holds the mutex and so
			//once the buffer is empty it stays so
			clear(p[i:])
			break
		}
	}
	return len(p), nil
}

type AtomicBuffer struct {
	storage []uint32
	head    atomic.Uint64
	tail    atomic.Uint64
	size    uint64
}

// the buffer size will be (1<<sizeShift)-1 to avoid modulo.
// note that % size == & size-1
func newAtomicBuffer(sizeShift int) *AtomicBuffer {
	mask := uint64(1 << sizeShift)
	return &AtomicBuffer{
		size:    mask - 1,
		storage: make([]uint32, mask),
	}
}

func (ab *AtomicBuffer) Write(sampleL, sampleR int16) {
	//go sign extends uint32(sample) so the inner cast isnt optional
	//oto ioreader in an int16 bit depth stream requests data in this order
	//read back as little endian (so left -> right)
	head := ab.head.Load()
	tail := ab.tail.Load()

	newHead := (head + 1)

	if newHead&ab.size != tail&ab.size {
		ab.storage[head&ab.size] = (uint32(uint16(sampleR)) << 16) | uint32(uint16(sampleL))
		ab.head.Store(newHead)
	}
}

func (ab *AtomicBuffer) Read(p []byte) (n int, err error) {
	for i := 0; i < len(p); i += 4 {
		head := ab.head.Load()
		tail := ab.tail.Load()

		if tail != head {
			binary.LittleEndian.PutUint32(p[i:], ab.storage[tail&ab.size])
			ab.tail.Store(tail + 1)
		} else {
			binary.LittleEndian.PutUint32(p[i:], 0)
		}
	}
	return len(p), nil
}
