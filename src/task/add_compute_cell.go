package task

import (
	"github.com/project-nano/framework"
	"modules"
	"log"
	"time"
)

type AddComputePoolExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *AddComputePoolExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	poolName, err := request.GetString(framework.ParamKeyPool)
	if err != nil{
		return err
	}
	cellName, err := request.GetString(framework.ParamKeyCell)
	if err != nil{
		return err
	}
	log.Printf("[%08X] request add cell '%s' to pool '%s' from %s.[%08X]", id, cellName, poolName,
		request.GetSender(), request.GetFromSession())
	var respChan= make(chan error)
	executor.ResourceModule.AddCell(poolName, cellName, respChan)

	resp, _ := framework.CreateJsonMessage(framework.AddComputePoolCellResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	defer executor.Sender.SendMessage(resp, request.GetSender())

	err = <-respChan
	if err != nil{
		resp.SetError(err.Error())
		log.Printf("[%08X] add compute cell fail: %s", id, err.Error())
		return
	}

	var computePool modules.ComputePoolInfo
	{
		var respChan = make(chan modules.ResourceResult, 1)
		executor.ResourceModule.GetComputePool(poolName, respChan)
		var result = <- respChan
		if result.Error != nil{
			err = result.Error
			resp.SetError(err.Error())
			log.Printf("[%08X] warning: get compute pool fail: %s", id, err.Error())
			return
		}
		computePool = result.ComputePoolInfo
	}

	{
		notify, _ := framework.CreateJsonMessage(framework.ComputePoolReadyEvent)
		notify.SetString(framework.ParamKeyPool, poolName)
		notify.SetString(framework.ParamKeyStorage, computePool.Storage)
		notify.SetString(framework.ParamKeyNetwork, computePool.Network)
		if "" != computePool.Storage {
			var respChan = make(chan modules.ResourceResult, 1)
			executor.ResourceModule.GetStoragePool(computePool.Storage, respChan)
			var result = <-respChan
			if result.Error != nil {
				err = result.Error
				resp.SetError(err.Error())
				log.Printf("[%08X] get storage pool fail: %s", id, err.Error())
				return
			}
			var storagePool = result.StoragePoolInfo
			notify.SetString(framework.ParamKeyType, storagePool.Type)
			notify.SetString(framework.ParamKeyHost, storagePool.Host)
			notify.SetString(framework.ParamKeyTarget, storagePool.Target)
		}
		if "" != computePool.Network{
			var respChan = make(chan modules.ResourceResult, 1)
			executor.ResourceModule.GetAddressPool(computePool.Network, respChan)
			var result = <- respChan
			if result.Error != nil{
				log.Printf("[%08X] get address pool fail: %s", id, result.Error.Error())
				return nil
			}
			var addressPool = result.AddressPoolStatus
			notify.SetString(framework.ParamKeyGateway, addressPool.Gateway)
			notify.SetStringArray(framework.ParamKeyServer, addressPool.DNS)
		}

		notify.SetFromSession(id)
		if err = executor.Sender.SendMessage(notify, cellName);err != nil{
			log.Printf("[%08X] notify cell fail: %s", id, err.Error())
			resp.SetError(err.Error())
			return
		}
		const (
			AddCellTimeout = 10 * time.Second
		)
		timer := time.NewTimer(AddCellTimeout)
		select {
		case cellResp := <- incoming:
			if cellResp.GetID() != framework.ComputeCellReadyEvent{
				log.Printf("[%08X] unexpected message [%08X] from %s.[%08X]", id, cellResp.GetID(),
					cellResp.GetSender(), cellResp.GetFromSession())
				resp.SetError("unexpected message received")
				return
			}
			if !cellResp.IsSuccess(){
				log.Printf("[%08X] wait cell ready fail: %s", id, cellResp.GetError())
				resp.SetError(cellResp.GetError())
			}else{
				resp.SetSuccess(true)
				log.Printf("[%08X] cell ready with storage pool '%s'", id, computePool.Storage)
			}
		case <- timer.C:
			//timeout
			log.Printf("[%08X] wait cell ready timeout", id)
			resp.SetError("wait cell ready timeout")
			return
		}
	}
	return nil
}
