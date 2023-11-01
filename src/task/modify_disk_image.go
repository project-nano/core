package task

import (
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"log"
	"time"
)

type ModifyDiskImageExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *ModifyDiskImageExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {

	var originSession = request.GetFromSession()
	var respChan = make(chan modules.ResourceResult, 1)
	executor.ResourceModule.GetImageServer(respChan)
	var result = <-respChan
	resp, _ := framework.CreateJsonMessage(framework.ModifyDiskImageResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	if result.Error != nil {
		err := result.Error
		log.Printf("[%08X] get image server fail: %s", id, err.Error())
		resp.SetError(err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}

	//forward to image server
	request.SetFromSession(id)
	request.SetToSession(0)
	var imageServer = result.Name

	if err = executor.Sender.SendMessage(request, imageServer); err != nil {
		log.Printf("[%08X] forward modify disk to image server fail: %s", id, err.Error())
		resp.SetError(err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	//wait response
	timer := time.NewTimer(modules.GetConfigurator().GetOperateTimeout())
	select {
	case forwardResp := <-incoming:
		if !forwardResp.IsSuccess() {
			log.Printf("[%08X] modify disk image fail: %s", id, forwardResp.GetError())
		}
		forwardResp.SetFromSession(id)
		forwardResp.SetToSession(originSession)
		forwardResp.SetTransactionID(request.GetTransactionID())
		//forward
		return executor.Sender.SendMessage(forwardResp, request.GetSender())

	case <-timer.C:
		//timeout
		log.Printf("[%08X] modify disk image timeout", id)
		resp.SetError("time out")
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
}
