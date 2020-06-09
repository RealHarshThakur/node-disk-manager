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

	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/openebs/node-disk-manager/cmd/ndm_daemonset/controller"

	"github.com/openebs/node-disk-manager/pkg/hierarchy"
	"github.com/openebs/node-disk-manager/pkg/ndm-grpc/server"
	"github.com/sirupsen/logrus"
)

// Node helps in using types defined in Server
type Node struct {
	*server.Node
}

// NewNode is a constructor
func NewNode(l *logrus.Logger) *Node {
	return &Node{&server.Node{Log: l}}
}

type disks []protos.BlockDevice

// ListDisks gives the status of iSCSI service
func (n *Node) ListDisks(ctx context.Context, null *protos.Null) (*protos.BlockDevices, error) {

	n.Log.Info("Listing block devices")

	ctrl, err := controller.NewController()
	if err != nil {
		n.Log.Errorf("Error creating a controller", err)
		return nil, status.Errorf(codes.NotFound, "Namespace not found")
	}

	err = ctrl.SetControllerOptions(controller.NDMOptions{ConfigFilePath: "/host/node-disk-manager.config"})
	if err != nil {
		n.Log.Error("Error setting config to controller ", err)
		return nil, status.Errorf(codes.Internal, "Error setting config to controller")
	}

	blockDeviceList, err := ctrl.ListBlockDeviceResource(false)
	if err != nil {
		n.Log.Errorf("Error listing block devices", err)
		return nil, status.Errorf(codes.Internal, "Error fetching list of disks")
	}

	if len(blockDeviceList.Items) == 0 {
		n.Log.Info("No items found")
	}

	blockDevices := make([]*protos.BlockDevice, 0)

	// Disks which have partitions are considered as Parent Disks
	parentNames, err := GetParentDisks(n, blockDeviceList)
	if err != nil {
		n.Log.Errorf("Error fetching Parent disks %v", err)
	}

	holderNames, err := GetHolders(n, blockDeviceList)
	if err != nil {
		n.Log.Errorf("Error fetching Holders %v", err)
	}

	slaveNames, err := GetSlaves(n, blockDeviceList)
	if err != nil {
		n.Log.Errorf("Error fetching slaves %v", err)
	}

	for _, name := range holderNames {

		blockDevices = append(blockDevices, &protos.BlockDevice{
			Name: name,
			Type: "Holder",
		})
	}

	for _, name := range slaveNames {

		blockDevices = append(blockDevices, &protos.BlockDevice{
			Name: name,
			Type: "Slaves",
		})
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

// GetHolders gets the holders of a particular device
func GetHolders(n *Node, BL *v1alpha1.BlockDeviceList) ([]string, error) {

	blockDeviceNames := make([]string, 0)
	for _, bd := range BL.Items {
		device := hierarchy.Device{Path: bd.Spec.Path}
		depDevices, err := device.GetDependents()
		if err != nil {
			n.Log.Errorf("Error fetching dependents of the disk", err)
			return nil, err
		}
		blockDeviceNames = depDevices.Holders

	}
	return blockDeviceNames, nil
}

// GetSlaves gets the holders of a particular device
func GetSlaves(n *Node, BL *v1alpha1.BlockDeviceList) ([]string, error) {

	blockDeviceNames := make([]string, 0)
	for _, bd := range BL.Items {
		device := hierarchy.Device{Path: bd.Spec.Path}
		depDevices, err := device.GetDependents()
		if err != nil {
			n.Log.Errorf("Error fetching dependents of the disk", err)
			return nil, err
		}
		blockDeviceNames = depDevices.Slaves

	}
	return blockDeviceNames, nil
}

// GetPartitions gets the partitions given a parent disk name
func GetPartitions(n *Node, name string) []string {

	device := hierarchy.Device{Path: name}

	depDevices, err := device.GetDependents()
	if err != nil {
		n.Log.Errorf("Error fetching dependents of the disk", err)
		return nil
	}

	if len(depDevices.Partitions) == 0 {
		n.Log.Infof("This device has no partitions")
		return nil
	}
	partitonNames := depDevices.Partitions

	return partitonNames

}

//GetParentDisks returns the names of disks which are parents(have partitions and aren't holders/slaves)
func GetParentDisks(n *Node, BL *v1alpha1.BlockDeviceList) ([]string, error) {

	blockDeviceNames := make([]string, 0)
	for _, bd := range BL.Items {
		device := hierarchy.Device{Path: bd.Spec.Path}
		depDevices, err := device.GetDependents()
		if err != nil {
			n.Log.Errorf("Error fetching dependents of the disk", err)
			return nil, err
		}
		if len(depDevices.Parent) == 0 {
			n.Log.Infof("This could be a holder or slave")
			return nil, err
		}
		blockDeviceNames = append(blockDeviceNames, depDevices.Parent)
	}
	return blockDeviceNames, nil
}
