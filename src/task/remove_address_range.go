package task

import (
	"github.com/project-nano/framework"
	"log"
	"github.com/project-nano/core/modules"
)

type RemoveAddressRangeExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}


func (executor *RemoveAddressRangeExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var poolName, rangeType, startAddress string
	if poolName, err = request.GetString(framework.ParamKeyAddress); err != nil{
		return
	}
	if rangeType, err = request.GetString(framework.ParamKeyType); err != nil{
		return
	}
	if startAddress, err = request.GetString(framework.ParamKeyStart); err != nil{
		return
	}
	var respChan = make(chan error, 1)
	executor.ResourceModule.RemoveAddressRange(poolName, rangeType, startAddress, respChan)
	resp, _ := framework.CreateJsonMessage(framework.RemoveAddressRangeResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)

	err = <- respChan
	if err != nil{
		resp.SetError(err.Error())
		log.Printf("[%08X] request remove address range from %s.[%08X] fail: %s",
			id, request.GetSender(), request.GetFromSession(), err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	resp.SetSuccess(true)
	log.Printf("[%08X] range '%s' removed from pool '%s' by %s.[%08X]",
		id, startAddress,
		poolName, request.GetSender(), request.GetFromSession())
	return executor.Sender.SendMessage(resp, request.GetSender())
}
