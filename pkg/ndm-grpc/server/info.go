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

package server

import (
	"os"

	protos "github.com/openebs/node-disk-manager/pkg/ndm-grpc/protos/ndm"
	"github.com/openebs/node-disk-manager/pkg/version"
	"github.com/sirupsen/logrus"

	"context"
)

// InfoType helps in creation of constructor
type InfoType struct {
	log *logrus.Logger
}

// NewInfo is a constructor
func NewInfo(l *logrus.Logger) *InfoType {
	return &InfoType{l}
}

// FindVersion detects the version and gitCommit of NDM
func (n *InfoType) FindVersion(ctx context.Context, null *protos.Null) (*protos.VersionInfo, error) {

	n.log.Infof("Print Version : %v , commit hash : %v", version.GetVersion(), version.GetGitCommit())

	return &protos.VersionInfo{Version: version.GetVersion(), GitCommit: version.GetGitCommit()}, nil

}

// FindNodeName is used to find the name of the worker node NDM is deployed on
func (n *InfoType) FindNodeName(ctx context.Context, null *protos.Null) (*protos.NodeName, error) {

	nodeName := os.Getenv("NODE_NAME")

	n.log.Infof("Node name is : %v", nodeName)

	return &protos.NodeName{NodeName: nodeName}, nil

}
