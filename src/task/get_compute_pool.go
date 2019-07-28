package task

import (
	"github.com/project-nano/framework"
	"modules"
	"log"
)

type GetComputePoolExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}


func (executor *GetComputePoolExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error){
	poolName, err := request.GetString(framework.ParamKeyPool)
	if err != nil{
		return
	}
	log.Printf("[%08X] get compute pool '%s' from %s.[%08X]", id, poolName, request.GetSender(), request.GetFromSession())
	var respChan = make(chan modules.ResourceResult)
	executor.ResourceModule.GetComputePool(poolName, respChan)
	result := <- respChan
	resp, _ := framework.CreateJsonMessage(framework.GetComputePoolResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	if result.Error != nil{
		err = result.Error
		resp.SetError(err.Error())
		log.Printf("[%08X] get compute pool fail: %s", id, err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	var poolInfo = result.ComputePoolInfo
	resp.SetString(framework.ParamKeyName, poolInfo.Name)
	resp.SetBoolean(framework.ParamKeyEnable, poolInfo.Enabled)
	resp.SetUInt(framework.ParamKeyCell, uint(poolInfo.CellCount))
	resp.SetString(framework.ParamKeyNetwork, poolInfo.Network)
	resp.SetString(framework.ParamKeyStorage, poolInfo.Storage)
	resp.SetBoolean(framework.ParamKeyOption, poolInfo.Failover)
	resp.SetSuccess(true)
	return executor.Sender.SendMessage(resp, request.GetSender())
}