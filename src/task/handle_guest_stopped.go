package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
)

type HandleGuestStoppedExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *HandleGuestStoppedExecutor)Execute(id framework.SessionID, event framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	instanceID, err := event.GetString(framework.ParamKeyInstance)
	if err != nil {
		return err
	}
	log.Printf("[%08X] recv guest '%s' stopped from %s.[%08X]", id, instanceID,
		event.GetSender(), event.GetFromSession())
	var status modules.InstanceStatus
	{
		var respChan = make(chan modules.ResourceResult)
		executor.ResourceModule.GetInstanceStatus(instanceID, respChan)
		result := <- respChan
		if result.Error != nil{
			errMsg := result.Error.Error()
			log.Printf("[%08X] fetch guest fail: %s", id, errMsg)
			return result.Error
		}
		status = result.InstanceStatus
	}
	status.Running = false
	{
		var respChan = make(chan error)
		executor.ResourceModule.UpdateInstanceStatus(status, respChan)
		err = <- respChan
		if err != nil{
			log.Printf("[%08X] warning: update stopped status fail: %s", id, err)
		}
		return nil
	}
}
