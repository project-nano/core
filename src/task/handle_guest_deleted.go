package task

import (
	"github.com/project-nano/framework"
	"log"
	"github.com/project-nano/core/modules"
)

type HandleGuestDeletedExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *HandleGuestDeletedExecutor)Execute(id framework.SessionID, event framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	instanceID, err := event.GetString(framework.ParamKeyInstance)
	if err != nil {
		return err
	}
	log.Printf("[%08X] recv guest '%s' deleted from %s.[%08X]", id, instanceID,
		event.GetSender(), event.GetFromSession())
	var respChan = make(chan error)
	executor.ResourceModule.DeallocateInstance(instanceID, nil, respChan)
	err = <- respChan
	if err != nil{
		log.Printf("[%08X] deallocate guest fail: %s", id, err.Error())
	}
	return nil
}
