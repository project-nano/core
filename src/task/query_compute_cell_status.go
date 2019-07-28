package task

import (
	"github.com/project-nano/framework"
	"modules"
	"log"
)

type QueryComputeCellStatusExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *QueryComputeCellStatusExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	poolName, err := request.GetString(framework.ParamKeyPool)
	if err != nil{
		return err
	}

	//log.Printf("[%08X] query cell status in pool '%s' from %s.[%08X]", id, poolName, request.GetSender(), request.GetFromSession())

	var respChan= make(chan modules.ResourceResult)
	executor.ResourceModule.QueryComputeCellStatus(poolName, respChan)
	result := <-respChan

	resp, _ := framework.CreateJsonMessage(framework.QueryComputePoolCellStatusResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	if result.Error != nil{
		err = result.Error
		resp.SetSuccess(false)
		resp.SetError(err.Error())
		log.Printf("[%08X] query compute cell status fail: %s", id, err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}

	resp.SetSuccess(true)
	var name, address []string
	var enabled, alive, instance, usage, cores, memory, disk, speed []uint64
	for _, s := range result.ComputeCellStatusList{
		name = append(name, s.Name)
		address = append(address, s.Address)
		if s.Enabled{
			enabled = append(enabled, 1)
		}else{
			enabled = append(enabled, 0)
		}
		if s.Alive{
			alive = append(alive, 1)
		}else{
			alive = append(alive, 0)
		}
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
	resp.SetStringArray(framework.ParamKeyAddress, address)
	resp.SetUIntArray(framework.ParamKeyEnable, enabled)
	resp.SetUIntArray(framework.ParamKeyStatus, alive)
	resp.SetUIntArray(framework.ParamKeyInstance, instance)
	resp.SetUIntArray(framework.ParamKeyUsage, usage)
	resp.SetUIntArray(framework.ParamKeyCore, cores)
	resp.SetUIntArray(framework.ParamKeyMemory, memory)
	resp.SetUIntArray(framework.ParamKeyDisk, disk)
	resp.SetUIntArray(framework.ParamKeySpeed, speed)
	//log.Printf("[%08X] %d compute cell status available", id, len(name))
	return executor.Sender.SendMessage(resp, request.GetSender())
}

