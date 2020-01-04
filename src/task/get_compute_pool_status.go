package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
)

type GetComputePoolStatusExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *GetComputePoolStatusExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	poolName, err := request.GetString(framework.ParamKeyPool)
	if err != nil{
		return err
	}

	//log.Printf("[%08X] get compute pool '%s' status from %s.[%08X]", id, poolName, request.GetSender(), request.GetFromSession())
	var respChan= make(chan modules.ResourceResult)

	executor.ResourceModule.GetComputePoolStatus(poolName, respChan)
	result := <-respChan

	resp, _ := framework.CreateJsonMessage(framework.GetComputePoolStatusResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	if result.Error != nil{
		err = result.Error
		resp.SetSuccess(false)
		resp.SetError(err.Error())
		log.Printf("[%08X] get compute pool status fail: %s", id, err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	var s = result.ComputePoolStatus

	resp.SetSuccess(true)
	//assemble
	resp.SetString(framework.ParamKeyName, s.Name)
	resp.SetBoolean(framework.ParamKeyEnable, s.Enabled)
	resp.SetUIntArray(framework.ParamKeyCell, []uint64{s.OfflineCells, s.OnlineCells})
	resp.SetUIntArray(framework.ParamKeyInstance, []uint64{s.StoppedInstances, s.RunningInstances, s.LostInstances, s.MigratingInstances})
	resp.SetFloat(framework.ParamKeyUsage, s.CpuUsage)
	resp.SetUInt(framework.ParamKeyCore, s.Cores)
	resp.SetUIntArray(framework.ParamKeyMemory, []uint64{s.MemoryAvailable, s.Memory})
	resp.SetUIntArray(framework.ParamKeyDisk, []uint64{s.DiskAvailable, s.Disk})
	resp.SetUIntArray(framework.ParamKeySpeed, []uint64{s.ReadSpeed, s.WriteSpeed, s.ReceiveSpeed, s.SendSpeed})

	return executor.Sender.SendMessage(resp, request.GetSender())
}

