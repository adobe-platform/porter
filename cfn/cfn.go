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
package cfn

import "fmt"

type (
	Template struct {
		Description string                    `json:"Description,omitempty"`
		Parameters  map[string]ParameterInput `json:"Parameters,omitempty"`
		Mappings    map[string]interface{}    `json:"Mappings,omitempty"`
		Resources   map[string]interface{}    `json:"Resources,omitempty"`
		Conditions  map[string]interface{}    `json:"Conditions,omitempty"`
		Outputs     interface{}               `json:"Outputs,omitempty"`

		// a reverse lookup of resource type to the logical names of that type
		typeToLogical map[string][]string
	}

	ParameterInput struct {
		Description           string   `json:"Description,omitempty"`
		Type                  string   `json:"Type,omitempty"`
		MinLength             int      `json:"MinLength,omitempty"`
		MaxLength             int      `json:"MaxLength,omitempty"`
		MinValue              string   `json:"MinValue,omitempty"`
		MaxValue              string   `json:"MaxValue,omitempty"`
		AllowedPattern        string   `json:"AllowedPattern,omitempty"`
		AllowedValues         []string `json:"AllowedValues,omitempty"`
		Default               string   `json:"Default,omitempty"`
		ConstraintDescription string   `json:"ConstraintDescription,omitempty"`
	}
)

func NewTemplate() *Template {
	return &Template{
		Parameters: make(map[string]ParameterInput),
		Mappings:   make(map[string]interface{}),
		Resources:  make(map[string]interface{}),
		Conditions: make(map[string]interface{}),

		typeToLogical: make(map[string][]string),
	}
}

func (recv *Template) ParseResources() {
	resources := recv.Resources
	recv.Resources = make(map[string]interface{})

	for resourceName, resourceRaw := range resources {
		if resourceMap, ok := resourceRaw.(map[string]interface{}); ok {
			recv.SetResource(resourceName, resourceMap)
		} else {
			panic("invariant violation: the resource " + resourceName + " is not a map[string]interface{}")
		}
	}
}

func (recv *Template) SetResource(logicalName string, resource map[string]interface{}) {
	recv.Resources[logicalName] = resource

	if resourceType, ok := resource["Type"].(string); ok && ValidType(resourceType) {
		var logicalNames []string
		var exists bool

		if logicalNames, exists = recv.typeToLogical[resourceType]; !exists {
			logicalNames = make([]string, 0)
		}

		for _, existingLogicalName := range logicalNames {
			if existingLogicalName == logicalName {
				panic("invariant violation: logical resource name collision")
			}
		}

		logicalNames = append(logicalNames, logicalName)

		recv.typeToLogical[resourceType] = logicalNames
	} else {
		panic("invariant violation: all resources must have a type")
	}
}

func (recv *Template) ResourceExists(resourceType string) bool {
	_, exists := recv.typeToLogical[resourceType]
	return exists
}

func (recv *Template) GetResourceNames(resourceType string) ([]string, bool) {
	logicalNames, exists := recv.typeToLogical[resourceType]
	return logicalNames, exists
}

func (recv *Template) GetResourcesByType(resourceType string) map[string]interface{} {
	resources := make(map[string]interface{})
	if logicalNames, exists := recv.typeToLogical[resourceType]; exists {
		for _, logicalName := range logicalNames {
			if resource, exists := recv.Resources[logicalName]; exists {
				resources[logicalName] = resource
			}
		}
	}
	return resources
}

func (recv *Template) GetResourceName(resourceType string) (string, error) {
	logicalNames, exists := recv.typeToLogical[resourceType]
	if !exists || len(logicalNames) == 0 {
		return "", fmt.Errorf("Resource of type %s doesn't exist", resourceType)
	}
	if len(logicalNames) > 1 {
		return "", fmt.Errorf("More than one resource of type %s exists", resourceType)
	}
	return logicalNames[0], nil
}
