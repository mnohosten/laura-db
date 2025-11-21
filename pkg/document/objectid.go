package document

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

// ObjectID is a unique 12-byte identifier similar to MongoDB's ObjectID
// Structure: [4-byte timestamp][5-byte random][3-byte counter]
type ObjectID [12]byte

var objectIDCounter uint32
var processUnique [5]byte

func init() {
	// Generate process-unique random bytes once at startup
	rand.Read(processUnique[:])
}

// NewObjectID generates a new ObjectID
func NewObjectID() ObjectID {
	var id ObjectID

	// 4 bytes: timestamp (seconds since epoch)
	timestamp := uint32(time.Now().Unix())
	binary.BigEndian.PutUint32(id[0:4], timestamp)

	// 5 bytes: process-unique identifier (static per process)
	copy(id[4:9], processUnique[:])

	// 3 bytes: counter (atomic, thread-safe)
	counter := atomic.AddUint32(&objectIDCounter, 1)
	id[9] = byte(counter >> 16)
	id[10] = byte(counter >> 8)
	id[11] = byte(counter)

	return id
}

// ObjectIDFromHex creates an ObjectID from a hex string
func ObjectIDFromHex(s string) (ObjectID, error) {
	var id ObjectID

	if len(s) != 24 {
		return id, fmt.Errorf("invalid ObjectID hex string length: %d", len(s))
	}

	b, err := hex.DecodeString(s)
	if err != nil {
		return id, fmt.Errorf("invalid ObjectID hex string: %w", err)
	}

	copy(id[:], b)
	return id, nil
}

// Hex returns the hex string representation of the ObjectID
func (id ObjectID) Hex() string {
	return hex.EncodeToString(id[:])
}

// String returns the string representation of the ObjectID
func (id ObjectID) String() string {
	return id.Hex()
}

// Timestamp returns the timestamp portion of the ObjectID
func (id ObjectID) Timestamp() time.Time {
	timestamp := binary.BigEndian.Uint32(id[0:4])
	return time.Unix(int64(timestamp), 0)
}

// IsZero returns true if the ObjectID is the zero value
func (id ObjectID) IsZero() bool {
	return id == ObjectID{}
}
