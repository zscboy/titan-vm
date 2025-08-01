package logic

import (
	"context"

	"titan-vm/vms/api/internal/svc"
	"titan-vm/vms/api/internal/types"
	"titan-vm/vms/vms"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReInstallVMLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReInstallVMLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReInstallVMLogic {
	return &ReInstallVMLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReInstallVMLogic) ReInstallVM(req *types.ReInstallVMRequest) error {
	request := &vms.ReinstallVMRequest{
		Id:     req.Id,
		VmName: req.VmName,
	}
	_, err := l.svcCtx.Vms.ReinstallVMWithLibvirt(l.ctx, request)
	return err
}
