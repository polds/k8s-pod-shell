package kube

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type Target struct {
	Kind      string
	Name      string
	Namespace string
}

type PodInfo struct {
	Name       string   `json:"name"`
	Containers []string `json:"container"`
	Ready      bool     `json:"ready"`
	Node       string   `json:"node"`
	Age        string   `json:"age"`
}

func ResolvePods(ctx context.Context, cs kubernetes.Interface, target Target) ([]PodInfo, error) {
	switch target.Kind {
	case "Deployment":
		return podsForDeployment(ctx, cs, target)
	case "StatefulSet":
		return podsForController(ctx, cs, target.Namespace, "statefulset", target.Name)
	case "DaemonSet":
		return podsForController(ctx, cs, target.Namespace, "daemonset", target.Name)
	default:
		return nil, fmt.Errorf("unsupported target kind: %s", target.Kind)
	}
}

func podsForDeployment(ctx context.Context, cs kubernetes.Interface, target Target) ([]PodInfo, error) {
	rsList, err := cs.AppsV1().ReplicaSets(target.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var all []PodInfo
	for _, rs := range rsList.Items {
		if !ownedBy(&rs, "Deployment", target.Name) {
			continue
		}
		selector, err := metav1.LabelSelectorAsSelector(rs.Spec.Selector)
		if err != nil {
			continue
		}
		pods, err := cs.CoreV1().Pods(target.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
		if err != nil {
			return nil, err
		}
		all = append(all, toPodInfos(pods.Items, "ReplicaSet", rs.Name)...)
	}
	return all, nil
}

func podsForController(ctx context.Context, cs kubernetes.Interface, ns, kind, name string) ([]PodInfo, error) {
	pods, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return toPodInfos(pods.Items, kind, name), nil
}

func toPodInfos(pods []corev1.Pod, ownerKind, ownerName string) []PodInfo {
	out := make([]PodInfo, 0, len(pods))
	for _, p := range pods {
		if !ownedByPod(&p, ownerKind, ownerName) {
			continue
		}
		if p.Status.Phase != corev1.PodRunning || !isPodReady(&p) {
			continue
		}
		containers := make([]string, 0, len(p.Spec.Containers))
		for _, c := range p.Spec.Containers {
			containers = append(containers, c.Name)
		}
		out = append(out, PodInfo{
			Name:       p.Name,
			Containers: containers,
			Ready:      true,
			Node:       p.Spec.NodeName,
			Age:        time.Since(p.CreationTimestamp.Time).Round(time.Second).String(),
		})
	}
	return out
}

func ownedBy(rs *appsv1.ReplicaSet, kind, name string) bool {
	for _, o := range rs.OwnerReferences {
		if o.Kind == kind && o.Name == name {
			return true
		}
	}
	return false
}

func ownedByPod(p *corev1.Pod, kind, name string) bool {
	for _, o := range p.OwnerReferences {
		if kind == "ReplicaSet" && o.Kind == "ReplicaSet" && o.Name == name {
			return true
		}
		if kind == "statefulset" && o.Kind == "StatefulSet" && o.Name == name {
			return true
		}
		if kind == "daemonset" && o.Kind == "DaemonSet" && o.Name == name {
			return true
		}
	}
	return false
}

func isPodReady(p *corev1.Pod) bool {
	for _, c := range p.Status.ContainerStatuses {
		if c.Ready {
			return true
		}
	}
	return false
}

func SelectorFromMap(m map[string]string) string {
	return labels.Set(m).AsSelector().String()
}
