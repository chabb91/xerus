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
	CYCLE_24 = uint64(12)
)

type DmaChannel struct {
	id int

	dmap  byte //control register
	bbad  byte //destination register
	ntlrx byte //hdma line counter register

	a1w  uint16 //dma source address/hdma table address register
	dasw uint16 //dma size register/hdma current indirect address register
	a2w  uint16 //hdma current table address register

	a1b  uint32 //dma source address bank/hdma table address register
	dasb uint32 //hdma indirect address register

	unknown1 byte //43xB and 43xF registers exist and they are aliases of each other but their purpose is not known
}

type Dma struct {
	Mdmaen      byte
	Hdmaen      byte
	HdmaenLatch byte //latching hdmaen so there is no way it changes between trigger and handoff

	dmaOp        *DmaOperation
	currentDmaOp *DmaOperation

	hdmaOp        [8]*HdmaOperation
	currentHdmaOp *HdmaOperation

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

		dma.Channels[i].dmap = 0xFF
		dma.Channels[i].bbad = 0xFF
		dma.Channels[i].ntlrx = 0xFF
		dma.Channels[i].a1w = 0xFFFF
		dma.Channels[i].dasw = 0xFFFF
		dma.Channels[i].a2w = 0xFFFF
		dma.Channels[i].a1b = 0xFF0000
		dma.Channels[i].dasb = 0xFF0000
		dma.Channels[i].unknown1 = 0xFF
	}

	for i := range dma.hdmaOp {
		dma.hdmaOp[i] = &HdmaOperation{
			bus:     bus,
			Hdmaen:  &dma.HdmaenLatch,
			channel: &dma.Channels[i],
		}
	}

	bus.RegisterRange(0x4300, 0x437F, dma, "DMA")
	return dma
}

func (dma *Dma) Step() uint64 {
	switch dma.DmaState {
	case HDMA_RELOAD_INIT:
		dma.initHdma()
		dma.DmaState = HDMA_RELOAD
		return CYCLE_18
	case HDMA_RELOAD:
		log.Printf("RELOADING HDMA with params %+v\n", dma.currentHdmaOp.channel)
		cycles := dma.currentHdmaOp.reload()
		if nextChannel := getNextActiveChannel(dma.HdmaenLatch, dma.currentHdmaOp.channel.id+1); nextChannel == -1 {
			dma.DmaState = HDMA_INACTIVE
		} else {
			dma.currentHdmaOp = dma.hdmaOp[nextChannel]
		}
		return cycles
	case HDMA_TRANSFER_INIT:
		dma.initHdma()
		dma.decideNextHdmaTransferState()
		return CYCLE_18
	case HDMA_TRANSFER_CH_OVERHEAD:
		cycles := dma.currentHdmaOp.stepLineCounter()
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
		if dma.Mdmaen != 0 {
			if dma.currentDmaOp == nil {
				nextChannel := getNextActiveChannel(dma.Mdmaen, 0)
				dma.currentDmaOp = dma.dmaOp.setup(&dma.Channels[nextChannel])
				log.Printf("Starting dma with params %+v\n", dma.currentDmaOp.channel)
			}
			if dma.currentDmaOp != nil {
				if dma.currentDmaOp.stepCycle() {
					dma.Mdmaen &= ^(1 << dma.currentDmaOp.channel.id)
					dma.currentDmaOp = nil
				}
			}
			return CYCLE_8
		}
		fallthrough
	default:
		panic(fmt.Errorf("Fatal: The dma chip is in an unexpected state!"))
	}
}

func (dma *Dma) decideNextHdmaTransferState() {
	channel := dma.currentHdmaOp
	if channel.doTransfer && !channel.isTerminated {
		dma.DmaState = HDMA_TRANSFER
	} else {
		dma.DmaState = HDMA_TRANSFER_CH_OVERHEAD
	}
}

// RELOAD init and TRANSFER init cancels all conflicting DMA transfers regardless if its in progress or qued
func (dma *Dma) cancelConflictingDmaTransfers(firstHdmaId int) {
	for i := firstHdmaId; i < 8; i++ {
		mask := byte(1) << i
		if dma.Mdmaen&mask != 0 && dma.HdmaenLatch&mask != 0 {
			log.Printf("Axing conflicting dma transfer: %+v\n", dma.Channels[i])
			dma.Mdmaen &= ^mask
			if dma.currentDmaOp != nil {
				dma.currentDmaOp = nil
			}
		}
	}
}

func (dma *Dma) initHdma() {
	//this has to be non -1 because the state cant be entered if hdmaen is 0
	nextChannel := getNextActiveChannel(dma.HdmaenLatch, 0)
	dma.currentHdmaOp = dma.hdmaOp[nextChannel]
	if dma.Mdmaen != 0 {
		dma.cancelConflictingDmaTransfers(nextChannel)
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
	for i := range 8 {
		channel := dma.hdmaOp[i]
		channel.isTerminated = false
		channel.doTransfer = dma.Hdmaen != 0
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

func (dma *Dma) Read(addr uint16) (byte, error) {
	b1, err := getChannelNum(addr)
	if err != nil {
		return 0, err
	}

	channel := &dma.Channels[b1]
	switch addr & 0xF {
	case 0x0:
		return channel.dmap, nil
	case 0x1:
		return channel.bbad, nil
	case 0x2:
		return byte(channel.a1w), nil
	case 0x3:
		return byte(channel.a1w >> 8), nil
	case 0x4:
		return byte(channel.a1b >> 16), nil
	case 0x5:
		return byte(channel.dasw), nil
	case 0x6:
		return byte(channel.dasw >> 8), nil
	case 0x7:
		return byte(channel.dasb >> 16), nil
	case 0x8:
		return byte(channel.a2w), nil
	case 0x9:
		return byte(channel.a2w >> 8), nil
	case 0xA:
		return channel.ntlrx, nil
	case 0xB, 0xF:
		return channel.unknown1, nil
	default:
		return 0, fmt.Errorf("undefined DMA register $%04X", addr)
	}
}

func (dma *Dma) Write(addr uint16, value byte) error {
	b1, err := getChannelNum(addr)
	if err != nil {
		return err
	}

	channel := &dma.Channels[b1]
	switch addr & 0xF {
	case 0x0:
		channel.dmap = value
		dma.hdmaOp[b1].setup()
		return nil
	case 0x1:
		channel.bbad = value
		return nil
	case 0x2:
		channel.a1w = channel.a1w&0xFF00 | uint16(value)
		return nil
	case 0x3:
		channel.a1w = channel.a1w&0x00FF | uint16(value)<<8
		return nil
	case 0x4:
		channel.a1b = uint32(value) << 16
		return nil
	case 0x5:
		channel.dasw = channel.dasw&0xFF00 | uint16(value)
		return nil
	case 0x6:
		channel.dasw = channel.dasw&0x00FF | uint16(value)<<8
		return nil
	case 0x7:
		channel.dasb = uint32(value) << 16
		return nil
	case 0x8:
		channel.a2w = channel.a2w&0xFF00 | uint16(value)
		return nil
	case 0x9:
		channel.a2w = channel.a2w&0x00FF | uint16(value)<<8
		return nil
	case 0xA:
		channel.ntlrx = value
		return nil
	case 0xB, 0xF:
		channel.unknown1 = value
		return nil
	default:
		return fmt.Errorf("undefined DMA register $%04X", addr)
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
