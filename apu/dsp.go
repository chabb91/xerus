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

type DSPInterface interface {
	ReadRegister(reg byte) byte
	WriteRegister(reg byte, val byte)
}

type DSP struct {
	state     int
	registers [DSP_REG_SIZE]byte

	counter int

	Buffer

	Voices [8]*Voice
}

func NewDsp(psram *SPCMemory) *DSP {
	dsp := &DSP{Buffer: newRingBuffer(11)}
	//dsp := &DSP{Buffer: newChannelBuffer(10)}
	for i := range len(dsp.Voices) {
		dsp.Voices[i] = newVoice(i, &dsp.registers, &psram.ram)
		dsp.Voices[i].envelope.advanceEnvelope = dsp.rateEvent
	}

	return dsp
}

func (dsp *DSP) Step() {
	dsp.state++
	if dsp.state <= 31 {
		if dsp.state == 29 {
			dsp.counter = (dsp.counter - 1) & 0x77FF
		}
		return
	}
	var out int32
	for _, v := range dsp.Voices {
		out += int32(v.Tick()) / 10
		/*if out > 16383 {
			out = 16383
		} else if out < -16384 {
			out = -16384
		}*/
		out = clamp16(out)
	}

	dsp.Buffer.Write(int16(out))
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
	return d.registers[reg&0x7F]
}

func (d *DSP) WriteRegister(reg byte, val byte) {
	if reg <= 0x7F {
		d.registers[reg] = val

		if reg == 0x4C {
			//fmt.Println("KEYON: ", val)
			for i := range 8 {
				if val&(1<<i) != 0 {
					d.Voices[i].keyOn()
				}
			}
		}
		if reg == 0x5C {
			//fmt.Println("KEYOFF: ", val)
			for i := range 8 {
				if val&(1<<i) != 0 {
					d.Voices[i].keyOff()
				}
			}
		}
		if reg&0x0F == 0x02 {
			pitch := &d.Voices[reg>>4].pitchValue
			*pitch = (*pitch & 0x3F00) | uint16(val)
		}
		if reg&0x0F == 0x03 {
			pitch := &d.Voices[reg>>4].pitchValue
			*pitch = (*pitch & 0xFF) | uint16(val&0x3F)<<8
		}
		if reg == 0x2D {
			fmt.Println("PMON: ", val)
		}
	}

}

type Buffer interface {
	Write(sample int16)
	Read(p []byte) (n int, err error)
}

type RingBuffer struct {
	sync.Mutex
	storage []int16
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
		storage: make([]int16, mask),
	}
}

func (ab *RingBuffer) Write(sample int16) {
	ab.Lock()
	defer ab.Unlock()

	ab.storage[ab.head] = sample
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

	for i := 0; i < len(p); i += 2 {
		if ab.count > 0 {
			sample := ab.storage[ab.tail]
			p[i] = byte(sample)
			p[i+1] = byte(sample >> 8)
			ab.tail = (ab.tail + 1) & ab.size
			ab.count--
		} else { //empty buffer
			p[i] = 0
			p[i+1] = 0
		}
	}
	return len(p), nil
}

type ChannelBuffer struct {
	ch chan int16
}

func newChannelBuffer(chanSize int) *ChannelBuffer {
	return &ChannelBuffer{ch: make(chan int16, chanSize)}
}

func (b *ChannelBuffer) Read(p []byte) (int, error) {
	for i := 0; i < len(p); i += 2 {
		s := <-b.ch
		p[i] = byte(s)
		p[i+1] = byte(s >> 8)
	}
	return len(p), nil
}

func (b *ChannelBuffer) Write(sample int16) {
	b.ch <- sample
}
