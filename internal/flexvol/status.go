/*
Copyright 2017 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package flexvol

// DriverStatus represents the return value of the driver callout.
type DriverStatus struct {
	// Status of the callout. One of "Success", "Failure" or "Not supported".
	Status Status `json:"status"`
	// Reason for success/failure.
	Message string `json:"message,omitempty"`
	// Path to the device attached. This field is valid only for attach calls.
	// ie: /dev/sdx
	DevicePath string `json:"device,omitempty"`
	// Cluster wide unique name of the volume.
	VolumeName string `json:"volumeName,omitempty"`
	// Represents volume is attached on the node
	Attached bool `json:"attached,omitempty"`
	// Returns capabilities of the driver.
	// By default we assume all the capabilities are supported.
	// If the plugin does not support a capability, it can return false for that capability.
	Capabilities *DriverCapabilities `json:"capabilities,omitempty"`
	// Returns the actual size of the volume after resizing is done, the size is in bytes.
	ActualVolumeSize int64 `json:"volumeNewSize,omitempty"`
}

// DriverCapabilities represents what driver can do
type DriverCapabilities struct {
	Attach           bool `json:"attach"`
	SELinuxRelabel   bool `json:"selinuxRelabel"`
	SupportsMetrics  bool `json:"supportsMetrics"`
	FSGroup          bool `json:"fsGroup"`
	RequiresFSResize bool `json:"requiresFSResize"`
}
