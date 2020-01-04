package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
	"time"
	"fmt"
)

type GetInstanceStatusExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *GetInstanceStatusExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	instanceID, err := request.GetString(framework.ParamKeyInstance)
	if err != nil {
		return err
	}

	//log.Printf("[%08X] request get instance '%s' status from %s.[%08X]", id, instanceID,
	//	request.GetSender(), request.GetFromSession())

	var ins modules.InstanceStatus

	resp, _ := framework.CreateJsonMessage(framework.GetInstanceStatusResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
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
	}
	var fromSession = request.GetFromSession()
	{
		//redirect request
		request.SetFromSession(id)
		if err = executor.Sender.SendMessage(request, ins.Cell); err != nil{
			log.Printf("[%08X] redirect query request to cell '%s' fail: %s", id, ins.Cell, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		timer := time.NewTimer(modules.DefaultOperateTimeout)
		select{
		case cellResp := <- incoming:
			if cellResp.IsSuccess(){
				//log.Printf("[%08X] cell get instance status success", id)
				//modify network info
				var internalMonitor = fmt.Sprintf("%s:%d", ins.InternalNetwork.MonitorAddress, ins.InternalNetwork.MonitorPort)
				var externalMonitor = fmt.Sprintf("%s:%d", ins.ExternalNetwork.MonitorAddress, ins.ExternalNetwork.MonitorPort)
				cellResp.SetStringArray(framework.ParamKeyMonitor, []string{internalMonitor,externalMonitor})
				cellResp.SetStringArray(framework.ParamKeyAddress, []string{ins.InternalNetwork.InstanceAddress, ins.ExternalNetwork.InstanceAddress})

			}else{
				log.Printf("[%08X] cell get instance status  fail: %s", id, cellResp.GetError())
			}
			cellResp.SetFromSession(id)
			cellResp.SetToSession(fromSession)
			//forward
			return executor.Sender.SendMessage(cellResp, request.GetSender())
		case <- timer.C:
			//timeout
			log.Printf("[%08X] wait query response timeout", id)
			resp.SetError("cell timeout")
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
}