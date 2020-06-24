package d2mpq

import (
	"encoding/binary"
	"errors"
)

type Reader struct {
	BlockTableEntry BlockTableEntry
	MPQData         *MPQ
	BlockSize       uint32

	// Only used for  and FileImplode
	BlockPositions []uint32
}

// NewReader creates a new io.Reader.
// Reads from the returned Reader read and decompress data from r

func newReader(mpq *MPQ, blockTableEntry BlockTableEntry, fileName string) (*Reader, error) {
	r := &Reader{
		MPQData:         mpq,
		BlockTableEntry: blockTableEntry,
	}
	r.BlockSize = 0x200 << r.MPQData.Data.BlockSize

	if r.BlockTableEntry.HasFlag(FilePatchFile) {
		return nil, errors.New("Patching is not supported")
	}

	// This condition is not the same as the one used inside the 'loadBlock' method. Not sure why.
	// We may end up using 'BlockPositions' despite it being not set here.
	if (r.BlockTableEntry.HasFlag(FileCompress) || r.BlockTableEntry.HasFlag(FileImplode)) &&
		!r.BlockTableEntry.HasFlag(FileSingleUnit) {

		if err := r.loadBlockOffsets(); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Populates the 'BlockPositions' field.
func (r *Reader) loadBlockOffsets() error {
	blockPositionCount := ((r.BlockTableEntry.UncompressedFileSize + r.BlockSize - 1) / r.BlockSize) + 1
	r.BlockPositions = make([]uint32, blockPositionCount)
	r.MPQData.File.Seek(int64(r.BlockTableEntry.FilePosition), 0)
	mpqBytes := make([]byte, blockPositionCount*4)
	r.MPQData.File.Read(mpqBytes)
	for i := range r.BlockPositions {
		idx := i * 4
		r.BlockPositions[i] = binary.LittleEndian.Uint32(mpqBytes[idx : idx+4])
	}
	blockPosSize := blockPositionCount << 2
	if r.BlockTableEntry.HasFlag(FileEncrypted) {
		decrypt(r.BlockPositions, r.BlockTableEntry.EncryptionSeed-1)
		if r.BlockPositions[0] != blockPosSize {
			return errors.New("decryption of MPQ failed")
		}
		if r.BlockPositions[1] > r.BlockSize+blockPosSize {
			return errors.New("decryption of MPQ failed")
		}
	}
	return nil
}
