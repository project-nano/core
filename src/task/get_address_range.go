package task

import (
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"log"
)

type GetAddressRangeExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}


func (executor *GetAddressRangeExecutor)Execute(id framework.SessionID, request framework.Message,
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
	var respChan = make(chan modules.ResourceResult, 1)
	executor.ResourceModule.GetAddressRange(poolName, rangeType, startAddress, respChan)
	resp, _ := framework.CreateJsonMessage(framework.GetAddressRangeResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)

	var result = <- respChan
	if result.Error != nil{
		err = result.Error
		resp.SetError(err.Error())
		log.Printf("[%08X] request get address range from %s.[%08X] fail: %s",
			id, request.GetSender(), request.GetFromSession(), err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	var status = result.AddressRangeStatus

	var addressArray, instanceArray []string
	for _, allocated := range status.Allocated{
		addressArray = append(addressArray, allocated.Address)
		instanceArray = append(instanceArray, allocated.Instance)
	}

	resp.SetSuccess(true)
	resp.SetString(framework.ParamKeyStart, status.Start)
	resp.SetString(framework.ParamKeyEnd, status.End)
	resp.SetString(framework.ParamKeyMask, status.Netmask)
	resp.SetUInt(framework.ParamKeyCount, uint(status.Capacity))
	resp.SetStringArray(framework.ParamKeyAddress, addressArray)
	resp.SetStringArray(framework.ParamKeyInstance, instanceArray)
	log.Printf("[%08X] reply status of address range '%s' to %s.[%08X]",
		id, startAddress, request.GetSender(), request.GetFromSession())
	return executor.Sender.SendMessage(resp, request.GetSender())
}
