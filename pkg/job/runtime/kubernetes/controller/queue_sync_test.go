/*
Copyright (c) 2022 PaddlePaddle Authors. All Rights Reserve.

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

package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic/dynamicinformer"
	fakedynamicclient "k8s.io/client-go/dynamic/fake"
	restclient "k8s.io/client-go/rest"
	"net/http/httptest"

	"paddleflow/pkg/apiserver/models"
	"paddleflow/pkg/common/database/db_fake"
	"paddleflow/pkg/common/k8s"
	"paddleflow/pkg/common/logger"
)

func newFakeQueueSyncController() *QueueSync {
	scheme := runtime.NewScheme()
	dynamicClient := fakedynamicclient.NewSimpleDynamicClient(scheme)

	var server = httptest.NewServer(DiscoveryHandlerFunc)
	defer server.Close()
	fakeDiscovery := discovery.NewDiscoveryClientForConfigOrDie(&restclient.Config{Host: server.URL})

	ctrl := &QueueSync{}
	opt := &k8s.DynamicClientOption{
		DynamicClient:   dynamicClient,
		DynamicFactory:  dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, 0),
		DiscoveryClient: fakeDiscovery,
	}
	ctrl.Initialize(opt)
	return ctrl
}

func TestQueueSync(t *testing.T) {
	tests := []struct {
		name   string
		oldObj *unstructured.Unstructured
		newObj *unstructured.Unstructured
	}{
		{
			name:   "queue create",
			oldObj: NewUnstructured(k8s.VCQueueGVK, "", "q1"),
			newObj: NewUnstructured(k8s.VCQueueGVK, "", "q1"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := &logger.RequestContext{UserName: "test"}
			db_fake.InitFakeDB()
			err := models.CreateQueue(ctx, &models.Queue{
				Name: "q1",
			})
			assert.Equal(t, nil, err)

			c := newFakeQueueSyncController()
			_, err = c.opt.DynamicClient.Resource(VCQueueGVR).Create(context.TODO(), test.newObj, metav1.CreateOptions{})
			assert.Equal(t, nil, err)
			c.updateQueue(test.oldObj, test.newObj)
			c.deleteQueue(test.newObj)

			stopCh := make(chan struct{})
			defer close(stopCh)
			c.Run(stopCh)
			time.Sleep(2 * time.Second)
		})
	}
}