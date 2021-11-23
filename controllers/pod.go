package controllers

import (
	"context"
	"strconv"

	"github.com/CodeMonk123/dlpipe-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *DLpipeJobReconciler) CreateWorker(ctx context.Context, dlpipeJob *v1alpha1.DLpipeJob, rank int) error {
	podTemplate := dlpipeJob.Spec.JobTemplate.DeepCopy()
	envWorldSize := corev1.EnvVar{Name: "WorldSize", Value: strconv.Itoa(int(*dlpipeJob.Spec.WorldSize))}
	envRank := corev1.EnvVar{Name: "RANK", Value: strconv.Itoa(rank)}
	envMasterAddr := corev1.EnvVar{Name: "MASTER_ADDR", Value: strconv.Itoa(rank)}
	for _, container := range podTemplate.Containers {
		container.Env = append(container.Env, envMasterAddr)
		container.Env = append(container.Env, envRank)
		container.Env = append(container.Env, envWorldSize)
	}

	workerPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dlpipeJob.Name + "-worker-" + strconv.Itoa(rank),
			Namespace: dlpipeJob.Namespace,
			Labels:    map[string]string{"job": dlpipeJob.Name},
		},
		Spec: *podTemplate,
	}
	controllerutil.SetOwnerReference(dlpipeJob, &workerPod, r.Scheme)
	err := r.Client.Create(ctx, &workerPod)
	return err
}

func (r *DLpipeJobReconciler) CreateMaster(ctx context.Context, dlpipeJob *v1alpha1.DLpipeJob) error {
	podTemplate := dlpipeJob.Spec.JobTemplate.DeepCopy()
	envWorldSize := corev1.EnvVar{Name: "WorldSize", Value: strconv.Itoa(int(*dlpipeJob.Spec.WorldSize))}
	for _, container := range podTemplate.Containers {
		container.Env = append(container.Env, envWorldSize)
	}
	masterPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dlpipeJob.Name + "-master",
			Namespace: dlpipeJob.Namespace,
			Labels:    map[string]string{"job": dlpipeJob.Name},
		},
		Spec: *podTemplate,
	}
	controllerutil.SetOwnerReference(dlpipeJob, &masterPod, r.Scheme)
	err := r.Client.Create(ctx, &masterPod)
	return err
}

func (r *DLpipeJobReconciler) IsMasterReady(ctx context.Context, dlpipeJob *v1alpha1.DLpipeJob) (error, bool) {
	masterPod := corev1.Pod{}
	masterNamespacedName := types.NamespacedName{
		Namespace: dlpipeJob.Namespace,
		Name:      dlpipeJob.Name + "-master",
	}

	err := r.Client.Get(ctx, masterNamespacedName, &masterPod)
	if err != nil {
		return err, false
	}

	if masterPod.Status.Phase == corev1.PodRunning {
		if masterPod.Status.PodIP != "" {
			return nil, true
		}
	}
	return nil, false
}

func (r *DLpipeJobReconciler) GetMasterAddr(ctx context.Context, dlpipeJob *v1alpha1.DLpipeJob) (error, string) {
	masterPod := corev1.Pod{}
	masterNamespacedName := types.NamespacedName{
		Namespace: dlpipeJob.Namespace,
		Name:      dlpipeJob.Name + "-master",
	}
	err := r.Client.Get(ctx, masterNamespacedName, &masterPod)
	if err != nil {
		return err, ""
	}
	return nil, masterPod.Status.PodIP
}

func (r *DLpipeJobReconciler) IsJobFailed(ctx context.Context, dlpipeJob *v1alpha1.DLpipeJob) bool {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"job": dlpipeJob.Name}}
	labelMap, _ := metav1.LabelSelectorAsMap(&labelSelector)
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	}

	pods, _ := r.KubeClient.CoreV1().Pods(dlpipeJob.Namespace).List(ctx, listOptions)
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodUnknown {
			return true
		}
	}
	return false
}

func (r *DLpipeJobReconciler) IsJobSucceeded(ctx context.Context, dlpipeJob *v1alpha1.DLpipeJob) bool {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"job": dlpipeJob.Name}}
	labelMap, _ := metav1.LabelSelectorAsMap(&labelSelector)
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	}

	pods, _ := r.KubeClient.CoreV1().Pods(dlpipeJob.Namespace).List(ctx, listOptions)
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodSucceeded {
			return false
		}
	}
	return true
}

func (r *DLpipeJobReconciler) DeleteAllPod(ctx context.Context, dlpipeJob *v1alpha1.DLpipeJob) {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"job": dlpipeJob.Name}}
	labelMap, _ := metav1.LabelSelectorAsMap(&labelSelector)
	listOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	}

	pods, _ := r.KubeClient.CoreV1().Pods(dlpipeJob.Namespace).List(ctx, listOptions)
	for _, pod := range pods.Items {
		_ = r.Client.Delete(ctx, &pod)
	}
}
