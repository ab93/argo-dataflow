// +build e2e

package e2e

import (
	"context"
	"fmt"
	. "github.com/argoproj-labs/argo-dataflow/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/pointer"
	"log"
	"reflect"
)

var (
	stepInterface = dynamicInterface.Resource(StepGroupVersionResource).Namespace(namespace)
)

func waitForStep(f func(Step) bool) {
	log.Printf("watching steps in pipeline %q\n", pipelineName)
	w, err := stepInterface.Watch(context.Background(), metav1.ListOptions{LabelSelector: KeyPipelineName + "=" + pipelineName, TimeoutSeconds: pointer.Int64Ptr(10)})
	if err != nil {
		panic(err)
	}
	defer w.Stop()
	for e := range w.ResultChan() {
		un, ok := e.Object.(*unstructured.Unstructured)
		if !ok {
			panic(fmt.Errorf("expected *unstructured.Unstructured, got %q", reflect.TypeOf(e.Object).Name()))
		}
		x := StepFromUnstructured(un)
		log.Println(fmt.Sprintf("step %q is %s %q", x.Name, x.Status.Phase, x.Status.Message))
		if f(x) {
			return
		}
	}
}