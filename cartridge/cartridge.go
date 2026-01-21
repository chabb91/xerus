package cartridge

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
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

const (
	romAddress = iota
	sramAddress
	unmappedAddress
)

type romMapper interface {
	getHeaderLocation() uint32
	getCartridgeType() int

	mapToCartridge(bank byte, offset uint16, hasSram bool) (index, addressType int)
}

type Cartridge struct {
	Mapper romMapper

	romData  []byte
	sramData []byte

	romMask  uint32
	sramMask uint32

	romPath  string
	sramPath string
}

func NewCartridge(romPath string) *Cartridge {
	cart := &Cartridge{}

	romData, err := load(romPath)
	if err != nil {
		panic(err)
	}

	cart.romData, cart.romMask = padROM(romData)
	cart.Mapper, err = findRomHeader(cart.romData)
	if err != nil {
		panic(err)
	}
	cart.romPath = romPath
	cart.DetectSram()

	return cart
}

func (cart *Cartridge) ReadByte(bank byte, offset uint16) (byte, error) {
	hasSram := cart.HasSram()
	index, addressType := cart.Mapper.mapToCartridge(bank, offset, hasSram)

	switch addressType {
	case romAddress:
		return cart.romData[uint32(index)&cart.romMask], nil
	case sramAddress:
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

	index, addressType := cart.Mapper.mapToCartridge(bank, offset, true)

	if addressType == sramAddress {
		cart.sramData[uint32(index)&cart.sramMask] = value
		return nil
	}

	return ErrUnmappedRomWrite
}

func (cart *Cartridge) HasSram() bool {
	return cart.sramData != nil
}

// TODO this always creates a new sram but if theres a battery and there is a SRAM file already it should be loaded instead
// loads (and creates if necessary) the .smc file from storage, based on the rom name and its path
func (cart *Cartridge) DetectSram() {
	romType, err := cart.ReadByte(0, 0xFFD6)
	if err != nil {
		cart.sramData = nil
		return
	}

	switch romType & 0x7 {
	case 0x1, 0x2, 0x4, 0x5:
		sizeVal, err := cart.ReadByte(0, 0xFFD8)
		if sizeVal == 0 || err != nil {
			cart.sramData = nil
		} else {
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
