package lsm

import (
	"encoding/binary"
	"hash/fnv"
)

// BloomFilter is a probabilistic data structure for membership testing
// False positives possible, false negatives impossible
type BloomFilter struct {
	bits      []byte // Bit array
	size      int    // Size in bits
	numHashes int    // Number of hash functions
}

// NewBloomFilter creates a new bloom filter
func NewBloomFilter(expectedItems int, numHashes int) *BloomFilter {
	// Calculate optimal size: m = -n*ln(p) / (ln(2)^2)
	// For p=0.01 (1% false positive rate): m ≈ 9.6 * n
	size := expectedItems * 10 // bits
	byteSize := (size + 7) / 8  // Convert to bytes

	return &BloomFilter{
		bits:      make([]byte, byteSize),
		size:      size,
		numHashes: numHashes,
	}
}

// Add adds a key to the bloom filter
func (bf *BloomFilter) Add(key []byte) {
	for i := 0; i < bf.numHashes; i++ {
		hash := bf.hash(key, i)
		bitIndex := hash % uint64(bf.size)
		byteIndex := bitIndex / 8
		bitOffset := bitIndex % 8
		bf.bits[byteIndex] |= (1 << bitOffset)
	}
}

// Contains checks if a key might be in the set
func (bf *BloomFilter) Contains(key []byte) bool {
	for i := 0; i < bf.numHashes; i++ {
		hash := bf.hash(key, i)
		bitIndex := hash % uint64(bf.size)
		byteIndex := bitIndex / 8
		bitOffset := bitIndex % 8
		if (bf.bits[byteIndex] & (1 << bitOffset)) == 0 {
			return false
		}
	}
	return true
}

// hash generates the i-th hash value for a key
func (bf *BloomFilter) hash(key []byte, i int) uint64 {
	h := fnv.New64a()
	h.Write(key)
	hash1 := h.Sum64()

	h.Reset()
	h.Write(append(key, byte(i)))
	hash2 := h.Sum64()

	// Double hashing: h(i) = h1 + i*h2
	return hash1 + uint64(i)*hash2
}

// Marshal serializes the bloom filter
func (bf *BloomFilter) Marshal() []byte {
	// Format: size(4) | numHashes(4) | bits
	buf := make([]byte, 8+len(bf.bits))
	binary.LittleEndian.PutUint32(buf[0:4], uint32(bf.size))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(bf.numHashes))
	copy(buf[8:], bf.bits)
	return buf
}

// UnmarshalBloomFilter deserializes a bloom filter
func UnmarshalBloomFilter(data []byte) (*BloomFilter, error) {
	if len(data) < 8 {
		return nil, ErrInvalidBloomFilter
	}

	size := int(binary.LittleEndian.Uint32(data[0:4]))
	numHashes := int(binary.LittleEndian.Uint32(data[4:8]))
	bits := make([]byte, len(data)-8)
	copy(bits, data[8:])

	return &BloomFilter{
		bits:      bits,
		size:      size,
		numHashes: numHashes,
	}, nil
}

// Stats returns bloom filter statistics
func (bf *BloomFilter) Stats() map[string]interface{} {
	// Count set bits
	setBits := 0
	for _, b := range bf.bits {
		for i := 0; i < 8; i++ {
			if (b & (1 << i)) != 0 {
				setBits++
			}
		}
	}

	fillRatio := float64(setBits) / float64(bf.size)

	// Estimate false positive rate: (1 - e^(-kn/m))^k
	// Simplified: fill_ratio^k
	fpr := 1.0
	for i := 0; i < bf.numHashes; i++ {
		fpr *= fillRatio
	}

	return map[string]interface{}{
		"size":                bf.size,
		"num_hashes":          bf.numHashes,
		"set_bits":            setBits,
		"fill_ratio":          fillRatio,
		"estimated_fpr":       fpr,
		"bytes":               len(bf.bits),
	}
}
