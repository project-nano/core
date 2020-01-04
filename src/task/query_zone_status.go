package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
)

type QueryZoneStatusExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *QueryZoneStatusExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {

	//log.Printf("[%08X] query zone status from %s.[%08X]", id, request.GetSender(), request.GetFromSession())
	var respChan= make(chan modules.ResourceResult)

	executor.ResourceModule.QueryZoneStatus(respChan)
	result := <-respChan

	resp, _ := framework.CreateJsonMessage(framework.QueryZoneStatusResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	if result.Error != nil{
		resp.SetSuccess(false)
		resp.SetError(result.Error.Error())
		log.Printf("[%08X] get zone status fail: %s", id, result.Error.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	resp.SetSuccess(true)
	//assemble
	var s = result.ZoneStatus
	resp.SetString(framework.ParamKeyName, s.Name)
	resp.SetUIntArray(framework.ParamKeyPool, []uint64{s.DisabledPools, s.EnabledPools})
	resp.SetUIntArray(framework.ParamKeyCell, []uint64{s.OfflineCells, s.OnlineCells})
	resp.SetUIntArray(framework.ParamKeyInstance, []uint64{s.StoppedInstances, s.RunningInstances, s.LostInstances, s.MigratingInstances})
	resp.SetFloat(framework.ParamKeyUsage, s.CpuUsage)
	resp.SetUInt(framework.ParamKeyCore, s.Cores)
	resp.SetUIntArray(framework.ParamKeyMemory, []uint64{s.MemoryAvailable, s.Memory})
	resp.SetUIntArray(framework.ParamKeyDisk, []uint64{s.DiskAvailable, s.Disk})
	resp.SetUIntArray(framework.ParamKeySpeed, []uint64{s.ReadSpeed, s.WriteSpeed, s.ReceiveSpeed, s.SendSpeed})
	resp.SetString(framework.ParamKeyStart, s.StartTime.Format(modules.TimeFormatLayout))

	return executor.Sender.SendMessage(resp, request.GetSender())
}
