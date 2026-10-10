package executor

import "testing"

func TestBuildContainerResources(t *testing.T) {
	if buildContainerResources(ContainerResources{}) != nil {
		t.Fatal("expected nil for empty resources")
	}
	got := buildContainerResources(ContainerResources{
		Requests: map[string]string{"cpu": "500m", "memory": "512Mi"},
		Limits:   map[string]string{"cpu": "2", "memory": "2Gi", "nvidia.com/gpu": "1"},
	})
	if got == nil {
		t.Fatal("expected resources block")
	}
	req, ok := got["requests"].(map[string]string)
	if !ok || req["cpu"] != "500m" {
		t.Fatalf("requests: %#v", got["requests"])
	}
	lim, ok := got["limits"].(map[string]string)
	if !ok || lim["nvidia.com/gpu"] != "1" {
		t.Fatalf("limits: %#v", got["limits"])
	}
}
