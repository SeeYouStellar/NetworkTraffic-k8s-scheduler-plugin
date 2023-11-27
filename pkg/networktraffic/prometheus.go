package networktraffic

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"k8s.io/klog/v2"
)

const (
	// nodeMeasureQueryTemplate is the template string to get the query for the node used bandwidth
	// nodeMeasureQueryTemplate = "sum_over_time(node_network_receive_bytes_total{instance=\"%s\"}[%s])"
	nodeMeasureQueryTemplate = "sum_over_time(node_network_receive_bytes_total{instance=\"%s\", device=\"%s\"}[%s])+sum_over_time(node_network_transmit_bytes_total{instance=\"%s\", device=\"%s\"}[%s])" 
)

// Handles the interaction of the networkplugin with Prometheus
type PrometheusHandle struct {
	networkInterface string
	timeRange        time.Duration
	address          string
	api              v1.API
}

func NewPrometheus(address, networkInterface string, timeRange time.Duration) *PrometheusHandle {
	client, err := api.NewClient(api.Config{
		Address: address,
	})
	if err != nil {
		klog.Fatalf("[NetworkTraffic] Error creating prometheus client: %s", err.Error())
	}

	return &PrometheusHandle{
		networkInterface: networkInterface,
		timeRange:        timeRange,
		address:          address,
		api:              v1.NewAPI(client),
	}
}

func (p *PrometheusHandle) GetNodeBandwidthMeasure(node string, networkInterface string) (*model.Sample, error) {
	fmt.Printf("[NetworkTraffic] GetNodeBandwidthMeasure: %s\n", node)
	var nodeip string
	if node == "k8s-node1" {
		nodeip = "10.10.10.171"
	} else if node == "k8s-node2" {
		nodeip = "10.10.10.177"
	} else {
		nodeip = "10.10.10.172"
	}
	nodeip += ":9100"
	query := getNodeBandwidthQuery(nodeip, networkInterface, p.timeRange)
	res, err := p.query(query)
	if err != nil {
		return nil, fmt.Errorf("[NetworkTraffic] Error querying prometheus: %w", err)
	}

	nodeMeasure := res.(model.Vector)
	if len(nodeMeasure) != 1 {
		return nil, fmt.Errorf("[NetworkTraffic] Invalid response, expected 1 value, got %d", len(nodeMeasure))
	}

	return nodeMeasure[0], nil
}

func getNodeBandwidthQuery(nodeip string, networkInterface string, timeRange time.Duration) string {
	return fmt.Sprintf(nodeMeasureQueryTemplate, nodeip, networkInterface, timeRange, nodeip, networkInterface, timeRange)
}

func (p *PrometheusHandle) query(query string) (model.Value, error) {
	results, warnings, err := p.api.Query(context.Background(), query, time.Now())

	if len(warnings) > 0 {
		klog.Warningf("[NetworkTraffic] Warnings: %v\n", warnings)
	}

	return results, err
}