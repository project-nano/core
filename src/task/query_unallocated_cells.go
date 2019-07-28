package task

import (
	"github.com/project-nano/framework"
	"modules"
	"log"
)

type QueryUnallocatedCellsExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *QueryUnallocatedCellsExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {

	log.Printf("[%08X] query unallocated compute cells from %s.[%08X]", id, request.GetSender(), request.GetFromSession())
	var respChan= make(chan modules.ResourceResult)

	executor.ResourceModule.GetUnallocatedCells(respChan)
	result := <-respChan

	resp, _ := framework.CreateJsonMessage(framework.QueryUnallocatedComputePoolCellResponse)
	resp.SetSuccess(true)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	modules.CellsToMessage(resp, result.ComputeCellInfoList)
	return executor.Sender.SendMessage(resp, request.GetSender())
}

