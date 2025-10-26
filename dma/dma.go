package dma

import (
	"SNES_emulator/memory"
	"fmt"
	"log"
)

const (
	HDMA_INACTIVE = iota
	HDMA_RELOAD_INIT
	HDMA_RELOAD
	HDMA_TRANSFER_INIT
	HDMA_TRANSFER_CH_OVERHEAD
	HDMA_TRANSFER_LOAD_INDIRECT
	HDMA_TRANSFER
)

type DmaChannel struct {
	dmap  byte //control register
	bbad  byte //destination register
	a1tl  byte //dma source address low/hdma table address register
	a1th  byte //dma source address high/hdma table address register
	a1b   byte //dma source address bank/hdma table address register
	dasl  byte //dma size register low/hdma indirect address register
	dash  byte //dma size register high/hdma indirect address register
	dasb  byte //hdma indirect address register
	a2al  byte //hdma mid frame table address register low
	a2ah  byte //hdma mid frame table address register high
	ntlrx byte //hdma line counter register
}

type Dma struct {
	bus memory.Bus

	Mdmaen byte
	Hdmaen byte

	dmaOp        *DmaOperation
	currentDmaOp *DmaOperation

	hdmaOp        [8]*HdmaOperation
	currentHdmaOp *HdmaOperation

	currentDmaId  int
	currentHdmaId int

	DmaState int

	Channels [8]DmaChannel
}

func NewDma(bus memory.Bus) *Dma {

	dma := &Dma{
		currentDmaOp: nil,
		bus:          bus,
		dmaOp:        &DmaOperation{bus: bus},
		hdmaOp:       [8]*HdmaOperation{},
		Channels:     [8]DmaChannel{}}

	for i := range dma.hdmaOp {
		dma.hdmaOp[i] = &HdmaOperation{bus: bus}
	}

	//TODO this probably isnt the best place to register it
	bus.RegisterRange(0x4300, 0x437F, dma, "DMA")
	return dma
}

func (dma *Dma) Step() bool {
	if dma.DmaState == HDMA_RELOAD_INIT {
		//this should always be >0 but who knows
		if dma.Hdmaen > 0 {
			dma.DmaState = HDMA_RELOAD
			dma.currentHdmaId = -1
			return false
		}
		dma.DmaState = HDMA_INACTIVE
		return true
	}
	if dma.DmaState == HDMA_RELOAD {
		dma.currentHdmaId = getNextActiveChannel(dma.Hdmaen, dma.currentHdmaId+1)
		if dma.currentHdmaId != -1 {
			dma.currentHdmaOp = nil
			dma.hdmaOp[dma.currentHdmaId].setup(dma.Channels[dma.currentHdmaId])
			dma.hdmaOp[dma.currentHdmaId].reload(dma.Channels[dma.currentHdmaId])
			log.Printf("RELOADING HDMA on channel %v with params %+v\n", dma.currentHdmaId, dma.Channels[dma.currentHdmaId])
			if getNextActiveChannel(dma.Hdmaen, dma.currentHdmaId+1) == -1 {
				dma.DmaState = HDMA_INACTIVE
				return true
			}
			return false
		}
	}
	//costs 18 master cycles
	if dma.DmaState == HDMA_TRANSFER_INIT {
		dma.currentHdmaId = -1
		dma.DmaState = HDMA_TRANSFER
		return false
	}
	if dma.DmaState == HDMA_TRANSFER {
		if dma.Hdmaen > 0 && dma.currentHdmaOp == nil {
			//this has to be non -1 the first time because ID= -1 and hdmaen >0
			dma.currentHdmaId = getNextActiveChannel(dma.Hdmaen, dma.currentHdmaId+1)
			if dma.hdmaOp[dma.currentHdmaId].isDoneForFrame() {
				if getNextActiveChannel(dma.Hdmaen, dma.currentHdmaId+1) == -1 {
					dma.DmaState = HDMA_INACTIVE
					return true
				}
				return false
			}
			dma.currentHdmaOp = dma.hdmaOp[dma.currentHdmaId]
			//dma and hdma on same channel cancels dma
			if dma.currentHdmaId == dma.currentDmaId && dma.currentDmaOp != nil {
				dma.currentDmaOp = nil
				dma.Mdmaen &= ^(1 << dma.currentDmaId)
			}
			return false
		}
		if dma.Hdmaen > 0 && dma.currentHdmaOp != nil {
			if dma.currentHdmaOp.stepCycle() {
				dma.currentHdmaOp = nil
				if getNextActiveChannel(dma.Hdmaen, dma.currentHdmaId+1) == -1 {
					dma.DmaState = HDMA_INACTIVE
					return true
				}
				return false
			}
		}
	}
	if dma.DmaState == HDMA_INACTIVE {
		if dma.isDmaActive() && dma.currentDmaOp == nil {
			dma.currentDmaOp = dma.dmaOp
			dma.currentDmaId = getNextActiveChannel(dma.Mdmaen, 0)
			dma.currentDmaOp.setup(dma.Channels[dma.currentDmaId])
			log.Printf("Starting dma on channel %v with params %+v\n", dma.currentDmaId, dma.Channels[dma.currentDmaId])
			return false
		}
		if dma.isDmaActive() && dma.currentDmaOp != nil {
			if dma.currentDmaOp.stepCycle() {
				dma.currentDmaOp = nil
				dma.Mdmaen &= ^(1 << dma.currentDmaId)
			}
			return dma.Mdmaen == 0
		}
	}
	return false
}

func (dma *Dma) IsInProgress() bool {
	return dma.isDmaActive() || dma.DmaState != HDMA_INACTIVE
}

func (dma *Dma) Reload() {
	if dma.Hdmaen > 0 {
		dma.DmaState = HDMA_RELOAD_INIT
	}
}

func (dma *Dma) DoTransfer() {
	//uncommenting the hdmaen check breaks things and im not entirely sure why
	//maybe the issue is the test rom uses channel 0 for uploading dma data
	//and hdma effects so the conflicting dma gets canceled and stuff glitches
	if dma.Hdmaen > 0 {
		dma.DmaState = HDMA_TRANSFER_INIT
	}
}

func (dma *Dma) isHdmaActive() bool {
	for v := range 8 {
		if (dma.Hdmaen&(1<<v)) != 0 && !dma.hdmaOp[v].isDoneForFrame() {
			//dma.currentHdmaId = v
			return true
		}
	}
	return false
}

// needed to properly enable midframe HDMA
func (dma *Dma) SetHdmaen(value byte) {
	for i := range 8 {
		if (dma.Hdmaen>>i)&1 == 0 && (value>>i)&1 != 0 {
			dma.hdmaOp[i].setup(dma.Channels[i])
		}
	}
	dma.Hdmaen = value
}

func (dma *Dma) isDmaActive() bool {
	return dma.Mdmaen != 0
}

func (dma *Dma) Read(addr uint16) (byte, error) {
	b1, err := getChannelNum(addr)
	if err != nil {
		return 0, err
	}

	b2, err := getRegister(&dma.Channels[b1], addr)
	if err != nil {
		return 0, err
	}

	return *b2, nil
}

func (dma *Dma) Write(addr uint16, value byte) error {
	b1, err := getChannelNum(addr)
	if err != nil {
		return err
	}

	b2, err := getRegister(&dma.Channels[b1], addr)
	if err != nil {
		return err
	}

	*b2 = value
	return nil
}

func getRegister(channel *DmaChannel, address uint16) (*byte, error) {

	switch address & 0xF {
	case 0x0:
		return &channel.dmap, nil
	case 0x1:
		return &channel.bbad, nil
	case 0x2:
		return &channel.a1tl, nil
	case 0x3:
		return &channel.a1th, nil
	case 0x4:
		return &channel.a1b, nil
	case 0x5:
		return &channel.dasl, nil
	case 0x6:
		return &channel.dash, nil
	case 0x7:
		return &channel.dasb, nil
	case 0x8:
		return &channel.a2al, nil
	case 0x9:
		return &channel.a2ah, nil
	case 0xA:
		return &channel.ntlrx, nil
	default:
		return nil, fmt.Errorf("undefined DMA register $%04X", address)
	}
}

func getChannelNum(address uint16) (byte, error) {
	ret := (address & 0x00F0) >> 4
	if ret < 8 {
		return byte(ret), nil
	} else {
		return 0, fmt.Errorf("undefined DMA channel $%04X", ret)
	}
}

func getNextActiveChannel(enabledChannels byte, from int) int {
	if enabledChannels == 0 {
		return -1
	}

	for i := from; i < 8; i++ {
		if (enabledChannels>>i)&1 == 1 {
			return i
		}
	}

	return -1
}
