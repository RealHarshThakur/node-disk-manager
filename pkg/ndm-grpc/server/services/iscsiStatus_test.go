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

	"github.com/sirupsen/logrus"
)

type Processes interface {
	Pid() int
	Ppid() int
	Executable() string
}

type Process struct {
}

func (*Process) Pid() int {

	return 100
}

func (*Process) PPid() int {

	return 40
}

func (*Process) Executable() string {

	return "iscsi"
}

func TestcheckISCSI(t *testing.T) {

	l := logrus.New()
	s := NewService(l)

	processList := &Process{Pid()}
	checkISCSI(s, processList, found)

}
