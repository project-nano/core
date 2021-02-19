package task

import (
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"log"
)

type SearchGuestsExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *SearchGuestsExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	var err error
	var condition modules.SearchGuestsCondition
	if condition.Pool, err = request.GetString(framework.ParamKeyPool); err != nil{
		return err
	}
	if condition.Cell, err = request.GetString(framework.ParamKeyCell); err != nil{
		return err
	}
	if condition.Keyword, err = request.GetString(framework.ParamKeyData); err != nil{
		return err
	}
	if condition.Limit, err = request.GetInt(framework.ParamKeyLimit); err != nil{
		return err
	}
	if condition.Offset, err = request.GetInt(framework.ParamKeyStart); err != nil{
		return err
	}

	resp, _ := framework.CreateJsonMessage(framework.SearchGuestResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	{
		var respChan = make(chan modules.ResourceResult, 1)
		executor.ResourceModule.SearchGuests(condition, respChan)
		result := <- respChan
		if result.Error != nil{
			err = result.Error
			log.Printf("[%08X] search guests fail: %s", id, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		var guests = result.InstanceList
		if err = modules.MarshalInstanceStatusListToMessage(guests, resp); err != nil{
			log.Printf("[%08X] build response message for search guests result fail: %s", id, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		var flags = []uint64{uint64(result.Total), uint64(result.Limit), uint64(result.Offset)}
		resp.SetUIntArray(framework.ParamKeyFlag, flags)
		//log.Printf("[%08X] %d guest(s) available", id, len(guests))
		resp.SetSuccess(true)
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
}
