package lxyscore

import (
	"context"
	"fmt"
	"math"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/klog/v2"
)

type LxyScore struct {
	handle framework.Handle
	
}

const Name = "LxyScore"

var _ framework.ScorePlugin = &LxyScore{}

func (ls *LxyScore) Name() string {
	return Name
}

func (ls *LxyScore) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	nodeInfo, err := ls.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("getting node %q from Snapshot: %v", nodeName, err))
	}
	var numOfPod int64
	numOfPod = int64(len(ls.handle.NominatedPodsForNode(nodeInfo.Node().Name)))
	klog.Infof("[lxyscore] node '%s' numOfPod: %d", nodeName, numOfPod)
	return numOfPod, nil
}

func (ls *LxyScore) ScoreExtensions() framework.ScoreExtensions {
	return ls
}

func (ls *LxyScore) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	// Find highest and lowest scores.
	var highest int64 = -math.MaxInt64
	var lowest int64 = math.MaxInt64
	for _, nodeScore := range scores {
		if nodeScore.Score > highest {
			highest = nodeScore.Score
		}
		if nodeScore.Score < lowest {
			lowest = nodeScore.Score
		}
	}

	// Transform the highest to lowest score range to fit the framework's min to max node score range.
	oldRange := highest - lowest
	newRange := framework.MaxNodeScore - framework.MinNodeScore
	for i, nodeScore := range scores {
		if oldRange == 0 {
			scores[i].Score = framework.MinNodeScore
		} else {
			scores[i].Score = ((nodeScore.Score - lowest) * newRange / oldRange) + framework.MinNodeScore
		}
	}
	klog.Infof("[lxyscore] Nodes final score: %v", scores)
	return nil
}

func New(_ runtime.Object, h framework.Handle) (framework.Plugin, error) {
	return &LxyScore{handle: h}, nil
}