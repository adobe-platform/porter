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
package conf

func (recv *Region) HasELB() bool {
	return recv.ELB != "" && recv.ELB != "none"
}

// inet is a superset of worker which are almost identical to cron
func (recv *Region) PrimaryTopology() (dominant string) {
	for _, container := range recv.Containers {
		switch container.Topology {
		case Topology_Inet:
			dominant = container.Topology
		case Topology_Worker:
			if dominant != Topology_Inet {
				dominant = container.Topology
			}
		}
	}
	return
}

func (recv *Region) HealthCheckMethod() string {
	for _, container := range recv.Containers {
		if container.Topology == Topology_Inet {
			return container.HealthCheck.Method
		}
	}
	return ""
}

func (recv *Region) HealthCheckPath() string {
	for _, container := range recv.Containers {
		if container.Topology == Topology_Inet {
			return container.HealthCheck.Path
		}
	}
	return ""
}
