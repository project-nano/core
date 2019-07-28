package task

import (
	"github.com/project-nano/framework"
	"log"
	"fmt"
	"modules"
)

type ModifyStoragePoolExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *ModifyStoragePoolExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var pool, storageType, host, target string
	pool, err = request.GetString(framework.ParamKeyStorage)
	if err != nil {
		return
	}
	if storageType, err = request.GetString(framework.ParamKeyType); err != nil{
		return
	}
	if host, err = request.GetString(framework.ParamKeyHost); err != nil{
		return
	}
	if target, err = request.GetString(framework.ParamKeyTarget); err != nil{
		return
	}

	log.Printf("[%08X] request modify storage pool '%s' from %s.[%08X]", id, pool,
		request.GetSender(), request.GetFromSession())

	resp, _ := framework.CreateJsonMessage(framework.ModifyStoragePoolResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	if err = QualifyNormalName(pool); err != nil{
		log.Printf("[%08X] invalid pool name '%s' : %s", id, pool, err.Error())
		err = fmt.Errorf("invalid pool name '%s': %s", pool, err.Error())
		resp.SetError(err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}

	var respChan= make(chan error)
	executor.ResourceModule.ModifyStoragePool(pool, storageType, host, target, respChan)
	err = <-respChan
	if err != nil{
		resp.SetError(err.Error())
		log.Printf("[%08X] modify storage pool fail: %s", id, err.Error())
	}else{
		resp.SetSuccess(true)
	}

	return executor.Sender.SendMessage(resp, request.GetSender())
}

