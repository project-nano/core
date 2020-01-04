package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
	"fmt"
	"time"
)

type StopInstanceExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *StopInstanceExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	instanceID, err := request.GetString(framework.ParamKeyInstance)
	if err != nil{
		return err
	}
	options, err := request.GetUIntArray(framework.ParamKeyOption)
	if err != nil{
		return err
	}
	const (
		ValidOptionCount = 2
	)
	if len(options) != ValidOptionCount{
		return fmt.Errorf("unexpected option count %d / %d", len(options), ValidOptionCount)
	}
	var reboot = 1 == options[0]
	var force = 1 == options[1]

	log.Printf("[%08X] request stop instance '%s' (reboot: %t, forece: %t ) from %s.[%08X]", id, instanceID,
		reboot, force, request.GetSender(), request.GetFromSession())

	var ins modules.InstanceStatus

	resp, _ := framework.CreateJsonMessage(framework.StopInstanceResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetTransactionID(request.GetTransactionID())
	resp.SetSuccess(false)
	{
		var respChan = make(chan modules.ResourceResult)
		executor.ResourceModule.GetInstanceStatus(instanceID, respChan)
		result := <- respChan
		if result.Error != nil{
			log.Printf("[%08X] fetch instance fail: %s", id, result.Error.Error())
			resp.SetError(result.Error.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		ins = result.InstanceStatus
		if !ins.Running{
			err = fmt.Errorf("instance '%s' already stopped", instanceID)
			log.Printf("[%08X] instance '%s' already stopped", id, instanceID)
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
	var fromSession = request.GetFromSession()
	{
		//request stop
		request.SetFromSession(id)
		if err = executor.Sender.SendMessage(request, ins.Cell); err != nil{
			log.Printf("[%08X] forward stop request to cell '%s' fail: %s", id, ins.Cell, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		timer := time.NewTimer(modules.DefaultOperateTimeout)
		select{
		case cellResp := <- incoming:
			if cellResp.IsSuccess(){
				log.Printf("[%08X] cell stop instance success", id)
			}else{
				log.Printf("[%08X] cell stop instance fail: %s", id, cellResp.GetError())
			}
			cellResp.SetFromSession(id)
			cellResp.SetToSession(fromSession)
			cellResp.SetTransactionID(request.GetTransactionID())
			//forward
			return executor.Sender.SendMessage(cellResp, request.GetSender())
		case <- timer.C:
			//timeout
			log.Printf("[%08X] wait stop response timeout", id)
			resp.SetError("cell timeout")
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
}