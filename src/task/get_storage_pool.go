package task

import (
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"log"
)

type GetStoragePoolExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}


func (executor *GetStoragePoolExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error){
	poolName, err := request.GetString(framework.ParamKeyStorage)
	if err != nil{
		return
	}
	log.Printf("[%08X] get storage pool '%s' from %s.[%08X]", id, poolName, request.GetSender(), request.GetFromSession())
	var respChan = make(chan modules.ResourceResult)
	executor.ResourceModule.GetStoragePool(poolName, respChan)
	result := <- respChan
	resp, _ := framework.CreateJsonMessage(framework.GetStoragePoolResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	if result.Error != nil{
		err = result.Error
		resp.SetError(err.Error())
		log.Printf("[%08X] get storage pool fail: %s", id, err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	var poolInfo = result.StoragePool
	resp.SetString(framework.ParamKeyName, poolInfo.Name)
	resp.SetString(framework.ParamKeyType, poolInfo.Type)
	resp.SetString(framework.ParamKeyHost, poolInfo.Host)
	resp.SetString(framework.ParamKeyTarget, poolInfo.Target)
	resp.SetSuccess(true)
	return executor.Sender.SendMessage(resp, request.GetSender())
}
