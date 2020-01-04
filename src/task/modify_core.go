package task

import (
	"github.com/project-nano/framework"
	"log"
	"time"
	"github.com/project-nano/core/modules"
	"errors"
)

type ModifyGuestCoreExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *ModifyGuestCoreExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	guestID, err := request.GetString(framework.ParamKeyGuest)
	if err != nil{
		return err
	}
	cores, err := request.GetUInt(framework.ParamKeyCore)
	if err != nil{
		return err
	}
	log.Printf("[%08X] request modifying cores of '%s' from %s.[%08X]", id, guestID,
		request.GetSender(), request.GetFromSession())
	var ins modules.InstanceStatus
	resp, _ := framework.CreateJsonMessage(framework.ModifyCoreResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetSuccess(false)
	{
		var respChan = make(chan modules.ResourceResult)
		executor.ResourceModule.GetInstanceStatus(guestID, respChan)
		result := <- respChan
		if result.Error != nil{
			log.Printf("[%08X] fetch instance fail: %s", id, result.Error.Error())
			resp.SetError(result.Error.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		ins = result.InstanceStatus
		if ins.Cores == cores{
			err = errors.New("no need to modify")
			log.Printf("[%08X] %s", id, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
	{
		//request modify core
		forward, _ := framework.CreateJsonMessage(framework.ModifyCoreRequest)
		forward.SetFromSession(id)
		forward.SetString(framework.ParamKeyGuest, guestID)
		forward.SetUInt(framework.ParamKeyCore, cores)
		if err = executor.Sender.SendMessage(forward, ins.Cell); err != nil{
			log.Printf("[%08X] forward modify core to cell '%s' fail: %s", id, ins.Cell, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		timer := time.NewTimer(modules.DefaultOperateTimeout)
		select{
		case cellResp := <- incoming:
			if cellResp.IsSuccess(){
				ins.Cores = cores
				//update
				var respChan = make(chan error)
				executor.ResourceModule.UpdateInstanceStatus(ins, respChan)
				err = <- respChan
				if err != nil{
					log.Printf("[%08X] update new cores fail: %s", id, err.Error())
					resp.SetError(err.Error())
					return executor.Sender.SendMessage(resp, request.GetSender())
				}
				log.Printf("[%08X] modify core success", id)
			}else{
				log.Printf("[%08X] modify core fail: %s", id, cellResp.GetError())
			}
			cellResp.SetFromSession(id)
			cellResp.SetToSession(request.GetFromSession())
			//forward
			return executor.Sender.SendMessage(cellResp, request.GetSender())
		case <- timer.C:
			//timeout
			log.Printf("[%08X] wait modify core response timeout", id)
			resp.SetError("request timeout")
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
}
