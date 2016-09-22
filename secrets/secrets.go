package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
)

const AesBytes = 32

type Payload struct {
	ContainerSecrets   map[string]string
	DockerRegistry     string
	DockerPullUsername string
	DockerPullPassword string
}

func GenerateKey() (symmetricKey []byte, err error) {
	symmetricKey = make([]byte, AesBytes)
	_, err = rand.Read(symmetricKey)
	return
}

func Encrypt(payload, symmetricKey []byte) (gcmPayload []byte, err error) {

	if len(symmetricKey) != AesBytes {
		err = errors.New("invalid symmetric key")
		return
	}

	aesCipher, err := aes.NewCipher(symmetricKey)
	if err != nil {
		return
	}

	gcmCipher, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return
	}

	nonce := make([]byte, gcmCipher.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return
	}

	gcmPayload = gcmCipher.Seal(nonce, nonce, payload, nil)
	return
}

func Decrypt(gcmPayload, symmetricKey []byte) (payload []byte, err error) {

	if len(symmetricKey) != AesBytes {
		err = errors.New("invalid symmetric key")
		return
	}

	aesCipher, err := aes.NewCipher(symmetricKey)
	if err != nil {
		return
	}

	gcmCipher, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return
	}

	nonceSize := gcmCipher.NonceSize()
	if len(gcmPayload) <= nonceSize {
		err = errors.New("gcmPayload was too small")
		return
	}

	nonce := gcmPayload[:nonceSize]
	encPayload := gcmPayload[nonceSize:]

	payload, err = gcmCipher.Open(nil, nonce, encPayload, nil)
	return
}
