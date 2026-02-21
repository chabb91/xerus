package apu

import (
	"fmt"
	"sync"
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

type DSP struct {
	state     int
	registers [DSP_REG_SIZE]byte

	counter int

	noiseRate      byte
	noiseSampleRaw uint16
	noiseSample    int16

	Buffer

	Voices [8]*Voice
}

func NewDsp(psram *SPCMemory) *DSP {
	dsp := &DSP{
		noiseSampleRaw: 0x4000,
		Buffer:         newRingBuffer(11),
	}
	for i := range len(dsp.Voices) {
		dsp.Voices[i] = newVoice(i, &dsp.registers, &psram.ram)
		dsp.Voices[i].envelope.advanceEnvelope = dsp.rateEvent
		dsp.Voices[i].currentNoiseSample = &dsp.noiseSample
	}

	return dsp
}

func (dsp *DSP) Step() {
	dsp.state++
	if dsp.state <= 31 {
		if dsp.state == 29 {
			dsp.counter = (dsp.counter - 1) & 0x77FF
		}
		if dsp.state == 28 {
			non := dsp.registers[NOn]
			for i := range 8 {
				dsp.Voices[i].useNoiseSample = non&dsp.Voices[i].idMask != 0
			}
			if dsp.rateEvent(dsp.noiseRate) {
				N := uint(dsp.noiseSampleRaw) //just storing the generated noise bits as unsigned
				dsp.noiseSampleRaw = uint16((N >> 1) | (((N << 14) ^ (N << 13)) & 0x4000))
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
		return
	}

	var outL, outR, echoL, echoR int32
	for _, v := range dsp.Voices {
		out := int32(v.Tick()) //>> 1
		outL += (out * int32(int8(dsp.registers[v.idReg|VxVolL]))) >> 7
		outL = clamp16(outL)
		outR += (out * int32(int8(dsp.registers[v.idReg|VxVolR]))) >> 7
		outR = clamp16(outR)
		if dsp.registers[EOn]&v.idMask != 0 {
			echoL = clamp16(echoL + outL)
			echoR = clamp16(echoR + outR)
		}
	}

	mainL := int16(clamp16(outL * int32(int8(dsp.registers[MVolL])) >> 7))
	mainR := int16(clamp16(outR * int32(int8(dsp.registers[MVolR])) >> 7))

	if dsp.registers[FLG]&0x40 == 0 {
		dsp.Buffer.Write(mainL, mainR)
	} else {
		dsp.Buffer.Write(0, 0)
	}
	//fmt.Printf("masterL: %v, masterR: %v\n", mainL, mainR)
	//fmt.Println(dsp.registers[MVolL])

	dsp.state = 0
}

func (d *DSP) rateEvent(rate byte) bool {
	rate &= 0x1F
	if rate == 0 {
		return false
	} else {
		return (d.counter+counter_offsets[rate])%counter_rates[rate] == 0
	}
}

func (d *DSP) ReadRegister(reg byte) byte {
	//fmt.Printf("READING ADDRESS: %x\n", reg)
	if dspReg(reg) == VxOutX {
		fmt.Println("READING OUTX ")
	}
	return d.registers[reg&0x7F]
}

func (d *DSP) WriteRegister(reg byte, val byte) {
	if reg <= 0x7F {
		d.registers[reg] = val

		reg := dspReg(reg)

		if reg == KOn {
			//fmt.Println("KEYON: ", val)
			for i := range 8 {
				if val&(1<<i) != 0 {
					d.Voices[i].keyOn()
				}
			}
		}
		if reg == KOff {
			//fmt.Println("KEYOFF: ", val)
			for i := range 8 {
				if val&(1<<i) != 0 {
					d.Voices[i].keyOff()
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
			fmt.Println("PMON: ", val)
		}
		if reg == FLG {
			fmt.Println("FLG: ", val)
			if val >= 0x80 {
				for i := range 8 {
					if val&(1<<i) != 0 {
						d.Voices[i].keyOff()
						d.Voices[i].envelope.reset()
					}
				}
			}
			d.noiseRate = val & 0x1F
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
			sample := ab.storage[ab.tail]
			p[i] = byte(sample)
			p[i+1] = byte(sample >> 8)
			p[i+2] = byte(sample >> 16)
			p[i+3] = byte(sample >> 24)
			ab.tail = (ab.tail + 1) & ab.size
			ab.count--
		} else { //empty buffer
			clear(p[i:]) //only works because this holds the mutex and so
			break        //once the buffer is empty it stays so
		}
	}
	return len(p), nil
}
