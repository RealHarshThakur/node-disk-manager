/*
Copyright 2019 The OpenEBS Authors
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

package services

import (
	"testing"

	apis "github.com/openebs/node-disk-manager/pkg/apis/openebs/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

// TestGetParentDisks tests the GetParentDisks function
func TestGetAllTypes(t *testing.T) {

	n := NewNode()

	mockDevice := &apis.BlockDevice{
		ObjectMeta: metav1.ObjectMeta{
			Labels: make(map[string]string),
			Name:   "dummy-blockdevice",
		},
	}

	mockblockDevices := make([]apis.BlockDevice, 0)
	mockblockDevices = append(mockblockDevices, *mockDevice)

	mockDeviceList := &apis.BlockDeviceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "BlockDevice",
			APIVersion: "",
		},
		Items: mockblockDevices,
	}
	pd, _, _, _, _, _, err := GetAllTypes(n, mockDeviceList)
	if err != nil {
		t.Errorf("Getting parent disks failed %v", err)
	}
	klog.Infof("Parent disks are: %v", pd)
}

// TestGetPartitions get the partitions of the block device
func TestGetPartitions(t *testing.T) {

	n := NewNode()

	mockDevice := apis.BlockDevice{
		ObjectMeta: metav1.ObjectMeta{
			Labels: make(map[string]string),
			Name:   "sda",
		},
	}

	partitions := GetPartitions(n, mockDevice.Name)

	if len(partitions) != 0 {
		klog.Infof("Partitions found are: %v", partitions)
	}
}
