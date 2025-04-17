package streams

import (
	"context"
	"encoding/json"
)

// JsonStream represents a stream of JSON tokens.
type JsonStream interface {
	// ReadJsonToken reads the next JSON token from the stream.
	// Returns the next token and any error encountered.
	ReadJsonToken(ctx context.Context) (json.Token, error)
}
