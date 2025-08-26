package datastores

import (
	_ "encoding" // for documentation links to [encoding]
	"encoding/base32"
	"errors"

	"github.com/google/uuid"
)

// uuid32 is [uuid.UUID] but uses [base32] for text marshaling.
type uuid32 struct{ uuid.UUID }

var (
	uuid32Encoding   = base32.StdEncoding.WithPadding(base32.NoPadding) //nolint: gochecknoglobals,nolintlint
	uuid32EncodedLen = uuid32Encoding.EncodedLen(len(uuid32{}.UUID))    //nolint: gochecknoglobals,nolintlint
)

func (id *uuid32) initV4() *uuid32 { id.UUID = uuid.Must(uuid.NewRandom()); return id }

// func (id *uuid32) initV7() *uuid32 { id.UUID = uuid.Must(uuid.NewV7()); return id }

// AppendText implements [encoding.TextAppender].
func (id *uuid32) AppendText(b []byte) ([]byte, error) {
	return uuid32Encoding.AppendEncode(b, id.UUID[:]), nil
}

// MarshalText implements [encoding.TextMarshaler].
func (id *uuid32) MarshalText() ([]byte, error) {
	return id.AppendText(nil)
}

// UnmarshalText implements [encoding.TextUnmarshaler].
func (id *uuid32) UnmarshalText(b []byte) error {
	if len(b) != uuid32EncodedLen {
		return errors.New("invalid length")
	}
	_, err := uuid32Encoding.Decode(id.UUID[:], b)
	return err
}
