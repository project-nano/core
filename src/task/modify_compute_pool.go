package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
	"fmt"
)

type ModifyComputePoolExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *ModifyComputePoolExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	pool, err := request.GetString(framework.ParamKeyPool)
	if err != nil{
		return err
	}
	storagePool, _ := request.GetString(framework.ParamKeyStorage)
	if "" == storagePool{
		log.Printf("[%08X] request modify compute pool '%s' using local storage from %s.[%08X]", id, pool, request.GetSender(), request.GetFromSession())
	}else{
		log.Printf("[%08X] request modify compute pool '%s' using storage pool '%s' from %s.[%08X]", id, pool, storagePool,
			request.GetSender(), request.GetFromSession())
	}
	addressPool, _ := request.GetString(framework.ParamKeyNetwork)
	var failover = false
	failover, _ = request.GetBoolean(framework.ParamKeyOption)


	resp, _ := framework.CreateJsonMessage(framework.ModifyComputePoolResponse)
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
	executor.ResourceModule.ModifyPool(pool, storagePool, addressPool, failover, respChan)
	err = <-respChan
	if err != nil{
		resp.SetError(err.Error())
		log.Printf("[%08X] modify compute pool fail: %s", id, err.Error())
	}else{
		resp.SetSuccess(true)
	}

	return executor.Sender.SendMessage(resp, request.GetSender())
}
