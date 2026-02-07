package engine

import (
	"testing"

	"github.com/mrhapile/fluid-ai-diagnoser/pkg/types"
)

func TestAnalyze_EmptyContext(t *testing.T) {
	ctx := types.DiagnosticContext{}
	result, err := Analyze(ctx)

	if err != nil {
		t.Errorf("Analyze returned error: %v", err)
	}

	if result.Engine != "rule-based" {
		t.Errorf("Expected engine 'rule-based', got '%s'", result.Engine)
	}

	if len(result.Hypotheses) != 0 {
		t.Errorf("Expected 0 hypotheses for empty context, got %d", len(result.Hypotheses))
	}
}

func TestAnalyze_FuseUnschedulable(t *testing.T) {
	ctx := types.DiagnosticContext{
		Graph: types.ResourceGraph{
			Pods: map[string]types.PodInfo{
				"mydata-fuse-abc123": {
					Name:      "mydata-fuse-abc123",
					Namespace: "default",
					Status:    "Pending",
					Labels:    map[string]string{"role": "fuse"},
					Conditions: []types.Condition{
						{
							Type:    "PodScheduled",
							Status:  "False",
							Reason:  "Unschedulable",
							Message: "node taints",
						},
					},
				},
			},
		},
	}

	result, err := Analyze(ctx)

	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if len(result.Hypotheses) == 0 {
		t.Fatal("Expected at least one hypothesis")
	}

	found := false
	for _, h := range result.Hypotheses {
		if h.Component == "Fuse" {
			found = true
			if h.Confidence < 0.5 {
				t.Errorf("Expected confidence >= 0.5, got %f", h.Confidence)
			}
			if len(h.Evidence) == 0 {
				t.Error("Expected evidence to be populated")
			}
		}
	}

	if !found {
		t.Error("Expected to find Fuse hypothesis")
	}
}

func TestAnalyze_WorkerPendingMemory(t *testing.T) {
	ctx := types.DiagnosticContext{
		Graph: types.ResourceGraph{
			Pods: map[string]types.PodInfo{
				"mydata-worker-0": {
					Name:      "mydata-worker-0",
					Namespace: "default",
					Status:    "Pending",
					Labels:    map[string]string{"role": "worker"},
					Conditions: []types.Condition{
						{
							Type:    "PodScheduled",
							Status:  "False",
							Reason:  "Unschedulable",
							Message: "Insufficient memory",
						},
					},
				},
			},
		},
	}

	result, err := Analyze(ctx)

	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	found := false
	for _, h := range result.Hypotheses {
		if h.Component == "Worker" {
			found = true
		}
	}

	if !found {
		t.Error("Expected to find Worker hypothesis")
	}
}

func TestAnalyze_RuntimePartiallyReady(t *testing.T) {
	ctx := types.DiagnosticContext{
		Graph: types.ResourceGraph{
			Runtimes: map[string]types.RuntimeInfo{
				"mydata": {
					Name:           "mydata",
					Namespace:      "default",
					MasterReplicas: 1,
					WorkerReplicas: 2,
					MasterReady:    1,
					WorkerReady:    0,
				},
			},
		},
	}

	result, err := Analyze(ctx)

	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	found := false
	for _, h := range result.Hypotheses {
		if h.Component == "Runtime" {
			found = true
		}
	}

	if !found {
		t.Error("Expected to find Runtime hypothesis")
	}
}

func TestAnalyze_PVCUnbound(t *testing.T) {
	ctx := types.DiagnosticContext{
		Graph: types.ResourceGraph{
			PVCs: map[string]types.PVCInfo{
				"mydata": {
					Name:      "mydata",
					Namespace: "default",
					Status:    "Pending",
				},
			},
		},
	}

	result, err := Analyze(ctx)

	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	found := false
	for _, h := range result.Hypotheses {
		if h.Component == "Storage" {
			found = true
		}
	}

	if !found {
		t.Error("Expected to find Storage hypothesis")
	}
}

func TestAnalyze_DatasetNotBound(t *testing.T) {
	ctx := types.DiagnosticContext{
		Graph: types.ResourceGraph{
			Datasets: map[string]types.DatasetInfo{
				"mydata": {
					Name:      "mydata",
					Namespace: "default",
					Status:    "NotBound",
				},
			},
		},
	}

	result, err := Analyze(ctx)

	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	found := false
	for _, h := range result.Hypotheses {
		if h.Component == "Dataset" {
			found = true
		}
	}

	if !found {
		t.Error("Expected to find Dataset hypothesis")
	}
}

func TestAnalyze_Deterministic(t *testing.T) {
	ctx := types.DiagnosticContext{
		Graph: types.ResourceGraph{
			Pods: map[string]types.PodInfo{
				"mydata-fuse-abc123": {
					Name:      "mydata-fuse-abc123",
					Namespace: "default",
					Status:    "Pending",
					Labels:    map[string]string{"role": "fuse"},
					Conditions: []types.Condition{
						{Type: "PodScheduled", Status: "False", Reason: "Unschedulable"},
					},
				},
			},
			Runtimes: map[string]types.RuntimeInfo{
				"mydata": {
					Name:           "mydata",
					Namespace:      "default",
					MasterReplicas: 1,
					WorkerReplicas: 2,
					MasterReady:    1,
					WorkerReady:    0,
				},
			},
		},
	}

	// Run analysis multiple times
	result1, _ := Analyze(ctx)
	result2, _ := Analyze(ctx)

	// Verify deterministic ordering
	if len(result1.Hypotheses) != len(result2.Hypotheses) {
		t.Fatal("Non-deterministic: different number of hypotheses")
	}

	for i := range result1.Hypotheses {
		if result1.Hypotheses[i].Rank != result2.Hypotheses[i].Rank {
			t.Errorf("Non-deterministic: different ranks at index %d", i)
		}
		if result1.Hypotheses[i].Component != result2.Hypotheses[i].Component {
			t.Errorf("Non-deterministic: different components at index %d", i)
		}
		if result1.Hypotheses[i].Issue != result2.Hypotheses[i].Issue {
			t.Errorf("Non-deterministic: different issues at index %d", i)
		}
	}
}

func TestAnalyze_RankingByConfidence(t *testing.T) {
	ctx := types.DiagnosticContext{
		Graph: types.ResourceGraph{
			Pods: map[string]types.PodInfo{
				"mydata-fuse-abc123": {
					Name:      "mydata-fuse-abc123",
					Namespace: "default",
					Status:    "Pending",
					Labels:    map[string]string{"role": "fuse"},
					Conditions: []types.Condition{
						{Type: "PodScheduled", Status: "False", Reason: "Unschedulable", Message: "taints"},
					},
				},
			},
			Runtimes: map[string]types.RuntimeInfo{
				"mydata": {
					Name:           "mydata",
					Namespace:      "default",
					MasterReplicas: 1,
					WorkerReplicas: 2,
					MasterReady:    1,
					WorkerReady:    0,
				},
			},
		},
	}

	result, _ := Analyze(ctx)

	// Verify ranking is sequential
	for i, h := range result.Hypotheses {
		if h.Rank != i+1 {
			t.Errorf("Expected rank %d at index %d, got %d", i+1, i, h.Rank)
		}
	}

	// Verify sorted by confidence (descending)
	for i := 1; i < len(result.Hypotheses); i++ {
		if result.Hypotheses[i].Confidence > result.Hypotheses[i-1].Confidence {
			t.Errorf("Hypotheses not sorted by confidence: %f > %f",
				result.Hypotheses[i].Confidence, result.Hypotheses[i-1].Confidence)
		}
	}
}
