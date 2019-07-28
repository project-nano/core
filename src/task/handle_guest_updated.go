package task

import (
	"github.com/project-nano/framework"
	"modules"
	"log"
	"errors"
)

type HandleGuestUpdatedExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *HandleGuestUpdatedExecutor)Execute(id framework.SessionID, event framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	instanceID, err := event.GetString(framework.ParamKeyInstance)
	if err != nil {
		return err
	}
	if !event.IsSuccess(){
		log.Printf("[%08X] guest '%s' create fail: %s", id, instanceID, event.GetError())
		err = errors.New(event.GetError())
		var respChan = make(chan error)
		executor.ResourceModule.DeallocateInstance(instanceID, err, respChan)
		<- respChan
		return nil
	}

	progress, err := event.GetUInt(framework.ParamKeyProgress)
	if err != nil{
		return err
	}

	log.Printf("[%08X] update guest '%s' progress to %d%% from %s.[%08X]", id, instanceID,
		progress, event.GetSender(), event.GetFromSession())

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
		if result.InstanceStatus.Created{
			log.Printf("[%08X] warning: guest already created", id)
			return nil
		}
		status = result.InstanceStatus
	}
	status.Progress = progress
	{
		var respChan = make(chan error)
		executor.ResourceModule.UpdateInstanceStatus(status, respChan)
		err = <- respChan
		if err != nil{
			log.Printf("[%08X] warning: update progress fail: %s", id, err)
		}
		return nil
	}
}