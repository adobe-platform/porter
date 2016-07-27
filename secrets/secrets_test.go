package secrets_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"crypto/rand"

	"github.com/adobe-platform/porter/secrets"
)

var _ = Describe("Secrets", func() {

	It("Encrypt/decrypt flow works", func() {

		symmetricKey, err := secrets.GenerateKey()
		Expect(err).To(BeNil())

		payload := []byte("Super secret message")
		encPayload, err := secrets.Encrypt(payload, symmetricKey)
		Expect(err).To(BeNil())

		originalPayload, err := secrets.Decrypt(encPayload, symmetricKey)
		Expect(err).To(BeNil())

		Expect(originalPayload).To(Equal(payload))
	})

})
