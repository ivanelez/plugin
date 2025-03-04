package plugins

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type MyK3SPlugin struct {
	handle framework.Handle
}

type NodeNameDistance struct {
	Name     string
	Distance int
}

const (
	// Name : name of plugin used in the plugin registry and configurations.
	Name = "MyK3SPlugin"
)

var _ = framework.FilterPlugin(&MyK3SPlugin{})
var _ = framework.ScorePlugin(&MyK3SPlugin{})

func (p *MyK3SPlugin) Name() string {
	return Name
}

func (p *MyK3SPlugin) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	// Filter nodes that can run the pod. If node cannot run the pod
	// method returns framework status Unschedulable.
	// Additionaly, method filters nodes that already have pod running
	// with the same application name.
	//
	fmt.Println("filter pod:", pod.Name, ", application name: ", pod.Labels["applicationName"], ", Node: ", nodeInfo.Node().Name)

	// Total node resources
	totalNodeCPU := nodeInfo.Node().Status.Capacity[v1.ResourceCPU]
	totalNodeMemory := nodeInfo.Node().Status.Capacity[v1.ResourceMemory]

	// Resources consumed by pods
	requestedCPU := resource.Quantity{}
	requestedMemory := resource.Quantity{}

	for _, p := range nodeInfo.Pods {
		requests := p.Pod.Spec.Containers[0].Resources.Requests
		requestedCPU.Add(requests[v1.ResourceCPU])
		requestedMemory.Add(requests[v1.ResourceMemory])
	}

	// Available resources
	availableCPU := totalNodeCPU.DeepCopy()
	availableCPU.Sub(requestedCPU)

	availableMemory := totalNodeMemory.DeepCopy()
	availableMemory.Sub(requestedMemory)

	//calculate allocated CPU for running all containers in current pod
	var podCPU resource.Quantity
	for _, container := range pod.Spec.Containers {
		if cpu, ok := container.Resources.Requests[v1.ResourceCPU]; ok {
			podCPU.Add(cpu)
		}
	}

	//If required resources for running pod are less than available resources on a node
	if availableCPU.Cmp(podCPU) > 0 {

		// Check if node already runs application with the same name
		pods := nodeInfo.Pods
		for _, p := range pods {
			labels := p.Pod.GetLabels()
			//if any pod running on current node already has pod with the label applicationName equal to pod given as parameter, then this app
			// already exists on this node.
			if applicationName, found := labels["applicationName"]; found && applicationName == pod.Labels["applicationName"] {

				fmt.Println("Application:", pod.Labels["applicationName"], "Already exists on node: ", nodeInfo.Node().Name)

				return framework.NewStatus(framework.Unschedulable, "Application already exists on this node")
			}
		}
		return framework.NewStatus(framework.Success)
	} else {
		return framework.NewStatus(framework.Unschedulable, "Not enough resources to run application: ", pod.Labels["applicationName"])
	}
}

func (p *MyK3SPlugin) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	rand.Seed(time.Now().UnixNano())
	score := rand.Int63n(101) // Random score between 0 and 100
	fmt.Println("Scoring node:", nodeName, "Score:", score)
	return score, nil
}

func (p *MyK3SPlugin) ScoreExtensions() framework.ScoreExtensions {
	return p
}

func (p *MyK3SPlugin) NormalizeScore(_ context.Context, _ *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	fmt.Println("----------NORMALIZE  SCORE -------------------------------------")
	var (
		highest int64 = 0
		lowest        = scores[0].Score
	)

	for _, nodeScore := range scores {
		if nodeScore.Score < lowest {
			lowest = nodeScore.Score
		}
		if nodeScore.Score > highest {
			highest = nodeScore.Score
		}
	}

	if highest == lowest {
		lowest--
	}

	// Set Range to [0-100]
	for i, nodeScore := range scores {
		scores[i].Score = (nodeScore.Score - lowest) * framework.MaxNodeScore / (highest - lowest)
		fmt.Println(scores[i].Name, scores[i].Score, pod.GetNamespace(), pod.GetName())
	}

	fmt.Println("----------NORMALIZE  SCORE END-------------------------------------")
	return nil
}

func New(obj runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	return &MyK3SPlugin{handle: handle}, nil
}
