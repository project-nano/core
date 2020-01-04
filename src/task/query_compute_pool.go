package task

import (
	"github.com/project-nano/framework"
	"log"
	"github.com/project-nano/core/modules"
)

type QueryComputePoolExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}


func (executor *QueryComputePoolExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error{
	log.Printf("[%08X] query compute pool from %s.[%08X]", id, request.GetSender(), request.GetFromSession())
	var respChan = make(chan modules.ResourceResult)
	executor.ResourceModule.GetAllComputePool(respChan)
	result := <- respChan
	var nameArray, networkArray, storageArray []string
	var cellArray, statusArray, failoverArray []uint64
	for _, info := range result.ComputePoolInfoList{
		if info.Enabled{
			statusArray = append(statusArray, 1)
		}else {
			statusArray = append(statusArray, 0)
		}
		if info.Failover{
			failoverArray = append(failoverArray, 1)
		}else {
			failoverArray = append(failoverArray, 0)
		}
		nameArray = append(nameArray, info.Name)
		cellArray = append(cellArray, info.CellCount)
		networkArray = append(networkArray, info.Network)
		storageArray = append(storageArray, info.Storage)
	}
	resp, _ := framework.CreateJsonMessage(framework.QueryComputePoolResponse)
	resp.SetSuccess(true)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetStringArray(framework.ParamKeyName, nameArray)
	resp.SetStringArray(framework.ParamKeyNetwork, networkArray)
	resp.SetStringArray(framework.ParamKeyStorage, storageArray)
	resp.SetUIntArray(framework.ParamKeyStatus, statusArray)
	resp.SetUIntArray(framework.ParamKeyCell, cellArray)
	resp.SetUIntArray(framework.ParamKeyOption, failoverArray)
	return executor.Sender.SendMessage(resp, request.GetSender())
}