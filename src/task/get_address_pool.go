package task

import (
	"log"
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
)

type GetAddressPoolExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}


func (executor *GetAddressPoolExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var poolName string
	if poolName, err = request.GetString(framework.ParamKeyAddress); err != nil{
		return
	}
	var respChan = make(chan modules.ResourceResult, 1)
	executor.ResourceModule.GetAddressPool(poolName, respChan)
	resp, _ := framework.CreateJsonMessage(framework.GetAddressPoolResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)

	var result = <- respChan
	if result.Error != nil{
		err = result.Error
		resp.SetError(err.Error())
		log.Printf("[%08X] get address pool from %s.[%08X] fail: %s",
			id, request.GetSender(), request.GetFromSession(), err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	var status = result.AddressPool
	var startArray, endArray, maskArray []string
	var capacityArray []uint64
	for _, addressRange := range status.Ranges{
		startArray = append(startArray, addressRange.Start)
		endArray = append(endArray, addressRange.End)
		maskArray = append(maskArray, addressRange.Netmask)
		capacityArray = append(capacityArray, uint64(addressRange.Capacity))
	}

	var addressArray, instanceArray []string
	for _, allocated := range status.Allocated{
		addressArray = append(addressArray, allocated.Address)
		instanceArray = append(instanceArray, allocated.Instance)
	}
	resp.SetSuccess(true)
	resp.SetString(framework.ParamKeyGateway, status.Gateway)
	resp.SetString(framework.ParamKeyMode, status.Provider)
	resp.SetStringArray(framework.ParamKeyServer, status.DNS)
	resp.SetStringArray(framework.ParamKeyStart, startArray)
	resp.SetStringArray(framework.ParamKeyEnd, endArray)
	resp.SetStringArray(framework.ParamKeyMask, maskArray)
	resp.SetUIntArray(framework.ParamKeyCount, capacityArray)
	resp.SetStringArray(framework.ParamKeyAddress, addressArray)
	resp.SetStringArray(framework.ParamKeyInstance, instanceArray)
	log.Printf("[%08X] reply status of address pool '%s' to %s.[%08X]",
		id, poolName, request.GetSender(), request.GetFromSession())
	return executor.Sender.SendMessage(resp, request.GetSender())
}