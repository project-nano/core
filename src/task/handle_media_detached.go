package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
)

type HandleMediaDetachedExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *HandleMediaDetachedExecutor)Execute(id framework.SessionID, event framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	instanceID, err := event.GetString(framework.ParamKeyInstance)
	if err != nil {
		return err
	}

	log.Printf("[%08X] media '%s' detached from guest '%s' from %s.[%08X]", id, instanceID,
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
		if !result.InstanceStatus.MediaAttached{
			log.Printf("[%08X] warning: media already detached", id)
			return nil
		}
		status = result.InstanceStatus
	}
	status.MediaAttached = false
	status.MediaSource = ""
	{
		var respChan = make(chan error)
		executor.ResourceModule.UpdateInstanceStatus(status, respChan)
		err = <- respChan
		if err != nil{
			log.Printf("[%08X] warning: update media status fail: %s", id, err)
		}
		return nil
	}
}
