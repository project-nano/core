package task

import (
	"github.com/project-nano/framework"
	"log"
	"github.com/project-nano/core/modules"
)

type QueryInstanceStatusExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *QueryInstanceStatusExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	poolName, err := request.GetString(framework.ParamKeyPool)
	if err != nil{
		return err
	}
	var inCell = false
	cellName, err := request.GetString(framework.ParamKeyCell)
	if err == nil{
		inCell = true
	}
	var respChan = make(chan modules.ResourceResult)
	if inCell{
		//log.Printf("[%08X] request query instance status in cell '%s' from %s.[%08X]", id, cellName,
		//	request.GetSender(), request.GetFromSession())
		executor.ResourceModule.QueryInstanceStatusInCell(poolName, cellName, respChan)
	}else{
		//log.Printf("[%08X] request query instance status in pool '%s' from %s.[%08X]", id, poolName,
		//	request.GetSender(), request.GetFromSession())
		executor.ResourceModule.QueryInstanceStatusInPool(poolName, respChan)
	}
	result := <- respChan

	resp, _ := framework.CreateJsonMessage(framework.QueryInstanceStatusResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetSuccess(false)
	if result.Error != nil{
		err = result.Error
		log.Printf("[%08X] query instance status fail: %s", id, err.Error())
		resp.SetError(err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}

	var instances = result.InstanceList
	modules.MarshalInstanceStatusListToMessage(instances, resp)
	resp.SetSuccess(true)
	//log.Printf("[%08X] %d instance(s) available", id, len(instances))
	return executor.Sender.SendMessage(resp, request.GetSender())
}
