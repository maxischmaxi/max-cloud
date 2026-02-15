package orchestrator

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/max-cloud/shared/pkg/models"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var knativeServiceGVR = schema.GroupVersionResource{
	Group:    "serving.knative.dev",
	Version:  "v1",
	Resource: "services",
}

// OrgNamespacePrefix is the prefix for organization namespaces.
const OrgNamespacePrefix = "mc-org-"

// KnativeOrchestrator erstellt Knative Services via k8s Dynamic Client.
type KnativeOrchestrator struct {
	client            dynamic.Interface
	clientset         kubernetes.Interface
	defaultNS         string
	logger            *slog.Logger
	registryURL       string
	registryJWTSecret string
}

// NewKnative erstellt einen KnativeOrchestrator mit kubeconfig.
func NewKnative(logger *slog.Logger, kubeconfigPath string, defaultNamespace string, registryURL string, registryJWTSecret string) (*KnativeOrchestrator, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("building kubeconfig: %w", err)
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating kubernetes clientset: %w", err)
	}

	return &KnativeOrchestrator{
		client:            client,
		clientset:         cs,
		defaultNS:         defaultNamespace,
		logger:            logger,
		registryURL:       registryURL,
		registryJWTSecret: registryJWTSecret,
	}, nil
}

// newKnativeFromClient erstellt einen KnativeOrchestrator mit vorhandenen Clients (für Tests).
func newKnativeFromClient(logger *slog.Logger, client dynamic.Interface, cs kubernetes.Interface, defaultNamespace string, registryURL string, registryJWTSecret string) *KnativeOrchestrator {
	return &KnativeOrchestrator{
		client:            client,
		clientset:         cs,
		defaultNS:         defaultNamespace,
		logger:            logger,
		registryURL:       registryURL,
		registryJWTSecret: registryJWTSecret,
	}
}

// namespaceForService returns the namespace for a service, using orgID if available.
func (k *KnativeOrchestrator) namespaceForService(svc models.Service) string {
	if svc.OrgID != "" {
		return OrgNamespacePrefix + svc.OrgID
	}
	return k.defaultNS
}

// NamespaceFromOrgID returns the Kubernetes namespace name for an organization.
func NamespaceFromOrgID(orgID string) string {
	return OrgNamespacePrefix + orgID
}

// CreateNamespace erstellt einen Kubernetes Namespace für eine Organisation.
func (k *KnativeOrchestrator) CreateNamespace(ctx context.Context, orgID string) error {
	nsName := NamespaceFromOrgID(orgID)

	_, err := k.clientset.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "max-cloud",
				"max-cloud.dev/org-id":         orgID,
			},
		},
	}, metav1.CreateOptions{})

	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			k.logger.Info("namespace already exists", "namespace", nsName)
			return nil
		}
		return fmt.Errorf("creating namespace %s: %w", nsName, err)
	}

	k.logger.Info("namespace created", "namespace", nsName, "org_id", orgID)
	return nil
}

// NamespaceExists prüft ob ein Namespace existiert.
func (k *KnativeOrchestrator) NamespaceExists(ctx context.Context, orgID string) (bool, error) {
	nsName := NamespaceFromOrgID(orgID)

	_, err := k.clientset.CoreV1().Namespaces().Get(ctx, nsName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("checking namespace %s: %w", nsName, err)
	}
	return true, nil
}

func (k *KnativeOrchestrator) Deploy(ctx context.Context, svc models.Service) (*DeployResult, error) {
	ns := k.namespaceForService(svc)
	obj := k.buildKnativeService(svc, ns)

	_, err := k.client.Resource(knativeServiceGVR).Namespace(ns).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			existing, err := k.client.Resource(knativeServiceGVR).Namespace(ns).Get(ctx, svc.Name, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("getting existing service: %w", err)
			}
			obj.SetResourceVersion(existing.GetResourceVersion())
			obj.SetUID(existing.GetUID())
			obj.SetCreationTimestamp(existing.GetCreationTimestamp())
			obj.SetGeneration(existing.GetGeneration())
			existingAnnotations := existing.GetAnnotations()
			if existingAnnotations != nil {
				objAnnotations := obj.GetAnnotations()
				if objAnnotations == nil {
					objAnnotations = make(map[string]string)
				}
				for k, v := range existingAnnotations {
					if _, ok := objAnnotations[k]; !ok {
						objAnnotations[k] = v
					}
				}
				obj.SetAnnotations(objAnnotations)
			}
			_, err = k.client.Resource(knativeServiceGVR).Namespace(ns).Update(ctx, obj, metav1.UpdateOptions{})
			if err != nil {
				return nil, fmt.Errorf("updating knative service: %w", err)
			}
			k.logger.Info("knative: service updated", "name", svc.Name, "namespace", ns)
		} else {
			return nil, fmt.Errorf("creating knative service: %w", err)
		}
	} else {
		k.logger.Info("knative: service created", "name", svc.Name, "namespace", ns)
	}

	return &DeployResult{Status: models.ServiceStatusPending}, nil
}

func (k *KnativeOrchestrator) Remove(ctx context.Context, svc models.Service) error {
	ns := k.namespaceForService(svc)
	err := k.client.Resource(knativeServiceGVR).Namespace(ns).Delete(ctx, svc.Name, metav1.DeleteOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting knative service: %w", err)
	}
	k.logger.Info("knative: service removed", "name", svc.Name, "namespace", ns)
	return nil
}

func (k *KnativeOrchestrator) Status(ctx context.Context, svc models.Service) (*DeployResult, error) {
	ns := k.namespaceForService(svc)
	obj, err := k.client.Resource(knativeServiceGVR).Namespace(ns).Get(ctx, svc.Name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting knative service: %w", err)
	}

	return k.parseStatus(obj), nil
}

func (k *KnativeOrchestrator) Logs(ctx context.Context, svc models.Service, opts LogsOptions) (io.ReadCloser, error) {
	ns := k.namespaceForService(svc)
	pods, err := k.clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("serving.knative.dev/service=%s", svc.Name),
	})
	if err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}

	var podName string
	var containerName string
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			podName = pod.Name
			if len(pod.Spec.Containers) > 0 {
				containerName = pod.Spec.Containers[0].Name
			}
			break
		}
	}
	if podName == "" {
		return nil, ErrNoPods
	}

	logOpts := &corev1.PodLogOptions{
		Follow:    opts.Follow,
		Container: containerName,
	}
	if opts.Tail > 0 {
		logOpts.TailLines = &opts.Tail
	}

	return k.clientset.CoreV1().Pods(ns).GetLogs(podName, logOpts).Stream(ctx)
}

func (k *KnativeOrchestrator) buildKnativeService(svc models.Service, ns string) *unstructured.Unstructured {
	container := map[string]interface{}{
		"image": svc.Image,
		"env":   buildEnvVars(svc.EnvVars),
	}

	if svc.Port > 0 {
		container["ports"] = []interface{}{
			map[string]interface{}{
				"containerPort": svc.Port,
			},
		}
	}

	if len(svc.Command) > 0 {
		container["command"] = svc.Command
	}

	if len(svc.Args) > 0 {
		container["args"] = svc.Args
	}

	containers := []interface{}{container}

	minScale := svc.MinScale
	maxScale := svc.MaxScale
	if maxScale == 0 {
		maxScale = 10
	}

	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "serving.knative.dev/v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      svc.Name,
				"namespace": ns,
				"labels": map[string]interface{}{
					"app.kubernetes.io/managed-by": "max-cloud",
				},
			},
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"annotations": map[string]interface{}{
							"autoscaling.knative.dev/minScale": fmt.Sprintf("%d", minScale),
							"autoscaling.knative.dev/maxScale": fmt.Sprintf("%d", maxScale),
						},
					},
					"spec": k.buildPodSpec(containers, svc.Image),
				},
			},
		},
	}

	return obj
}

func (k *KnativeOrchestrator) buildPodSpec(containers []interface{}, image string) map[string]interface{} {
	podSpec := map[string]interface{}{
		"containers": containers,
	}

	if k.usesPrivateRegistry(image) {
		podSpec["imagePullSecrets"] = []interface{}{
			map[string]interface{}{
				"name": "registry-pull-secret",
			},
		}
	}

	return podSpec
}

func (k *KnativeOrchestrator) usesPrivateRegistry(image string) bool {
	if k.registryURL == "" {
		return false
	}
	return len(image) > len(k.registryURL) && image[:len(k.registryURL)] == k.registryURL
}

func buildEnvVars(envVars map[string]string) []interface{} {
	if len(envVars) == 0 {
		return nil
	}
	result := make([]interface{}, 0, len(envVars))
	for k, v := range envVars {
		result = append(result, map[string]interface{}{
			"name":  k,
			"value": v,
		})
	}
	return result
}

func (k *KnativeOrchestrator) parseStatus(obj *unstructured.Unstructured) *DeployResult {
	result := &DeployResult{Status: models.ServiceStatusPending}

	// URL aus status.url lesen
	url, found, err := unstructured.NestedString(obj.Object, "status", "url")
	if err == nil && found {
		result.URL = url
	}

	// status.conditions nach type=Ready suchen
	conditions, found, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil || !found {
		return result
	}

	for _, c := range conditions {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		condType, _ := cond["type"].(string)
		if condType != "Ready" {
			continue
		}
		condStatus, _ := cond["status"].(string)
		switch condStatus {
		case "True":
			result.Status = models.ServiceStatusReady
		case "False":
			result.Status = models.ServiceStatusFailed
		default:
			result.Status = models.ServiceStatusPending
		}
		break
	}

	return result
}
