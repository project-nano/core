package task

import (
	"github.com/project-nano/framework"
	"modules"
	"log"
)

type HandleGuestCreatedExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *HandleGuestCreatedExecutor)Execute(id framework.SessionID, event framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	instanceID, err := event.GetString(framework.ParamKeyInstance)
	if err != nil {
		return err
	}
	monitorPort, err := event.GetUInt(framework.ParamKeyMonitor)
	if err != nil{
		return err
	}
	var monitorSecret = ""
	monitorSecret, _ = event.GetString(framework.ParamKeySecret)
	log.Printf("[%08X] recv guest '%s' created from %s.[%08X], monitor port %d", id, instanceID,
		event.GetSender(), event.GetFromSession(), monitorPort)
	var respChan = make(chan error)
	executor.ResourceModule.ConfirmInstance(instanceID, monitorPort, monitorSecret, respChan)
	err = <- respChan
	if err != nil{
		log.Printf("[%08X] confirm instance fail: %s", id, err.Error())
	}
	return nil
}
