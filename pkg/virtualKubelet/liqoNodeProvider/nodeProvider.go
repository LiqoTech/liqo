package liqonodeprovider

import (
	"context"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// LiqoNodeProvider is a node provider that manages the Liqo resources.
type LiqoNodeProvider struct {
	client    kubernetes.Interface
	dynClient dynamic.Interface

	node              *corev1.Node
	terminating       bool
	lastAppliedLabels map[string]string

	nodeName         string
	foreignClusterID string
	kubeletNamespace string
	resyncPeriod     time.Duration

	networkReady       bool
	podProviderStopper chan struct{}
	networkReadyChan   chan struct{}

	onNodeChangeCallback func(*corev1.Node)
	updateMutex          sync.Mutex
}

// Ping just implements the NodeProvider interface.
// It returns the error from the passed in context only.
func (p *LiqoNodeProvider) Ping(ctx context.Context) error {
	return ctx.Err()
}

// NotifyNodeStatus implements the NodeProvider interface.
func (p *LiqoNodeProvider) NotifyNodeStatus(ctx context.Context, f func(*corev1.Node)) {
	p.onNodeChangeCallback = f
}

// IsTerminating indicates if the node is in terminating (and in the draining phase).
func (p *LiqoNodeProvider) IsTerminating() bool {
	p.updateMutex.Lock()
	defer p.updateMutex.Unlock()
	return p.terminating
}
