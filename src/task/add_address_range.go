package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
)

type AddAddressRangeExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}


func (executor *AddAddressRangeExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var poolName, rangeType string
	if poolName, err = request.GetString(framework.ParamKeyAddress); err != nil{
		return
	}
	if rangeType, err = request.GetString(framework.ParamKeyType); err != nil{
		return
	}
	var config modules.AddressRangeConfig
	if config.Start, err = request.GetString(framework.ParamKeyStart); err != nil{
		return
	}
	if config.End, err = request.GetString(framework.ParamKeyEnd); err != nil{
		return
	}
	if config.Netmask, err = request.GetString(framework.ParamKeyMask); err != nil{
		return
	}
	var respChan = make(chan error, 1)
	executor.ResourceModule.AddAddressRange(poolName, rangeType, config, respChan)
	resp, _ := framework.CreateJsonMessage(framework.AddAddressRangeResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)

	err = <- respChan
	if err != nil{
		resp.SetError(err.Error())
		log.Printf("[%08X] request add address range from %s.[%08X] fail: %s",
			id, request.GetSender(), request.GetFromSession(), err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	resp.SetSuccess(true)
	log.Printf("[%08X] range '%s ~ %s/%s' added to pool '%s' from %s.[%08X]",
		id, config.Start, config.End, config.Netmask,
		poolName, request.GetSender(), request.GetFromSession())
	return executor.Sender.SendMessage(resp, request.GetSender())
}
