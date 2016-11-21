/*
 *  Copyright 2016 Adobe Systems Incorporated. All rights reserved.
 *  This file is licensed to you under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License. You may obtain a copy
 *  of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software distributed under
 *  the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 *  OF ANY KIND, either express or implied. See the License for the specific language
 *  governing permissions and limitations under the License.
 */
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
