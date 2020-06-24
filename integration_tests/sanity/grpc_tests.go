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

package sanity

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openebs/node-disk-manager/integration_tests/k8s"
	"github.com/openebs/node-disk-manager/integration_tests/udev"
	"github.com/openebs/node-disk-manager/integration_tests/utils"

	protos "github.com/openebs/node-disk-manager/pkg/ndm-grpc/protos/ndm"
	"google.golang.org/grpc"
)

var _ = Describe("gRPC tests", func() {

	var err error
	var k8sClient k8s.K8sClient
	physicalDisk := udev.NewDisk(DiskImageSize)
	err = physicalDisk.AttachDisk()

	BeforeEach(func() {
		Expect(err).NotTo(HaveOccurred())
		By("getting a new client set")
		k8sClient, _ = k8s.GetClientSet()

		By("creating the NDM Daemonset")
		err = k8sClient.CreateNDMDaemonSet()
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the daemonset pod to be running")
		ok := WaitForPodToBeRunningEventually(DaemonSetPodPrefix)
		Expect(ok).To(BeTrue())

		k8s.WaitForReconciliation()

	})
	AfterEach(func() {
		By("deleting the NDM deamonset")
		err := k8sClient.DeleteNDMDaemonSet()
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the pod to be removed")
		ok := WaitForPodToBeDeletedEventually(DaemonSetPodPrefix)
		Expect(ok).To(BeTrue())
	})
	Context("gRPC services", func() {

		It("iSCSI test", func() {
			conn, err := grpc.Dial("0.0.0.0:9090", grpc.WithInsecure())
			Expect(err).NotTo(HaveOccurred())
			defer conn.Close()

			isc := protos.NewISCSIClient(conn)
			ctx := context.Background()
			null := &protos.Null{}

			By("Checking when ISCSI is disabled")
			err = utils.RunCommandWithSudo("sudo systemctl stop iscsid")
			Expect(err).NotTo(HaveOccurred())
			res, err := isc.Status(ctx, null)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.GetStatus()).To(BeFalse())

			By("Checking when ISCSI is enabled ")
			err = utils.RunCommandWithSudo("sudo systemctl enable iscsid")
			Expect(err).NotTo(HaveOccurred())
			err = utils.RunCommandWithSudo("sudo systemctl start iscsid")
			Expect(err).NotTo(HaveOccurred())
			res, err = isc.Status(ctx, null)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.GetStatus()).To(BeTrue())

		})

		It("List Block Devices test", func() {
			conn, err := grpc.Dial("0.0.0.0:9090", grpc.WithInsecure())
			Expect(err).NotTo(HaveOccurred())
			defer conn.Close()

			ns := protos.NewNodeClient(conn)

			ctx := context.Background()
			null := &protos.Null{}
			res, err := ns.ListBlockDevices(ctx, null)
			Expect(err).NotTo(HaveOccurred())

			// utils.ExecCommandWithSudo("partx -a " + physicalDisk.Name)
			output, err := utils.ExecCommandWithSudo("ln -s " + physicalDisk.Name + " /dev/sdz")
			Expect(err).NotTo(HaveOccurred())
			By(output)

			// partitionName := physicalDisk.Name + "p1"
			// utils.ExecCommandWithSudo("ln -s " + partitionName + "/dev/sdz1")
			partitions := make([]string, 0)
			// partitions = append(partitions, "/dev/sdz1")

			bd := &protos.BlockDevice{
				Name:       "/dev/sdz",
				Type:       "Disk",
				Partitions: partitions,
			}
			bds := make([]*protos.BlockDevice, 0)
			bds = append(bds, bd)
			_ = &protos.BlockDevices{
				Blockdevices: bds,
			}
			Expect(res.GetBlockdevices()).To(Equal(nil))

		})

	})
})
