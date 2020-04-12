package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"fmt"
	"log"
)

type QueryGuestConfigExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *QueryGuestConfigExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	var err error
	var condition modules.GuestQueryCondition
	if condition.Pool, err = request.GetString(framework.ParamKeyPool); err != nil{
		return err
	}
	options, err := request.GetUIntArray(framework.ParamKeyOption)
	if err != nil{
		return err
	}
	const (
		CellOption = iota
		OwnerOption
		GroupOption
		StatusOption
		CreatedOption
		ValidOptionCount = 5
	)
	if ValidOptionCount != len(options){
		return fmt.Errorf("unexpected options params count %d / %d", len(options), ValidOptionCount)
	}
	if 1 == options[CellOption]{
		condition.InCell = true
		if condition.Cell, err = request.GetString(framework.ParamKeyCell);err != nil{
			return err
		}
	}
	if 1 == options[OwnerOption]{
		condition.WithOwner = true
		if condition.Owner, err = request.GetString(framework.ParamKeyUser);err != nil{
			return err
		}
	}
	if 1 == options[GroupOption]{
		condition.WithGroup = true
		if condition.Group, err = request.GetString(framework.ParamKeyGroup);err != nil{
			return err
		}
	}
	if 1 == options[StatusOption]{
		condition.WithStatus = true
		if status, err := request.GetUInt(framework.ParamKeyStatus);err != nil{
			return err
		}else{
			condition.Status = int(status)
		}
	}
	if 1 == options[CreatedOption]{
		condition.WithCreateFlag = true
		if condition.Created, err = request.GetBoolean(framework.ParamKeyEnable);err != nil{
			return err
		}
	}
	//log.Printf("[%08X] recv query guest requet from %s.[%08X]", id, request.GetSender(), request.GetFromSession())

	resp, _ := framework.CreateJsonMessage(framework.QueryGuestResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	{
		var respChan = make(chan modules.ResourceResult)
		executor.ResourceModule.SearchGuestConfig(condition, respChan)
		result := <- respChan
		if result.Error != nil{
			err = result.Error
			log.Printf("[%08X] search guest config fail: %s", id, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		var guests = result.InstanceList
		if err = modules.MarshalInstanceStatusListToMessage(guests, resp); err != nil{
			log.Printf("[%08X] build response message fail: %s", id, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		//log.Printf("[%08X] %d guest(s) available", id, len(guests))
		resp.SetSuccess(true)
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
}
