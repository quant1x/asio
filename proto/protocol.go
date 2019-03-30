package proto

type Encoder interface {
	Append(data [] byte) error
}

type Decoder interface {
	Append(data []byte) error
}
