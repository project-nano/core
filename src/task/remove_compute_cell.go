package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
)

type RemoveComputePoolExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *RemoveComputePoolExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	pool, err := request.GetString(framework.ParamKeyPool)
	if err != nil{
		return err
	}
	cellName, err := request.GetString(framework.ParamKeyCell)
	if err != nil{
		return err
	}
	log.Printf("[%08X] request remove cell '%s' from pool '%s' from %s.[%08X]", id, cellName, pool,
		request.GetSender(), request.GetFromSession())
	var respChan= make(chan error)
	executor.ResourceModule.RemoveCell(pool, cellName, respChan)

	resp, _ := framework.CreateJsonMessage(framework.RemoveComputePoolCellResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	err = <-respChan
	if err != nil{
		resp.SetError(err.Error())
		log.Printf("[%08X] remove compute cell fail: %s", id, err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	resp.SetSuccess(true)
	if err = executor.Sender.SendMessage(resp, request.GetSender()); err != nil{
		log.Printf("[%08X] warning: send cell removed response to '%s' fail: %s", id, request.GetSender(), err.Error())
	}
	event, _ := framework.CreateJsonMessage(framework.ComputeCellRemovedEvent)
	if err = executor.Sender.SendMessage(event, cellName); err != nil{
		log.Printf("[%08X] warning: notify cell removed to '%s' fail: %s", id, cellName, err.Error())
	}
	return nil
}
