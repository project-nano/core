package task

import (
	"errors"
	"fmt"
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"log"
	"time"
)

type QueryStoragePathsExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *QueryStoragePathsExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var targetCell string
	if targetCell, err = request.GetString(framework.ParamKeyCell); err != nil {
		err = fmt.Errorf("get target cell fail: %s", err.Error())
		return
	}
	resp, _ := framework.CreateJsonMessage(framework.QueryCellStorageResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetSuccess(false)

	var fromSession = request.GetFromSession()
	{
		//redirect request
		request.SetFromSession(id)
		if err = executor.Sender.SendMessage(request, targetCell); err != nil{
			log.Printf("[%08X] redirect query storage request to cell '%s' fail: %s", id, targetCell, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		timer := time.NewTimer(modules.DefaultOperateTimeout)
		select{
		case cellResp := <- incoming:
			if !cellResp.IsSuccess(){
				err = errors.New(cellResp.GetError())
				log.Printf("[%08X] cell query storage paths fail: %s", id, cellResp.GetError())
			}else{
				cellResp.SetSuccess(true)
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