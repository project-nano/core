package task

import (
	"log"
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
)

type HandleMediaAttachedExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *HandleMediaAttachedExecutor)Execute(id framework.SessionID, event framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	instanceID, err := event.GetString(framework.ParamKeyInstance)
	if err != nil {
		return err
	}

	mediaSource, err := event.GetString(framework.ParamKeyMedia)
	if err != nil{
		return err
	}

	log.Printf("[%08X] media '%s' attached to guest '%s' from %s.[%08X]", id, mediaSource, instanceID,
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
		if result.Instance.MediaAttached{
			log.Printf("[%08X] warning: media already attached", id)
			return nil
		}
		status = result.Instance
	}
	status.MediaAttached = true
	status.MediaSource = mediaSource
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
