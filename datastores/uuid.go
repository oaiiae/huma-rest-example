package datastores

import (
	"encoding/base64"
	"errors"

	"github.com/google/uuid"
)

// UUID is a [uuid.UUID] that uses [base64.RawURLEncoding]
// to marshal to and from text.
type UUID uuid.UUID

func newUUID() UUID { return UUID(uuid.Must(uuid.NewV7())) }

func (*UUID) encoding() *base64.Encoding { return base64.RawURLEncoding }

func (id *UUID) encodedLen() int {
	return id.encoding().EncodedLen(len(id))
}

func (id *UUID) AppendText(b []byte) ([]byte, error) {
	return id.encoding().AppendEncode(b, id[:]), nil
}

func (id *UUID) MarshalText() ([]byte, error) {
	return id.AppendText(nil)
}

func (id *UUID) UnmarshalText(b []byte) error {
	if len(b) != id.encodedLen() {
		return errors.New("invalid length")
	}
	_, err := id.encoding().Decode(id[:], b)
	return err
}
