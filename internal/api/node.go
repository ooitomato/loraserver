package api

import (
	pb "github.com/brocaar/loraserver/api"
	"github.com/brocaar/loraserver/internal/loraserver"
	"github.com/brocaar/loraserver/internal/storage"
	"github.com/brocaar/loraserver/models"
	"github.com/brocaar/lorawan"
	"golang.org/x/net/context"
)

// NodeAPI exports the Node related functions.
type NodeAPI struct {
	ctx loraserver.Context
}

// NewNodeAPI creates a new NodeAPI.
func NewNodeAPI(ctx loraserver.Context) *NodeAPI {
	return &NodeAPI{
		ctx: ctx,
	}
}

// Create creates the given Node.
func (a *NodeAPI) Create(ctx context.Context, req *pb.CreateNodeRequest) (*pb.CreateNodeResponse, error) {
	var appEUI, devEUI lorawan.EUI64
	var appKey lorawan.AES128Key

	if err := appEUI.UnmarshalText([]byte(req.AppEUI)); err != nil {
		return nil, err
	}
	if err := devEUI.UnmarshalText([]byte(req.DevEUI)); err != nil {
		return nil, err
	}
	if err := appKey.UnmarshalText([]byte(req.AppKey)); err != nil {
		return nil, err
	}

	node := models.Node{
		DevEUI: devEUI,
		AppEUI: appEUI,
		AppKey: appKey,

		RXDelay:     uint8(req.RxDelay),
		RX1DROffset: uint8(req.Rx1DROffset),
	}
	if req.ChannelListID > 0 {
		node.ChannelListID = &req.ChannelListID
	}

	if err := storage.CreateNode(a.ctx.DB, node); err != nil {
		return nil, err
	}

	return &pb.CreateNodeResponse{}, nil
}

// Get returns the Node for the given DevEUI.
func (a *NodeAPI) Get(ctx context.Context, req *pb.GetNodeRequest) (*pb.GetNodeResponse, error) {
	var eui lorawan.EUI64
	if err := eui.UnmarshalText([]byte(req.DevEUI)); err != nil {
		return nil, err
	}

	node, err := storage.GetNode(a.ctx.DB, eui)
	if err != nil {
		return nil, err
	}

	devEUI, err := node.DevEUI.MarshalText()
	if err != nil {
		return nil, err
	}
	appEUI, err := node.AppEUI.MarshalText()
	if err != nil {
		return nil, err
	}
	appKey, err := node.AppKey.MarshalText()
	if err != nil {
		return nil, err
	}

	resp := pb.GetNodeResponse{
		DevEUI:      string(devEUI),
		AppEUI:      string(appEUI),
		AppKey:      string(appKey),
		RxDelay:     uint32(node.RXDelay),
		Rx1DROffset: uint32(node.RX1DROffset),
	}

	if node.ChannelListID != nil {
		resp.ChannelListID = *node.ChannelListID
	}

	return &resp, nil
}

// GetList returns a list of nodes (given a limit and offset).
func (a *NodeAPI) List(ctx context.Context, req *pb.ListNodeRequest) (*pb.ListNodeResponse, error) {
	nodes, err := storage.GetNodes(a.ctx.DB, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, err
	}
	count, err := storage.GetNodesCount(a.ctx.DB)
	return a.returnList(count, nodes)
}

// GetListByAppEUI returns a list of nodes (given an AppEUI, limit and offset).
func (a *NodeAPI) ListByAppEUI(ctx context.Context, req *pb.ListNodeByAppEUIRequest) (*pb.ListNodeResponse, error) {
	var eui lorawan.EUI64
	if err := eui.UnmarshalText([]byte(req.AppEUI)); err != nil {
		return nil, err
	}

	nodes, err := storage.GetNodesForAppEUI(a.ctx.DB, eui, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, err
	}
	count, err := storage.GetNodesForAppEUICount(a.ctx.DB, eui)
	if err != nil {
		return nil, err
	}
	return a.returnList(count, nodes)
}

// Update updates the node matching the given DevEUI.
func (a *NodeAPI) Update(ctx context.Context, req *pb.UpdateNodeRequest) (*pb.UpdateNodeResponse, error) {
	var appEUI, devEUI lorawan.EUI64
	var appKey lorawan.AES128Key

	if err := appEUI.UnmarshalText([]byte(req.AppEUI)); err != nil {
		return nil, err
	}
	if err := devEUI.UnmarshalText([]byte(req.DevEUI)); err != nil {
		return nil, err
	}
	if err := appKey.UnmarshalText([]byte(req.AppKey)); err != nil {
		return nil, err
	}

	node, err := storage.GetNode(a.ctx.DB, devEUI)
	if err != nil {
		return nil, err
	}

	node.AppEUI = appEUI
	node.AppKey = appKey
	node.RXDelay = uint8(req.RxDelay)
	node.RX1DROffset = uint8(req.Rx1DROffset)
	if req.ChannelListID > 0 {
		node.ChannelListID = &req.ChannelListID
	} else {
		node.ChannelListID = nil
	}

	if err := storage.UpdateNode(a.ctx.DB, node); err != nil {
		return nil, err
	}

	return &pb.UpdateNodeResponse{}, nil
}

// Delete deletes the node matching the given DevEUI.
func (a *NodeAPI) Delete(ctx context.Context, req *pb.DeleteNodeRequest) (*pb.DeleteNodeResponse, error) {
	var eui lorawan.EUI64
	if err := eui.UnmarshalText([]byte(req.DevEUI)); err != nil {
		return nil, err
	}

	if err := storage.DeleteNode(a.ctx.DB, eui); err != nil {
		return nil, err
	}

	return &pb.DeleteNodeResponse{}, nil
}

// FlushTXPayloadQueue flushes the tx payload queue for the given DevEUI.
func (a *NodeAPI) FlushTXPayloadQueue(ctx context.Context, req *pb.FlushTXPayloadQueueRequest) (*pb.FlushTXPayloadQueueResponse, error) {
	var eui lorawan.EUI64
	if err := eui.UnmarshalText([]byte(req.DevEUI)); err != nil {
		return nil, err
	}
	if err := storage.FlushTXPayloadQueue(a.ctx.RedisPool, eui); err != nil {
		return nil, err
	}
	return &pb.FlushTXPayloadQueueResponse{}, nil
}

func (a *NodeAPI) returnList(count int, nodes []models.Node) (*pb.ListNodeResponse, error) {
	resp := pb.ListNodeResponse{
		TotalCount: int64(count),
	}
	for _, node := range nodes {
		appEUI, err := node.AppEUI.MarshalText()
		if err != nil {
			return nil, err
		}
		devEUI, err := node.DevEUI.MarshalText()
		if err != nil {
			return nil, err
		}
		appKey, err := node.AppKey.MarshalText()
		if err != nil {
			return nil, err
		}

		item := pb.GetNodeResponse{
			DevEUI:      string(devEUI),
			AppEUI:      string(appEUI),
			AppKey:      string(appKey),
			RxDelay:     uint32(node.RXDelay),
			Rx1DROffset: uint32(node.RX1DROffset),
		}

		if node.ChannelListID != nil {
			item.ChannelListID = *node.ChannelListID
		}

		resp.Result = append(resp.Result, &item)
	}
	return &resp, nil
}
