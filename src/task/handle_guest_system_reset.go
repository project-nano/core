package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
	"errors"
)

type HandleGuestSystemResetExecutor struct {
	ResourceModule modules.ResourceModule
}

func (executor *HandleGuestSystemResetExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var guestID string
	if guestID, err = request.GetString(framework.ParamKeyGuest); err != nil{
		return
	}
	var respChan = make(chan error, 1)
	if !request.IsSuccess(){
		err = errors.New(request.GetError())
	}
	executor.ResourceModule.FinishResetSystem(guestID, err, respChan)
	err = <- respChan
	if err != nil{
		log.Printf("[%08X] recv guest reset finish, but update fail: %s", id, err.Error())
	}else{
		log.Printf("[%08X] reset system of guest '%s' finished", id, guestID)
	}
	return nil
}
