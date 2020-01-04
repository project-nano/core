package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
)

type HandleCellStatusUpdatedExecutor struct {
	ResourceModule modules.ResourceModule
}

func (executor *HandleCellStatusUpdatedExecutor)Execute(id framework.SessionID, event framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	var usage = modules.CellStatusReport{}
	if err := usage.FromMessage(event); err != nil{
		log.Printf("handle cell usage fail: %s", err.Error())
		return err
	}
	executor.ResourceModule.UpdateCellStatus(usage)
	return nil
}