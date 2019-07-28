package task

import (
	"github.com/project-nano/framework"
	"modules"
	"log"
)

type GetBatchStopGuestExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *GetBatchStopGuestExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var batchID string
	if batchID, err = request.GetString(framework.ParamKeyID);err != nil{
		return err
	}
	resp, _ := framework.CreateJsonMessage(framework.GetBatchStopGuestResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)

	var respChan = make(chan modules.ResourceResult, 1)
	executor.ResourceModule.GetBatchStopGuestStatus(batchID, respChan)
	var result = <- respChan
	if result.Error != nil{
		err = result.Error
		log.Printf("[%08X] get batch stop status from %s.[%08X] fail: %s", id, request.GetSender(), request.GetFromSession(), err.Error())
		resp.SetError(err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}

	var guestStatus []uint64
	var guestID, guestName, stopError []string

	for _, status := range result.BatchStop{
		guestStatus = append(guestStatus, uint64(status.Status))
		guestID = append(guestID, status.ID)
		guestName = append(guestName, status.Name)
		stopError = append(stopError, status.Error)
	}
	resp.SetSuccess(true)
	resp.SetStringArray(framework.ParamKeyName, guestName)
	resp.SetStringArray(framework.ParamKeyGuest, guestID)
	resp.SetStringArray(framework.ParamKeyError, stopError)
	resp.SetUIntArray(framework.ParamKeyStatus, guestStatus)
	return executor.Sender.SendMessage(resp, request.GetSender())
}

