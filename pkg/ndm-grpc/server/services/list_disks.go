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
	protos "github.com/openebs/node-disk-manager/pkg/ndm-grpc/protos/ndm"

	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/openebs/node-disk-manager/cmd/ndm_daemonset/controller"

	"github.com/openebs/node-disk-manager/pkg/ndm-grpc/server"
	"github.com/sirupsen/logrus"
)

// Node helps in using types defined in Server
type Node struct {
	server.Node
}

// NewNode is a constructor
func NewNode(l *logrus.Logger) *Node {
	return &Node{server.Node{Log: l}}
}

type disks []protos.Disk

// ListDisks gives the status of iSCSI service
func (n *Node) ListDisks(ctx context.Context, null *protos.Null) (*protos.Disks, error) {

	n.Log.Info("Listing devices")

	ctrl, err := controller.NewController()
	if err != nil {
		n.Log.Errorf("Error creating a controller", err)
		return nil, status.Errorf(codes.NotFound, "Namespace not found")
	}

	diskList, err := ctrl.ListBlockDeviceResource(false)
	if err != nil {
		n.Log.Errorf("Error listing block devices", err)
		return nil, status.Errorf(codes.Internal, "Error fetching list of disks")
	}

	n.Log.Info(diskList)
	if len(diskList.Items) == 0 {
		n.Log.Info("No items found")
	}

	for _, item := range diskList.Items {
		n.Log.Info("Printing item")
		n.Log.Info(item.Spec.Path)
	}

	// disks := make([]*protos.Disk, 0, len(names))

	// for _, name := range names {
	//   disks = append(disks, &protos.Disk{
	// 	Name: name,
	//   })
	// }

	// return &protos.Disks{
	// 	Disks: disks,
	//  }, nil
	return &protos.Disks{}, nil
}

// func mapNamesToDisks(names string) []protos.Disk {
// 	mapped := make([]protos.Disk, 0, len(names))

// 	for _, name := range names {
// 		mapped = append(mapped, protos.Disk{
// 			Name: name,
// 		})
// 	}
// 	return mapped
// }
