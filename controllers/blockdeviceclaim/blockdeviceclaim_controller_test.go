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

package blockdeviceclaim

import (
	"context"
	"fmt"
	"testing"
	"time"

	bdapis "github.com/openebs/node-disk-manager/apis/blockdevice/v1alpha1"
	bdcapis "github.com/openebs/node-disk-manager/apis/blockdeviceclaim/v1alpha1"
	ndm "github.com/openebs/node-disk-manager/cmd/ndm_daemonset/controller"
	"github.com/openebs/node-disk-manager/db/kubernetes"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	fakeHostName                   = "fake-hostname"
	diskName                       = "disk-example"
	deviceName                     = "blockdevice-example"
	blockDeviceClaimName           = "blockdeviceclaim-example"
	blockDeviceClaimUID  types.UID = "blockDeviceClaim-example-UID"
	namespace                      = ""
	capacity             uint64    = 1024000
	claimCapacity                  = resource.MustParse("1024000")
	fakeRecorder                   = record.NewFakeRecorder(50)
)

// TestBlockDeviceClaimController runs ReconcileBlockDeviceClaim.Reconcile() against a
// fake client that tracks a BlockDeviceClaim object.
func TestBlockDeviceClaimController(t *testing.T) {

	// Set the logger to development mode for verbose logs.
	logf.SetLogger(logf.ZapLogger(true))

	// Create a fake client to mock API calls.
	cl, s := CreateFakeClient()
	deviceR := GetFakeDeviceObject(deviceName, capacity)
	deviceR.Labels[kubernetes.KubernetesHostNameLabel] = fakeHostName

	deviceClaimR := GetFakeBlockDeviceClaimObject()
	// Create a new blockdevice obj
	err := cl.Create(context.TODO(), deviceR)
	if err != nil {
		fmt.Println("BlockDevice object is not created", err)
	}

	// Create a new deviceclaim obj
	err = cl.Create(context.TODO(), deviceClaimR)
	if err != nil {
		fmt.Println("BlockDeviceClaim object is not created", err)
	}

	// Create a ReconcileDevice object with the scheme and fake client.
	r := &BlockDeviceClaimReconciler{Client: cl, Scheme: s, recorder: fakeRecorder}

	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      blockDeviceClaimName,
			Namespace: namespace,
		},
	}

	// Check status of deviceClaim it should be empty(Not bound)
	r.CheckBlockDeviceClaimStatus(t, req, bdcapis.BlockDeviceClaimStatusEmpty)

	// Fetch the BlockDeviceClaim CR and change capacity to invalid
	// Since Capacity is invalid, it delete device claim CR
	r.InvalidCapacityTest(t, req)

	// Create new BlockDeviceClaim CR with right capacity,
	// trigger reconciliation event. This time, it should
	// bound.
	deviceClaim := &bdcapis.BlockDeviceClaim{}
	err = r.Client.Get(context.TODO(), req.NamespacedName, deviceClaim)
	if err != nil {
		t.Errorf("Get deviceClaim: (%v)", err)
	}

	deviceClaim.Spec.Resources.Requests[bdcapis.ResourceStorage] = claimCapacity
	// resetting status to empty
	deviceClaim.Status.Phase = bdcapis.BlockDeviceClaimStatusEmpty
	err = r.Client.Update(context.TODO(), deviceClaim)
	if err != nil {
		t.Errorf("Update deviceClaim: (%v)", err)
	}

	res, err := r.Reconcile(req)
	if err != nil {
		t.Logf("reconcile: (%v)", err)
	}

	// Check the result of reconciliation to make sure it has the desired state.
	if !res.Requeue {
		t.Log("reconcile did not requeue request as expected")
	}
	r.CheckBlockDeviceClaimStatus(t, req, bdcapis.BlockDeviceClaimStatusDone)

	r.DeviceRequestedHappyPathTest(t, req)
	//TODO: Need to find a way to update deletion timestamp
	//r.DeleteBlockDeviceClaimedTest(t, req)
}

func (r *BlockDeviceClaimReconciler) DeleteBlockDeviceClaimedTest(t *testing.T,
	req reconcile.Request) {

	devRequestInst := &bdcapis.BlockDeviceClaim{}

	// Fetch the BlockDeviceClaim CR
	err := r.Client.Get(context.TODO(), req.NamespacedName, devRequestInst)
	if err != nil {
		t.Errorf("Get devClaimInst: (%v)", err)
	}

	err = r.Client.Delete(context.TODO(), devRequestInst)
	if err != nil {
		t.Errorf("Delete devClaimInst: (%v)", err)
	}

	res, err := r.Reconcile(req)
	if err != nil {
		t.Logf("reconcile: (%v)", err)
	}

	// Check the result of reconciliation to make sure it has the desired state.
	if !res.Requeue {
		t.Log("reconcile did not requeue request as expected")
	}

	dvRequestInst := &bdcapis.BlockDeviceClaim{}
	err = r.Client.Get(context.TODO(), req.NamespacedName, dvRequestInst)
	if errors.IsNotFound(err) {
		t.Logf("BlockDeviceClaim is deleted, expected")
		err = nil
	} else if err != nil {
		t.Errorf("Get dvClaimInst: (%v)", err)
	}

	time.Sleep(10 * time.Second)
	// Fetch the BlockDevice CR
	devInst := &bdapis.BlockDevice{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: deviceName, Namespace: namespace}, devInst)
	if err != nil {
		t.Errorf("get devInst: (%v)", err)
	}

	if devInst.Spec.ClaimRef.UID == dvRequestInst.ObjectMeta.UID {
		t.Logf("BlockDevice ObjRef UID:%v match expected deviceRequest UID:%v",
			devInst.Spec.ClaimRef.UID, dvRequestInst.ObjectMeta.UID)
	} else {
		t.Fatalf("BlockDevice ObjRef UID:%v did not match expected deviceRequest UID:%v",
			devInst.Spec.ClaimRef.UID, dvRequestInst.ObjectMeta.UID)
	}

	if devInst.Status.ClaimState == bdapis.BlockDeviceClaimed {
		t.Logf("BlockDevice Obj state:%v match expected state:%v",
			devInst.Status.ClaimState, bdapis.BlockDeviceClaimed)
	} else {
		t.Fatalf("BlockDevice Obj state:%v did not match expected state:%v",
			devInst.Status.ClaimState, bdapis.BlockDeviceClaimed)
	}
}

func (r *BlockDeviceClaimReconciler) DeviceRequestedHappyPathTest(t *testing.T,
	req reconcile.Request) {

	devRequestInst := &bdcapis.BlockDeviceClaim{}
	// Fetch the BlockDeviceClaim CR
	err := r.Client.Get(context.TODO(), req.NamespacedName, devRequestInst)
	if err != nil {
		t.Errorf("Get devRequestInst: (%v)", err)
	}

	// Fetch the BlockDevice CR
	devInst := &bdapis.BlockDevice{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: deviceName, Namespace: namespace}, devInst)
	if err != nil {
		t.Errorf("get devInst: (%v)", err)
	}

	if devInst.Spec.ClaimRef.UID == devRequestInst.ObjectMeta.UID {
		t.Logf("BlockDevice ObjRef UID:%v match expected deviceRequest UID:%v",
			devInst.Spec.ClaimRef.UID, devRequestInst.ObjectMeta.UID)
	} else {
		t.Fatalf("BlockDevice ObjRef UID:%v did not match expected deviceRequest UID:%v",
			devInst.Spec.ClaimRef.UID, devRequestInst.ObjectMeta.UID)
	}

	if devInst.Status.ClaimState == bdapis.BlockDeviceClaimed {
		t.Logf("BlockDevice Obj state:%v match expected state:%v",
			devInst.Status.ClaimState, bdapis.BlockDeviceClaimed)
	} else {
		t.Fatalf("BlockDevice Obj state:%v did not match expected state:%v",
			devInst.Status.ClaimState, bdapis.BlockDeviceClaimed)
	}
}

func (r *BlockDeviceClaimReconciler) InvalidCapacityTest(t *testing.T,
	req reconcile.Request) {

	devRequestInst := &bdcapis.BlockDeviceClaim{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, devRequestInst)
	if err != nil {
		t.Errorf("Get devRequestInst: (%v)", err)
	}

	devRequestInst.Spec.Resources.Requests[bdcapis.ResourceStorage] = resource.MustParse("0")
	err = r.Client.Update(context.TODO(), devRequestInst)
	if err != nil {
		t.Errorf("Update devRequestInst: (%v)", err)
	}

	res, err := r.Reconcile(req)
	if err != nil {
		t.Logf("reconcile: (%v)", err)
	}

	// Check the result of reconciliation to make sure it has the desired state.
	if !res.Requeue {
		t.Log("reconcile did not requeue request as expected")
	}

	dvC := &bdcapis.BlockDeviceClaim{}
	err = r.Client.Get(context.TODO(), req.NamespacedName, dvC)
	if err != nil {
		t.Errorf("Get devRequestInst: (%v)", err)
	}
	r.CheckBlockDeviceClaimStatus(t, req, bdcapis.BlockDeviceClaimStatusPending)
}

func TestBlockDeviceClaimsLabelSelector(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	logf.SetLogger(logf.ZapLogger(true))

	tests := map[string]struct {
		bdLabels           map[string]string
		selector           *metav1.LabelSelector
		expectedClaimPhase bdcapis.DeviceClaimPhase
	}{
		"only hostname label is present and no selector": {
			bdLabels: map[string]string{
				kubernetes.KubernetesHostNameLabel: fakeHostName,
			},
			selector:           nil,
			expectedClaimPhase: bdcapis.BlockDeviceClaimStatusDone,
		},
		"custom label and hostname present on bd and selector": {
			bdLabels: map[string]string{
				ndm.KubernetesHostNameLabel: fakeHostName,
				"ndm.io/test":               "1234",
			},
			selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					ndm.KubernetesHostNameLabel: fakeHostName,
					"ndm.io/test":               "1234",
				},
			},
			expectedClaimPhase: bdcapis.BlockDeviceClaimStatusDone,
		},
		"custom labels and hostname": {
			bdLabels: map[string]string{
				ndm.KubernetesHostNameLabel: fakeHostName,
				"ndm.io/test":               "1234",
			},
			selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"ndm.io/test": "1234",
				},
			},
			expectedClaimPhase: bdcapis.BlockDeviceClaimStatusDone,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// pinning the variables
			bdLabels := test.bdLabels
			selector := test.selector
			expectedClaimPhase := test.expectedClaimPhase

			// Create a fake client to mock API calls.
			cl, s := CreateFakeClient()

			// Create a ReconcileDevice object with the scheme and fake client.
			r := &BlockDeviceClaimReconciler{Client: cl, Scheme: s, recorder: fakeRecorder}

			// Mock request to simulate Reconcile() being called on an event for a
			// watched resource .
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      blockDeviceClaimName,
					Namespace: namespace,
				},
			}

			bdcR := GetFakeBlockDeviceClaimObject()
			if err := cl.Create(context.TODO(), bdcR); err != nil {
				t.Fatal(err)
			}

			bd := GetFakeDeviceObject("bd-1", capacity*10)

			bd.Labels = bdLabels
			// mark the device as unclaimed
			bd.Spec.ClaimRef = nil
			bd.Status.ClaimState = bdapis.BlockDeviceUnclaimed

			err := cl.Create(context.TODO(), bd)
			if err != nil {
				t.Fatalf("error updating BD. %v", err)
			}
			bdc := GetFakeBlockDeviceClaimObject()
			bdc.Spec.BlockDeviceName = ""
			bdc.Spec.Selector = selector
			err = cl.Update(context.TODO(), bdc)
			if err != nil {
				t.Fatalf("error updating BDC. %v", err)
			}
			_, _ = r.Reconcile(req)

			err = cl.Get(context.TODO(), req.NamespacedName, bdc)
			if err != nil {
				t.Fatalf("error getting BDC. %v", err)
			}

			err = cl.Delete(context.TODO(), bd)
			if err != nil {
				t.Fatalf("error deleting BDC. %v", err)
			}
			assert.Equal(t, expectedClaimPhase, bdc.Status.Phase)
		})
	}
}

func (r *BlockDeviceClaimReconciler) CheckBlockDeviceClaimStatus(t *testing.T,
	req reconcile.Request, phase bdcapis.DeviceClaimPhase) {

	devRequestCR := &bdcapis.BlockDeviceClaim{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, devRequestCR)
	if err != nil {
		t.Errorf("get devRequestCR : (%v)", err)
	}

	// BlockDeviceClaim should yet to bound.
	if devRequestCR.Status.Phase == phase {
		t.Logf("BlockDeviceClaim Object status:%v match expected status:%v",
			devRequestCR.Status.Phase, phase)
	} else {
		t.Fatalf("BlockDeviceClaim Object status:%v did not match expected status:%v",
			devRequestCR.Status.Phase, phase)
	}
}

func GetFakeBlockDeviceClaimObject() *bdcapis.BlockDeviceClaim {
	deviceRequestCR := &bdcapis.BlockDeviceClaim{}

	TypeMeta := metav1.TypeMeta{
		Kind:       "BlockDeviceClaim",
		APIVersion: ndm.NDMVersion,
	}

	ObjectMeta := metav1.ObjectMeta{
		Labels:    make(map[string]string),
		Name:      blockDeviceClaimName,
		Namespace: namespace,
		UID:       blockDeviceClaimUID,
	}

	Requests := corev1.ResourceList{bdcapis.ResourceStorage: claimCapacity}

	Requirements := bdcapis.DeviceClaimResources{
		Requests: Requests,
	}

	Spec := bdcapis.DeviceClaimSpec{
		Resources:  Requirements,
		DeviceType: "",
		HostName:   fakeHostName,
	}

	deviceRequestCR.ObjectMeta = ObjectMeta
	deviceRequestCR.TypeMeta = TypeMeta
	deviceRequestCR.Spec = Spec
	deviceRequestCR.Status.Phase = bdcapis.BlockDeviceClaimStatusEmpty
	return deviceRequestCR
}

func GetFakeDeviceObject(bdName string, bdCapacity uint64) *bdapis.BlockDevice {
	device := &bdapis.BlockDevice{}

	TypeMeta := metav1.TypeMeta{
		Kind:       ndm.NDMBlockDeviceKind,
		APIVersion: ndm.NDMVersion,
	}

	ObjectMeta := metav1.ObjectMeta{
		Labels:    make(map[string]string),
		Name:      bdName,
		Namespace: namespace,
	}

	Spec := bdapis.DeviceSpec{
		Path: "dev/disk-fake-path",
		Capacity: bdapis.DeviceCapacity{
			Storage: bdCapacity, // Set blockdevice size.
		},
		DevLinks:    make([]bdapis.DeviceDevLink, 0),
		Partitioned: ndm.NDMNotPartitioned,
	}

	device.ObjectMeta = ObjectMeta
	device.TypeMeta = TypeMeta
	device.Status.ClaimState = bdapis.BlockDeviceUnclaimed
	device.Status.State = ndm.NDMActive
	device.Spec = Spec
	return device
}

func CreateFakeClient() (client.Client, *runtime.Scheme) {

	deviceR := GetFakeDeviceObject(deviceName, capacity)

	deviceList := &bdapis.BlockDeviceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "BlockDevice",
			APIVersion: "",
		},
	}

	deviceClaimR := GetFakeBlockDeviceClaimObject()
	deviceclaimList := &bdcapis.BlockDeviceClaimList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "BlockDeviceClaim",
			APIVersion: "",
		},
	}

	s := scheme.Scheme

	s.AddKnownTypes(bdcapis.GroupVersion, deviceR)
	s.AddKnownTypes(bdcapis.GroupVersion, deviceList)
	s.AddKnownTypes(bdcapis.GroupVersion, deviceClaimR)
	s.AddKnownTypes(bdcapis.GroupVersion, deviceclaimList)

	fakeNdmClient := fake.NewFakeClientWithScheme(s)
	if fakeNdmClient == nil {
		fmt.Println("NDMClient is not created")
	}

	return fakeNdmClient, s
}

func TestGenerateSelector(t *testing.T) {
	tests := map[string]struct {
		bdc  bdcapis.BlockDeviceClaim
		want *metav1.LabelSelector
	}{
		"hostname/node attributes not given and no selector": {
			bdc: bdcapis.BlockDeviceClaim{
				Spec: bdcapis.DeviceClaimSpec{},
			},
			want: &metav1.LabelSelector{
				MatchLabels: make(map[string]string),
			},
		},
		"hostname is given, node attributes not given and no selector": {
			bdc: bdcapis.BlockDeviceClaim{
				Spec: bdcapis.DeviceClaimSpec{
					HostName: "hostname",
				},
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					ndm.KubernetesHostNameLabel: "hostname",
				},
			},
		},
		"hostname is not given, node attribute is given and no selector": {
			bdc: bdcapis.BlockDeviceClaim{
				Spec: bdcapis.DeviceClaimSpec{
					BlockDeviceNodeAttributes: bdcapis.BlockDeviceNodeAttributes{
						HostName: "hostname",
					},
				},
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					ndm.KubernetesHostNameLabel: "hostname",
				},
			},
		},
		"same hostname, node attribute is given and no selector": {
			bdc: bdcapis.BlockDeviceClaim{
				Spec: bdcapis.DeviceClaimSpec{
					HostName: "hostname",
					BlockDeviceNodeAttributes: bdcapis.BlockDeviceNodeAttributes{
						HostName: "hostname",
					},
				},
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					ndm.KubernetesHostNameLabel: "hostname",
				},
			},
		},
		"different hostname and node attributes is given and no selector": {
			bdc: bdcapis.BlockDeviceClaim{
				Spec: bdcapis.DeviceClaimSpec{
					HostName: "hostname1",
					BlockDeviceNodeAttributes: bdcapis.BlockDeviceNodeAttributes{
						HostName: "hostname2",
					},
				},
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					ndm.KubernetesHostNameLabel: "hostname2",
				},
			},
		},
		"no hostname and custom selector is given": {
			bdc: bdcapis.BlockDeviceClaim{
				Spec: bdcapis.DeviceClaimSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"ndm.io/test": "test",
						},
					},
				},
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"ndm.io/test": "test",
				},
			},
		},
		"hostname given and selector also contains custom label name": {
			bdc: bdcapis.BlockDeviceClaim{
				Spec: bdcapis.DeviceClaimSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							ndm.KubernetesHostNameLabel: "hostname1",
							"ndm.io/test":               "test",
						},
					},
					HostName: "hostname2",
				},
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					ndm.KubernetesHostNameLabel: "hostname2",
					"ndm.io/test":               "test",
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := generateSelector(test.bdc)
			assert.Equal(t, test.want, got)
		})
	}
}
