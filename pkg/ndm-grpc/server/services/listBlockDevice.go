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
	"github.com/openebs/node-disk-manager/pkg/apis/openebs/v1alpha1"
	protos "github.com/openebs/node-disk-manager/pkg/ndm-grpc/protos/ndm"
	"k8s.io/klog"

	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/openebs/node-disk-manager/cmd/ndm_daemonset/controller"

	"github.com/openebs/node-disk-manager/pkg/hierarchy"
	"github.com/openebs/node-disk-manager/pkg/ndm-grpc/server"
)

// Node helps in using types defined in Server
type Node struct {
	server.Node
}

// NewNode is a constructor
func NewNode() *Node {
	return &Node{server.Node{}}
}

type disks []protos.BlockDevice

// ListBlockDevices gives the status of iSCSI service
func (n *Node) ListBlockDevices(ctx context.Context, null *protos.Null) (*protos.BlockDevices, error) {

	klog.Info("Listing block devices")

	ctrl, err := controller.NewController()
	if err != nil {
		klog.Errorf("Error creating a controller %v", err)
		return nil, status.Errorf(codes.NotFound, "Namespace not found")
	}

	err = ctrl.SetControllerOptions(controller.NDMOptions{ConfigFilePath: "/host/node-disk-manager.config"})
	if err != nil {
		klog.Errorf("Error setting config to controller %v", err)
		return nil, status.Errorf(codes.Internal, "Error setting config to controller")
	}

	blockDeviceList, err := ctrl.ListBlockDeviceResource(false)
	if err != nil {
		klog.Errorf("Error listing block devices %v", err)
		return nil, status.Errorf(codes.Internal, "Error fetching list of disks")
	}

	if len(blockDeviceList.Items) == 0 {
		klog.Info("No items found")
	}

	blockDevices := make([]*protos.BlockDevice, 0)

	// Currently, we have support for only parentNames(lsblk equivalent is a Disk)
	// Reason to leave out partitions from this is to have a parent-partition relationship in response
	parentNames, _, _, err := GetAllTypes(n, blockDeviceList)
	if err != nil {
		klog.Errorf("Error fetching Parent disks %v", err)
	}

	for _, name := range parentNames {

		blockDevices = append(blockDevices, &protos.BlockDevice{
			Name:       name,
			Type:       "Disk",
			Partitions: GetPartitions(n, name),
		})
	}

	return &protos.BlockDevices{
		Blockdevices: blockDevices,
	}, nil
}

// GetAllTypes takes a list of all block devices found on nodes and returns Parents, Holders and Slaves( in the same order as slices)
func GetAllTypes(n *Node, BL *v1alpha1.BlockDeviceList) ([]string, []string, []string, error) {
	ParentDeviceNames := make([]string, 0)
	HolderDeviceNames := make([]string, 0)
	SlaveDeviceNames := make([]string, 0)

	for _, bd := range BL.Items {
		device := hierarchy.Device{Path: bd.Spec.Path}
		depDevices, err := device.GetDependents()
		if err != nil {
			klog.Errorf("Error fetching dependents of the disk %v", err)
			return nil, nil, nil, err
		}
		// Handling of disks types except partitions. This is to ensure we have parent-partition relationship while sending a response
		if len(depDevices.Parent) == 0 {
			klog.Infof("There are no parent disks for %v", bd.Name)
		} else {
			ParentDeviceNames = append(ParentDeviceNames, depDevices.Parent)
		}
		if len(depDevices.Holders) == 0 {
			klog.Infof("There are no Holder disks for %v", bd.Name)
		} else {
			HolderDeviceNames = append(HolderDeviceNames, depDevices.Holders...)
		}
		if len(depDevices.Slaves) == 0 {
			klog.Infof("There are no Slave disks for %v", bd.Name)
		} else {
			SlaveDeviceNames = append(SlaveDeviceNames, depDevices.Slaves...)
		}

	}

	return ParentDeviceNames, HolderDeviceNames, SlaveDeviceNames, nil
}

// GetPartitions gets the partitions given a parent disk name
func GetPartitions(n *Node, name string) []string {

	device := hierarchy.Device{Path: name}

	depDevices, err := device.GetDependents()
	if err != nil {
		klog.Errorf("Error fetching dependents of the disk %v", err)
		return nil
	}

	if len(depDevices.Partitions) == 0 {
		klog.Infof("This device has no partitions")
		return nil
	}
	partitonNames := depDevices.Partitions

	return partitonNames

}
