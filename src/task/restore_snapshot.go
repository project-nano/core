package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
	"time"
	"fmt"
)

type RestoreSnapshotExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *RestoreSnapshotExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	instanceID, err := request.GetString(framework.ParamKeyInstance)
	if err != nil{
		return err
	}
	snapshot, err := request.GetString(framework.ParamKeyName)
	if err != nil{
		return err
	}
	log.Printf("[%08X] request restore guest '%s' to snapshot '%s' from %s.[%08X]", id, instanceID, snapshot,
		request.GetSender(), request.GetFromSession())

	var ins modules.InstanceStatus
	resp, _ := framework.CreateJsonMessage(framework.RestoreSnapshotResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetSuccess(false)
	if err = QualifySnapshotName(snapshot); err != nil{
		log.Printf("[%08X] invalid snapshot name '%s' : %s", id, snapshot, err.Error())
		err = fmt.Errorf("invalid snapshot name '%s': %s", snapshot, err.Error())
		resp.SetError(err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
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
	{
		//forward request
		forward, _ := framework.CreateJsonMessage(framework.RestoreSnapshotRequest)
		forward.SetFromSession(id)
		forward.SetString(framework.ParamKeyInstance, instanceID)
		forward.SetString(framework.ParamKeyName, snapshot)
		if err = executor.Sender.SendMessage(forward, ins.Cell); err != nil{
			log.Printf("[%08X] forward restore snapshot to cell '%s' fail: %s", id, ins.Cell, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		timer := time.NewTimer(modules.DefaultOperateTimeout)
		select{
		case cellResp := <- incoming:
			if cellResp.IsSuccess(){
				log.Printf("[%08X] cell restore snapshot success", id)
			}else{
				log.Printf("[%08X] cell restore snapshot fail: %s", id, cellResp.GetError())
			}
			cellResp.SetFromSession(id)
			cellResp.SetToSession(request.GetFromSession())
			//forward
			return executor.Sender.SendMessage(cellResp, request.GetSender())
		case <- timer.C:
			//timeout
			log.Printf("[%08X] wait restore snapshot response timeout", id)
			resp.SetError("cell timeout")
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
	}
}
