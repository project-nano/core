package task

import (
	"fmt"
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"log"
)

type DeleteSystemTemplateExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *DeleteSystemTemplateExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error){

	var templateID string
	if templateID, err = request.GetString(framework.ParamKeyTemplate); err != nil{
		err = fmt.Errorf("get template id fail: %s", err.Error())
		return
	}
	var respChan = make(chan error, 1)
	executor.ResourceModule.DeleteSystemTemplate(templateID, respChan)
	err = <- respChan

	resp, _ := framework.CreateJsonMessage(framework.DeleteTemplateResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	if err != nil{
		resp.SetError(err.Error())
		log.Printf("[%08X] handle delete system template from %s.[%08X] fail: %s",
			id, request.GetSender(), request.GetFromSession(), err.Error())
	}else{
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
