package util_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"errors"

	"github.com/adobe-platform/porter/util"
)

var _ = Describe("Retryers", func() {

	Context("SuccessRetryer", func() {

		It("Propagates success", func() {
			var i int
			success := util.SuccessRetryer(1, func(int) {}, func() bool {
				i++
				return true
			})
			Expect(i).To(Equal(1))
			Expect(success).To(BeTrue())
		})

		It("Retries, messages, and propagates failure", func() {
			var i, retryMsg int
			success := util.SuccessRetryer(2, func(int) { retryMsg++ }, func() bool {
				i++
				return false
			})
			Expect(retryMsg).To(Equal(1))
			Expect(i).To(Equal(2))
			Expect(success).To(BeFalse())
		})
	})

	Context("ErrorRetryer", func() {

		It("Propagates no errors", func() {
			var i int
			err := util.ErrorRetryer(1, func(int) {}, func() error {
				i++
				return nil
			})
			Expect(i).To(Equal(1))
			Expect(err).To(BeNil())
		})

		It("Retries, messages, and propagates errors", func() {
			var i, retryMsg int
			err := util.ErrorRetryer(2, func(int) { retryMsg++ }, func() error {
				i++
				return errors.New("uh oh")
			})
			Expect(retryMsg).To(Equal(1))
			Expect(i).To(Equal(2))
			Expect(err).To(MatchError("uh oh"))
		})
	})

})
