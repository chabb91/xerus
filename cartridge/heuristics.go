package cartridge

import (
	"fmt"
	"log"
	"math/bits"
)

func computeChecksum(romData []byte) (checksum uint16) {
	for _, v := range romData {
		checksum += uint16(v)
	}

	log.Printf("Cartridge: The computed checksum is: %x", checksum)

	return
}

func testCandidateHeader(romType int, headerLocation int, romData []byte, checksum uint16) int {
	if len(romData) < headerLocation+64 {
		return -1
	}

	headerData := make([]byte, 64)
	copy(headerData, romData[headerLocation:])

	points := int(0)

	candidateChecksum := uint16(headerData[0x1F])<<8 | uint16(headerData[0x1E])
	candidateChecksumComplement := uint16(headerData[0x1D])<<8 | uint16(headerData[0x1C])
	resetVector := uint16(headerData[0x3D])<<8 | uint16(headerData[0x3C])

	if checksum == candidateChecksum {
		points += 10
	}
	if candidateChecksum == ^candidateChecksumComplement {
		points++
	}
	if containsOnlyASCIIBytes(headerData[:0x15]) {
		points++
	}
	if headerData[0x15]&0xF == byte(romType) {
		points++
	}
	if isPowerOfTwo(len(romData)) &&
		//assuming we are dealing with padded rom data and size is power of 2 this should be true
		headerData[0x17] == byte(bits.TrailingZeros(uint(len(romData))))-10 {
		points++
	}
	if resetVector >= 0x8000 {
		points++
	} else {
		return -1
	}

	return points
}

// any automated header detection in chat
func findRomHeader(romData []byte) (romMapper, error) {
	yellow := "\033[33m"
	reset := "\033[0m"
	checksum := computeChecksum(romData)

	loRomPt := testCandidateHeader(LoROM, 0x7FC0, romData, checksum)
	hiRomPt := testCandidateHeader(HiROM, 0xFFC0, romData, checksum)
	exHiRomPt := testCandidateHeader(ExHiROM, 0x40FFC0, romData, checksum)

	bestResult := max(loRomPt, hiRomPt, exHiRomPt)
	if bestResult > 0 {
		if bestResult < 10 {
			log.Printf("Cartridge: %s[WARNING]%s Checksum mismatch!", yellow, reset)
		}
		if bestResult == loRomPt {
			log.Printf("Cartridge: LoROM detected with a fitness of: %v", bestResult)
			return NewLoRom(), nil
		}
		if bestResult == hiRomPt {
			log.Printf("Cartridge: HiROM detected with a fitness of: %v", bestResult)
			return NewHiRom(), nil
		}
		if bestResult == exHiRomPt {
			log.Printf("Cartridge: ExHiROM detected with a fitness of: %v", bestResult)
			return nil, fmt.Errorf("Cartridge: ExHiROM is WIP.") //TODO implement NewExHiRom
		}
	}
	log.Printf("Cartridge: [WARNING] Failed to detect the rom header.")
	return nil, fmt.Errorf("Cartridge: [FATAL] The rom header could not be located.")
}

func padROM(data []byte) ([]byte, uint32) {
	if len(data)%1024 == 512 {
		data = data[512:] //remove the 0x200 header padding if its present
		log.Printf("Cartridge: Removing the 0x200 header padding...")
	}

	log.Printf("Cartridge: Padding the rom data...")
	size := len(data)
	if size == 0 {
		log.Printf("Cartridge: WARNING: rom file is empty!")
		return data, 0
	}

	endSize := nextPow2(size)
	if endSize == size {
		log.Printf("Cartridge: Rom size is %v which is a power of 2, no intervention necessary.", size)
		return data, uint32(endSize - 1)
	}

	largePart := prevPow2(size)

	remainder := data[largePart:]
	remSize := len(remainder)

	smallPart := nextPow2(remSize)

	var paddedRem []byte
	if smallPart == remSize {
		log.Printf("Cartridge: The small part of the rom is a power of 2(%v), no padding needed.", remSize)
		paddedRem = remainder
	} else {
		log.Printf("Cartridge: The small part of the rom is not a power of 2(%v), padding with zeroes.", remSize)
		paddedRem = make([]byte, smallPart)
		copy(paddedRem, remainder)
	}

	finalROM := make([]byte, endSize)
	copy(finalROM, data[:largePart])
	cnt := largePart >> bits.TrailingZeros(uint(smallPart)) //largePart/smallPart

	for i := range cnt {
		copy(finalROM[largePart+(i*smallPart):], paddedRem)
	}

	log.Printf("Cartridge: The small part was copied %v times to match the size of the large. Total rom size: %v.", cnt, endSize)
	return finalROM, uint32(endSize - 1)
}

func isPowerOfTwo(n int) bool {
	return n > 0 && (n&(n-1) == 0)
}

func nextPow2(x int) int {
	if x <= 1 {
		return 1
	}
	return 1 << bits.Len(uint(x-1))
}

func prevPow2(x int) int {
	if x <= 1 {
		return 1
	}
	return 1 << (bits.Len(uint(x)) - 1)
}

func containsOnlyASCIIBytes(b []byte) bool {
	for i := range b {
		if b[i] >= 128 {
			return false
		}
	}
	return true
}
