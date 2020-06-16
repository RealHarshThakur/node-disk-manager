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

	ps "github.com/mitchellh/go-ps"

	"github.com/sirupsen/logrus"
)

type Process struct {
	PidMock        int
	PPidMock       int
	ExecutableMock string
}

func (p Process) Pid() int {

	return p.PidMock
}

func (p Process) PPid() int {

	return p.PPidMock
}

func (p Process) Executable() string {

	return p.ExecutableMock
}

//TestcheckISCSI checks the ISCSI service
func TestcheckISCSI(t *testing.T) {

	l := logrus.New()
	s := NewService(l)

	mockProcess := Process{
		PidMock:  100,
		PPidMock: 200,

		ExecutableMock: "iscsid",
	}
	processList := make([]ps.Process, 0)
	_ = append(processList, mockProcess)

	var found *bool
	checkISCSI(s, processList, found)

}
