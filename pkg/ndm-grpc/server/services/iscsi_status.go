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
	ps "github.com/mitchellh/go-ps"
	protos "github.com/openebs/node-disk-manager/pkg/ndm-grpc/protos/ndm"

	"context"
	"strings"

	"github.com/openebs/node-disk-manager/pkg/ndm-grpc/server"
	"github.com/sirupsen/logrus"
)

// Service helps in using types defined in Server
type Service server.Service

// NewService is a constructor
func NewService(l *logrus.Logger) *Service {
	return &Service{Log: l}
}

// Status gives the status of iSCSI service
func (s *Service) Status(ctx context.Context, null *protos.Null) (*protos.ISCSIStatus, error) {

	s.Log.Info("Finding ISCSI status")

	// This will fetch the processes regardless of which OS is being used
	processList, err := ps.Processes()
	if err != nil {
		s.Log.Error(err)
	}

	var found bool

	for _, p := range processList {

		if strings.Contains(p.Executable(), "iscsid") {
			s.Log.Infof("%v is running with process id %v", p.Executable(), p.Pid())
			found = true
		}
	}
	if !found {
		// Note: When using clients like grpcurl, they might return empty output as response when converting to json
		// Set the appropriate flags to avoid that. In case of grpcurl, it is -emit-defaults
		return &protos.ISCSIStatus{Status: false}, nil
	}

	return &protos.ISCSIStatus{Status: true}, nil

}
