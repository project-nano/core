package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
)

type CreateAddressPoolExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}


func (executor *CreateAddressPoolExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var config modules.AddressPoolConfig
	var poolName string
	if poolName, err = request.GetString(framework.ParamKeyAddress); err != nil{
		return
	}
	config.Name = poolName
	if config.Gateway, err = request.GetString(framework.ParamKeyGateway); err != nil{
		return
	}
	if config.DNS, err = request.GetStringArray(framework.ParamKeyServer); err != nil{
		return
	}
	var respChan = make(chan error, 1)
	executor.ResourceModule.CreateAddressPool(config, respChan)
	resp, _ := framework.CreateJsonMessage(framework.CreateAddressPoolResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)

	err = <- respChan
	if err != nil{
		resp.SetError(err.Error())
		log.Printf("[%08X] request create address pool from %s.[%08X] fail: %s",
			id, request.GetSender(), request.GetFromSession(), err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}

	resp.SetSuccess(true)
	log.Printf("[%08X] address pool '%s' created from %s.[%08X]",
		id, poolName, request.GetSender(), request.GetFromSession())
	return executor.Sender.SendMessage(resp, request.GetSender())
}
