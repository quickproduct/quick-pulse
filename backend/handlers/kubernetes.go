package handlers

import (
	"net/http"
)

var namespaces = []string{"default", "kube-system", "monitoring", "production", "staging"}

var nodes = []map[string]interface{}{
	{
		"name": "node-01", "role": "control-plane", "status": "Ready", "version": "v1.29.3",
		"cpu": "4", "memory": "8Gi", "os": "Ubuntu 22.04", "arch": "amd64",
		"age_seconds": 3600000, "conditions": []map[string]string{{"type": "Ready", "status": "True"}},
	},
	{
		"name": "node-02", "role": "worker", "status": "Ready", "version": "v1.29.3",
		"cpu": "8", "memory": "16Gi", "os": "Ubuntu 22.04", "arch": "amd64",
		"age_seconds": 3598000, "conditions": []map[string]string{{"type": "Ready", "status": "True"}},
	},
	{
		"name": "node-03", "role": "worker", "status": "Ready", "version": "v1.29.3",
		"cpu": "8", "memory": "16Gi", "os": "Ubuntu 22.04", "arch": "amd64",
		"age_seconds": 3550000, "conditions": []map[string]string{{"type": "Ready", "status": "True"}},
	},
}

var pods = []map[string]interface{}{
	{
		"name": "quickpulse-backend-7d9f8b-kxp2l", "namespace": "production", "status": "Running",
		"ready": "1/1", "restarts": 0, "age_seconds": 86400, "node": "node-02",
		"cpu": "45m", "memory": "128Mi", "image": "quickpulse/backend:latest",
		"conditions": []map[string]string{{"type": "Ready", "status": "True"}},
	},
	{
		"name": "quickpulse-frontend-5c7d9f-mn4rs", "namespace": "production", "status": "Running",
		"ready": "1/1", "restarts": 0, "age_seconds": 86400, "node": "node-03",
		"cpu": "12m", "memory": "64Mi", "image": "quickpulse/frontend:latest",
		"conditions": []map[string]string{{"type": "Ready", "status": "True"}},
	},
	{
		"name": "quickpulse-db-0", "namespace": "production", "status": "Running",
		"ready": "1/1", "restarts": 0, "age_seconds": 172800, "node": "node-02",
		"cpu": "180m", "memory": "512Mi", "image": "timescale/timescaledb:latest-pg16",
		"conditions": []map[string]string{{"type": "Ready", "status": "True"}},
	},
	{
		"name": "quickpulse-redis-0", "namespace": "production", "status": "Running",
		"ready": "1/1", "restarts": 1, "age_seconds": 172800, "node": "node-03",
		"cpu": "8m", "memory": "32Mi", "image": "redis:7-alpine",
		"conditions": []map[string]string{{"type": "Ready", "status": "True"}},
	},
	{
		"name": "prometheus-0", "namespace": "monitoring", "status": "Running",
		"ready": "1/1", "restarts": 0, "age_seconds": 604800, "node": "node-02",
		"cpu": "220m", "memory": "384Mi", "image": "prom/prometheus:v2.52.0",
		"conditions": []map[string]string{{"type": "Ready", "status": "True"}},
	},
	{
		"name": "grafana-6f8b9d-wr5kp", "namespace": "monitoring", "status": "Running",
		"ready": "1/1", "restarts": 0, "age_seconds": 604800, "node": "node-03",
		"cpu": "35m", "memory": "96Mi", "image": "grafana/grafana:10.4.2",
		"conditions": []map[string]string{{"type": "Ready", "status": "True"}},
	},
	{
		"name": "coredns-7db6d8ff4-nt9zp", "namespace": "kube-system", "status": "Running",
		"ready": "2/2", "restarts": 0, "age_seconds": 3600000, "node": "node-01",
		"cpu": "6m", "memory": "18Mi", "image": "registry.k8s.io/coredns/coredns:v1.11.3",
		"conditions": []map[string]string{{"type": "Ready", "status": "True"}},
	},
	{
		"name": "kube-apiserver-node-01", "namespace": "kube-system", "status": "Running",
		"ready": "1/1", "restarts": 2, "age_seconds": 3600000, "node": "node-01",
		"cpu": "95m", "memory": "256Mi", "image": "registry.k8s.io/kube-apiserver:v1.29.3",
		"conditions": []map[string]string{{"type": "Ready", "status": "True"}},
	},
	{
		"name": "staging-api-deploy-78c9d-qp2kr", "namespace": "staging", "status": "Pending",
		"ready": "0/1", "restarts": 0, "age_seconds": 120, "node": "",
		"cpu": "0m", "memory": "0Mi", "image": "quickpulse/backend:edge",
		"conditions": []map[string]string{{"type": "PodScheduled", "status": "False", "reason": "Unschedulable"}},
	},
	{
		"name": "batch-job-xkcd9-wrkr", "namespace": "default", "status": "Failed",
		"ready": "0/1", "restarts": 5, "age_seconds": 3600, "node": "node-02",
		"cpu": "0m", "memory": "0Mi", "image": "quickpulse/batch:1.2.0",
		"conditions": []map[string]string{{"type": "Ready", "status": "False"}},
	},
}

var deployments = []map[string]interface{}{
	{
		"name": "quickpulse-backend", "namespace": "production",
		"desired": 2, "ready": 2, "available": 2, "updated": 2,
		"age_seconds": 86400, "image": "quickpulse/backend:latest", "strategy": "RollingUpdate",
	},
	{
		"name": "quickpulse-frontend", "namespace": "production",
		"desired": 2, "ready": 2, "available": 2, "updated": 2,
		"age_seconds": 86400, "image": "quickpulse/frontend:latest", "strategy": "RollingUpdate",
	},
	{
		"name": "grafana", "namespace": "monitoring",
		"desired": 1, "ready": 1, "available": 1, "updated": 1,
		"age_seconds": 604800, "image": "grafana/grafana:10.4.2", "strategy": "Recreate",
	},
	{
		"name": "coredns", "namespace": "kube-system",
		"desired": 2, "ready": 2, "available": 2, "updated": 2,
		"age_seconds": 3600000, "image": "registry.k8s.io/coredns/coredns:v1.11.3", "strategy": "RollingUpdate",
	},
	{
		"name": "staging-api-deploy", "namespace": "staging",
		"desired": 3, "ready": 1, "available": 1, "updated": 3,
		"age_seconds": 7200, "image": "quickpulse/backend:edge", "strategy": "RollingUpdate",
	},
}

var services = []map[string]interface{}{
	{
		"name": "quickpulse-backend", "namespace": "production", "type": "ClusterIP",
		"cluster_ip": "10.96.45.12", "external_ip": nil,
		"ports":    []map[string]interface{}{{"port": 8000, "target_port": 8000, "protocol": "TCP"}},
		"selector": map[string]string{"app": "quickpulse-backend"}, "age_seconds": 86400,
	},
	{
		"name": "quickpulse-frontend", "namespace": "production", "type": "LoadBalancer",
		"cluster_ip": "10.96.45.13", "external_ip": "203.0.113.42",
		"ports":    []map[string]interface{}{{"port": 80, "target_port": 80, "protocol": "TCP"}},
		"selector": map[string]string{"app": "quickpulse-frontend"}, "age_seconds": 86400,
	},
	{
		"name": "quickpulse-db", "namespace": "production", "type": "ClusterIP",
		"cluster_ip": "10.96.45.14", "external_ip": nil,
		"ports":    []map[string]interface{}{{"port": 5432, "target_port": 5432, "protocol": "TCP"}},
		"selector": map[string]string{"app": "quickpulse-db"}, "age_seconds": 172800,
	},
	{
		"name": "quickpulse-redis", "namespace": "production", "type": "ClusterIP",
		"cluster_ip": "10.96.45.15", "external_ip": nil,
		"ports":    []map[string]interface{}{{"port": 6379, "target_port": 6379, "protocol": "TCP"}},
		"selector": map[string]string{"app": "quickpulse-redis"}, "age_seconds": 172800,
	},
	{
		"name": "prometheus", "namespace": "monitoring", "type": "NodePort",
		"cluster_ip": "10.96.78.10", "external_ip": nil,
		"ports":    []map[string]interface{}{{"port": 9090, "target_port": 9090, "node_port": 30090, "protocol": "TCP"}},
		"selector": map[string]string{"app": "prometheus"}, "age_seconds": 604800,
	},
	{
		"name": "grafana", "namespace": "monitoring", "type": "NodePort",
		"cluster_ip": "10.96.78.11", "external_ip": nil,
		"ports":    []map[string]interface{}{{"port": 3000, "target_port": 3000, "node_port": 30030, "protocol": "TCP"}},
		"selector": map[string]string{"app": "grafana"}, "age_seconds": 604800,
	},
	{
		"name": "kubernetes", "namespace": "default", "type": "ClusterIP",
		"cluster_ip": "10.96.0.1", "external_ip": nil,
		"ports":    []map[string]interface{}{{"port": 443, "target_port": 6443, "protocol": "TCP"}},
		"selector": map[string]string{}, "age_seconds": 3600000,
	},
}

var events = []map[string]interface{}{
	{
		"name": "quickpulse-backend.17d1f2a9b3", "namespace": "production",
		"type": "Normal", "reason": "Pulled", "object": "Pod/quickpulse-backend-7d9f8b-kxp2l",
		"message": "Successfully pulled image \"quickpulse/backend:latest\" in 2.341s",
		"count":   1, "age_seconds": 86400,
	},
	{
		"name": "quickpulse-backend.17d1f2b3c4", "namespace": "production",
		"type": "Normal", "reason": "Started", "object": "Pod/quickpulse-backend-7d9f8b-kxp2l",
		"message": "Started container quickpulse-backend",
		"count":   1, "age_seconds": 86395,
	},
	{
		"name": "staging-api.17d1f2d8e9", "namespace": "staging",
		"type": "Warning", "reason": "FailedScheduling", "object": "Pod/staging-api-deploy-78c9d-qp2kr",
		"message": "0/3 nodes are available: 3 Insufficient cpu. preemption: 0/3 nodes are available: 3 No preemption victims found for incoming pod.",
		"count":   12, "age_seconds": 120,
	},
	{
		"name": "batch-job.17d1f1a2b3", "namespace": "default",
		"type": "Warning", "reason": "BackOff", "object": "Pod/batch-job-xkcd9-wrkr",
		"message": "Back-off restarting failed container batch-worker in pod batch-job-xkcd9-wrkr_default",
		"count":   5, "age_seconds": 3600,
	},
	{
		"name": "quickpulse-redis.17d1f0c5d6", "namespace": "production",
		"type": "Normal", "reason": "Killing", "object": "Pod/quickpulse-redis-0",
		"message": "Stopping container redis",
		"count":   1, "age_seconds": 172800,
	},
}

// K8sOverviewHandler handles GET /api/v1/kubernetes/overview
func K8sOverviewHandler(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"nodes":        len(nodes),
		"nodes_ready":  3,
		"pods_total":   len(pods),
		"pods_running": 8,
		"pods_pending": 1,
		"pods_failed":  1,
		"namespaces":   len(namespaces),
		"source":       "mock",
	})
}

// K8sNodesHandler handles GET /api/v1/kubernetes/nodes
func K8sNodesHandler(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, nodes)
}

// K8sPodsHandler handles GET /api/v1/kubernetes/pods
func K8sPodsHandler(w http.ResponseWriter, r *http.Request) {
	ns := r.URL.Query().Get("namespace")
	if ns == "" {
		WriteJSON(w, http.StatusOK, pods)
		return
	}
	filtered := []map[string]interface{}{}
	for _, p := range pods {
		if p["namespace"] == ns {
			filtered = append(filtered, p)
		}
	}
	WriteJSON(w, http.StatusOK, filtered)
}

// K8sDeploymentsHandler handles GET /api/v1/kubernetes/deployments
func K8sDeploymentsHandler(w http.ResponseWriter, r *http.Request) {
	ns := r.URL.Query().Get("namespace")
	if ns == "" {
		WriteJSON(w, http.StatusOK, deployments)
		return
	}
	filtered := []map[string]interface{}{}
	for _, d := range deployments {
		if d["namespace"] == ns {
			filtered = append(filtered, d)
		}
	}
	WriteJSON(w, http.StatusOK, filtered)
}

// K8sServicesHandler handles GET /api/v1/kubernetes/services
func K8sServicesHandler(w http.ResponseWriter, r *http.Request) {
	ns := r.URL.Query().Get("namespace")
	if ns == "" {
		WriteJSON(w, http.StatusOK, services)
		return
	}
	filtered := []map[string]interface{}{}
	for _, s := range services {
		if s["namespace"] == ns {
			filtered = append(filtered, s)
		}
	}
	WriteJSON(w, http.StatusOK, filtered)
}

// K8sNamespacesHandler handles GET /api/v1/kubernetes/namespaces
func K8sNamespacesHandler(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, namespaces)
}

// K8sEventsHandler handles GET /api/v1/kubernetes/events
func K8sEventsHandler(w http.ResponseWriter, r *http.Request) {
	ns := r.URL.Query().Get("namespace")
	if ns == "" {
		WriteJSON(w, http.StatusOK, events)
		return
	}
	filtered := []map[string]interface{}{}
	for _, e := range events {
		if e["namespace"] == ns {
			filtered = append(filtered, e)
		}
	}
	WriteJSON(w, http.StatusOK, filtered)
}
