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
	protos "github.com/openebs/node-disk-manager/pkg/ndm-grpc/protos/ndm"
	"google.golang.org/grpc"
	"k8s.io/klog"
)

var _ = Describe("gRPC tests", func() {

	var err error
	var k8sClient k8s.K8sClient

	BeforeEach(func() {
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

		conn, err := grpc.Dial("localhost:9090", grpc.WithInsecure())
		if err != nil {
			klog.Errorf("connection failed: %v", err)
		}
		defer conn.Close()

		It("iSCSI test", func() {

			isc := protos.NewISCSIClient(conn)
			ctx := context.Background()
			null := &protos.Null{}

			By("Checking when ISCSI is disabled")
			res, err := isc.Status(ctx, null)
			Expect(err).NotTo(HaveOccured())
			Expect(res.GetStatus()).To(Equal(false))
		})
	})
})
