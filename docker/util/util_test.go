package util_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/adobe-platform/porter/docker/util"
)

var _ = Describe("Util to clean --env-file", func() {

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
