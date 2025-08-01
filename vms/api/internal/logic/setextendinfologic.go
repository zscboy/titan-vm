package logic

import (
	"context"

	"titan-vm/vms/api/internal/svc"
	"titan-vm/vms/api/internal/types"
	"titan-vm/vms/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetExtendInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetExtendInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetExtendInfoLogic {
	return &SetExtendInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetExtendInfoLogic) SetExtendInfo(req *types.SetNodeExtendInfoRequest) error {
	node, err := model.GetNode(l.svcCtx.Redis, req.Id)
	if err != nil {
		return err
	}
	logx.Info("extend:", req.Extend)
	node.Extend = req.Extend
	return model.SaveNode(l.svcCtx.Redis, node)
}
