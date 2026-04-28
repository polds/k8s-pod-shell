package kube

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestResolvePodsDeployment(t *testing.T) {
	cs := fake.NewSimpleClientset(
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-rs",
				Namespace: "demo",
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "Deployment", Name: "foo"},
				},
			},
			Spec: appsv1.ReplicaSetSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "foo"}},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-0",
				Namespace: "demo",
				Labels:    map[string]string{"app": "foo"},
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "ReplicaSet", Name: "foo-rs"},
				},
			},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}},
			Status: corev1.PodStatus{
				Phase:             corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{{Name: "main", Ready: true}},
			},
		},
	)

	pods, err := ResolvePods(context.Background(), cs, Target{
		Kind:      "Deployment",
		Name:      "foo",
		Namespace: "demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pods) != 1 || pods[0].Name != "foo-0" {
		t.Fatalf("unexpected pods: %+v", pods)
	}
}
