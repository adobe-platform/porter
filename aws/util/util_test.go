package util_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/adobe-platform/porter/aws/util"
)

var _ = Describe("AWS util", func() {

	It("ValidRegion validates regions", func() {
		usWest2IsValid := util.ValidRegion("us-west-2")
		Expect(usWest2IsValid).To(BeTrue())

		somethingElse := util.ValidRegion("not a region")
		Expect(somethingElse).To(BeFalse())
	})

})
