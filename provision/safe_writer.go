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
package provision

import (
	"bytes"
	"sync"
)

// purpose-built for concurrent writes to an underlying buffer
type safeWriter struct {
	mutex sync.Mutex
	buf   bytes.Buffer
}

func (recv *safeWriter) Write(p []byte) (n int, err error) {
	recv.mutex.Lock()
	n, err = recv.buf.Write(p)
	recv.mutex.Unlock()
	return
}

func (recv safeWriter) String() (s string) {
	recv.mutex.Lock()
	s = recv.buf.String()
	recv.mutex.Unlock()
	return
}
