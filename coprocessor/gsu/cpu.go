package gsu

import (
	"SNES_emulator/coprocessor"
	"SNES_emulator/internal/constants"
	"fmt"
)

type immediateInstructionFunc func(gsu *GSU)

const SRAM_BASE_BANK byte = 0x70
const trace bool = false

type GSU struct {
	cartridge        coprocessor.CartridgeDataSource
	interruptManager coprocessor.InterruptManager

	r registers

	cache      [0x200]byte
	cacheFlags uint32

	immediateBytes       [3]byte
	immediateOpcode      byte
	immediateInstruction immediateInstructionFunc

	prevRamAddr uint32 //the full address expanded with the SRAM_BASE_BANK.

	sReg, dReg byte

	currentOpcode byte

	waitState

	tracer *tracer
}

func New() coprocessor.Coprocessor {
	gsu := &GSU{}
	if trace {
		gsu.tracer = newTracer(600_000, 25)
	}
	gsu.r.fetchFunc = gsu.preFetchByte
	gsu.r.cpuRegister15Buffer = R15_NOT_BRANCHING
	return gsu
}

func (gsu *GSU) Step() uint64 {
	if gsu.r.SFR&FlagGo == 0 || gsu.waiting {
		return constants.CYCLE_2
	}

	if trace {
		gsu.tracer.trace(gsu)
	}
	gsu.processByte()
	gsu.preFetchByte()
	//TODO if gsu is in WAIT the proper cycle cost should be returned AFTER its not
	//waiting anymore
	return constants.CYCLE_2
}

// the gsu is execute -> fetch.
// TODO prefetch determines cycle cost
func (gsu *GSU) preFetchByte() {
	pc := gsu.r.cpuRegisters[0xF]
	var opcode byte
	var err error

	if idx := pc - gsu.r.CBR; idx < 0x200 {
		cacheMask := uint32((1 << (idx >> 4)))
		if gsu.cacheFlags&cacheMask == 0 {
			rowBaseIdx := idx & 0x1F0
			rowBasePc := pc & 0xFFF0
			for i := range uint16(16) {
				//TODO this read has to add cumulative overhead
				opcode, err = gsu.Read8(gsu.r.PBR, rowBasePc+i)

				if err != nil {
					panic(err.Error())
				}

				gsu.cache[rowBaseIdx+i] = opcode
			}
			gsu.cacheFlags |= cacheMask
		}
		opcode = gsu.cache[idx]
	} else {
		opcode, err = gsu.Read8(gsu.r.PBR, pc)

		if err != nil {
			panic(err.Error())
		}
	}
	gsu.currentOpcode = opcode
	if pcVal := gsu.r.cpuRegister15Buffer; pcVal != R15_NOT_BRANCHING {
		gsu.r.cpuRegisters[0xF] = uint16(pcVal)
		gsu.r.cpuRegister15Buffer = R15_NOT_BRANCHING
	} else {
		gsu.r.cpuRegisters[0xF]++
	}
	//fmt.Printf("%02x\n", opcode)
}

func (gsu *GSU) GetRegisterMap() coprocessor.RegisterMap {
	return coprocessor.RegisterMap{Start: 0x3000, End: 0x347F, Name: "GSU"}
}

func (gsu *GSU) SetCartridge(cartridge coprocessor.CartridgeDataSource) {
	gsu.cartridge = cartridge
}

func (gsu *GSU) SetInterruptManager(manager coprocessor.InterruptManager) {
	gsu.interruptManager = manager
}

// every coprocessor carries its own mapper
// which then it can use to get data using the cartridge data source
func (gsu *GSU) Read8(bank byte, offset uint16) (byte, error) {
	if bank < 0x40 {
		gsu.verifyRomOwnership(gsu.r.SCMR)
		offset = (offset & 0x7FFF) | (uint16(bank&1) << 15)
		return gsu.cartridge.ReadRom(int(bank>>1)<<16 | int(offset)), nil //lorom
	}
	if bank-0x40 < 0x20 { //0x40-0x5F
		gsu.verifyRomOwnership(gsu.r.SCMR)
		return gsu.cartridge.ReadRom(int(bank&0x3F)<<16 | int(offset)), nil //hirom
	}
	if bank-0x70 < 2 {
		gsu.verifyRamOwnership(gsu.r.SCMR)
		return gsu.cartridge.ReadRam(int(bank&1)<<16 | int(offset)), nil
	}
	return 0, fmt.Errorf("GSU: Trying to read unmapped memory"+
		" at $%02x%04x", bank, offset)
}

func (gsu *GSU) Write8(bank byte, offset uint16, value byte) error {
	if bank-0x70 < 2 {
		gsu.verifyRamOwnership(gsu.r.SCMR)
		gsu.cartridge.WriteRam(int(bank&1)<<16|int(offset), value)
		return nil
	}
	return fmt.Errorf("GSU: Trying to write unmapped or read only memory"+
		" at $%02x%04x", bank, offset)
}

// tracks if an instruction accessed rom/ram when RON/RAN was disabled.
// this causes the cpu to WAIT till it is re-enabled.
type waitState struct {
	waitForRom, waitForRam bool
	waiting                bool
}

func (w *waitState) updateWait(scmr byte) {
	if w.waitForRam {
		w.waitForRam = scmr&RAN == 0
	}
	if w.waitForRom {
		w.waitForRom = scmr&RON == 0
	}
	w.waiting = w.waitForRam || w.waitForRom
}

func (w *waitState) verifyRomOwnership(scmr byte) {
	w.waitForRom = scmr&RON == 0
	w.waiting = w.waitForRam || w.waitForRom
}

func (w *waitState) verifyRamOwnership(scmr byte) {
	w.waitForRam = scmr&RAN == 0
	w.waiting = w.waitForRam || w.waitForRom
}
