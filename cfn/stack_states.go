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
package cfn

const (
	CREATE_COMPLETE                              = "CREATE_COMPLETE"
	CREATE_FAILED                                = "CREATE_FAILED"
	CREATE_IN_PROGRESS                           = "CREATE_IN_PROGRESS"
	DELETE_COMPLETE                              = "DELETE_COMPLETE"
	DELETE_FAILED                                = "DELETE_FAILED"
	DELETE_IN_PROGRESS                           = "DELETE_IN_PROGRESS"
	ROLLBACK_COMPLETE                            = "ROLLBACK_COMPLETE"
	ROLLBACK_FAILED                              = "ROLLBACK_FAILED"
	ROLLBACK_IN_PROGRESS                         = "ROLLBACK_IN_PROGRESS"
	UPDATE_COMPLETE                              = "UPDATE_COMPLETE"
	UPDATE_COMPLETE_CLEANUP_IN_PROGRESS          = "UPDATE_COMPLETE_CLEANUP_IN_PROGRESS"
	UPDATE_IN_PROGRESS                           = "UPDATE_IN_PROGRESS"
	UPDATE_ROLLBACK_COMPLETE                     = "UPDATE_ROLLBACK_COMPLETE"
	UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS = "UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS"
	UPDATE_ROLLBACK_FAILED                       = "UPDATE_ROLLBACK_FAILED"
	UPDATE_ROLLBACK_IN_PROGRESS                  = "UPDATE_ROLLBACK_IN_PROGRESS"
)

func PrunableStatus(status string) bool {
	switch status {
	case CREATE_COMPLETE,
		CREATE_FAILED,
		ROLLBACK_COMPLETE,
		ROLLBACK_FAILED,
		ROLLBACK_IN_PROGRESS,
		UPDATE_COMPLETE,
		UPDATE_COMPLETE_CLEANUP_IN_PROGRESS,
		UPDATE_IN_PROGRESS,
		UPDATE_ROLLBACK_COMPLETE,
		UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS,
		UPDATE_ROLLBACK_FAILED,
		UPDATE_ROLLBACK_IN_PROGRESS:
		return true
	default:
		return false
	}
}

// Valid states that should be used when considering hot swap eligibility are
// CREATE_COMPLETE plus any variation on updating (a previous hotswap)
//
// A separate check is done before actually peforming the hot swap
func CheckHotswapStatus(status string) bool {
	switch status {
	case CREATE_COMPLETE,
		UPDATE_COMPLETE,
		UPDATE_COMPLETE_CLEANUP_IN_PROGRESS,
		UPDATE_IN_PROGRESS,
		UPDATE_ROLLBACK_COMPLETE,
		UPDATE_ROLLBACK_COMPLETE_CLEANUP_IN_PROGRESS,
		UPDATE_ROLLBACK_FAILED,
		UPDATE_ROLLBACK_IN_PROGRESS:
		return true
	default:
		return false
	}
}

func AnyStatus(status string) bool {
	return true
}
