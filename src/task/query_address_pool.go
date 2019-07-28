package task

import (
	"github.com/project-nano/framework"
	"modules"
	"log"
)

type QueryAddressPoolExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}


func (executor *QueryAddressPoolExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var respChan = make(chan modules.ResourceResult, 1)
	executor.ResourceModule.QueryAddressPool(respChan)
	resp, _ := framework.CreateJsonMessage(framework.QueryAddressPoolResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)

	var result = <- respChan
	if result.Error != nil{
		err = result.Error
		resp.SetError(err.Error())
		log.Printf("[%08X] query address pool from %s.[%08X] fail: %s",
			id, request.GetSender(), request.GetFromSession(), err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	var nameArray, gatewayArray []string
	var addressArray, allocateArray []uint64
	for _, pool := range result.AddressPoolStatusList{
		nameArray = append(nameArray, pool.Name)
		gatewayArray = append(gatewayArray, pool.Gateway)
		var addressCount uint32 = 0
		allocateArray = append(allocateArray, uint64(len(pool.Allocated)))
		for _, addressRange := range pool.Ranges{
			addressCount += addressRange.Capacity
		}
		addressArray = append(addressArray, uint64(addressCount))
	}
	resp.SetSuccess(true)
	resp.SetStringArray(framework.ParamKeyName, nameArray)
	resp.SetStringArray(framework.ParamKeyGateway, gatewayArray)
	resp.SetUIntArray(framework.ParamKeyAddress, addressArray)
	resp.SetUIntArray(framework.ParamKeyAllocate, allocateArray)
	log.Printf("[%08X] reply %d address pool(s) to %s.[%08X]",
		id, len(result.AddressPoolStatusList), request.GetSender(), request.GetFromSession())
	return executor.Sender.SendMessage(resp, request.GetSender())
}

