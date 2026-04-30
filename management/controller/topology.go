package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"wireflow/api/v1alpha1"
	"wireflow/internal/infra"
	"wireflow/management/models"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (m *monitorController) GetWorkspaceTopology(ctx context.Context, wsID string) (*models.TopologyResponse, error) {
	if m.client == nil {
		return nil, fmt.Errorf("k8s client is not available")
	}

	namespace, err := m.resolveNamespace(ctx, wsID)
	if err != nil {
		return nil, err
	}

	var peerList v1alpha1.WireflowPeerList
	if err := m.client.GetAPIReader().List(ctx, &peerList, ctrlclient.InNamespace(namespace)); err != nil {
		return nil, err
	}

	resp := &models.TopologyResponse{
		Nodes: make([]models.TopoNode, 0, len(peerList.Items)),
		Links: make([]models.TopoLink, 0),
	}

	positions := buildNodePositions(len(peerList.Items))
	nodesByName := make(map[string]models.TopoNode, len(peerList.Items))
	nodesByAppID := make(map[string]models.TopoNode, len(peerList.Items))

	for idx, peer := range peerList.Items {
		node := models.TopoNode{
			ID:     peer.Name,
			Name:   peer.Spec.AppId,
			IP:     derefString(peer.Status.AllocatedAddress),
			X:      int(positions[idx][0]),
			Y:      int(positions[idx][1]),
			Status: toTopologyNodeStatus(peer),
			Type:   toTopologyNodeType(peer),
		}
		if node.Name == "" {
			node.Name = peer.Name
		}

		resp.Nodes = append(resp.Nodes, node)
		nodesByName[peer.Name] = node
		if peer.Spec.AppId != "" {
			nodesByAppID[peer.Spec.AppId] = node
		}
	}

	seenLinks := make(map[string]struct{})
	for _, peer := range peerList.Items {
		computedPeers, err := m.loadComputedPeers(ctx, namespace, peer.Name)
		if err != nil {
			m.log.Warn("load computed peers failed", "peer", peer.Name, "err", err)
		}

		for _, computedPeer := range computedPeers {
			fromNode, ok := nodesByName[peer.Name]
			if !ok {
				continue
			}

			toNode, ok := nodesByName[computedPeer.Name]
			if !ok && computedPeer.AppID != "" {
				toNode, ok = nodesByAppID[computedPeer.AppID]
			}
			if !ok {
				continue
			}

			linkKey := canonicalLinkKey(fromNode.ID, toNode.ID)
			if _, exists := seenLinks[linkKey]; exists {
				continue
			}
			seenLinks[linkKey] = struct{}{}

			resp.Links = append(resp.Links, models.TopoLink{
				ID:      fmt.Sprintf("%s__%s", fromNode.ID, toNode.ID),
				From:    fromNode.ID,
				To:      toNode.ID,
				Quality: toTopologyLinkQuality(fromNode, toNode),
				Latency: syntheticLatency(fromNode, toNode),
			})
		}
	}

	if len(resp.Links) == 0 {
		resp.Links = fallbackLinks(resp.Nodes)
	}

	return resp, nil
}

func (m *monitorController) loadComputedPeers(ctx context.Context, namespace, peerName string) ([]*infra.Peer, error) {
	var configMap corev1.ConfigMap
	if err := m.client.GetAPIReader().Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      fmt.Sprintf("%s-config", peerName),
	}, &configMap); err != nil {
		return nil, err
	}

	rawConfig := configMap.Data["config.json"]
	if rawConfig == "" {
		return nil, nil
	}

	var message infra.Message
	if err := json.Unmarshal([]byte(rawConfig), &message); err != nil {
		return nil, err
	}

	return message.ComputedPeers, nil
}

func buildNodePositions(count int) [][2]float64 {
	if count <= 0 {
		return nil
	}
	if count == 1 {
		return [][2]float64{{420, 260}}
	}

	positions := make([][2]float64, 0, count)
	centerX, centerY := 420.0, 260.0
	columns := 3
	if count <= 4 {
		columns = 2
	}

	gapX, gapY := 250.0, 170.0
	startX := centerX - gapX*float64(columns-1)/2
	rows := (count + columns - 1) / columns
	startY := centerY - gapY*float64(rows-1)/2

	for i := 0; i < count; i++ {
		col := i % columns
		row := i / columns
		positions = append(positions, [2]float64{
			startX + float64(col)*gapX,
			startY + float64(row)*gapY,
		})
	}

	return positions
}

func toTopologyNodeStatus(peer v1alpha1.WireflowPeer) string {
	if peer.Status.Phase == v1alpha1.NodePhaseReady && peer.Status.AllocatedAddress != nil {
		return "online"
	}
	return "offline"
}

func toTopologyNodeType(peer v1alpha1.WireflowPeer) string {
	if peer.Spec.Network != nil && strings.Contains(strings.ToLower(*peer.Spec.Network), "relay") {
		return "relay"
	}

	switch strings.ToLower(peer.Spec.Platform) {
	case "ios", "android", "darwin", "windows":
		return "client"
	default:
		return "edge"
	}
}

func toTopologyLinkQuality(fromNode, toNode models.TopoNode) string {
	if fromNode.Status == "online" && toNode.Status == "online" {
		return "good"
	}
	if fromNode.Status == "online" || toNode.Status == "online" {
		return "warn"
	}
	return "error"
}

func syntheticLatency(fromNode, toNode models.TopoNode) int {
	switch toTopologyLinkQuality(fromNode, toNode) {
	case "good":
		return 20
	case "warn":
		return 80
	default:
		return 0
	}
}

func fallbackLinks(nodes []models.TopoNode) []models.TopoLink {
	if len(nodes) < 2 {
		return nil
	}

	links := make([]models.TopoLink, 0, len(nodes)-1)
	for i := 1; i < len(nodes); i++ {
		links = append(links, models.TopoLink{
			ID:      fmt.Sprintf("%s__%s", nodes[0].ID, nodes[i].ID),
			From:    nodes[0].ID,
			To:      nodes[i].ID,
			Quality: toTopologyLinkQuality(nodes[0], nodes[i]),
			Latency: syntheticLatency(nodes[0], nodes[i]),
		})
	}
	return links
}

func canonicalLinkKey(a, b string) string {
	if a < b {
		return a + "::" + b
	}
	return b + "::" + a
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
