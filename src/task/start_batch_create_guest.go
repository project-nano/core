package task

import (
	"fmt"
	"log"
	"time"
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"errors"
)

type StartBatchCreateGuestExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}


func (executor *StartBatchCreateGuestExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var poolName, namePrefix string
	var nameRule, guestCount uint
	if namePrefix, err = request.GetString(framework.ParamKeyName);err != nil{
		return err
	}
	if poolName, err = request.GetString(framework.ParamKeyPool);err != nil{
		return err
	}
	if nameRule, err = request.GetUInt(framework.ParamKeyMode);err != nil{
		return err
	}
	if guestCount, err = request.GetUInt(framework.ParamKeyCount);err != nil{
		return err
	}
	var templateID, adminName string
	var templateOptions []uint64
	if templateID, err = request.GetString(framework.ParamKeyTemplate); err != nil{
		err = fmt.Errorf("get template id fail: %s", err.Error())
		return
	}else{
		var respChan = make(chan modules.ResourceResult, 1)
		executor.ResourceModule.GetSystemTemplate(templateID, respChan)
		var result = <- respChan
		if result.Error != nil{
			err = fmt.Errorf("get template fail: %s", result.Error)
			return
		}
		var t = result.Template
		if adminName, err = request.GetString(framework.ParamKeyAdmin); err != nil{
			adminName = t.Admin
		}
		if templateOptions, err = t.ToOptions(); err != nil{
			err = fmt.Errorf("invalid template: %s", err.Error())
			return
		}
	}

	log.Printf("[%08X] recv batch create %d guests from %s.[%08X]", id, guestCount, request.GetSender(), request.GetFromSession())

	resp, _ := framework.CreateJsonMessage(framework.StartBatchCreateGuestResponse)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())
	resp.SetSuccess(false)
	var originalSender = request.GetSender()

	var batchID string
	var guestList []string
	{
		var respChan = make(chan modules.ResourceResult, 1)
		var bathRequest = modules.BatchCreateRequest{modules.BatchCreateNameRule(nameRule), namePrefix, poolName, int(guestCount)}
		executor.ResourceModule.StartBatchCreateGuest(bathRequest, respChan)
		var result = <- respChan
		if result.Error != nil{
			err = result.Error
			log.Printf("[%08X] start batch create guest fail: %s", id, err.Error())
			resp.SetError(err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		batchID = result.Batch
		for _, status := range result.BatchCreate{
			guestList = append(guestList, status.Name)
		}
	}

	var targets = map[framework.TransactionID]string{}
	//forward request
	for index, guestName := range guestList{
		var forward = framework.CloneJsonMessage(request)
		var transID = framework.TransactionID(index)
		forward.SetID(framework.CreateGuestRequest)
		forward.SetString(framework.ParamKeyAdmin, adminName)
		forward.SetString(framework.ParamKeyName, guestName)
		forward.SetUIntArray(framework.ParamKeyTemplate, templateOptions)
		forward.SetToSession(0)
		forward.SetFromSession(id)
		forward.SetTransactionID(transID)
		targets[transID] = guestName
		if err = executor.Sender.SendToSelf(forward); err != nil{
			log.Printf("[%08X] warning: forward create guest '%s' fail: %s", id, guestName, err.Error())
		}
	}
	log.Printf("[%08X] new batch create '%s' started", id, batchID)
	resp.SetSuccess(true)
	resp.SetString(framework.ParamKeyID, batchID)
	executor.Sender.SendMessage(resp, originalSender)

	var lastUpdate = time.Now()
	const (
		CheckInterval = time.Second * 5
		UpdateTimeout = time.Second * 10
	)
	var checkTicker = time.NewTicker(CheckInterval)
	for len(targets) > 0{
		select {
		case <- checkTicker.C:
			//check
			if lastUpdate.Add(UpdateTimeout).Before(time.Now()){
				log.Printf("[%08X] warning: receive create response timeout", id)
				return
			}
		case createResponse := <-incoming:
			var transID = createResponse.GetTransactionID()
			guestName, exists := targets[transID]
			if !exists{
				log.Printf("[%08X] warning: invalid create response with trans [%08X] from [%08X]",
					id, transID, createResponse.GetFromSession())
				break
			}
			var errChan = make(chan error, 1)
			if createResponse.IsSuccess(){
				var guestID string
				if guestID, err = createResponse.GetString(framework.ParamKeyInstance); err != nil{
					log.Printf("[%08X] warning: guest '%s' created, but get id fail", id, guestName)
					break
				}
				log.Printf("[%08X] create guest '%s'('%s') started", id, guestName, guestID)
				executor.ResourceModule.SetBatchCreateGuestStart(batchID, guestName, guestID, errChan)
			}else{
				var createError = errors.New(createResponse.GetError())
				log.Printf("[%08X] create guest '%s' fail: %s", id, guestName, createError.Error())
				executor.ResourceModule.SetBatchCreateGuestFail(batchID, guestName, createError, errChan)
			}
			var result = <- errChan
			if result != nil{
				log.Printf("[%08X] warning:update create status fail: %s", id, result.Error())
			}
			lastUpdate = time.Now()
			delete(targets, transID)
		}
	}
	//all targets finished
	log.Printf("[%08X] all create request finished in batch '%s'", id, batchID)
	return nil
}
