package task

import (
	"github.com/project-nano/framework"
	"log"
	"github.com/project-nano/core/modules"
)

type QueryCellsByPoolExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}


func (executor *QueryCellsByPoolExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error{
	poolName, err := request.GetString(framework.ParamKeyPool)
	if err != nil{
		return err
	}
	//log.Printf("[%08X] query cells by pool from %s.[%08X]", id, request.GetSender(), request.GetFromSession())
	var respChan = make(chan modules.ResourceResult)
	executor.ResourceModule.QueryCellsInPool(poolName, respChan)
	result := <- respChan
	resp, _ := framework.CreateJsonMessage(framework.QueryComputePoolResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	if result.Error != nil{
		resp.SetError(result.Error.Error())
		log.Printf("[%08X] query cells fail: %s", id, result.Error.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}

	//log.Printf("[%08X] %d cells available in pool '%s'", id, len(result.ComputeCellInfoList), poolName)
	resp.SetSuccess(true)
	modules.CellsToMessage(resp, result.ComputeCellInfoList)

	return executor.Sender.SendMessage(resp, request.GetSender())
}
