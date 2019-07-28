package task

import (
	"github.com/project-nano/framework"
	"modules"
	"log"
)

type HandleAddressChangedExecutor struct {
	ResourceModule modules.ResourceModule
}

func (executor *HandleAddressChangedExecutor)Execute(id framework.SessionID, event framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	instanceID, err := event.GetString(framework.ParamKeyInstance)
	if err != nil {
		return err
	}
	address, err := event.GetString(framework.ParamKeyAddress)
	if err != nil{
		return err
	}

	log.Printf("[%08X] address of guest '%s' changed to %s, notify from %s.[%08X]", id, instanceID,
		address, event.GetSender(), event.GetFromSession())
	var respChan = make(chan error)
	executor.ResourceModule.UpdateInstanceAddress(instanceID, address, respChan)
	err = <- respChan
	if err != nil{
		log.Printf("[%08X] update address fail: %s", id, err.Error())
	}
	return nil
}