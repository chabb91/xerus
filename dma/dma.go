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
	//HDMA_TRANSFER_LOAD_INDIRECT
	HDMA_TRANSFER
)

const (
	CYCLE_8  = uint64(4)
	CYCLE_18 = uint64(9)
)

type DmaChannel struct {
	id int

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
	Mdmaen      byte
	Hdmaen      byte
	HdmaenLatch byte //latching hdmaen so there is no way it changes between trigger and handoff

	dmaOp        *DmaOperation
	currentDmaOp *DmaOperation

	hdmaOp        [8]*HdmaOperation
	currentHdmaOp *HdmaOperation

	currentDmaId int

	DmaState int

	Channels [8]DmaChannel
}

func NewDma(bus memory.Bus) *Dma {

	dma := &Dma{
		currentDmaOp: nil,
		dmaOp:        &DmaOperation{bus: bus},
		Channels:     [8]DmaChannel{}}

	for i := range dma.Channels {
		dma.Channels[i].id = i
	}

	for i := range dma.hdmaOp {
		dma.hdmaOp[i] = &HdmaOperation{
			bus:     bus,
			Hdmaen:  &dma.HdmaenLatch,
			channel: &dma.Channels[i],
		}
	}

	//TODO this probably isnt the best place to register it
	bus.RegisterRange(0x4300, 0x437F, dma, "DMA")
	return dma
}

func (dma *Dma) Step() uint64 {
	switch dma.DmaState {
	case HDMA_RELOAD_INIT:
		dma.DmaState = HDMA_RELOAD
		dma.currentHdmaOp = dma.hdmaOp[getNextActiveChannel(dma.HdmaenLatch, 0)]
		return CYCLE_18
	case HDMA_RELOAD:
		cycles := CYCLE_8
		cycles += dma.currentHdmaOp.reload()
		log.Printf("RELOADING HDMA with params %+v\n", dma.currentHdmaOp.channel)
		if nextChannel := getNextActiveChannel(dma.HdmaenLatch, dma.currentHdmaOp.channel.id+1); nextChannel == -1 {
			dma.DmaState = HDMA_INACTIVE
		} else {
			dma.currentHdmaOp = dma.hdmaOp[nextChannel]
		}
		return cycles
	case HDMA_TRANSFER_INIT:
		//this has to be non -1 because the state cant be entered if hdmaen is 0
		dma.currentHdmaOp = dma.hdmaOp[getNextActiveChannel(dma.HdmaenLatch, 0)]
		dma.decideNextHdmaTransferState()
		return CYCLE_18
	case HDMA_TRANSFER_CH_OVERHEAD:
		cycles := CYCLE_8
		if !dma.currentHdmaOp.isTerminated {
			cycles += dma.currentHdmaOp.stepLineCounter()
		}
		if nextChannel := getNextActiveChannel(dma.HdmaenLatch, dma.currentHdmaOp.channel.id+1); nextChannel == -1 {
			dma.DmaState = HDMA_INACTIVE
		} else {
			dma.currentHdmaOp = dma.hdmaOp[nextChannel]
			dma.decideNextHdmaTransferState()
		}
		return cycles
	case HDMA_TRANSFER:
		if dma.currentHdmaOp.stepCycle() {
			dma.DmaState = HDMA_TRANSFER_CH_OVERHEAD
		}
		return CYCLE_8
	case HDMA_INACTIVE:
		if dma.Mdmaen != 0 && dma.currentDmaOp == nil {
			dma.currentDmaOp = dma.dmaOp
			dma.currentDmaId = getNextActiveChannel(dma.Mdmaen, 0)
			dma.currentDmaOp.setup(dma.Channels[dma.currentDmaId])
			log.Printf("Starting dma on channel %v with params %+v\n", dma.currentDmaId, dma.Channels[dma.currentDmaId])
			return CYCLE_8
		}
		if dma.Mdmaen != 0 && dma.currentDmaOp != nil {
			if dma.currentDmaOp.stepCycle() {
				dma.currentDmaOp = nil
				dma.Mdmaen &= ^(1 << dma.currentDmaId)
			}
			return CYCLE_8
		}
		fallthrough
	default:
		log.Println("WARNING: The dma chip is in an unexpected state")
		return CYCLE_8
	}
}

func (dma *Dma) decideNextHdmaTransferState() {
	channel := dma.currentHdmaOp
	if channel.doTransfer && !channel.isTerminated {
		dma.DmaState = HDMA_TRANSFER
		//dma and hdma on same channel cancels dma
		if channel.channel.id == dma.currentDmaId && dma.currentDmaOp != nil {
			dma.currentDmaOp = nil
			dma.Mdmaen &= ^(1 << dma.currentDmaId)
		}
	} else {
		dma.DmaState = HDMA_TRANSFER_CH_OVERHEAD
	}
}

func (dma *Dma) IsInProgress() bool {
	return dma.Mdmaen != 0 || dma.DmaState != HDMA_INACTIVE
}

func (dma *Dma) Reload() {
	if dma.Hdmaen > 0 {
		dma.DmaState = HDMA_RELOAD_INIT
		dma.HdmaenLatch = dma.Hdmaen
	}
}

func (dma *Dma) DoTransfer() {
	//uncommenting the hdmaen check breaks things and im not entirely sure why
	//maybe the issue is the test rom uses channel 0 for uploading dma data
	//and hdma effects so the conflicting dma gets canceled and stuff glitches
	if dma.Hdmaen > 0 {
		dma.DmaState = HDMA_TRANSFER_INIT
		dma.HdmaenLatch = dma.Hdmaen
	}
}

// needed to properly enable midframe HDMA
func (dma *Dma) SetHdmaen(value byte) {
	for i := range 8 {
		if /*(dma.Hdmaen>>i)&1 == 0 &&*/ (value>>i)&1 != 0 {
			dma.hdmaOp[i].setup()
		}
	}
	dma.Hdmaen = value
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
