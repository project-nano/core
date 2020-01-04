package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
)

type QueryComputePoolStatusExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *QueryComputePoolStatusExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {

	//log.Printf("[%08X] query compute pool status from %s.[%08X]", id, request.GetSender(), request.GetFromSession())
	var respChan= make(chan modules.ResourceResult)

	executor.ResourceModule.QueryComputePoolStatus(respChan)
	result := <-respChan

	resp, _ := framework.CreateJsonMessage(framework.QueryComputePoolStatusResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	if result.Error != nil{
		err = result.Error
		resp.SetSuccess(false)
		resp.SetError(err.Error())
		log.Printf("[%08X] query compute pool status fail: %s", id, err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	resp.SetSuccess(true)
	var name []string
	var enabled, cells, instance, usage, cores, memory, disk, speed []uint64
	for _, s := range result.ComputePoolStatusList{
		name = append(name, s.Name)
		if s.Enabled{
			enabled = append(enabled, 1)
		}else{
			enabled = append(enabled, 0)
		}
		cells = append(cells, s.OfflineCells)
		cells = append(cells, s.OnlineCells)
		instance = append(instance, s.StoppedInstances)
		instance = append(instance, s.RunningInstances)
		instance = append(instance, s.LostInstances)
		instance = append(instance, s.MigratingInstances)
		usage = append(usage, uint64(s.CpuUsage))//todo: tripped decimal
		cores = append(cores, uint64(s.Cores))
		memory = append(memory, s.MemoryAvailable)
		memory = append(memory, s.Memory)
		disk = append(disk, s.DiskAvailable)
		disk = append(disk, s.Disk)
		speed = append(speed, s.ReadSpeed)
		speed = append(speed, s.WriteSpeed)
		speed = append(speed, s.ReceiveSpeed)
		speed = append(speed, s.SendSpeed)
	}

	//assemble
	resp.SetStringArray(framework.ParamKeyName, name)
	resp.SetUIntArray(framework.ParamKeyEnable, enabled)
	resp.SetUIntArray(framework.ParamKeyCell, cells)
	resp.SetUIntArray(framework.ParamKeyInstance, instance)
	resp.SetUIntArray(framework.ParamKeyUsage, usage)
	resp.SetUIntArray(framework.ParamKeyCore, cores)
	resp.SetUIntArray(framework.ParamKeyMemory, memory)
	resp.SetUIntArray(framework.ParamKeyDisk, disk)
	resp.SetUIntArray(framework.ParamKeySpeed, speed)
	//log.Printf("[%08X] %d compute pool status available", id, len(name))
	return executor.Sender.SendMessage(resp, request.GetSender())
}

