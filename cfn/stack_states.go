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
