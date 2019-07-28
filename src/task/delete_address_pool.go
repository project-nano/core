package task

import (
	"github.com/project-nano/framework"
	"log"
	"modules"
)

type DeleteAddressPoolExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}


func (executor *DeleteAddressPoolExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var poolName string
	if poolName, err = request.GetString(framework.ParamKeyAddress); err != nil{
		return
	}
	var respChan = make(chan error, 1)
	executor.ResourceModule.DeleteAddressPool(poolName, respChan)
	resp, _ := framework.CreateJsonMessage(framework.DeleteAddressPoolResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)

	err = <- respChan
	if err != nil{
		resp.SetError(err.Error())
		log.Printf("[%08X] request delete address pool from %s.[%08X] fail: %s",
			id, request.GetSender(), request.GetFromSession(), err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}

	resp.SetSuccess(true)
	log.Printf("[%08X] address pool '%s' deleted from %s.[%08X]",
		id, poolName, request.GetSender(), request.GetFromSession())
	return executor.Sender.SendMessage(resp, request.GetSender())
}
