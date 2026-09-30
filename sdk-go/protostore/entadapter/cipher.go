package entadapter

var _ Cipher = CipherFuncs{}

type (
	Cipher interface {
		Encrypt([]byte) ([]byte, error)
		Decrypt([]byte) ([]byte, error)
	}
	CipherFuncs struct {
		EncryptFunc
		DecryptFunc
	}
	EncryptFunc func([]byte) ([]byte, error)
	DecryptFunc func([]byte) ([]byte, error)
)

func (e EncryptFunc) Encrypt(data []byte) ([]byte, error) {
	return e(data)
}

func (d DecryptFunc) Decrypt(data []byte) ([]byte, error) {
	return d(data)
}
