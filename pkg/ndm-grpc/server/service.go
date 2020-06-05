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
	"context"
	"strings"

	ps "github.com/mitchellh/go-ps"
	protos "github.com/openebs/node-disk-manager/pkg/ndm-grpc/protos/ndm"
	"github.com/sirupsen/logrus"
)

// Service helps in creation of constructor
type Service struct {
	log *logrus.Logger
}

// NewService is a constructor
func NewService(l *logrus.Logger) *Service {
	return &Service{l}
}

// Status returns the status of the service
func (s *Service) Status(ctx context.Context, null *protos.Null) (*protos.ISCSIStatus, error) {
	s.log.Info("Handling ISCSI status")

	processList, err := ps.Processes()
	if err != nil {
		s.log.Error(err)
	}

	var found bool

	for _, p := range processList {
		if strings.Contains(p.Executable(), "iscsid") {
			s.log.Infof("%v is running with process id %v", p.Executable(), p.Pid())
			found = true
		}
	}
	if !found {
		// Note: When using clients like grpcurl, they might output empty response when converting to json
		// Set the appropriate flags to avoid that.
		return &protos.ISCSIStatus{Status: false}, nil
	}

	return &protos.ISCSIStatus{Status: true}, nil

}
