package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
	"time"
	"fmt"
)

type HandleCellAvailableExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *HandleCellAvailableExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) error {
	cellName, err := request.GetString(framework.ParamKeyCell)
	if err != nil{
		return err
	}
	cellAddress, err := request.GetString(framework.ParamKeyAddress)
	if err != nil{
		return err
	}
	log.Printf("[%08X] cell '%s' (address %s) available", id, cellName, cellAddress)
	{
		var respChan = make(chan error)
		executor.ResourceModule.UpdateCellInfo(cellName, cellAddress, respChan)
		err = <- respChan
		if err != nil{
			log.Printf("[%08X] update cell info fail: %s", id, err.Error())
			return nil
		}
	}
	var poolName string
	var cellStatus modules.ComputeCellStatus
	{
		//get pool name
		respChan := make(chan modules.ResourceResult, 1)
		executor.ResourceModule.GetCellStatus(cellName, respChan)
		result := <- respChan
		if result.Error != nil{
			log.Printf("[%08X] get cell status fail: %s", id, result.Error.Error())
			return nil
		}
		if result.Pool == ""{
			log.Printf("[%08X] cell not allocated", id)
			return nil
		}
		poolName = result.Pool
		cellStatus = result.ComputeCellStatus
	}
	if cellStatus.PurgeAppending{
		//need purge
		detach, _ := framework.CreateJsonMessage(framework.DetachInstanceRequest)
		detach.SetFromSession(id)
		detach.SetStringArray(framework.ParamKeyInstance, []string{})
		if err = executor.Sender.SendMessage(detach, cellName); err != nil{
			log.Printf("[%08X] try purge cell '%s' fail: %s", id, cellName, err.Error())
			return nil
		}
		log.Printf("[%08X] try purge remote cell '%s'...", id, cellName)
		timer := time.NewTimer(10*time.Second)
		select {
		case purgeResp := <- incoming:
			if purgeResp.GetID() != framework.DetachInstanceResponse{
				log.Printf("[%08X] unexpected message [%08X] from %s.[%08X]", id, purgeResp.GetID(),
					purgeResp.GetSender(), purgeResp.GetFromSession())
				return nil
			}
			if !purgeResp.IsSuccess(){
				log.Printf("[%08X] purge remote cell '%s' fail: %s", id, purgeResp.GetSender(), purgeResp.GetError())
				return nil
			}
			var respChan = make(chan error, 1)
			executor.ResourceModule.PurgeInstance(cellName, respChan)
			err = <- respChan
			if err != nil{
				log.Printf("[%08X] purge cell instance fail: %s", id, err.Error())
			}else{
				log.Printf("[%08X] instances on cell '%s' purged", id, cellName)
			}
		case <- timer.C:
			//timeout
			log.Printf("[%08X] wait purge response timeout", id)
			return nil
		}
		return nil
	}

	var computePool modules.ComputePoolInfo
	{
		var respChan = make(chan modules.ResourceResult, 1)
		executor.ResourceModule.GetComputePool(poolName, respChan)
		var result = <- respChan
		if result.Error != nil{
			log.Printf("[%08X] get compute pool fail: %s", id, result.Error.Error())
			return nil
		}
		computePool = result.ComputePoolInfo
	}
	var cellResp framework.Message
	{
		//notify cell
		notify, _ := framework.CreateJsonMessage(framework.ComputePoolReadyEvent)
		notify.SetString(framework.ParamKeyPool, poolName)
		notify.SetString(framework.ParamKeyStorage, computePool.Storage)
		notify.SetString(framework.ParamKeyNetwork, computePool.Network)
		if "" != computePool.Storage{
			var respChan = make(chan modules.ResourceResult, 1)
			executor.ResourceModule.GetStoragePool(computePool.Storage, respChan)
			var result = <- respChan
			if result.Error != nil{
				log.Printf("[%08X] get storage pool fail: %s", id, result.Error.Error())
				return nil
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
			return nil
		}

		timer := time.NewTimer(modules.DefaultOperateTimeout)
		select {
		case cellResp = <- incoming:
			if cellResp.GetID() != framework.ComputeCellReadyEvent{
				log.Printf("[%08X] unexpected message [%08X] from %s.[%08X]", id, cellResp.GetID(),
					cellResp.GetSender(), cellResp.GetFromSession())
				return nil
			}
			if !cellResp.IsSuccess(){
				log.Printf("[%08X] cell using storage '%s' fail: %s", id, computePool.Storage, cellResp.GetError())
				return nil
			}
		case <- timer.C:
			//timeout
			log.Printf("[%08X] wait cell instance timeout", id)
			return nil
		}
	}

	count, err := cellResp.GetUInt(framework.ParamKeyCount)
	if err != nil{
		return err
	}
	var instances []modules.InstanceStatus
	if 0 == count{
		log.Printf("[%08X] no instance available in cell '%s'", id, cellName)
	}else {

		var names, ids, users, groups, secrets, addresses, systems, internal, external, createTime, hardware []string
		var cores, options, enables, progress, status, monitors, memories, disks, diskCounts, cpuPriorities, ioLimits []uint64
		if names, err = cellResp.GetStringArray(framework.ParamKeyName); err != nil {
			return err
		}
		if ids, err = cellResp.GetStringArray(framework.ParamKeyInstance); err != nil {
			return err
		}
		if users, err = cellResp.GetStringArray(framework.ParamKeyUser); err != nil {
			return err
		}
		if groups, err = cellResp.GetStringArray(framework.ParamKeyGroup); err != nil {
			return err
		}
		if secrets, err = cellResp.GetStringArray(framework.ParamKeySecret); err != nil {
			return err
		}
		if addresses, err = cellResp.GetStringArray(framework.ParamKeyAddress); err != nil{
			return err
		}
		if systems, err = cellResp.GetStringArray(framework.ParamKeySystem); err != nil{
			return err
		}
		if internal, err = cellResp.GetStringArray(framework.ParamKeyInternal); err != nil{
			return err
		}
		if external, err = cellResp.GetStringArray(framework.ParamKeyExternal); err != nil{
			return err
		}
		if createTime, err = cellResp.GetStringArray(framework.ParamKeyCreate); err != nil{
			return err
		}
		if hardware, err = cellResp.GetStringArray(framework.ParamKeyHardware); err != nil{
			return err
		}

		if cores, err = cellResp.GetUIntArray(framework.ParamKeyCore); err != nil {
			return err
		}
		if options, err = cellResp.GetUIntArray(framework.ParamKeyOption); err != nil {
			return err
		}
		if enables, err = cellResp.GetUIntArray(framework.ParamKeyEnable); err != nil {
			return err
		}
		if progress, err = cellResp.GetUIntArray(framework.ParamKeyProgress); err != nil {
			return err
		}
		if status, err = cellResp.GetUIntArray(framework.ParamKeyStatus); err != nil {
			return err
		}
		if monitors, err = cellResp.GetUIntArray(framework.ParamKeyMonitor); err != nil {
			return err
		}
		if memories, err = cellResp.GetUIntArray(framework.ParamKeyMemory); err != nil {
			return err
		}
		if diskCounts, err = cellResp.GetUIntArray(framework.ParamKeyCount); err != nil {
			return err
		}
		if disks, err = cellResp.GetUIntArray(framework.ParamKeyDisk); err != nil {
			return err
		}

		if cpuPriorities, err = cellResp.GetUIntArray(framework.ParamKeyPriority); err != nil{
			return err
		}

		if ioLimits, err = cellResp.GetUIntArray(framework.ParamKeyLimit); err != nil{
			return err
		}

		const (
			ReadSpeedOffset           = iota
			WriteSpeedOffset
			ReadIOPSOffset
			WriteIOPSOffset
			ReceiveOffset
			SendOffset
			ValidLimitParametersCount = 6
		)

		if len(cpuPriorities) != int(count){
			return fmt.Errorf("unpected priority array size %d / %d", len(cpuPriorities), count)
		}

		if len(ioLimits) != int(count * ValidLimitParametersCount){
			return fmt.Errorf("unpected limit array size %d / %d", len(ioLimits), count * ValidLimitParametersCount)
		}

		var diskOffset= 0
		for i := 0; i < int(count); i++ {
			var config modules.InstanceStatus
			config.Pool = poolName
			config.Cell = cellName
			config.Name = names[i]
			config.ID = ids[i]
			config.User = users[i]
			config.Group = groups[i]
			config.Cores = uint(cores[i])
			config.Memory = uint(memories[i])
			var diskCount= int(diskCounts[i])
			for _, diskSize := range disks[diskOffset:diskOffset+diskCount] {
				config.Disks = append(config.Disks, diskSize)
			}
			diskOffset += diskCount
			config.InternalNetwork.MonitorPort = uint(monitors[i])
			config.MonitorSecret = secrets[i]
			config.Progress = uint(progress[i])
			config.AutoStart = options[i] == 1
			config.Created = enables[i] == 1
			config.Running = status[i] == modules.InstanceStatusRunning
			config.System = systems[i]
			config.InternalNetwork.InstanceAddress = addresses[i]
			config.InternalNetwork.AssignedAddress = internal[i]
			config.ExternalNetwork.AssignedAddress = external[i]
			config.CreateTime = createTime[i]
			config.HardwareAddress = hardware[i]
			config.CPUPriority = modules.PriorityEnum(cpuPriorities[i])
			config.ReadSpeed = ioLimits[ ValidLimitParametersCount * i + ReadSpeedOffset ]
			config.WriteSpeed = ioLimits[ ValidLimitParametersCount * i + WriteSpeedOffset ]
			config.ReadIOPS = ioLimits[ ValidLimitParametersCount * i + ReadIOPSOffset ]
			config.WriteIOPS = ioLimits[ ValidLimitParametersCount * i + WriteIOPSOffset ]
			config.ReceiveSpeed = ioLimits[ ValidLimitParametersCount * i + ReceiveOffset ]
			config.SendSpeed = ioLimits[ ValidLimitParametersCount * i + SendOffset ]
			instances = append(instances, config)
		}
	}
	{
		var respChan = make(chan error)
		executor.ResourceModule.BatchUpdateInstanceStatus(poolName, cellName, instances, respChan)
		err = <- respChan
		if err != nil{
			log.Printf("[%08X] update cell instance fail: %s", id, err.Error())
			return nil
		}
		log.Printf("[%08X] %d instance(s) updated in cell '%s'", id, len(instances), cellName)
	}
	return nil
}
