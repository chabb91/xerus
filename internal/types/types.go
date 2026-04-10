package types

////CARTRIDGE and COPROCESSOR share some of the same types
////They are defined here.

type RomRegionType int

const (
	RomAddress RomRegionType = iota
	SramAddress
	UnmappedAddress
	RomOwnedByCoprocessor
	RamOwnedByCoprocessor
)

type RomMapper func(bank byte, offset uint16, hasSram bool) (int, RomRegionType)
