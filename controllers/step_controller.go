/*


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

package controllers

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dfv1 "github.com/argoproj-labs/argo-dataflow/api/v1alpha1"
)

// StepReconciler reconciles a Step object
type StepReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	RESTConfig *rest.Config
	Kubernetes kubernetes.Interface
}

// +kubebuilder:rbac:groups=dataflow.argoproj.io,resources=steps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dataflow.argoproj.io,resources=steps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=,resources=pods,verbs=get;watch;list;create
func (r *StepReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("step", req.NamespacedName)

	step := &dfv1.Step{}
	if err := r.Get(ctx, req.NamespacedName, step); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if step.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}

	pipelineName := step.GetLabels()[dfv1.KeyPipelineName]
	lastReplicas := step.Status.GetReplicas()
	newReplicas := step.Spec.GetReplicas().Calculate(step.Status.GetSourceStatues().GetPending())

	replicas := lastReplicas
	if step.Status.LastScaleTime == nil || time.Since(step.Status.LastScaleTime.Time) > time.Minute {
		replicas = newReplicas
	}

	log.Info("reconciling", "lastReplicas", lastReplicas, "newReplicas", newReplicas, "replicas", replicas, "pipelineName", pipelineName)

	volMnt := corev1.VolumeMount{Name: "var-run-argo-dataflow", MountPath: "/var/run/argo-dataflow"}
	container := step.Spec.GetContainer()
	container.VolumeMounts = append(container.VolumeMounts, volMnt)

	for replica := 0; replica < replicas; replica++ {
		podName := fmt.Sprintf("%s-%d", step.Name, replica)
		log.Info("creating pod (if not exists)", "podName", podName)
		envVars := []corev1.EnvVar{
			{Name: dfv1.EnvPipelineName, Value: pipelineName},
			{Name: dfv1.EnvReplica, Value: strconv.Itoa(replica)},
			{Name: dfv1.EnvStep, Value: dfv1.Json(step)},
		}
		if err := r.Client.Create(
			ctx,
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        podName,
					Namespace:   step.Namespace,
					Labels:      map[string]string{dfv1.KeyStepName: step.Name, dfv1.KeyPipelineName: pipelineName},
					Annotations: map[string]string{dfv1.KeyReplica: strconv.Itoa(replica)},
					OwnerReferences: []metav1.OwnerReference{
						*metav1.NewControllerRef(step.GetObjectMeta(), dfv1.GroupVersion.WithKind("Step")),
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: step.Spec.GetRestartPolicy(),
					Volumes: append(
						step.Spec.GetVolumes(),
						corev1.Volume{
							Name:         volMnt.Name,
							VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
						},
					),
					InitContainers: []corev1.Container{
						{
							Name:            dfv1.CtrInit,
							Image:           runnerImage,
							Args:            []string{"init"},
							Env:             envVars,
							ImagePullPolicy: imagePullPolicy,
							VolumeMounts:    []corev1.VolumeMount{volMnt},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            dfv1.CtrSidecar,
							Image:           runnerImage,
							ImagePullPolicy: imagePullPolicy,
							Args:            []string{"sidecar"},
							Env:             envVars,
							VolumeMounts:    []corev1.VolumeMount{volMnt},
						},
						container,
					},
				},
			},
		); IgnoreAlreadyExists(err) != nil {
			return ctrl.Result{}, err
		}
	}

	pods := &corev1.PodList{}
	selector, _ := labels.Parse(dfv1.KeyStepName + "=" + step.Name)
	if err := r.Client.List(ctx, pods, &client.ListOptions{LabelSelector: selector}); err != nil {
		return ctrl.Result{}, err
	}

	newStatus := &dfv1.StepStatus{Phase: dfv1.StepUnknown, Replicas: uint32(replicas)}

	if lastReplicas != replicas {
		newStatus.LastScaleTime = &metav1.Time{Time: time.Now()}
	}

	for _, pod := range pods.Items {
		if i, _ := strconv.Atoi(pod.GetAnnotations()[dfv1.KeyReplica]); i >= replicas {
			if err := r.Client.Delete(ctx, &pod); client.IgnoreNotFound(err) != nil {
				return ctrl.Result{}, err
			}
		} else {
			phase := inferPhase(pod)
			log.Info("inspecting pod", "name", pod.Name, "phase", phase, "message", pod.Status.Message)
			newStatus.Phase = dfv1.MinStepPhase(newStatus.Phase, phase)
			if newStatus.Phase.Completed() && pod.Status.Message != "" {
				newStatus.Message = pod.Status.Message
			}

			for _, s := range pod.Status.ContainerStatuses {
				if s.Name != "main" || s.State.Terminated == nil {
					continue
				}
				url := r.Kubernetes.CoreV1().RESTClient().Post().
					Resource("pods").
					Name(pod.Name).
					Namespace(pod.Namespace).
					SubResource("exec").
					Param("container", dfv1.CtrSidecar).
					Param("stdout", "true").
					Param("stderr", "true").
					Param("tty", "false").
					Param("command", "/runner").
					Param("command", "kill").
					URL()
				log.Info("killing sidecar", "url", url)
				exec, err := remotecommand.NewSPDYExecutor(r.RESTConfig, "POST", url)
				if err != nil {
					return ctrl.Result{}, fmt.Errorf("failed to exec %w", err)
				}
				if err := exec.Stream(remotecommand.StreamOptions{
					Stdout: os.Stdout,
					Stderr: os.Stderr,
					Tty:    true,
				}); IgnoreContainerNotFound(err) != nil {
					return ctrl.Result{}, fmt.Errorf("failed to stream %w", err)
				}
			}
		}
	}

	if !reflect.DeepEqual(step.Status, newStatus) {
		log.Info("patching func status (phase/message)", "phase", newStatus.Phase)
		if err := r.Status().
			Patch(ctx, step, client.RawPatch(types.MergePatchType, []byte(dfv1.Json(&dfv1.Step{Status: newStatus}))));
			IgnoreConflict(err) != nil { // conflict is ok, we will reconcile again soon
			return ctrl.Result{}, fmt.Errorf("failed to patch status: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *StepReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dfv1.Step{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}