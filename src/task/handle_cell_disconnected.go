package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
)

type HandleCellDisconnectedExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *HandleCellDisconnectedExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	cellName, err := request.GetString(framework.ParamKeyCell)
	if err != nil {
		return
	}
	gracefullyClose, err := request.GetBoolean(framework.ParamKeyFlag)
	if err != nil{
		return
	}
	if gracefullyClose{
		var respChan = make(chan error, 1)
		executor.ResourceModule.SetCellDead(cellName, respChan)
		err = <- respChan
		if err != nil{
			log.Printf("[%08X] set cell dead fail: %s", id, err.Error())
		}else{
			log.Printf("[%08X] remote cell '%s' closed", id, cellName)
		}
		return
	}
	var plan map[string][]string
	{
		var respChan = make(chan modules.ResourceResult, 1)
		executor.ResourceModule.BuildFailoverPlan(cellName, respChan)
		var result = <- respChan
		if result.Error != nil{
			err = result.Error
			log.Printf("[%08X] handle cell '%s' disconnected fail: %s", id, cellName, err.Error())
			return nil
		}
		if 0 == len(result.FailoverPlan){
			//no plan need execute
			log.Printf("[%08X] cell '%s' lost", id, cellName)
			return nil
		}
		plan = result.FailoverPlan
	}
	var instanceCount = 0
	for targetName, instances := range plan{
		migrate, _ := framework.CreateJsonMessage(framework.AttachInstanceRequest)
		migrate.SetStringArray(framework.ParamKeyInstance, instances)
		migrate.SetBoolean(framework.ParamKeyImmediate, true)
		migrate.SetString(framework.ParamKeyCell, cellName)
		if err = executor.Sender.SendMessage(migrate, targetName);err != nil{
			log.Printf("[%08X] warning: send migrate request to '%s' fail: %s", id, targetName, err.Error())
		}else{
			log.Printf("[%08X] try migrate %d instances to cell '%s'", id, len(instances), targetName)
		}
		instanceCount += len(instances)
	}
	log.Printf("[%08X] %d instance(s) on cell '%s' dispatched to %d new cells by automated failover.",
		id, instanceCount, cellName, len(plan))
	return nil
}
