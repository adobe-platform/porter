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
package util

var regions = []string{
	"ap-northeast-1",
	"ap-southeast-1",
	"ap-southeast-2",
	"eu-central-1",
	"eu-west-1",
	"sa-east-1",
	"us-east-1",
	"us-west-1",
	"us-west-2",
}

// ValidRegion validates the input is one of
//
// ap-northeast-1
// ap-southeast-1
// ap-southeast-2
// eu-central-1
// eu-west-1
// sa-east-1
// us-east-1
// us-west-1
// us-west-2
func ValidRegion(region string) bool {
	for _, validRegion := range regions {
		if region == validRegion {
			return true
		}
	}
	return false
}
