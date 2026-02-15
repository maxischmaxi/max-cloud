package orchestrator

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/max-cloud/shared/pkg/models"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func newTestKnative() (*KnativeOrchestrator, *dynamicfake.FakeDynamicClient, *kubefake.Clientset) {
	scheme := runtime.NewScheme()
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			knativeServiceGVR: "ServiceList",
		},
	)
	cs := kubefake.NewSimpleClientset()
	orch := newKnativeFromClient(slog.Default(), client, cs, "default", "registry.maxcloud.dev", "test-secret")
	return orch, client, cs
}

func TestKnativeDeploy(t *testing.T) {
	orch, client, _ := newTestKnative()
	ctx := context.Background()

	result, err := orch.Deploy(ctx, models.Service{
		Name:     "myapp",
		Image:    "nginx:latest",
		MinScale: 0,
		MaxScale: 5,
		EnvVars:  map[string]string{"PORT": "8080"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != models.ServiceStatusPending {
		t.Fatalf("expected pending, got %s", result.Status)
	}

	_, err = client.Resource(knativeServiceGVR).Namespace("default").Get(ctx, "myapp", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected knative service to exist: %v", err)
	}
}

func TestKnativeDeployIdempotent(t *testing.T) {
	orch, _, _ := newTestKnative()
	ctx := context.Background()

	svc := models.Service{Name: "myapp", Image: "nginx:latest", MaxScale: 10}

	_, err := orch.Deploy(ctx, svc)
	if err != nil {
		t.Fatalf("unexpected error on first deploy: %v", err)
	}

	svc.Image = "nginx:1.25"
	result, err := orch.Deploy(ctx, svc)
	if err != nil {
		t.Fatalf("unexpected error on second deploy: %v", err)
	}
	if result.Status != models.ServiceStatusPending {
		t.Fatalf("expected pending, got %s", result.Status)
	}
}

func TestKnativeRemove(t *testing.T) {
	orch, _, _ := newTestKnative()
	ctx := context.Background()

	svc := models.Service{Name: "myapp", Image: "nginx:latest", MaxScale: 10}
	if _, err := orch.Deploy(ctx, svc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := orch.Remove(ctx, svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKnativeRemoveIdempotent(t *testing.T) {
	orch, _, _ := newTestKnative()
	ctx := context.Background()

	err := orch.Remove(ctx, models.Service{Name: "nonexistent"})
	if err != nil {
		t.Fatalf("expected no error on remove of nonexistent, got %v", err)
	}
}

func TestKnativeStatusNotFound(t *testing.T) {
	orch, _, _ := newTestKnative()
	ctx := context.Background()

	_, err := orch.Status(ctx, models.Service{Name: "nonexistent"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestKnativeStatusReady(t *testing.T) {
	orch, client, _ := newTestKnative()
	ctx := context.Background()

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "serving.knative.dev/v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      "myapp",
				"namespace": "default",
			},
			"status": map[string]interface{}{
				"url": "https://myapp.default.example.com",
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Ready",
						"status": "True",
					},
				},
			},
		},
	}
	if _, err := client.Resource(knativeServiceGVR).Namespace("default").Create(ctx, obj, metav1.CreateOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := orch.Status(ctx, models.Service{Name: "myapp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != models.ServiceStatusReady {
		t.Fatalf("expected ready, got %s", result.Status)
	}
	if result.URL != "https://myapp.default.example.com" {
		t.Fatalf("expected URL https://myapp.default.example.com, got %s", result.URL)
	}
}

func TestKnativeStatusFailed(t *testing.T) {
	orch, client, _ := newTestKnative()
	ctx := context.Background()

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "serving.knative.dev/v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      "failapp",
				"namespace": "default",
			},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Ready",
						"status": "False",
					},
				},
			},
		},
	}
	if _, err := client.Resource(knativeServiceGVR).Namespace("default").Create(ctx, obj, metav1.CreateOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := orch.Status(ctx, models.Service{Name: "failapp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != models.ServiceStatusFailed {
		t.Fatalf("expected failed, got %s", result.Status)
	}
}

func TestKnativeLogsNoPods(t *testing.T) {
	orch, _, _ := newTestKnative()
	ctx := context.Background()

	_, err := orch.Logs(ctx, models.Service{Name: "myapp"}, LogsOptions{Tail: 10})
	if !errors.Is(err, ErrNoPods) {
		t.Fatalf("expected ErrNoPods, got %v", err)
	}
}

func TestKnativeStatusPending(t *testing.T) {
	orch, client, _ := newTestKnative()
	ctx := context.Background()

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "serving.knative.dev/v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      "pendingapp",
				"namespace": "default",
			},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":   "Ready",
						"status": "Unknown",
					},
				},
			},
		},
	}
	if _, err := client.Resource(knativeServiceGVR).Namespace("default").Create(ctx, obj, metav1.CreateOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := orch.Status(ctx, models.Service{Name: "pendingapp"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != models.ServiceStatusPending {
		t.Fatalf("expected pending, got %s", result.Status)
	}
}

func TestKnativeCreateNamespace(t *testing.T) {
	orch, _, cs := newTestKnative()
	ctx := context.Background()

	orgID := "test-org-123"
	err := orch.CreateNamespace(ctx, orgID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nsName := NamespaceFromOrgID(orgID)
	ns, err := cs.CoreV1().Namespaces().Get(ctx, nsName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected namespace to exist: %v", err)
	}
	if ns.Name != nsName {
		t.Fatalf("expected namespace name %s, got %s", nsName, ns.Name)
	}
	if ns.Labels["max-cloud.dev/org-id"] != orgID {
		t.Fatalf("expected org-id label %s, got %s", orgID, ns.Labels["max-cloud.dev/org-id"])
	}
}

func TestKnativeCreateNamespaceIdempotent(t *testing.T) {
	orch, _, _ := newTestKnative()
	ctx := context.Background()

	orgID := "test-org-456"

	err := orch.CreateNamespace(ctx, orgID)
	if err != nil {
		t.Fatalf("unexpected error on first create: %v", err)
	}

	err = orch.CreateNamespace(ctx, orgID)
	if err != nil {
		t.Fatalf("unexpected error on second create: %v", err)
	}
}

func TestKnativeNamespaceExists(t *testing.T) {
	orch, _, _ := newTestKnative()
	ctx := context.Background()

	orgID := "test-org-789"

	exists, err := orch.NamespaceExists(ctx, orgID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatal("expected namespace to not exist")
	}

	if err := orch.CreateNamespace(ctx, orgID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exists, err = orch.NamespaceExists(ctx, orgID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("expected namespace to exist")
	}
}

func TestKnativeDeployToOrgNamespace(t *testing.T) {
	orch, client, _ := newTestKnative()
	ctx := context.Background()

	orgID := "test-org-namespace"
	svc := models.Service{
		Name:     "myapp",
		Image:    "nginx:latest",
		OrgID:    orgID,
		MaxScale: 10,
	}

	if err := orch.CreateNamespace(ctx, orgID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := orch.Deploy(ctx, svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != models.ServiceStatusPending {
		t.Fatalf("expected pending, got %s", result.Status)
	}

	expectedNS := NamespaceFromOrgID(orgID)
	_, err = client.Resource(knativeServiceGVR).Namespace(expectedNS).Get(ctx, "myapp", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected knative service in org namespace %s: %v", expectedNS, err)
	}
}
