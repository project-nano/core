package task

import (
	"github.com/project-nano/framework"
	"log"
	"time"
	"github.com/project-nano/core/modules"
	"fmt"
)

type InsertMediaExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *InsertMediaExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	instanceID, err := request.GetString(framework.ParamKeyInstance)
	if err != nil{
		return err
	}
	mediaSource, err := request.GetString(framework.ParamKeyMedia)
	if err != nil{
		return err
	}

	log.Printf("[%08X] request insert media '%s' into guest '%s' from %s.[%08X]", id, mediaSource, instanceID,
		request.GetSender(), request.GetFromSession())

	var ins modules.InstanceStatus
	resp, _ := framework.CreateJsonMessage(framework.InsertMediaResponse)
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
		ins = result.Instance
		if !ins.Running{
			err = fmt.Errorf("instance '%s' is stopped", instanceID)
			log.Printf("[%08X] instance '%s' is stopped", id, instanceID)
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
	{
		//forward request
		forward, _ := framework.CreateJsonMessage(framework.InsertMediaRequest)
		forward.SetFromSession(id)
		forward.SetString(framework.ParamKeyInstance, instanceID)
		forward.SetString(framework.ParamKeyMedia, mediaSource)

		//todo: get media name for display
		var respChan = make(chan modules.ResourceResult)
		executor.ResourceModule.GetImageServer(respChan)
		var result = <-respChan
		if result.Error != nil {
			errMsg := result.Error.Error()
			log.Printf("[%08X] get image server fail: %s", id, errMsg)
			resp.SetError(errMsg)
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		log.Printf("[%08X] select image server '%s:%d' for image '%s'", id, result.Host, result.Port, mediaSource)
		forward.SetString(framework.ParamKeyHost, result.Host)
		forward.SetUInt(framework.ParamKeyPort, uint(result.Port))

		if err = executor.Sender.SendMessage(forward, ins.Cell); err != nil{
			log.Printf("[%08X] forward insert media to cell '%s' fail: %s", id, ins.Cell, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		timer := time.NewTimer(modules.DefaultOperateTimeout)
		select{
		case cellResp := <- incoming:
			if cellResp.IsSuccess(){
				log.Printf("[%08X] cell insert media success", id)
			}else{
				log.Printf("[%08X] cell insert media fail: %s", id, cellResp.GetError())
			}
			cellResp.SetFromSession(id)
			cellResp.SetToSession(request.GetFromSession())
			//forward
			return executor.Sender.SendMessage(cellResp, request.GetSender())
		case <- timer.C:
			//timeout
			log.Printf("[%08X] wait insert response timeout", id)
			resp.SetError("cell timeout")
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
}
