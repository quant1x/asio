package util

// InputStream is a helper type for managing input streams from inside
// the Data event.
type Stream struct {
	b []byte
}

func NewStream(data []byte) *Stream {
	return &Stream{b:data}
}

func (is *Stream) Len() (int) {
	return len(is.b)
}

// Begin accepts a new packet and returns a working sequence of
// unprocessed bytes.
func (is *Stream) Begin(packet []byte) (data []byte) {
	data = packet
	if len(is.b) > 0 {
		is.b = append(is.b, data...)
		data = is.b
	}
	return data
}

// End shifts the stream to match the unprocessed data.
func (is *Stream) End(data []byte) {
	if len(data) > 0 {
		if len(data) != len(is.b) {
			is.b = append(is.b[:0], data...)
		}
	} else if len(is.b) > 0 {
		is.b = is.b[:0]
	}
}
