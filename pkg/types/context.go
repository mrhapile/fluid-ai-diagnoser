package types

type DiagnosticContext struct {
	Summary  Summary           `json:"summary"`
	Graph    ResourceGraph     `json:"graph"`
	Findings []FailureHint     `json:"findings"`
	Events   []Event           `json:"events"`
	Logs     map[string]string `json:"logs"`
	Metadata Metadata          `json:"metadata"`
}

type Summary struct {
	ClusterVersion string `json:"clusterVersion"`
	Namespace      string `json:"namespace"`
}

type ResourceGraph struct {
	Nodes    map[string]NodeInfo    `json:"nodes,omitempty"`
	Pods     map[string]PodInfo     `json:"pods,omitempty"`
	PVCs     map[string]PVCInfo     `json:"pvcs,omitempty"`
	Datasets map[string]DatasetInfo `json:"datasets,omitempty"`
	Runtimes map[string]RuntimeInfo `json:"runtimes,omitempty"`
}

type NodeInfo struct {
	Name          string            `json:"name"`
	Taints        []Taint           `json:"taints,omitempty"`
	Allocatable   map[string]string `json:"allocatable,omitempty"`
	Capacity      map[string]string `json:"capacity,omitempty"`
	Unschedulable bool              `json:"unschedulable,omitempty"`
}

type Taint struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Effect string `json:"effect"`
}

type PodInfo struct {
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	Status          string            `json:"status"` // Pending, Running, Failed, etc.
	Conditions      []Condition       `json:"conditions,omitempty"`
	Events          []Event           `json:"events,omitempty"`
	OwnerReferences []OwnerReference  `json:"ownerReferences,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
}

type Condition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type OwnerReference struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type PVCInfo struct {
	Name       string      `json:"name"`
	Namespace  string      `json:"namespace"`
	Status     string      `json:"status"` // Bound, Pending, Lost
	VolumeName string      `json:"volumeName,omitempty"`
	Conditions []Condition `json:"conditions,omitempty"`
}

type DatasetInfo struct {
	Name       string      `json:"name"`
	Namespace  string      `json:"namespace"`
	Status     string      `json:"status"` // Bound, NotBound
	Conditions []Condition `json:"conditions,omitempty"`
}

type RuntimeInfo struct {
	Name            string      `json:"name"`
	Namespace       string      `json:"namespace"`
	Type            string      `json:"type"` // Alluxio, JuiceFS, etc.
	MasterReplicas  int32       `json:"masterReplicas"`
	WorkerReplicas  int32       `json:"workerReplicas"`
	MasterReady     int32       `json:"masterReady"`
	WorkerReady     int32       `json:"workerReady"`
	Phase           string      `json:"phase"`
	Conditions      []Condition `json:"conditions,omitempty"`
	FusePhase       string      `json:"fusePhase,omitempty"`
	FuseReady       int32       `json:"fuseReady,omitempty"`
	FuseUnavailable int32       `json:"fuseUnavailable,omitempty"`
}

type FailureHint struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	Source  string `json:"source"` // component name
}

type Event struct {
	Reason        string `json:"reason"`
	Message       string `json:"message"`
	Type          string `json:"type"` // Normal, Warning
	Count         int32  `json:"count"`
	LastTimestamp string `json:"lastTimestamp"`
	InvolvedObject ObjectReference `json:"involvedObject"`
}

type ObjectReference struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type Metadata struct {
	CreationTimestamp string `json:"creationTimestamp"`
	CollectorVersion  string `json:"collectorVersion"`
}
