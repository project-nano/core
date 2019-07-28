package task

import (
	"github.com/project-nano/framework"
	"modules"
	"log"
	"time"
	"fmt"
)

type ModifyGuestDiskThresholdExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *ModifyGuestDiskThresholdExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var guestID string
	guestID, err = request.GetString(framework.ParamKeyGuest)
	if err != nil{
		return err
	}
	limitParameters, err := request.GetUIntArray(framework.ParamKeyLimit)
	if err != nil {
		return err
	}
	const (
		ReadSpeedOffset           = iota
		WriteSpeedOffset
		ReadIOPSOffset
		WriteIOPSOffset
		ValidLimitParametersCount = 4
	)

	if ValidLimitParametersCount != len(limitParameters){
		var err = fmt.Errorf("invalid QoS parameters count %d", len(limitParameters))
		return err
	}
	var readSpeed = limitParameters[ReadSpeedOffset]
	var writeSpeed = limitParameters[WriteSpeedOffset]
	var readIOPS = limitParameters[ReadIOPSOffset]
	var writeIOPS = limitParameters[WriteIOPSOffset]

	log.Printf("[%08X] request modifying disk threshold of guest '%s' from %s.[%08X]", id, guestID,
		request.GetSender(), request.GetFromSession())

	resp, _ := framework.CreateJsonMessage(framework.ModifyDiskThresholdResponse)
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
		ins = result.InstanceStatus
		if ins.Running{
			resp.SetError(fmt.Sprintf("instance %s ('%s') is still running", ins.Name, guestID))
			log.Printf("[%08X] instance %s ('%s') is still running", id, ins.Name, guestID)
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
	{
		//forward request
		forward, _ := framework.CreateJsonMessage(framework.ModifyDiskThresholdRequest)
		forward.SetFromSession(id)
		forward.SetString(framework.ParamKeyGuest, guestID)
		forward.SetUIntArray(framework.ParamKeyLimit, []uint64{readSpeed, writeSpeed, readIOPS, writeIOPS})
		if err = executor.Sender.SendMessage(forward, ins.Cell); err != nil{
			log.Printf("[%08X] forward modify disk threshold to cell '%s' fail: %s", id, ins.Cell, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		timer := time.NewTimer(modules.DefaultOperateTimeout)
		select{
		case cellResp := <- incoming:
			if cellResp.IsSuccess(){
				//update
				var respChan = make(chan error, 1)
				executor.ResourceModule.UpdateInstanceDiskThreshold(guestID, readSpeed, readIOPS, writeSpeed, writeIOPS, respChan)
				err = <- respChan
				if err != nil{
					log.Printf("[%08X] update disk threshold fail: %s", id, err.Error())
					resp.SetError(err.Error())
					return executor.Sender.SendMessage(resp, request.GetSender())
				}
				log.Printf("[%08X] modify disk threshold success", id)
			}else{
				log.Printf("[%08X] modify disk threshold fail: %s", id, cellResp.GetError())
			}
			cellResp.SetFromSession(id)
			cellResp.SetToSession(request.GetFromSession())
			//forward
			return executor.Sender.SendMessage(cellResp, request.GetSender())
		case <- timer.C:
			//timeout
			log.Printf("[%08X] wait modify disk threshold response timeout", id)
			resp.SetError("request timeout")
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
}