package task

import (
	"fmt"
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"log"
)

type ModifySystemTemplateExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *ModifySystemTemplateExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error){
	var templateID string
	if templateID, err = request.GetString(framework.ParamKeyTemplate); err != nil{
		err = fmt.Errorf("get template id fail: %s", err.Error())
		return
	}
	var config modules.SystemTemplateConfig
	if config.Name, err = request.GetString(framework.ParamKeyName); err != nil{
		err = fmt.Errorf("get template name fail: %s", err.Error())
		return
	}
	if config.Admin, err = request.GetString(framework.ParamKeyAdmin); err != nil{
		err = fmt.Errorf("get admin name fail: %s", err.Error())
		return
	}
	if config.OperatingSystem, err = request.GetString(framework.ParamKeySystem); err != nil{
		err = fmt.Errorf("get oprating system fail: %s", err.Error())
		return
	}
	if config.Disk, err = request.GetString(framework.ParamKeyDisk); err != nil{
		err = fmt.Errorf("get disk option fail: %s", err.Error())
		return
	}
	if config.Network, err = request.GetString(framework.ParamKeyNetwork); err != nil{
		err = fmt.Errorf("get network option fail: %s", err.Error())
		return
	}
	if config.Display, err = request.GetString(framework.ParamKeyDisplay); err != nil{
		err = fmt.Errorf("get display option fail: %s", err.Error())
		return
	}
	if config.Control, err = request.GetString(framework.ParamKeyMonitor); err != nil{
		err = fmt.Errorf("get control option fail: %s", err.Error())
		return
	}
	if config.USB, err = request.GetString(framework.ParamKeyDevice); err != nil{
		err = fmt.Errorf("get usb option fail: %s", err.Error())
		return
	}
	if config.Tablet, err = request.GetString(framework.ParamKeyInterface); err != nil{
		err = fmt.Errorf("get tablet option fail: %s", err.Error())
		return
	}

	var respChan = make(chan error, 1)
	executor.ResourceModule.ModifySystemTemplate(templateID, config, respChan)
	err = <- respChan

	resp, _ := framework.CreateJsonMessage(framework.ModifyTemplateResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	if err != nil{
		resp.SetError(err.Error())
		log.Printf("[%08X] handle modify system template from %s.[%08X] fail: %s",
			id, request.GetSender(), request.GetFromSession(), err.Error())
	}else{
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
