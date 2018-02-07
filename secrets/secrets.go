/*
 * (c) 2016-2018 Adobe. All rights reserved.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 */
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
)

const AesBytes = 32

type Payload struct {
	HostSecrets        []byte
	ContainerSecrets   map[string]string
	DockerRegistry     string
	DockerPullUsername string
	DockerPullPassword string
	PemFile            []byte
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
