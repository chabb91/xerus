package cartridge

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/chabb91/xerus/coprocessor"
	"github.com/chabb91/xerus/coprocessor/gsu"
	"github.com/chabb91/xerus/internal/constants"
	"github.com/chabb91/xerus/internal/types"
)

var ErrUnmappedSramRead = errors.New("Trying to read SRAM but the cartridge doesnt have one")
var ErrUnmappedRomRead = errors.New("Trying to read unmapped address space.")
var ErrUnmappedSramWrite = errors.New("No SRAM present so writes arent allowed")
var ErrUnmappedRomWrite = errors.New("Trying to write to read only or unmapped region")

const (
	LoROM   = 0
	HiROM   = 1
	ExHiROM = 5
)

type CartridgeCountry int

const (
	CountryJapan       CartridgeCountry = 0x00
	CountryUSA         CartridgeCountry = 0x01
	CountryEurope      CartridgeCountry = 0x02
	CountryScandinavia CartridgeCountry = 0x03
	CountryFrench      CartridgeCountry = 0x06
	CountryDutch       CartridgeCountry = 0x07
	CountrySpanish     CartridgeCountry = 0x08
	CountryGerman      CartridgeCountry = 0x09
	CountryItalian     CartridgeCountry = 0x0A
	CountryChinese     CartridgeCountry = 0x0B
	CountryKorean      CartridgeCountry = 0x0D
	CountryCommon      CartridgeCountry = 0x0E
	CountryCanada      CartridgeCountry = 0x0F
	CountryBrazil      CartridgeCountry = 0x10
	CountryAustralia   CartridgeCountry = 0x11
)

type CoprocessorId int

const (
	CpuDSP    CoprocessorId = 0x00
	CpuGSU    CoprocessorId = 0x10
	CpuOBC1   CoprocessorId = 0x20
	CpuSA1    CoprocessorId = 0x30
	CpuSDD1   CoprocessorId = 0x40
	CpuSRTC   CoprocessorId = 0x50
	CpuOther  CoprocessorId = 0xE0 //super game boy/satellaview
	CpuCustom CoprocessorId = 0xF0
)

type CoprocessorCustom int

const (
	CpuSPC7110 CoprocessorCustom = 0x0
	CpuST01x   CoprocessorCustom = 0x1
	CpuST018   CoprocessorCustom = 0x2
	CpuCX4     CoprocessorCustom = 0x3
)

type Cartridge struct {
	Mapper types.RomMapper

	romData  []byte
	sramData []byte

	romMask  uint32
	sramMask uint32

	romPath  string
	sramPath string

	Coprocessor coprocessor.Coprocessor
}

func NewCartridge(romPath string) *Cartridge {
	cart := &Cartridge{}

	romData, err := load(romPath)
	if err != nil {
		panic(err)
	}

	cart.romData, cart.romMask = padROM(romData)
	cart.Mapper, err = getRomMapper(cart.romData)
	if err != nil {
		panic(err)
	}
	cart.Coprocessor = cart.DetectCoprocessor()
	cart.romPath = romPath
	cart.InitSram()

	if cart.Coprocessor != nil {
		cart.Coprocessor.SetCartridge(cart)
		if mapperOverride := cart.Coprocessor.OverrideCartridgeMapper(); mapperOverride != nil {
			cart.Mapper = mapperOverride
		}
	}

	return cart
}

func (cart *Cartridge) ReadByte(bank byte, offset uint16) (byte, error) {
	hasSram := cart.HasSram()
	index, addressType := cart.Mapper(bank, offset, hasSram)

	switch addressType {
	case types.RomAddress:
		return cart.romData[uint32(index)&cart.romMask], nil
	case types.RomOwnedByCoprocessor:
		return byte(index), nil
	case types.SramAddress:
		if hasSram {
			return cart.sramData[uint32(index)&cart.sramMask], nil
		} else {
			return 0, ErrUnmappedSramRead
		}
	default:
		//unmappedAddress
		return 0, ErrUnmappedRomRead
	}
}

func (cart *Cartridge) WriteByte(bank byte, offset uint16, value byte) error {
	if !cart.HasSram() {
		return ErrUnmappedSramWrite
	}

	index, addressType := cart.Mapper(bank, offset, true)

	if addressType == types.SramAddress {
		cart.sramData[uint32(index)&cart.sramMask] = value
		return nil
	}

	return ErrUnmappedRomWrite
}

func (cart *Cartridge) HasSram() bool {
	return cart.sramData != nil
}

// coprocessors carry their own internal mappers, they produce and index to the data
func (cart *Cartridge) ReadRam(index int) byte {
	return cart.sramData[uint32(index)&cart.sramMask]
}

func (cart *Cartridge) ReadRom(index int) byte {
	return cart.romData[uint32(index)&cart.romMask]
}

func (cart *Cartridge) WriteRam(index int, value byte) {
	cart.sramData[uint32(index)&cart.sramMask] = value
}

// loads (and creates if necessary) the .smc file from storage, based on the rom name and its path
func (cart *Cartridge) InitSram() {
	romType, err := cart.ReadByte(0, 0xFFD6)
	if err != nil {
		panic("Cannot read the rom header, the mapper is broken.")
	}

	sram := func(cart *Cartridge, sizeVal byte) {
		log.Printf("Cartridge: Sram detected with the size of: %dkb", 1<<sizeVal)
		size := 1 << (10 + sizeVal) //(1<<sizeVal)*1024)
		cart.sramMask = uint32(size - 1)

		romExt := filepath.Ext(cart.romPath)
		basePath := strings.TrimSuffix(cart.romPath, romExt)
		cart.sramPath = basePath + ".srm"
		cart.sramData = make([]byte, size)

		info, err := os.Stat(cart.sramPath)

		if errors.Is(err, fs.ErrNotExist) {
			log.Printf("Cartridge: Sram file(.srm) missing. Creating: %s", cart.sramPath)
			cart.SaveSramToFile()
		} else if info.Size() != int64(size) {
			log.Printf("Cartridge: Sram file size incorrect. Expected %d.", size)
			sramBackupPath := cart.sramPath + ".bak"
			log.Printf("Cartridge: Backing up old sram file to: %s", sramBackupPath)
			err := os.Rename(cart.sramPath, sramBackupPath)
			if err != nil {
				log.Printf("Cartridge: [WARNING] Failed to back up the old sram file to: %s", sramBackupPath)
			}
			cart.SaveSramToFile()
		} else {
			log.Printf("Cartridge: Sram file(.srm) found. Loading: %s", cart.sramPath)
			sramData, err := load(cart.sramPath)
			if err != nil {
				panic(err)
			}
			copy(cart.sramData, sramData)
		}
	}

	sizeVal, err := cart.sramSizeOverride()
	if err == nil {
		log.Printf("Cartridge: Sram override necessary. Forcing Sram size to: %dkb", 1<<sizeVal)
		sram(cart, sizeVal)
		return
	}

	switch romType & 0x7 {
	case 4, 5:
		if romType&0xF0 == byte(CpuGSU) { //for coprocessors that specify sram in the extended header.
			if !cart.isValidExtendedHeaderPresent() {
				panic("Cartridge: Extended header could not be located. Exiting.")
			}

			log.Printf("Cartridge: Checking the extended header for Sram size...")
			sizeVal, err = cart.ReadByte(0, 0xFFBD)
			if err != nil {
				panic("Cannot read the rom header, the mapper is broken.")
			}
			sram(cart, sizeVal)
			return
		}
		fallthrough
	case 1, 2:
		sizeVal, err = cart.ReadByte(0, 0xFFD8)
		if err != nil {
			panic("Cannot read the rom header, the mapper is broken.")
		}
		sram(cart, sizeVal)
		return
	default:
		cart.sramData = nil
	}
}

func (cart *Cartridge) SaveSramToFile() {
	if cart.sramData == nil || cart.sramPath == "" {
		return
	}
	err := os.WriteFile(cart.sramPath, cart.sramData, 0o644)
	if err != nil {
		log.Printf("Cartridge: [WARNING] Failed to save SRAM to %s.", cart.sramPath)
		return
	}
	log.Printf("Cartridge: SRAM has been successfully saved to %s.", cart.sramPath)
}

func load(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	} else {
		return data, nil
	}
}

func (cart *Cartridge) GetRomName() string {
	romExt := filepath.Ext(cart.romPath)
	return strings.TrimSuffix(filepath.Base(cart.romPath), romExt)
}

func (cart *Cartridge) GetHeaderTitle() string {
	from, _ := cart.Mapper(0, 0xFFC0, false)
	to, _ := cart.Mapper(0, 0xFFD5, false)
	return string(cart.romData[from:to])
}

func (cart *Cartridge) IsPal() bool {
	country, _ := cart.Mapper(0, 0xFFD9, false)
	countryId := CartridgeCountry(cart.romData[country])
	switch countryId {
	case CountryEurope, CountryScandinavia, CountryFrench, CountryDutch,
		CountrySpanish, CountryGerman, CountryItalian:
		log.Printf("Cartridge: PAL game detected.")
		return true
	default:
		log.Printf("Cartridge: NTSC game detected.")
		return false
	}
}

// TODO unfinished function
func (cart *Cartridge) DetectCoprocessor() coprocessor.Coprocessor {
	chipsetAddr, _ := cart.Mapper(0, 0xFFD6, false)
	chipset := cart.romData[chipsetAddr]

	if chipset&0xF >= 3 {
		switch CoprocessorId(chipset & 0xF0) {
		case CpuDSP:
		case CpuGSU:
			log.Printf("Cartridge: GSU detected.")
			return gsu.New()
		case CpuOBC1:
		case CpuSA1:
			log.Printf("Cartridge: SA-1 detected.")
		case CpuSDD1:
		case CpuSRTC:
		case CpuOther:
		case CpuCustom:
			customAddr, _ := cart.Mapper(0, 0xFFBF, false)
			customCpu := cart.romData[customAddr]

			switch CoprocessorCustom(customCpu) {
			case CpuSPC7110:
			case CpuST01x:
			case CpuST018:
			case CpuCX4:
			}
		}
		if constants.PanicIfComponentMissing {
			panic("Unimplemented coprocessor detected. Exiting...:(")
		}
	}
	return nil
}

// some roms contain SRAM but they dont specify it
func (cart *Cartridge) sramSizeOverride() (sizeVal byte, err error) {
	switch strings.TrimSpace(cart.GetHeaderTitle()) {
	case "STAR FOX":
		return 5, nil
	case "POWERSLIDE":
		return 5, nil
	default:
		return 0, fmt.Errorf("NO OVERRIDE NEEDED")
	}
}

func (cart *Cartridge) isValidExtendedHeaderPresent() bool {
	from, err := cart.ReadByte(0, 0xFFDA)
	return from == 0x33 && err == nil
}
