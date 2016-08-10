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
package conf

import (
	"errors"
	"fmt"
	"time"
)

func (recv *Environment) GetELBForRegion(reg string, elb string) (string, error) {
	region, err := recv.GetRegion(reg)
	if err != nil {
		return "", err
	}

	// always return this if defined. it supersedes the old scheme
	if region.ELB != "" {
		return region.ELB, nil
	}

	// backward compatibility with old scheme
	for _, loadBalancer := range region.ELBs {
		if loadBalancer.ELBTag == elb {
			return loadBalancer.Name, nil
		}
	}

	return "", fmt.Errorf("ELB tagged %s doesn't exist in the config for region %s", elb, reg)
}

func (recv *Environment) GetRegion(regionName string) (*Region, error) {
	for _, region := range recv.Regions {
		if region.Name == regionName {
			return region, nil
		}
	}
	return nil, fmt.Errorf("Region %s missing in environment %s", regionName, recv.Name)
}

func (recv *Environment) GetRoleARN(regionName string) (string, error) {
	region, err := recv.GetRegion(regionName)
	if err != nil {
		return "", err
	}

	if region.RoleARN != "" {
		return region.RoleARN, nil
	}

	if recv.RoleARN == "" {
		return "", errors.New("previous validation failure led to missing RoleARN")
	}

	return recv.RoleARN, nil
}

func (recv *Environment) GetStackDefinitionPath(regionName string) (string, error) {
	region, err := recv.GetRegion(regionName)
	if err != nil {
		return "", err
	}

	if region.StackDefinitionPath != "" {
		return region.StackDefinitionPath, nil
	}

	return recv.StackDefinitionPath, nil
}

func (recv *Environment) IsWithinBlackoutWindow() error {
	now := time.Now()

	for _, window := range recv.BlackoutWindows {
		startTime, err := time.Parse(time.RFC3339, window.StartTime)
		if err != nil {
			return err
		}

		endTime, err := time.Parse(time.RFC3339, window.EndTime)
		if err != nil {
			return err
		}

		if startTime.After(endTime) {
			return errors.New("start_time is after end_time")
		}

		if now.After(startTime) && now.Before(endTime) {
			return errors.New(now.Format(time.RFC3339) + " is currently within a blackout window")
		}
	}

	return nil
}
