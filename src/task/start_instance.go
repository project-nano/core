package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
	"fmt"
	"time"
)

type StartInstanceExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *StartInstanceExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	instanceID, err := request.GetString(framework.ParamKeyInstance)
	if err != nil {
		return err
	}
	mediaOption, err := request.GetUInt(framework.ParamKeyOption)
	if err != nil{
		return err
	}
	log.Printf("[%08X] request start instance '%s' from %s.[%08X]", id, instanceID,
		request.GetSender(), request.GetFromSession())

	resp, _ := framework.CreateJsonMessage(framework.StartInstanceResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetSuccess(false)

	var ins modules.InstanceStatus
	{
		//check status first
		var respChan = make(chan modules.ResourceResult)
		executor.ResourceModule.GetInstanceStatus(instanceID, respChan)
		result := <- respChan
		if result.Error != nil{
			errMsg := result.Error.Error()
			log.Printf("[%08X] get instance fail: %s", id, errMsg)
			resp.SetError(errMsg)
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		ins = result.InstanceStatus
		if ins.Running{
			errMsg := fmt.Sprintf("instance '%s' already started", instanceID)
			log.Printf("[%08X] start instance fail: %s", id, errMsg)
			resp.SetError(errMsg)
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
	var fromSession = request.GetFromSession()
	{
		forward, _ := framework.CreateJsonMessage(framework.StartInstanceRequest)
		forward.SetFromSession(id)
		forward.SetString(framework.ParamKeyInstance, instanceID)
		forward.SetUInt(framework.ParamKeyOption, mediaOption)
		{

			switch mediaOption {
			case modules.InstanceMediaOptionNone:
				break
			case modules.InstanceMediaOptionImage:
				mediaSource, err := request.GetString(framework.ParamKeySource)
				if err != nil{
					return err
				}
				{
					//todo: get media name for display
					var respChan = make(chan modules.ResourceResult)
					executor.ResourceModule.GetImageServer(respChan)
					var result = <- respChan
					if result.Error != nil{
						errMsg := result.Error.Error()
						log.Printf("[%08X] get image server fail: %s", id, errMsg)
						resp.SetError(errMsg)
						return executor.Sender.SendMessage(resp, request.GetSender())
					}
					log.Printf("[%08X] select image server '%s:%d' for image '%s'", id, result.Host, result.Port, mediaSource)
					forward.SetString(framework.ParamKeySource, mediaSource)
					forward.SetString(framework.ParamKeyHost, result.Host)
					forward.SetUInt(framework.ParamKeyPort, uint(result.Port))
				}
			default:
				return fmt.Errorf("unsupported media option %d", mediaOption)
			}
		}
		if err = executor.Sender.SendMessage(forward, ins.Cell);err != nil{
			log.Printf("[%08X] forward start request to cell '%s' fail: %s", id, ins.Cell, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		timer := time.NewTimer(modules.DefaultOperateTimeout)
		select{
		case cellResp := <- incoming:
			if cellResp.IsSuccess(){
				log.Printf("[%08X] cell start instance success", id)
			}else{
				log.Printf("[%08X] cell start instance fail: %s", id, cellResp.GetError())
			}
			cellResp.SetFromSession(id)
			cellResp.SetToSession(fromSession)
			//forward
			return executor.Sender.SendMessage(cellResp, request.GetSender())
		case <- timer.C:
			//timeout
			log.Printf("[%08X] wait start response timeout", id)
			resp.SetError("cell timeout")
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
}