package util_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"strings"

	"github.com/adobe-platform/porter/docker/util"
)

var _ = Describe("Docker util", func() {

	Context("CleanEnvFile", func() {
		It("Removes comments", func() {
			inString := `FOO=bar
#comment`

			outString := util.CleanEnvFile(inString)
			Expect(outString).To(Equal("FOO=bar"))
		})

		It("Removes invalid keys", func() {
			inString := `FOO=bar
hy-phen=foo`

			outString := util.CleanEnvFile(inString)
			Expect(outString).To(Equal("FOO=bar"))
		})

		It("Allows empty values", func() {
			inString := `FOO=`

			outString := util.CleanEnvFile(inString)
			Expect(outString).To(Equal("FOO="))
		})

		It("Allows more than one =", func() {
			inString := `FOO=bar=baz`

			outString := util.CleanEnvFile(inString)
			Expect(outString).To(Equal("FOO=bar=baz"))
		})
	})

	Context("NetworkNameToId", func() {
		It("Produces a network name to id mapping", func() {
			// output of `docker network ls`
			input := `NETWORK ID          NAME                DRIVER
e9653d9b9df3        bridge              bridge
6dacac8877d2        porter-a            bridge
03b9c5233205        porter-b            bridge`

			mapping, err := util.NetworkNameToId(strings.NewReader(input))
			Expect(err).To(BeNil())
			Expect(mapping).To(HaveKeyWithValue("bridge", "e9653d9b9df3"))
			Expect(mapping).To(HaveKeyWithValue("porter-a", "6dacac8877d2"))
			Expect(mapping).To(HaveKeyWithValue("porter-b", "03b9c5233205"))
			Expect(mapping).To(HaveLen(3))
		})
	})

})
