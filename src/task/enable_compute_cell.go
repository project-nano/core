package task

import (
	"github.com/project-nano/framework"
	"modules"
	"log"
)

type EnableComputeCellExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *EnableComputeCellExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	poolName, err := request.GetString(framework.ParamKeyPool)
	if err != nil {
		return err
	}
	cellName, err := request.GetString(framework.ParamKeyCell)
	if err != nil {
		return err
	}
	//log.Printf("[%08X] request enable cell '%s' in pool '%s' from %s.[%08X]", id, cellName, poolName,
	//	request.GetSender(), request.GetFromSession())
	var respChan = make(chan error, 1)
	executor.ResourceModule.EnableCell(poolName, cellName, respChan)

	resp, _ := framework.CreateJsonMessage(framework.EnableComputePoolCellResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	err = <-respChan
	if err != nil {
		resp.SetError(err.Error())
		log.Printf("[%08X] enable compute cell fail: %s", id, err.Error())
	}else{
		resp.SetSuccess(true)
		log.Printf("[%08X] cell '%s' enabled in pool %s", id, cellName, poolName)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}