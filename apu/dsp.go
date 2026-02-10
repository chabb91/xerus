package apu

import (
	"fmt"
	"sync"
)

const DSP_REG_SIZE = 0x80

type DSPInterface interface {
	ReadRegister(reg byte) byte
	WriteRegister(reg byte, val byte)
}

type DSP struct {
	state     int
	registers [DSP_REG_SIZE]byte

	*AudioBuffer
	*AudioBuffer2
	*ChannelReader
	Samples chan int16

	Voices [8]*Voice
}

func NewDsp(psram *SPCMemory) *DSP {
	dsp := &DSP{AudioBuffer: newAudioBuffer(11)}
	//dsp := &DSP{AudioBuffer2: &AudioBuffer2{storage: make([]int16, 0, 100), buffers: make(chan []int16, 30)}}
	dsp.Samples = make(chan int16)
	dsp.ChannelReader = &ChannelReader{dsp.Samples}
	for i := range len(dsp.Voices) {
		dsp.Voices[i] = newVoice(i, &dsp.registers, &psram.ram)
	}

	return dsp
}

func (dsp *DSP) Step() {
	dsp.state++
	if dsp.state <= 31 {
		return
	}

	var out int32
	for _, v := range dsp.Voices {
		out += int32(v.Tick()) / 20
		if out > 32767 {
			out = 32767
		}
		if out < -32768 {
			out = -32768
		}
	}

	//dsp.AudioBuffer.Write(int16(out))
	/*select {
	//case SampleChan <- int16(out):
	case dsp.Samples <- int16(out):

	}*/
	dsp.Samples <- int16(out)
	dsp.state = 0
}

func (d *DSP) ReadRegister(reg byte) byte {
	return d.registers[reg&0x7F]
}

func (d *DSP) WriteRegister(reg byte, val byte) {
	if reg <= 0x7F {
		d.registers[reg] = val

		if reg == 0x4C {
			fmt.Println("KEYON: ", val)
			for i := range 8 {
				if val&(1<<i) != 0 {
					d.Voices[i].keyOn()
				}
			}
		}
		if reg == 0x5C {
			fmt.Println("KEYOFF: ", val)
			for i := range 8 {
				if val&(1<<i) != 0 {
					d.Voices[i].keyOff()
				}
			}
		}
	}

}

type AudioBuffer struct {
	sync.Mutex
	storage []int16
	head    int
	tail    int
	count   int
	size    int
}

// the buffer size will be (1<<sizeShift)-1 to avoid modulo.
// note that % size == & size-1
func newAudioBuffer(sizeShift int) *AudioBuffer {
	mask := 1 << sizeShift
	return &AudioBuffer{
		size:    mask - 1,
		storage: make([]int16, mask),
	}
}

func (ab *AudioBuffer) Write(sample int16) {
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

func (ab *AudioBuffer) Read(p []byte) (n int, err error) {
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

type AudioBuffer2 struct {
	buffers         chan []int16
	storage         []int16
	activeBuffer    []int16
	activeBufferPos int
}

func newAudioBuffer2(bufferCap, storageSize int) *AudioBuffer2 {
	return &AudioBuffer2{
		buffers: make(chan []int16, bufferCap),
		storage: make([]int16, 0, storageSize),
	}
}

func (ab *AudioBuffer2) Write(sample int16) {
	if len(ab.storage) == cap(ab.storage) {
		send := make([]int16, len(ab.storage))
		copy(send, ab.storage)
		ab.storage = ab.storage[:0]
		ab.buffers <- send
	}
	ab.storage = append(ab.storage, sample)
}

func (ab *AudioBuffer2) Read(p []byte) (n int, err error) {
	for i := 0; i < len(p); i += 2 {
		if ab.activeBufferPos >= len(ab.activeBuffer) {
			ab.activeBuffer = <-ab.buffers
			ab.activeBufferPos = 0
		}

		sample := ab.activeBuffer[ab.activeBufferPos]
		p[i] = byte(sample)
		p[i+1] = byte(sample >> 8)

		ab.activeBufferPos++
	}
	return len(p), nil
}

type ChannelReader struct {
	ch chan int16
}

func (b *ChannelReader) Read(p []byte) (int, error) {
	for i := 0; i < len(p); i += 2 {
		s := <-b.ch
		p[i] = byte(s)
		p[i+1] = byte(s >> 8)
	}
	return len(p), nil
}
