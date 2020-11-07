package main

import (
	"github.com/project-nano/framework"
	"log"
	"github.com/project-nano/core/modules"
)

const (
	CurrentVersion = "1.3.0"
)

type CoreService struct {
	framework.EndpointService //base class
	ConfigPath      string
	DataPath        string
	resourceManager *modules.ResourceManager
	transManager    *CoreTransactionManager
	apiModule       *modules.APIModule
}

func (core *CoreService) GetAPIServiceAddress() string{
	if nil != core.apiModule{
		return core.apiModule.GetServiceAddress()
	}
	return ""
}

func (core *CoreService)GetVersion() string{
	return CurrentVersion
}


func (core *CoreService)OnMessageReceived(msg framework.Message){

	if targetSession := msg.GetToSession(); targetSession != 0{
		if err := core.transManager.PushMessage(msg);err != nil{
			log.Printf("<core> push message [%08X] from %s to session [%08X] fail: %s", msg.GetID(), msg.GetSender(), targetSession, err.Error())
		}
		return
	}
	var err error
	switch msg.GetID() {
	case framework.QueryComputePoolRequest:
	case framework.GetComputePoolRequest:
	case framework.CreateComputePoolRequest:
	case framework.DeleteComputePoolRequest:
	case framework.ModifyComputePoolRequest:

	case framework.QueryStoragePoolRequest:
	case framework.GetStoragePoolRequest:
	case framework.CreateStoragePoolRequest:
	case framework.DeleteStoragePoolRequest:
	case framework.ModifyStoragePoolRequest:
	case framework.QueryAddressPoolRequest:
	case framework.GetAddressPoolRequest:
	case framework.CreateAddressPoolRequest:
	case framework.ModifyAddressPoolRequest:
	case framework.DeleteAddressPoolRequest:
	case framework.QueryAddressRangeRequest:
	case framework.GetAddressRangeRequest:
	case framework.AddAddressRangeRequest:
	case framework.RemoveAddressRangeRequest:

	case framework.QueryComputePoolCellRequest:
	case framework.GetComputePoolCellRequest:
	case framework.AddComputePoolCellRequest:
	case framework.RemoveComputePoolCellRequest:
	case framework.QueryUnallocatedComputePoolCellRequest:
	case framework.QueryZoneStatusRequest:
	case framework.QueryComputePoolStatusRequest:
	case framework.GetComputePoolStatusRequest:
	case framework.QueryComputePoolCellStatusRequest:
	case framework.GetComputePoolCellStatusRequest:
	case framework.EnableComputePoolCellRequest:
	case framework.DisableComputePoolCellRequest:
	case framework.QueryCellStorageRequest:
	case framework.ModifyCellStorageRequest:
	case framework.MigrateInstanceRequest:
	case framework.InstanceMigratedEvent:
	case framework.InstancePurgedEvent:

	case framework.ComputeCellAvailableEvent:
	case framework.ImageServerAvailableEvent:

	case framework.QueryGuestRequest:
	case framework.GetGuestRequest:
	case framework.CreateGuestRequest:
	case framework.DeleteGuestRequest:
	case framework.ResetSystemRequest:
	case framework.QueryInstanceStatusRequest:
	case framework.GetInstanceStatusRequest:
	case framework.StartInstanceRequest:
	case framework.StopInstanceRequest:
	case framework.ResetSecretRequest:
	case framework.GuestCreatedEvent:
	case framework.GuestDeletedEvent:
	case framework.GuestStartedEvent:
	case framework.GuestStoppedEvent:
	case framework.GuestUpdatedEvent:
	case framework.CellStatusReportEvent:
	case framework.AddressChangedEvent:
	case framework.SystemResetEvent:
	case framework.StartBatchCreateGuestRequest:
	case framework.GetBatchCreateGuestRequest:
	case framework.StartBatchDeleteGuestRequest:
	case framework.GetBatchDeleteGuestRequest:
	case framework.StartBatchStopGuestRequest:
	case framework.GetBatchStopGuestRequest:
	case framework.ModifyPriorityRequest:
	case framework.ModifyDiskThresholdRequest:
	case framework.ModifyNetworkThresholdRequest:

	case framework.InsertMediaRequest:
	case framework.EjectMediaRequest:
	case framework.MediaAttachedEvent:
	case framework.MediaDetachedEvent:

	case framework.ModifyGuestNameRequest:
	case framework.ModifyCoreRequest:
	case framework.ModifyMemoryRequest:
	case framework.ModifyAuthRequest:
	case framework.GetAuthRequest:
	case framework.ResizeDiskRequest:
	case framework.ShrinkDiskRequest:

	case framework.QueryDiskImageRequest:
	case framework.GetDiskImageRequest:
	case framework.CreateDiskImageRequest:
	case framework.DeleteDiskImageRequest:
	case framework.ModifyDiskImageRequest:
	case framework.SynchronizeDiskImageRequest:

	case framework.QueryMediaImageRequest:
	case framework.GetMediaImageRequest:
	case framework.CreateMediaImageRequest:
	case framework.DeleteMediaImageRequest:
	case framework.ModifyMediaImageRequest:
	case framework.SynchronizeMediaImageRequest:

	case framework.QuerySnapshotRequest:
	case framework.GetSnapshotRequest:
	case framework.CreateSnapshotRequest:
	case framework.DeleteSnapshotRequest:
	case framework.RestoreSnapshotRequest:
	case framework.SnapshotResumedEvent:

	case framework.QueryMigrationRequest:
	case framework.GetMigrationRequest:
	case framework.CreateMigrationRequest:
	case framework.QueryTemplateRequest:
	case framework.GetTemplateRequest:
	case framework.CreateTemplateRequest:
	case framework.ModifyTemplateRequest:
	case framework.DeleteTemplateRequest:
	case framework.ComputeCellDisconnectedEvent:
	//security policy group
	case framework.QueryPolicyGroupRequest:
	case framework.GetPolicyGroupRequest:
	case framework.CreatePolicyGroupRequest:
	case framework.ModifyPolicyGroupRequest:
	case framework.DeletePolicyGroupRequest:
	case framework.QueryPolicyRuleRequest:
	case framework.AddPolicyRuleRequest:
	case framework.ModifyPolicyRuleRequest:
	case framework.ChangePolicyRuleOrderRequest:
	case framework.RemovePolicyRuleRequest:

	//guest security policy
	case framework.GetGuestRuleRequest:
	case framework.ChangeGuestRuleOrderRequest:
	case framework.ChangeGuestRuleDefaultActionRequest:
	case framework.AddGuestRuleRequest:
	case framework.ModifyGuestRuleRequest:
	case framework.RemoveGuestRuleRequest:
	default:
		core.handleIncomingMessage(msg)
		return
	}
	//Invoke transaction
	err = core.transManager.InvokeTask(msg)
	if err != nil{
		log.Printf("<core> invoke transaction with message [%08X] fail: %s", msg.GetID(), err.Error())
	}
}
func (core *CoreService) handleIncomingMessage(msg framework.Message){
	switch msg.GetID() {
	default:
		log.Printf("<core> message [%08X] from %s.[%08X] ignored", msg.GetID(), msg.GetSender(), msg.GetFromSession())
	}
}

func (core *CoreService)OnServiceConnected(name string, t framework.ServiceType, remoteAddress string){
	log.Printf("<core> service %s connected, type %d", name, t)
	switch t {
	case framework.ServiceTypeCell:
		event, _ := framework.CreateJsonMessage(framework.ComputeCellAvailableEvent)
		event.SetString(framework.ParamKeyCell, name)
		event.SetString(framework.ParamKeyAddress, remoteAddress)
		core.SendToSelf(event)
	default:
		break
	}
}

func (core *CoreService)OnServiceDisconnected(nodeName string, t framework.ServiceType, gracefullyClose bool){
	if gracefullyClose{
		log.Printf("<core> service %s closed by remote, type %d", nodeName, t)
	}else{
		log.Printf("<core> service %s lost, type %d", nodeName, t)
	}

	switch t {
	case framework.ServiceTypeCell:
		event, _ := framework.CreateJsonMessage(framework.ComputeCellDisconnectedEvent)
		event.SetString(framework.ParamKeyCell, nodeName)
		event.SetBoolean(framework.ParamKeyFlag, gracefullyClose)
		core.SendToSelf(event)
	case framework.ServiceTypeImage:
		core.resourceManager.RemoveImageServer(nodeName)
	default:
		break
	}
}

func (core *CoreService)OnDependencyReady(){
	core.SetServiceReady()
}

func (core *CoreService)InitialEndpoint() (err error){
	log.Printf("<core> initial core service, v %s", CurrentVersion)
	log.Printf("<core> domain %s, group address %s:%d", core.GetDomain(), core.GetGroupAddress(), core.GetGroupPort())

	core.resourceManager, err = modules.CreateResourceManager(core.DataPath)
	if err != nil{
		return err
	}
	core.transManager, err = CreateTransactionManager(core, core.resourceManager)
	if err != nil{
		return err
	}

	core.apiModule, err = modules.CreateAPIModule(core.ConfigPath, core, core.resourceManager)
	if err != nil{
		return err
	}
	//register submodules
	if err = core.RegisterSubmodule(core.apiModule.GetModuleName(), core.apiModule.GetResponseChannel());err != nil{
		return err
	}
	return nil
}

func (core *CoreService)OnEndpointStarted() (err error){
	if err = core.resourceManager.Start(); err != nil{
		return err
	}
	if err = core.transManager.Start(); err != nil{
		return err
	}
	if err = core.apiModule.Start();err != nil{
		return err
	}
	log.Print("<core> started")
	return nil
}

func (core *CoreService)OnEndpointStopped(){
	if err := core.apiModule.Stop(); err != nil{
		log.Printf("<core> stop api module fail: %s", err.Error())
	}
	if err := core.transManager.Stop(); err != nil{
		log.Printf("<core> stop transaction manager fail: %s", err.Error())
	}
	if err := core.resourceManager.Stop(); err != nil{
		log.Printf("<core> stop compute pool module fail: %s", err.Error())
	}
	log.Print("<core> stopped")
}
