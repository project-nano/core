package main

import (
	"github.com/project-nano/framework"
	"task"
	"modules"
	"net/http"
	"crypto/tls"
)

type CoreTransactionManager struct {
	*framework.TransactionEngine
}

func CreateTransactionManager(sender framework.MessageSender, resourceModule modules.ResourceModule) (*CoreTransactionManager, error) {
	engine, err := framework.CreateTransactionEngine()
	if err != nil{
		return nil, err
	}
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}

	var manager = CoreTransactionManager{engine}
	if err = manager.RegisterExecutor(framework.QueryComputePoolRequest,
		&task.QueryComputePoolExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetComputePoolRequest,
		&task.GetComputePoolExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}

	if err = manager.RegisterExecutor(framework.CreateComputePoolRequest,
		&task.CreateComputePoolExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ModifyComputePoolRequest,
		&task.ModifyComputePoolExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.DeleteComputePoolRequest,
		&task.DeleteComputePoolExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	//storage pools
	if err = manager.RegisterExecutor(framework.QueryStoragePoolRequest,
		&task.QueryStoragePoolExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetStoragePoolRequest,
		&task.GetStoragePoolExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}

	if err = manager.RegisterExecutor(framework.CreateStoragePoolRequest,
		&task.CreateStoragePoolExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ModifyStoragePoolRequest,
		&task.ModifyStoragePoolExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.DeleteStoragePoolRequest,
		&task.DeleteStoragePoolExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}

	//address pool&range
	if err = manager.RegisterExecutor(framework.QueryAddressPoolRequest,
		&task.QueryAddressPoolExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetAddressPoolRequest,
		&task.GetAddressPoolExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.CreateAddressPoolRequest,
		&task.CreateAddressPoolExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ModifyAddressPoolRequest,
		&task.ModifyAddressPoolExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.DeleteAddressPoolRequest,
		&task.DeleteAddressPoolExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.QueryAddressRangeRequest,
		&task.QueryAddressRangeExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetAddressRangeRequest,
		&task.GetAddressRangeExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.AddAddressRangeRequest,
		&task.AddAddressRangeExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.RemoveAddressRangeRequest,
		&task.RemoveAddressRangeExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	
	if err = manager.RegisterExecutor(framework.QueryComputePoolCellRequest,
		&task.QueryCellsByPoolExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetComputePoolCellRequest,
		&task.GetComputeCellExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.AddComputePoolCellRequest,
		&task.AddComputePoolExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.RemoveComputePoolCellRequest,
		&task.RemoveComputePoolExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.QueryUnallocatedComputePoolCellRequest,
		&task.QueryUnallocatedCellsExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.QueryZoneStatusRequest,
		&task.QueryZoneStatusExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.QueryComputePoolStatusRequest,
		&task.QueryComputePoolStatusExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetComputePoolStatusRequest,
		&task.GetComputePoolStatusExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.QueryComputePoolCellStatusRequest,
		&task.QueryComputeCellStatusExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetComputePoolCellStatusRequest,
		&task.GetComputeCellStatusExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.EnableComputePoolCellRequest,
		&task.EnableComputeCellExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.DisableComputePoolCellRequest,
		&task.DisableComputeCellExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ComputeCellAvailableEvent,
		&task.HandleCellAvailableExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetGuestRequest,
		&task.GetGuestConfigExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.QueryGuestRequest,
		&task.QueryGuestConfigExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.CreateGuestRequest,
		&task.CreateGuestExecutor{sender, resourceModule, client}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.DeleteGuestRequest,
		&task.DeleteGuestExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ModifyGuestNameRequest,
		&task.ModifyGuestNameExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ModifyCoreRequest,
		&task.ModifyGuestCoreExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ModifyMemoryRequest,
		&task.ModifyGuestMemoryExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}

	if err = manager.RegisterExecutor(framework.ModifyPriorityRequest,
		&task.ModifyGuestPriorityExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ModifyNetworkThresholdRequest,
		&task.ModifyGuestNetworkThresholdExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ModifyDiskThresholdRequest,
		&task.ModifyGuestDiskThresholdExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ResizeDiskRequest,
		&task.ResizeGuestDiskExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ShrinkDiskRequest,
		&task.ShrinkGuestDiskExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ResetSystemRequest,
		&task.ResetGuestSystemExecutor{sender, resourceModule, client}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ModifyAuthRequest,
		&task.ModifyGuestPasswordExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetAuthRequest,
		&task.GetGuestPasswordExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}

	if err = manager.RegisterExecutor(framework.GetInstanceStatusRequest,
		&task.GetInstanceStatusExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.QueryInstanceStatusRequest,
		&task.QueryInstanceStatusExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}

	if err = manager.RegisterExecutor(framework.StartInstanceRequest,
		&task.StartInstanceExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.StopInstanceRequest,
		&task.StopInstanceExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}

	//media images
	if err = manager.RegisterExecutor(framework.QueryMediaImageRequest,
		&task.QueryMediaImageExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetMediaImageRequest,
		&task.GetMediaImageExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.CreateMediaImageRequest,
		&task.CreateMediaImageExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ModifyMediaImageRequest,
		&task.ModifyMediaImageExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.DeleteMediaImageRequest,
		&task.DeleteMediaImageExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}

	//disk images
	if err = manager.RegisterExecutor(framework.QueryDiskImageRequest,
		&task.QueryDiskImageExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetDiskImageRequest,
		&task.GetDiskImageExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.CreateDiskImageRequest,
		&task.CreateDiskImageExecutor{sender, resourceModule, client}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ModifyDiskImageRequest,
		&task.ModifyDiskImageExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.DeleteDiskImageRequest,
		&task.DeleteDiskImageExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}

	if err = manager.RegisterExecutor(framework.GuestCreatedEvent,
		&task.HandleGuestCreatedExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GuestDeletedEvent,
		&task.HandleGuestDeletedExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}

	if err = manager.RegisterExecutor(framework.GuestStartedEvent,
		&task.HandleGuestStartedExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GuestStoppedEvent,
		&task.HandleGuestStoppedExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GuestUpdatedEvent,
		&task.HandleGuestUpdatedExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.SystemResetEvent,
		&task.HandleGuestSystemResetExecutor{resourceModule}); err != nil{
		return nil, err
	}
	//batch
	if err = manager.RegisterExecutor(framework.StartBatchCreateGuestRequest,
		&task.StartBatchCreateGuestExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetBatchCreateGuestRequest,
		&task.GetBatchCreateGuestExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.StartBatchDeleteGuestRequest,
		&task.StartBatchDeleteGuestExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetBatchDeleteGuestRequest,
		&task.GetBatchDeleteGuestExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.StartBatchStopGuestRequest,
		&task.StartBatchStopGuestExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetBatchStopGuestRequest,
		&task.GetBatchStopGuestExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	//instance media
	if err = manager.RegisterExecutor(framework.InsertMediaRequest,
		&task.InsertMediaExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.EjectMediaRequest,
		&task.EjectMediaExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.MediaAttachedEvent,
		&task.HandleMediaAttachedExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.MediaDetachedEvent,
		&task.HandleMediaDetachedExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}

	//snapshot
	if err = manager.RegisterExecutor(framework.QuerySnapshotRequest,
		&task.QuerySnapshotExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetSnapshotRequest,
		&task.GetSnapshotExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.CreateSnapshotRequest,
		&task.CreateSnapshotExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.DeleteSnapshotRequest,
		&task.DeleteSnapshotExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.RestoreSnapshotRequest,
		&task.RestoreSnapshotExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.QueryMigrationRequest,
		&task.QueryMigrationExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetMigrationRequest,
		&task.GetMigrationExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.CreateMigrationRequest,
		&task.CreateMigrationExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.InstanceMigratedEvent,
		&task.HandleInstanceMigratedExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.CellStatusReportEvent,
		&task.HandleCellStatusUpdatedExecutor{resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.AddressChangedEvent,
		&task.HandleAddressChangedExecutor{resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ComputeCellDisconnectedEvent,
		&task.HandleCellDisconnectedExecutor{sender, resourceModule}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ImageServerAvailableEvent,
		&task.SyncImageServerExecutor{sender, resourceModule, client}); err != nil{
		return nil, err
	}

	return &manager, nil
}
