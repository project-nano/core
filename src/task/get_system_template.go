package task

import (
	"fmt"
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"log"
)

type GetSystemTemplateExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *GetSystemTemplateExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error){
	var templateID string
	if templateID, err = request.GetString(framework.ParamKeyTemplate); err != nil{
		err = fmt.Errorf("get template id fail: %s", err.Error())
		return
	}
	var respChan = make(chan modules.ResourceResult, 1)
	executor.ResourceModule.GetSystemTemplate(templateID, respChan)
	var result = <- respChan

	resp, _ := framework.CreateJsonMessage(framework.GetTemplateResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	if result.Error != nil{
		err = result.Error
		resp.SetError(err.Error())
		log.Printf("[%08X] handle get system template from %s.[%08X] fail: %s",
			id, request.GetSender(), request.GetFromSession(), err.Error())
	}else{
		var t = result.Template
		resp.SetSuccess(true)
		resp.SetString(framework.ParamKeyID, t.ID)
		resp.SetString(framework.ParamKeyName, t.Name)
		resp.SetString(framework.ParamKeyAdmin, t.Admin)
		resp.SetString(framework.ParamKeySystem, t.OperatingSystem)
		resp.SetString(framework.ParamKeyDisk, t.Disk)
		resp.SetString(framework.ParamKeyNetwork, t.Network)
		resp.SetString(framework.ParamKeyDisplay, t.Display)
		resp.SetString(framework.ParamKeyMonitor, t.Control)
		resp.SetString(framework.ParamKeyDevice, t.USB)
		resp.SetString(framework.ParamKeyInterface, t.Tablet)
		resp.SetString(framework.ParamKeyCreate, t.CreatedTime)
		resp.SetString(framework.ParamKeyModify, t.ModifiedTime)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
