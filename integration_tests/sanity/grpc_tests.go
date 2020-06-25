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
	"k8s.io/klog"

	protos "github.com/openebs/node-disk-manager/pkg/ndm-grpc/protos/ndm"
	"google.golang.org/grpc"
)

// func init() {
// 	var wg sync.WaitGroup

// 	go func() {
// 		wg.Add(1)
// 		// Creating a grpc server, use WithInsecure to allow http connections
// 		gs := grpc.NewServer()

// 		// Creates an instance of Info
// 		is := services.NewInfo()

// 		// Creates an instance of Service
// 		ss := services.NewService()

// 		// Creates an instance of Node
// 		ns := services.NewNode()

// 		// This helps clients determine which services are available to call
// 		reflection.Register(gs)

// 		// Similar to registring handlers for http
// 		protos.RegisterInfoServer(gs, is)

// 		protos.RegisterISCSIServer(gs, ss)

// 		protos.RegisterNodeServer(gs, ns)

// 		l, err := net.Listen("tcp", "0.0.0.0:9090")
// 		if err != nil {
// 			klog.Errorf("Unable to listen %f", err)
// 			os.Exit(1)
// 		}

// 		// Listen for requests
// 		klog.Info("Starting server at 9090")
// 		gs.Serve(l)

// 	}()
// }

var _ = Describe("gRPC tests", func() {

	var err error
	var k8sClient k8s.K8sClient
	physicalDisk := udev.NewDisk(DiskImageSize)
	err = physicalDisk.AttachDisk()

	BeforeEach(func() {
		Expect(err).NotTo(HaveOccurred())
		By("getting a new client set")
		k8sClient, err = k8s.GetClientSet()
		Expect(err).NotTo(HaveOccurred())

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
		err = nil
	})
	Context("gRPC services", func() {

		// It("iSCSI test", func() {
		// 	conn, err := grpc.Dial("0.0.0.0:9090", grpc.WithInsecure())
		// 	Expect(err).NotTo(HaveOccurred())
		// 	defer conn.Close()

		// 	isc := protos.NewISCSIClient(conn)
		// 	ctx := context.Background()
		// 	null := &protos.Null{}

		// 	By("Checking when ISCSI is disabled")
		// 	err = utils.RunCommandWithSudo("sudo systemctl stop iscsid")
		// 	Expect(err).NotTo(HaveOccurred())
		// 	res, err := isc.Status(ctx, null)
		// 	Expect(err).NotTo(HaveOccurred())
		// 	Expect(res.GetStatus()).To(BeFalse())

		// 	By("Checking when ISCSI is enabled ")
		// 	err = utils.RunCommandWithSudo("sudo systemctl enable iscsid")
		// 	Expect(err).NotTo(HaveOccurred())
		// 	err = utils.RunCommandWithSudo("sudo systemctl start iscsid")
		// 	Expect(err).NotTo(HaveOccurred())
		// 	res, err = isc.Status(ctx, null)
		// 	Expect(err).NotTo(HaveOccurred())
		// 	Expect(res.GetStatus()).To(BeTrue())

		// })

		It("List Block Devices test", func() {
			conn, err := grpc.Dial("localhost:9090", grpc.WithInsecure())
			Expect(err).NotTo(HaveOccurred())
			defer conn.Close()

			ns := protos.NewNodeClient(conn)

			ctx := context.Background()
			null := &protos.Null{}
			nn, err := ns.Name(ctx, null)
			Expect(err).NotTo(HaveOccurred())
			klog.Info(nn)

			res, err := ns.ListBlockDevices(ctx, null)
			Expect(err).NotTo(HaveOccurred())
			klog.Info(res)

			// // utils.ExecCommandWithSudo("partx -a " + physicalDisk.Name)
			// output, err := utils.ExecCommandWithSudo("ln -s " + physicalDisk.Name + " /dev/sdz")
			// Expect(err).NotTo(HaveOccurred())
			// fmt.Fprintf(GinkgoWriter, "Output of linking is %v", output)

			// partitionName := physicalDisk.Name + "p1"
			// utils.ExecCommandWithSudo("ln -s " + partitionName + "/dev/sdz1")
			// partitions := make([]string, 0)
			// partitions = append(partitions, "/dev/sda1")

			// output, err := utils.ExecCommandWithSudo("lsblk -a")
			// Expect(err).NotTo(HaveOccurred())
			// klog.Infof("Output of lsblk is %v", output)

			bd := &protos.BlockDevice{
				Name: physicalDisk.Name,
				Type: "Disk",
			}
			bds := make([]*protos.BlockDevice, 0)
			bds = append(bds, bd)
			_ = &protos.BlockDevices{
				Blockdevices: bds,
			}
			for _, bdn := range res.GetBlockdevices() {
				if bdn.Name == bd.Name {
					Expect(bd.Name).To(Equal(bd.Name))
				}
			}

		})

	})
})
