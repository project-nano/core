package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
)

type GetComputeCellStatusExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *GetComputeCellStatusExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	poolName, err := request.GetString(framework.ParamKeyPool)
	if err != nil{
		return err
	}
	cellName, err := request.GetString(framework.ParamKeyCell)
	if err != nil{
		return err
	}

	//log.Printf("[%08X] get compute cell '%s' status from %s.[%08X]", id, cellName, request.GetSender(), request.GetFromSession())
	var respChan= make(chan modules.ResourceResult)

	executor.ResourceModule.GetComputeCellStatus(poolName, cellName, respChan)
	result := <-respChan

	resp, _ := framework.CreateJsonMessage(framework.GetComputePoolCellStatusResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	if result.Error != nil{
		err = result.Error
		resp.SetSuccess(false)
		resp.SetError(err.Error())
		log.Printf("[%08X] get compute cell status fail: %s", id, err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	var s = result.ComputeCell

	resp.SetSuccess(true)
	//assemble
	resp.SetString(framework.ParamKeyName, s.Name)
	resp.SetString(framework.ParamKeyAddress, s.Address)
	resp.SetBoolean(framework.ParamKeyEnable, s.Enabled)
	resp.SetBoolean(framework.ParamKeyStatus, s.Alive)
	resp.SetUIntArray(framework.ParamKeyInstance, []uint64{s.StoppedInstances, s.RunningInstances, s.LostInstances, s.MigratingInstances})
	resp.SetFloat(framework.ParamKeyUsage, s.CpuUsage)
	resp.SetUInt(framework.ParamKeyCore, s.Cores)
	resp.SetUIntArray(framework.ParamKeyMemory, []uint64{s.MemoryAvailable, s.Memory})
	resp.SetUIntArray(framework.ParamKeyDisk, []uint64{s.DiskAvailable, s.Disk})
	resp.SetUIntArray(framework.ParamKeySpeed, []uint64{s.ReadSpeed, s.WriteSpeed, s.ReceiveSpeed, s.SendSpeed})

	return executor.Sender.SendMessage(resp, request.GetSender())
}

