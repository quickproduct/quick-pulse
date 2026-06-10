package handlers

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// errK8sNotConfigured is returned when neither a kubeconfig file nor an
// in-cluster service account is available — i.e. the operator simply hasn't
// connected a cluster yet. Distinct from a kubeconfig that exists but fails
// to load, so the UI can show a friendlier reason.
var errK8sNotConfigured = errors.New("no kubeconfig found and not running in a cluster")

// runningInContainer returns true when the process is running inside a Docker
// container. Used to decide whether to rewrite loopback URLs in kubeconfig
// (which point at the host, not the container).
func runningInContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	return false
}

// rewriteLoopbackHost replaces 0.0.0.0 / 127.0.0.1 / localhost in the
// kubeconfig server URL with host.docker.internal so the container can reach
// the API server published on the host (k3d, kind, OrbStack, Docker Desktop
// all bind to loopback on the host). No-op when running outside a container
// or when the host is already non-loopback.
func rewriteLoopbackHost(config *rest.Config) {
	if !runningInContainer() || config == nil || config.Host == "" {
		return
	}
	for _, loopback := range []string{"//0.0.0.0:", "//127.0.0.1:", "//localhost:"} {
		if strings.Contains(config.Host, loopback) {
			replacement := strings.Replace(loopback, loopback[2:len(loopback)-1], "host.docker.internal", 1)
			config.Host = strings.Replace(config.Host, loopback, replacement, 1)
			// Loopback certs are issued for the loopback hostname; once we
			// connect via host.docker.internal the SAN won't match. Most
			// local clusters (k3d/kind/OrbStack) use self-signed certs the
			// user already trusts implicitly, so disable TLS verification
			// for the rewritten host — same trust model as K8S_SERVER_OVERRIDE.
			config.TLSClientConfig.Insecure = true
			config.TLSClientConfig.CAData = nil
			config.TLSClientConfig.CAFile = ""
			return
		}
	}
}

// disconnectedReason renders a short human-readable explanation for why the
// dashboard could not reach a cluster. Surfaced to the UI verbatim.
func disconnectedReason(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, errK8sNotConfigured) {
		return "No kubeconfig found and not running in a cluster."
	}
	return err.Error()
}

// kubeconfigPath returns the path the backend uses for kubeconfig
// (KUBECONFIG env var, falling back to ~/.kube/config). Returns "" if neither
// is set/available.
func kubeconfigPath() string {
	if p := os.Getenv("KUBECONFIG"); p != "" {
		return p
	}
	if homeDir, _ := os.UserHomeDir(); homeDir != "" {
		return filepath.Join(homeDir, ".kube", "config")
	}
	return ""
}

// getK8sClient builds a clientset for the kubeconfig's current-context
// (or in-cluster config when running inside a pod).
func getK8sClient() (*kubernetes.Clientset, error) {
	return getK8sClientForContext("")
}

// getK8sClientForContext builds a clientset for the named kubeconfig context.
// An empty contextName means "current-context" (back-compat behaviour).
// In-cluster auth is used if no kubeconfig file is present.
func getK8sClientForContext(contextName string) (*kubernetes.Clientset, error) {
	var (
		config *rest.Config
		err    error
	)

	kubeconfig := kubeconfigPath()
	kubeconfigExists := false
	if kubeconfig != "" {
		if _, statErr := os.Stat(kubeconfig); statErr == nil {
			kubeconfigExists = true
			loader := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
			overrides := &clientcmd.ConfigOverrides{CurrentContext: contextName}
			config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides).ClientConfig()
			if err != nil {
				return nil, fmt.Errorf("failed to load kubeconfig %s (context=%q): %w", kubeconfig, contextName, err)
			}
		}
	}

	if config == nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			if errors.Is(err, rest.ErrNotInCluster) && !kubeconfigExists {
				return nil, errK8sNotConfigured
			}
			return nil, fmt.Errorf("in-cluster config failed: %w", err)
		}
	}

	if override := os.Getenv("K8S_SERVER_OVERRIDE"); override != "" && contextName == "" {
		// Override only applies to the default/current-context client.
		// When the UI explicitly asks for a named context, respect kubeconfig's URL.
		config.Host = override
		config.TLSClientConfig.Insecure = true
		config.TLSClientConfig.CAData = nil
		config.TLSClientConfig.CAFile = ""
	} else {
		rewriteLoopbackHost(config)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubernetes client: %w", err)
	}
	return clientset, nil
}

// contextFromRequest returns the cluster context name from the ?context=
// query param, or "" for current-context.
func contextFromRequest(r *http.Request) string {
	return r.URL.Query().Get("context")
}

// k8sCallTimeout bounds a single (non-streaming) Kubernetes API call so a
// hung API server cannot wedge a handler. Streaming endpoints (log follow)
// manage their own lifetimes and must not use this.
const k8sCallTimeout = 15 * time.Second

func k8sCtx(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), k8sCallTimeout)
}

// KubeContextInfo describes a single kubeconfig context for the UI dropdown.
type KubeContextInfo struct {
	Name    string `json:"name"`
	Cluster string `json:"cluster"`
	Server  string `json:"server"`
	Current bool   `json:"current"`
}

// K8sContextsHandler handles GET /api/v1/kubernetes/contexts. Lists every
// context defined in the loaded kubeconfig so the UI can render a cluster
// switcher. Returns an empty list (not an error) when no kubeconfig is
// configured — the UI just shows "no clusters configured" in that case.
func K8sContextsHandler(w http.ResponseWriter, r *http.Request) {
	path := kubeconfigPath()
	if path == "" {
		WriteJSON(w, http.StatusOK, []KubeContextInfo{})
		return
	}
	if _, err := os.Stat(path); err != nil {
		WriteJSON(w, http.StatusOK, []KubeContextInfo{})
		return
	}
	apiCfg, err := clientcmd.LoadFromFile(path)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to read kubeconfig: %v", err))
		return
	}
	out := make([]KubeContextInfo, 0, len(apiCfg.Contexts))
	for name, ctx := range apiCfg.Contexts {
		server := ""
		if cluster, ok := apiCfg.Clusters[ctx.Cluster]; ok {
			server = cluster.Server
		}
		out = append(out, KubeContextInfo{
			Name:    name,
			Cluster: ctx.Cluster,
			Server:  server,
			Current: name == apiCfg.CurrentContext,
		})
	}
	// Stable order: current first, then alphabetical.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Current != out[j].Current {
			return out[i].Current
		}
		return out[i].Name < out[j].Name
	})
	WriteJSON(w, http.StatusOK, out)
}

// disconnectedOverview is the honest empty-state payload for /kubernetes/overview
// when no cluster is reachable. Zero counts so the UI renders an empty state
// instead of fabricated pods/nodes.
func disconnectedOverview(reason string) map[string]interface{} {
	return map[string]interface{}{
		"nodes":        0,
		"nodes_ready":  0,
		"pods_total":   0,
		"pods_running": 0,
		"pods_pending": 0,
		"pods_failed":  0,
		"namespaces":   0,
		"source":       "disconnected",
		"connected":    false,
		"reason":       reason,
	}
}

// K8sOverviewHandler handles GET /api/v1/kubernetes/overview
func K8sOverviewHandler(w http.ResponseWriter, r *http.Request) {
	clientset, err := getK8sClientForContext(contextFromRequest(r))
	if err != nil {
		WriteJSON(w, http.StatusOK, disconnectedOverview(disconnectedReason(err)))
		return
	}

	ctx, cancel := k8sCtx(r)
	defer cancel()
	nodesList, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteJSON(w, http.StatusOK, disconnectedOverview(disconnectedReason(err)))
		return
	}

	podsList, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteJSON(w, http.StatusOK, disconnectedOverview(disconnectedReason(err)))
		return
	}

	namespacesList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteJSON(w, http.StatusOK, disconnectedOverview(disconnectedReason(err)))
		return
	}

	readyNodes := 0
	for _, n := range nodesList.Items {
		for _, cond := range n.Status.Conditions {
			if cond.Type == "Ready" && cond.Status == "True" {
				readyNodes++
				break
			}
		}
	}

	runningPods := 0
	pendingPods := 0
	failedPods := 0
	for _, p := range podsList.Items {
		switch p.Status.Phase {
		case corev1.PodRunning:
			runningPods++
		case corev1.PodPending:
			pendingPods++
		case corev1.PodFailed:
			failedPods++
		}
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"nodes":        len(nodesList.Items),
		"nodes_ready":  readyNodes,
		"pods_total":   len(podsList.Items),
		"pods_running": runningPods,
		"pods_pending": pendingPods,
		"pods_failed":  failedPods,
		"namespaces":   len(namespacesList.Items),
		"source":       "live",
		"connected":    true,
		"reason":       "",
	})
}

// K8sNodesHandler handles GET /api/v1/kubernetes/nodes
func K8sNodesHandler(w http.ResponseWriter, r *http.Request) {
	clientset, err := getK8sClientForContext(contextFromRequest(r))
	if err != nil {
		WriteJSON(w, http.StatusOK, []map[string]interface{}{})
		return
	}

	ctx, cancel := k8sCtx(r)
	defer cancel()
	nodeList, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteJSON(w, http.StatusOK, []map[string]interface{}{})
		return
	}

	var resp []map[string]interface{}
	for _, node := range nodeList.Items {
		age := int(time.Since(node.CreationTimestamp.Time).Seconds())
		status := "NotReady"
		conds := []map[string]string{}
		for _, c := range node.Status.Conditions {
			conds = append(conds, map[string]string{
				"type":   string(c.Type),
				"status": string(c.Status),
			})
			if c.Type == "Ready" && c.Status == "True" {
				status = "Ready"
			}
		}

		cpuStr := node.Status.Allocatable.Cpu().String()
		memStr := node.Status.Allocatable.Memory().String()

		role := "worker"
		for label := range node.Labels {
			if strings.HasPrefix(label, "node-role.kubernetes.io/") {
				role = strings.TrimPrefix(label, "node-role.kubernetes.io/")
				break
			}
		}

		resp = append(resp, map[string]interface{}{
			"name":        node.Name,
			"role":        role,
			"status":      status,
			"version":     node.Status.NodeInfo.KubeletVersion,
			"cpu":         cpuStr,
			"memory":      memStr,
			"os":          node.Status.NodeInfo.OSImage,
			"arch":        node.Status.NodeInfo.Architecture,
			"age_seconds": age,
			"conditions":  conds,
		})
	}
	WriteJSON(w, http.StatusOK, resp)
}

// K8sPodsHandler handles GET /api/v1/kubernetes/pods
func K8sPodsHandler(w http.ResponseWriter, r *http.Request) {
	ns := r.URL.Query().Get("namespace")
	clientset, err := getK8sClientForContext(contextFromRequest(r))
	if err != nil {
		WriteJSON(w, http.StatusOK, []map[string]interface{}{})
		return
	}

	ctx, cancel := k8sCtx(r)
	defer cancel()
	podList, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteJSON(w, http.StatusOK, []map[string]interface{}{})
		return
	}

	var resp []map[string]interface{}
	for _, pod := range podList.Items {
		readyContainers := 0
		totalContainers := len(pod.Spec.Containers)
		restarts := 0

		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				readyContainers++
			}
			restarts += int(cs.RestartCount)
		}

		readyStr := fmt.Sprintf("%d/%d", readyContainers, totalContainers)

		var cpuReq, memReq string
		for _, container := range pod.Spec.Containers {
			if reqCpu := container.Resources.Requests.Cpu(); !reqCpu.IsZero() {
				cpuReq = reqCpu.String()
			}
			if reqMem := container.Resources.Requests.Memory(); !reqMem.IsZero() {
				memReq = reqMem.String()
			}
		}
		if cpuReq == "" {
			cpuReq = "0m"
		}
		if memReq == "" {
			memReq = "0Mi"
		}

		image := ""
		if len(pod.Spec.Containers) > 0 {
			image = pod.Spec.Containers[0].Image
		}

		age := int(time.Since(pod.CreationTimestamp.Time).Seconds())

		conds := []map[string]string{}
		for _, c := range pod.Status.Conditions {
			conds = append(conds, map[string]string{
				"type":   string(c.Type),
				"status": string(c.Status),
			})
		}

		resp = append(resp, map[string]interface{}{
			"name":        pod.Name,
			"namespace":   pod.Namespace,
			"status":      string(pod.Status.Phase),
			"ready":       readyStr,
			"restarts":    restarts,
			"age_seconds": age,
			"node":        pod.Spec.NodeName,
			"cpu":         cpuReq,
			"memory":      memReq,
			"image":       image,
			"conditions":  conds,
		})
	}
	WriteJSON(w, http.StatusOK, resp)
}

// K8sDeploymentsHandler handles GET /api/v1/kubernetes/deployments
func K8sDeploymentsHandler(w http.ResponseWriter, r *http.Request) {
	ns := r.URL.Query().Get("namespace")
	clientset, err := getK8sClientForContext(contextFromRequest(r))
	if err != nil {
		WriteJSON(w, http.StatusOK, []map[string]interface{}{})
		return
	}

	ctx, cancel := k8sCtx(r)
	defer cancel()
	deployList, err := clientset.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteJSON(w, http.StatusOK, []map[string]interface{}{})
		return
	}

	var resp []map[string]interface{}
	for _, deploy := range deployList.Items {
		image := ""
		if len(deploy.Spec.Template.Spec.Containers) > 0 {
			image = deploy.Spec.Template.Spec.Containers[0].Image
		}

		desired := 0
		if deploy.Spec.Replicas != nil {
			desired = int(*deploy.Spec.Replicas)
		}

		resp = append(resp, map[string]interface{}{
			"name":        deploy.Name,
			"namespace":   deploy.Namespace,
			"desired":     desired,
			"ready":       int(deploy.Status.ReadyReplicas),
			"available":   int(deploy.Status.AvailableReplicas),
			"updated":     int(deploy.Status.UpdatedReplicas),
			"age_seconds": int(time.Since(deploy.CreationTimestamp.Time).Seconds()),
			"image":       image,
			"strategy":    string(deploy.Spec.Strategy.Type),
		})
	}
	WriteJSON(w, http.StatusOK, resp)
}

// K8sServicesHandler handles GET /api/v1/kubernetes/services
func K8sServicesHandler(w http.ResponseWriter, r *http.Request) {
	ns := r.URL.Query().Get("namespace")
	clientset, err := getK8sClientForContext(contextFromRequest(r))
	if err != nil {
		WriteJSON(w, http.StatusOK, []map[string]interface{}{})
		return
	}

	ctx, cancel := k8sCtx(r)
	defer cancel()
	svcList, err := clientset.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteJSON(w, http.StatusOK, []map[string]interface{}{})
		return
	}

	var resp []map[string]interface{}
	for _, svc := range svcList.Items {
		ports := []map[string]interface{}{}
		for _, p := range svc.Spec.Ports {
			portMap := map[string]interface{}{
				"port":        p.Port,
				"target_port": p.TargetPort.String(),
				"protocol":    string(p.Protocol),
			}
			if p.NodePort > 0 {
				portMap["node_port"] = p.NodePort
			}
			ports = append(ports, portMap)
		}

		var externalIP interface{} = nil
		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			if svc.Status.LoadBalancer.Ingress[0].IP != "" {
				externalIP = svc.Status.LoadBalancer.Ingress[0].IP
			} else if svc.Status.LoadBalancer.Ingress[0].Hostname != "" {
				externalIP = svc.Status.LoadBalancer.Ingress[0].Hostname
			}
		} else if len(svc.Spec.ExternalIPs) > 0 {
			externalIP = svc.Spec.ExternalIPs[0]
		}

		resp = append(resp, map[string]interface{}{
			"name":        svc.Name,
			"namespace":   svc.Namespace,
			"type":        string(svc.Spec.Type),
			"cluster_ip":  svc.Spec.ClusterIP,
			"external_ip": externalIP,
			"ports":       ports,
			"selector":    svc.Spec.Selector,
			"age_seconds": int(time.Since(svc.CreationTimestamp.Time).Seconds()),
		})
	}
	WriteJSON(w, http.StatusOK, resp)
}

// K8sNamespacesHandler handles GET /api/v1/kubernetes/namespaces
func K8sNamespacesHandler(w http.ResponseWriter, r *http.Request) {
	clientset, err := getK8sClientForContext(contextFromRequest(r))
	if err != nil {
		WriteJSON(w, http.StatusOK, []string{})
		return
	}

	ctx, cancel := k8sCtx(r)
	defer cancel()
	nsList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteJSON(w, http.StatusOK, []string{})
		return
	}

	var resp []string
	for _, ns := range nsList.Items {
		resp = append(resp, ns.Name)
	}
	WriteJSON(w, http.StatusOK, resp)
}

// K8sEventsHandler handles GET /api/v1/kubernetes/events.
// Optional query params: namespace, type (Normal|Warning), limit (default 200, max 1000).
func K8sEventsHandler(w http.ResponseWriter, r *http.Request) {
	ns := r.URL.Query().Get("namespace")
	clientset, err := getK8sClientForContext(contextFromRequest(r))
	if err != nil {
		WriteJSON(w, http.StatusOK, []map[string]interface{}{})
		return
	}

	limit := 200
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = min(l, 1000)
	}

	listOpts := metav1.ListOptions{}
	if evType := r.URL.Query().Get("type"); evType == "Normal" || evType == "Warning" {
		listOpts.FieldSelector = "type=" + evType
	}

	ctx, cancel := k8sCtx(r)
	defer cancel()
	eventList, err := clientset.CoreV1().Events(ns).List(ctx, listOpts)
	if err != nil {
		WriteJSON(w, http.StatusOK, []map[string]interface{}{})
		return
	}

	var resp []map[string]interface{}
	for _, ev := range eventList.Items {
		objRef := fmt.Sprintf("%s/%s", ev.InvolvedObject.Kind, ev.InvolvedObject.Name)

		count := int(ev.Count)
		if count == 0 {
			count = 1
		}

		var age int
		if !ev.LastTimestamp.IsZero() {
			age = int(time.Since(ev.LastTimestamp.Time).Seconds())
		} else if !ev.FirstTimestamp.IsZero() {
			age = int(time.Since(ev.FirstTimestamp.Time).Seconds())
		} else if !ev.EventTime.IsZero() {
			age = int(time.Since(ev.EventTime.Time).Seconds())
		} else {
			age = int(time.Since(ev.CreationTimestamp.Time).Seconds())
		}

		resp = append(resp, map[string]interface{}{
			"name":        ev.Name,
			"namespace":   ev.Namespace,
			"type":        ev.Type,
			"reason":      ev.Reason,
			"object":      objRef,
			"message":     ev.Message,
			"count":       count,
			"age_seconds": age,
		})
	}
	sort.Slice(resp, func(i, j int) bool {
		return resp[i]["age_seconds"].(int) < resp[j]["age_seconds"].(int)
	})
	if len(resp) > limit {
		resp = resp[:limit]
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetK8sPodLogsHandler handles GET /api/v1/kubernetes/pods/{namespace}/{pod_name}/logs
func GetK8sPodLogsHandler(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	podName := r.PathValue("pod_name")
	if namespace == "" || podName == "" {
		WriteError(w, http.StatusBadRequest, "Missing namespace or pod name")
		return
	}

	tailStr := r.URL.Query().Get("tail")
	tail := int64(100)
	if tailStr != "" {
		if t, err := strconv.Atoi(tailStr); err == nil {
			tail = int64(t)
		}
	}
	if tail < 1 {
		tail = 1
	}
	if tail > 500 {
		tail = 500
	}

	clientset, err := getK8sClientForContext(contextFromRequest(r))
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, fmt.Sprintf("Kubernetes unavailable: %s", disconnectedReason(err)))
		return
	}

	ctx, cancel := k8sCtx(r)
	defer cancel()
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		WriteError(w, http.StatusNotFound, fmt.Sprintf("Pod %s not found in namespace %s: %v", podName, namespace, err))
		return
	}

	if len(pod.Spec.Containers) == 0 {
		WriteError(w, http.StatusBadRequest, "Pod has no containers")
		return
	}
	containerName := pod.Spec.Containers[0].Name

	logOpts := &corev1.PodLogOptions{
		Container: containerName,
		TailLines: &tail,
		Follow:    false,
	}

	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, logOpts)
	stream, err := req.Stream(ctx)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to stream logs: %v", err))
		return
	}
	defer stream.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, stream)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to read logs stream: %v", err))
		return
	}

	lines := strings.Split(buf.String(), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"logs": lines,
	})
}

// HandleWSK8sLogs handles streaming pod logs via WebSocket /ws/kubernetes/logs/{namespace}/{pod_name}
func HandleWSK8sLogs(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	podName := r.PathValue("pod_name")
	if namespace == "" || podName == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Missing namespace or pod name"))
		return
	}

	_, ok := validateWSAuth(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Unauthorized"))
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade failed for pod logs: %v", err)
		return
	}
	defer conn.Close()

	clientset, err := getK8sClientForContext(contextFromRequest(r))
	if err != nil {
		_ = conn.WriteJSON(map[string]string{"error": "Kubernetes unavailable: " + disconnectedReason(err)})
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		_ = conn.WriteJSON(map[string]string{"error": fmt.Sprintf("Pod not found: %v", err)})
		return
	}

	if len(pod.Spec.Containers) == 0 {
		_ = conn.WriteJSON(map[string]string{"error": "Pod has no containers"})
		return
	}
	containerName := pod.Spec.Containers[0].Name

	tail := int64(100)
	logOpts := &corev1.PodLogOptions{
		Container: containerName,
		TailLines: &tail,
		Follow:    true,
	}

	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, logOpts)
	stream, err := req.Stream(ctx)
	if err != nil {
		_ = conn.WriteJSON(map[string]string{"error": fmt.Sprintf("Failed to stream logs: %v", err)})
		return
	}
	defer stream.Close()

	paused := false
	var mu sync.Mutex

	// Read logs and stream to WS. When the log stream ends (pod gone, kubelet
	// rotation, API error) tell the client and close the conn so the command
	// loop below unblocks instead of leaving a silent, dead "Live" view.
	go func() {
		reader := bufio.NewReader(stream)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if ctx.Err() == nil {
					_ = conn.WriteJSON(map[string]interface{}{"eof": true})
				}
				_ = conn.Close()
				return
			}
			line = strings.TrimSuffix(line, "\n")
			line = strings.TrimSuffix(line, "\r")

			mu.Lock()
			isPaused := paused
			mu.Unlock()

			if !isPaused {
				err = conn.WriteJSON(map[string]interface{}{
					"line": line,
				})
				if err != nil {
					return
				}
			}
		}
	}()

	// Read commands from websocket
	for {
		var msg map[string]string
		err := conn.ReadJSON(&msg)
		if err != nil {
			break
		}

		action := msg["action"]
		mu.Lock()
		if action == "pause" {
			paused = true
		} else if action == "resume" {
			paused = false
		}
		mu.Unlock()
	}
}

var mockK8sMu sync.Mutex

type ScaleDeploymentRequest struct {
	Replicas int `json:"replicas"`
}

// ScaleDeploymentHandler handles POST /api/v1/kubernetes/deployments/{namespace}/{name}/scale
func ScaleDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")
	if namespace == "" || name == "" {
		WriteError(w, http.StatusBadRequest, "Missing namespace or deployment name")
		return
	}

	var req ScaleDeploymentRequest
	if err := ParseJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.Replicas < 0 || req.Replicas > 1000 {
		WriteError(w, http.StatusBadRequest, "Replicas must be between 0 and 1000")
		return
	}

	clientset, err := getK8sClientForContext(contextFromRequest(r))
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, "Kubernetes unavailable: "+disconnectedReason(err))
		return
	}

	ctx, cancel := k8sCtx(r)
	defer cancel()
	scale, err := clientset.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get deployment scale: %v", err))
		return
	}

	scale.Spec.Replicas = int32(req.Replicas)
	_, err = clientset.AppsV1().Deployments(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to scale deployment: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Deployment %s/%s scaled to %d", namespace, name, req.Replicas),
	})
}

// DeletePodHandler handles DELETE /api/v1/kubernetes/pods/{namespace}/{name}
func DeletePodHandler(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")
	if namespace == "" || name == "" {
		WriteError(w, http.StatusBadRequest, "Missing namespace or pod name")
		return
	}

	clientset, err := getK8sClientForContext(contextFromRequest(r))
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, "Kubernetes unavailable: "+disconnectedReason(err))
		return
	}

	// Graceful delete by default; ?force=true skips the grace period the way
	// `kubectl delete --force --grace-period=0` does.
	deleteOpts := metav1.DeleteOptions{}
	if r.URL.Query().Get("force") == "true" {
		gracePeriod := int64(0)
		deleteOpts.GracePeriodSeconds = &gracePeriod
	}
	ctx, cancel := k8sCtx(r)
	defer cancel()
	err = clientset.CoreV1().Pods(namespace).Delete(ctx, name, deleteOpts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete pod: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Pod %s/%s deleted successfully", namespace, name),
	})
}
