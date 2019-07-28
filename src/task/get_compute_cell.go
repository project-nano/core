package task

import (
	"github.com/project-nano/framework"
	"modules"
	"log"
	"fmt"
	"time"
	"github.com/pkg/errors"
)

type GetComputeCellExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *GetComputeCellExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	poolName, err := request.GetString(framework.ParamKeyPool)
	if err != nil{
		return err
	}
	cellName, err := request.GetString(framework.ParamKeyCell)
	if err != nil{
		return err
	}

	log.Printf("[%08X] get compute cell '%s.%s' from %s.[%08X]", id, poolName, cellName,
		request.GetSender(), request.GetFromSession())

	var respChan= make(chan modules.ResourceResult)

	executor.ResourceModule.GetComputeCellStatus(poolName, cellName, respChan)
	result := <-respChan

	resp, _ := framework.CreateJsonMessage(framework.GetComputePoolCellResponse)
	resp.SetFromSession(id)
	resp.SetSuccess(false)
	resp.SetToSession(request.GetFromSession())
	if result.Error != nil{
		err = result.Error
		resp.SetError(err.Error())
		log.Printf("[%08X] get compute cell fail: %s", id, err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	var s = result.ComputeCellStatus


	//assemble
	resp.SetString(framework.ParamKeyName, s.Name)
	resp.SetString(framework.ParamKeyAddress, s.Address)
	resp.SetBoolean(framework.ParamKeyEnable, s.Enabled)
	resp.SetBoolean(framework.ParamKeyStatus, s.Alive)
	if !s.Alive || !s.Enabled{
		resp.SetSuccess(true)
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	forward, _ := framework.CreateJsonMessage(framework.GetComputePoolCellRequest)
	forward.SetFromSession(id)
	forward.SetString(framework.ParamKeyCell, cellName)
	if err = executor.Sender.SendMessage(forward, cellName);err != nil{
		log.Printf("[%08X] forward to cell '%s' fail: %s", id, cellName, err.Error())
		err = fmt.Errorf("forward to cell '%s' fail: %s", cellName, err.Error())
		resp.SetError(err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	var timer = time.NewTimer(modules.DefaultOperateTimeout)
	select {
	case <- timer.C:
		log.Printf("[%08X] wait cell status timeout", id)
		err = errors.New("wait cell status timeout")
		resp.SetError(err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	case cellResp := <- incoming:
		if !cellResp.IsSuccess(){
			err = errors.New(cellResp.GetError())
			log.Printf("[%08X] get remote cell status fail: %s", id, err)
			resp.SetError(err.Error())
		}else{
			//success
			var storage, errors []string
			var attached []uint64
			storage, _ = cellResp.GetStringArray(framework.ParamKeyStorage)
			errors, _ = cellResp.GetStringArray(framework.ParamKeyError)
			attached, _ = cellResp.GetUIntArray(framework.ParamKeyAttach)
			resp.SetStringArray(framework.ParamKeyStorage, storage)
			resp.SetStringArray(framework.ParamKeyError, errors)
			resp.SetUIntArray(framework.ParamKeyAttach, attached)
			resp.SetSuccess(true)
		}
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
}
