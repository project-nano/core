package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
)

type GetGuestConfigExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *GetGuestConfigExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	var err error
	instanceID, err := request.GetString(framework.ParamKeyInstance)
	if err != nil{
		return err
	}
	//log.Printf("[%08X] request get guest '%s' config from %s.[%08X]", id, instanceID,
	//	request.GetSender(), request.GetFromSession())

	resp, _ := framework.CreateJsonMessage(framework.QueryGuestResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetSuccess(false)

	var config modules.InstanceStatus
	{
		var respChan = make(chan modules.ResourceResult)
		executor.ResourceModule.GetInstanceStatus(instanceID, respChan)
		result := <- respChan
		if result.Error != nil{
			errMsg := result.Error.Error()
			log.Printf("[%08X] get config fail: %s", id, errMsg)
			resp.SetError(errMsg)
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		config = result.Instance
	}
	config.Marshal(resp)
	resp.SetSuccess(true)
	return executor.Sender.SendMessage(resp, request.GetSender())
}
