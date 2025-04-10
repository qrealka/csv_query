package streams

import (
	"encoding/json"
	"io"

	iface "propertytreeanalyzer/api/streams"
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
func (j *jsonReader) ReadJsonToken() (json.Token, error) {
	if j == nil || j.decoder == nil {
		return nil, io.EOF
	}
	return j.decoder.Token()
}
