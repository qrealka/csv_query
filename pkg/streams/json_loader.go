package streams

import (
	"context"
	"encoding/json"
	"io"

	iface "propertytreeanalyzer/pkg/api/streams"
)

type jsonReader struct {
	decoder *json.Decoder
}

var _ iface.JsonStream = (*jsonReader)(nil)

func NewJsonStream(reader io.Reader) iface.JsonStream {
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	return &jsonReader{decoder: decoder}
}

// ReadJsonToken implements JsonStream.
func (j *jsonReader) ReadJsonToken(ctx context.Context) (json.Token, error) {
	if j == nil || j.decoder == nil {
		return nil, io.EOF
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return j.decoder.Token()
	}
}
