package task

import (
	"github.com/project-nano/framework"
	"modules"
	"log"
)

type DeleteStoragePoolExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *DeleteStoragePoolExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	pool, err := request.GetString(framework.ParamKeyStorage)
	if err != nil{
		return err
	}
	log.Printf("[%08X] request delete storage pool '%s' from %s.[%08X]", id, pool, request.GetSender(), request.GetFromSession())
	var respChan= make(chan error)

	executor.ResourceModule.DeleteStoragePool(pool, respChan)

	resp, _ := framework.CreateJsonMessage(framework.DeleteStoragePoolResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	err = <-respChan
	if err != nil{
		resp.SetError(err.Error())
		log.Printf("[%08X] delete storage pool fail: %s", id, err.Error())
	}else{
		resp.SetSuccess(true)
	}

	return executor.Sender.SendMessage(resp, request.GetSender())
}
