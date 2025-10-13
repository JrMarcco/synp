package codec

//go:generate mockgen -source=./codec.go -destination=./mock/codec.mock.go -package=codecmock -typed Codec

// Codec 是消息编码/解码接口。
type Codec interface {
	Name() string
	Marshal(val any) ([]byte, error)
	Unmarshal(data []byte, val any) error
}
