package task

import (
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
)

type QueryStoragePoolExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}


func (executor *QueryStoragePoolExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error{
	//log.Printf("[%08X] query storage pool from %s.[%08X]", id, request.GetSender(), request.GetFromSession())
	var respChan = make(chan modules.ResourceResult)
	executor.ResourceModule.QueryStoragePool(respChan)
	result := <- respChan
	var nameArray, typeArray, hostArray, targetArray []string
	for _, info := range result.StoragePoolList {
		nameArray = append(nameArray, info.Name)
		typeArray = append(typeArray, info.Type)
		hostArray = append(hostArray, info.Host)
		targetArray = append(targetArray, info.Target)
	}
	resp, _ := framework.CreateJsonMessage(framework.QueryStoragePoolResponse)
	resp.SetSuccess(true)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetStringArray(framework.ParamKeyName, nameArray)
	resp.SetStringArray(framework.ParamKeyType, typeArray)
	resp.SetStringArray(framework.ParamKeyHost, hostArray)
	resp.SetStringArray(framework.ParamKeyTarget, targetArray)
	return executor.Sender.SendMessage(resp, request.GetSender())
}
