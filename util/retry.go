/*
 * (c) 2016-2017 Adobe. All rights reserved.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 */
package util

import (
	"math"
	"time"
)

// SuccessRetryer does exponential backoff of the action function which returns
// a boolean
//
// Here's some math around the count input
// count  1 == cumulative time of 2    seconds or ~0 mins
// count  2 == cumulative time of 6    seconds or ~0 mins
// count  3 == cumulative time of 14   seconds or ~0 mins
// count  4 == cumulative time of 30   seconds or ~0 mins
// count  5 == cumulative time of 62   seconds or ~1 mins
// count  6 == cumulative time of 126  seconds or ~2 mins
// count  7 == cumulative time of 254  seconds or ~4 mins
// count  8 == cumulative time of 510  seconds or ~8 mins
// count  9 == cumulative time of 1022 seconds or ~17 mins
// count 10 == cumulative time of 2046 seconds or ~34 mins
func SuccessRetryer(count int, retryMsg func(int), action func() bool) (success bool) {
	var i int
	for i = 0; i < count; i++ {

		if i > 0 {
			duration := time.Duration(math.Pow(2, float64(i)))
			time.Sleep(duration * time.Second)
			retryMsg(i)
		}

		if action() {
			break
		}
	}
	if i == count {
		return
	}

	success = true
	return
}

// ErrorRetryer does exponential backoff of the action function which returns
// an error
func ErrorRetryer(count int, retryMsg func(int), action func() error) (err error) {
	var i int
	for i = 0; i < count; i++ {

		if i > 0 {
			duration := time.Duration(math.Pow(2, float64(i)))
			time.Sleep(duration * time.Second)
			retryMsg(i)
		}

		if err = action(); err == nil {
			break
		}
	}

	return
}
