package task

import (
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"log"
)

type QuerySystemTemplatesExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *QuerySystemTemplatesExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error){
	var respChan = make(chan modules.ResourceResult, 1)
	executor.ResourceModule.QuerySystemTemplates(respChan)
	var result = <- respChan

	resp, _ := framework.CreateJsonMessage(framework.QueryTemplateResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	if result.Error != nil{
		err = result.Error
		resp.SetError(err.Error())
		log.Printf("[%08X] handle query system templates from %s.[%08X] fail: %s",
			id, request.GetSender(), request.GetFromSession(), err.Error())
	}else{
		var idList, nameList, osList, createList, modifiedList []string
		for _, t := range result.TemplateList {
			idList = append(idList, t.ID)
			nameList = append(nameList, t.Name)
			osList = append(osList, t.OperatingSystem)
			createList = append(createList, t.CreatedTime)
			modifiedList = append(modifiedList, t.ModifiedTime)
		}
		resp.SetSuccess(true)
		resp.SetStringArray(framework.ParamKeyID, idList)
		resp.SetStringArray(framework.ParamKeyName, nameList)
		resp.SetStringArray(framework.ParamKeySystem, osList)
		resp.SetStringArray(framework.ParamKeyCreate, createList)
		resp.SetStringArray(framework.ParamKeyModify, modifiedList)
	}

	return executor.Sender.SendMessage(resp, request.GetSender())
}
