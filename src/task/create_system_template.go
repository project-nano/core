package task

import (
	"fmt"
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"log"
)

type CreateSystemTemplateExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *CreateSystemTemplateExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error){
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

	var respChan = make(chan modules.ResourceResult, 1)
	executor.ResourceModule.CreateSystemTemplate(config, respChan)
	var result = <- respChan

	resp, _ := framework.CreateJsonMessage(framework.ModifyTemplateResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	if result.Error != nil{
		err = result.Error
		resp.SetError(err.Error())
		log.Printf("[%08X] handle create system template from %s.[%08X] fail: %s",
			id, request.GetSender(), request.GetFromSession(), err.Error())
	}else{
		var t = result.Template
		resp.SetSuccess(true)
		resp.SetString(framework.ParamKeyID, t.ID)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
