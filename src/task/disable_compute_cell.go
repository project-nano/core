package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
)

type DisableComputeCellExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *DisableComputeCellExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	poolName, err := request.GetString(framework.ParamKeyPool)
	if err != nil {
		return err
	}
	cellName, err := request.GetString(framework.ParamKeyCell)
	if err != nil {
		return err
	}
	var respChan = make(chan error, 1)
	executor.ResourceModule.DisableCell(poolName, cellName, false, respChan)

	resp, _ := framework.CreateJsonMessage(framework.DisableComputePoolCellResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	err = <-respChan
	if err != nil {
		resp.SetError(err.Error())
		log.Printf("[%08X] disable compute cell fail: %s", id, err.Error())
	}else{
		resp.SetSuccess(true)
		log.Printf("[%08X] cell '%s' disabled in pool %s", id, cellName, poolName)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
