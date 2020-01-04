package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
)

type HandleGuestCreatedExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *HandleGuestCreatedExecutor)Execute(id framework.SessionID, event framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var instanceID, monitorSecret, ethernet string
	var monitorPort uint
	if instanceID, err = event.GetString(framework.ParamKeyInstance); err != nil {
		return
	}

	if monitorPort, err = event.GetUInt(framework.ParamKeyMonitor); err != nil{
		return
	}

	if monitorSecret, err = event.GetString(framework.ParamKeySecret); err != nil{
		return
	}
	if ethernet, err = event.GetString(framework.ParamKeyHardware); err != nil{
		return
	}
	log.Printf("[%08X] recv guest '%s' created from %s.[%08X], monitor port %d", id, instanceID,
		event.GetSender(), event.GetFromSession(), monitorPort)
	var respChan = make(chan error)
	executor.ResourceModule.ConfirmInstance(instanceID, monitorPort, monitorSecret, ethernet, respChan)
	err = <- respChan
	if err != nil{
		log.Printf("[%08X] confirm instance fail: %s", id, err.Error())
	}
	return nil
}
