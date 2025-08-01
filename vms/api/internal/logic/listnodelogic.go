package logic

import (
	"context"

	"titan-vm/vms/api/internal/svc"
	"titan-vm/vms/api/internal/types"
	"titan-vm/vms/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListNodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListNodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListNodeLogic {
	return &ListNodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListNodeLogic) ListNode(req *types.ListNodeReqeust) (resp *types.ListNodeResponse, err error) {
	modelNodes, err := model.ListNode(l.ctx, l.svcCtx.Redis, int(req.Start), int(req.End))
	if err != nil {
		return nil, err
	}

	nodes := make([]*types.Node, 0, len(modelNodes))
	for _, modelNode := range modelNodes {
		node := &types.Node{
			Id:          modelNode.Id,
			OS:          modelNode.OS,
			VmType:      modelNode.VmAPI,
			TotalCpu:    int(modelNode.CPU),
			TotalMemory: int(modelNode.Memory),
			IP:          modelNode.IP,
			Online:      modelNode.Online,
			Extend:      modelNode.Extend,
		}
		nodes = append(nodes, node)
	}

	len, err := model.GetNodeLen(l.svcCtx.Redis)
	if err != nil {
		return nil, err
	}

	return &types.ListNodeResponse{Nodes: nodes, Total: int32(len)}, nil
}
