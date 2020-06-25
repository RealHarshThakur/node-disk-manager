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
	parentNames, _, loopNames, _, _, sparseNames, err := GetAllTypes(n, blockDeviceList)
	if err != nil {
		klog.Errorf("Error fetching Parent disks %v", err)
	}

	klog.Infof("Disks are : %v", parentNames)
	klog.Infof("Loop devices are : %v", loopNames)
	klog.Infof(" Sparse images are : %v", sparseNames)

	for _, name := range parentNames {

		blockDevices = append(blockDevices, &protos.BlockDevice{
			Name:       name,
			Type:       "Disk",
			Partitions: GetPartitions(n, name),
		})
	}

	for _, name := range loopNames {

		blockDevices = append(blockDevices, &protos.BlockDevice{
			Name:       name,
			Type:       "Loop",
			Partitions: GetPartitions(n, name),
		})
	}

	for _, name := range sparseNames {

		blockDevices = append(blockDevices, &protos.BlockDevice{
			Name: name,
			Type: "Sparse",
		})
	}

	return &protos.BlockDevices{
		Blockdevices: blockDevices,
	}, nil
}

// GetAllTypes takes a list of all block devices found on nodes and returns Parents, Partitions, Loop,  Holders, Slaves, Sparse( in the same order as slices)
func GetAllTypes(n *Node, BL *v1alpha1.BlockDeviceList) ([]string, []string, []string, []string, []string, []string, error) {
	ParentDeviceNames := make([]string, 0)
	HolderDeviceNames := make([]string, 0)
	SlaveDeviceNames := make([]string, 0)
	PartitionNames := make([]string, 0)
	LoopNames := make([]string, 0)
	SparseNames := make([]string, 0)

	for _, bd := range BL.Items {
		klog.Infof("Devices are: %v ", bd.Spec.Path)

		if bd.Spec.Details.DeviceType == "sparse" {
			SparseNames = append(SparseNames, bd.Spec.Path)
			continue
		}
		device := hierarchy.Device{Path: bd.Spec.Path}
		depDevices, err := device.GetDependents()
		if err != nil {
			klog.Errorf("Error fetching dependents of the disk name: %v, err: %v", bd.Spec.Path, err)
			continue
		}

		if depDevices.Parent == "" {
			klog.Infof(bd.Spec.Details.DeviceType)
			klog.Infof("There are no parent disks for %v or this is a parent disk", bd.Spec.Path)
			if bd.Spec.Details.DeviceType == "disk" {
				klog.Info("This block device is a disk :  %v", bd.Spec.Path)
				ParentDeviceNames = append(ParentDeviceNames, bd.Spec.Path)
			} else if bd.Spec.Details.DeviceType == "loop" {
				klog.Info("This block device is a Loop device: %v", bd.Spec.Path)
				LoopNames = append(LoopNames, bd.Spec.Path)
			}
		} else {
			for _, pn := range ParentDeviceNames {
				if pn != depDevices.Parent {
					ParentDeviceNames = append(ParentDeviceNames, depDevices.Parent)
				}
			}
		}

		if len(depDevices.Partitions) == 0 {
			klog.Infof("There are partitions for %v", bd.Spec.Path)
		} else {
			PartitionNames = append(PartitionNames, depDevices.Partitions...)
		}

		if len(depDevices.Holders) == 0 {
			klog.Infof("There are no Holder disks for %v", bd.Spec.Path)
		} else {
			HolderDeviceNames = append(HolderDeviceNames, depDevices.Holders...)
		}
		if len(depDevices.Slaves) == 0 {
			klog.Infof("There are no Slave disks for %v", bd.Spec.Path)
		} else {
			SlaveDeviceNames = append(SlaveDeviceNames, depDevices.Slaves...)
		}

	}

	return ParentDeviceNames, PartitionNames, LoopNames, HolderDeviceNames, SlaveDeviceNames, SparseNames, nil
}

// GetPartitions gets the partitions given a parent disk name(Will change this later to filter out partitions given a disk name and partitions slice)
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
