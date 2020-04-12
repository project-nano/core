package task

import (
	"log"
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
)

type QueryAddressRangeExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}


func (executor *QueryAddressRangeExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var poolName, rangeType string
	if poolName, err = request.GetString(framework.ParamKeyAddress); err != nil{
		return
	}
	if rangeType, err = request.GetString(framework.ParamKeyType); err != nil{
		return
	}
	var respChan = make(chan modules.ResourceResult, 1)
	executor.ResourceModule.QueryAddressRange(poolName, rangeType, respChan)
	resp, _ := framework.CreateJsonMessage(framework.QueryAddressRangeResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)

	var result = <- respChan
	if result.Error != nil{
		err = result.Error
		resp.SetError(err.Error())
		log.Printf("[%08X] query address range from %s.[%08X] fail: %s",
			id, request.GetSender(), request.GetFromSession(), err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	var startArray, endArray, maskArray []string
	for _, status := range result.AddressRangeList {
		startArray = append(startArray, status.Start)
		endArray = append(endArray, status.End)
		maskArray = append(maskArray, status.Netmask)
	}
	resp.SetSuccess(true)
	resp.SetStringArray(framework.ParamKeyStart, startArray)
	resp.SetStringArray(framework.ParamKeyEnd, endArray)
	resp.SetStringArray(framework.ParamKeyMask, maskArray)
	log.Printf("[%08X] reply %d address range(s) to %s.[%08X]",
		id, len(result.AddressRangeList), request.GetSender(), request.GetFromSession())
	return executor.Sender.SendMessage(resp, request.GetSender())
}