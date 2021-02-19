package task

import (
	"fmt"
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"log"
	"time"
)

type ModifyGuestAutoStartExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *ModifyGuestAutoStartExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var guestID string
	var enable bool
	if guestID, err = request.GetString(framework.ParamKeyGuest); err != nil{
		err = fmt.Errorf("get guest ID fail: %s", err.Error())
		return
	}
	if enable, err = request.GetBoolean(framework.ParamKeyEnable); err != nil{
		err = fmt.Errorf("get enable option fail: %s", err.Error())
		return
	}

	resp, _ := framework.CreateJsonMessage(framework.ModifyAutoStartResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetSuccess(false)

	var ins modules.InstanceStatus
	{
		var respChan = make(chan modules.ResourceResult, 1)
		executor.ResourceModule.GetInstanceStatus(guestID, respChan)
		result := <- respChan
		if result.Error != nil{
			log.Printf("[%08X] fetch instance fail: %s", id, result.Error.Error())
			resp.SetError(result.Error.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		ins = result.Instance
		if ins.AutoStart == enable{
			if enable{
				err = fmt.Errorf("auto start of guest '%s' already enabled", ins.Name)
			}else{
				err = fmt.Errorf("auto start of guest '%s' already disabled", ins.Name)
			}
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
	{
		//forward request
		forward, _ := framework.CreateJsonMessage(framework.ModifyAutoStartRequest)
		forward.SetFromSession(id)
		forward.SetString(framework.ParamKeyGuest, guestID)
		forward.SetBoolean(framework.ParamKeyEnable, enable)
		if err = executor.Sender.SendMessage(forward, ins.Cell); err != nil{
			log.Printf("[%08X] forward modify auto start to cell '%s' fail: %s", id, ins.Cell, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		timer := time.NewTimer(modules.DefaultOperateTimeout)
		select{
		case cellResp := <- incoming:
			if cellResp.IsSuccess(){
				//update
				var respChan = make(chan error, 1)
				executor.ResourceModule.UpdateGuestAutoStart(guestID, enable, respChan)
				err = <- respChan
				if err != nil{
					log.Printf("[%08X] update auto start status of guest '%s' fail: %s", id, ins.Name, err.Error())
					resp.SetError(err.Error())
					return executor.Sender.SendMessage(resp, request.GetSender())
				}
				if enable{
					log.Printf("[%08X] guest '%s' enable auto start", id, guestID)
				}else{
					log.Printf("[%08X] guest '%s' disable auto start", id, guestID)
				}

			}else{
				log.Printf("[%08X] modify auto start fail: %s", id, cellResp.GetError())
			}
			cellResp.SetFromSession(id)
			cellResp.SetToSession(request.GetFromSession())
			//forward
			return executor.Sender.SendMessage(cellResp, request.GetSender())
		case <- timer.C:
			//timeout
			log.Printf("[%08X] wait modify auto start response timeout", id)
			resp.SetError("request timeout")
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
}
