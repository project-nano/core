package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
)

type ModifyAddressPoolExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}


func (executor *ModifyAddressPoolExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var config modules.AddressPoolConfig
	var poolName string
	if poolName, err = request.GetString(framework.ParamKeyAddress); err != nil{
		return
	}
	config.Name = poolName
	if config.Gateway, err = request.GetString(framework.ParamKeyGateway); err != nil{
		return
	}
	if config.DNS, err = request.GetStringArray(framework.ParamKeyServer); err != nil{
		return
	}
	var respChan = make(chan modules.ResourceResult, 1)
	executor.ResourceModule.ModifyAddressPool(config, respChan)
	resp, _ := framework.CreateJsonMessage(framework.ModifyAddressPoolResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)

	var result = <- respChan
	if result.Error != nil{
		err = result.Error
		resp.SetError(err.Error())
		log.Printf("[%08X] request modify address pool from %s.[%08X] fail: %s",
			id, request.GetSender(), request.GetFromSession(), err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	resp.SetSuccess(true)
	log.Printf("[%08X] address pool '%s' modified from %s.[%08X]",
		id, poolName, request.GetSender(), request.GetFromSession())
	if err = executor.Sender.SendMessage(resp, request.GetSender()); err != nil{
		log.Printf("[%08X] warning: send modify address pool response fail: %s", id, err.Error())
	}
	if 0 != len(result.ComputeCellInfoList) {
		notify, _ := framework.CreateJsonMessage(framework.AddressPoolChangedEvent)
		notify.SetString(framework.ParamKeyAddress, poolName)
		notify.SetString(framework.ParamKeyGateway, config.Gateway)
		notify.SetStringArray(framework.ParamKeyServer, config.DNS)
		notify.SetFromSession(id)
		for _, cell := range result.ComputeCellInfoList {
			if err = executor.Sender.SendMessage(notify, cell.Name); err != nil{
				log.Printf("[%08X] warning: notify address pool change to '%s' fail: %s", id, cell.Name, err.Error())
			}
		}
		log.Printf("[%08X] notified address pool changed to %d affected cell", id, len(result.ComputeCellInfoList))
	}
	return nil
}
