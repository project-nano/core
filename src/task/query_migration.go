package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
)

type QueryMigrationExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *QueryMigrationExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error){
	resp, _ := framework.CreateJsonMessage(framework.QueryMigrationResponse)
	resp.SetSuccess(false)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	var respChan = make(chan modules.ResourceResult, 1)
	executor.ResourceModule.QueryMigration(respChan)
	var result = <- respChan
	if result.Error != nil{
		err = result.Error
		log.Printf("[%08X] query migration fail: %s", err.Error())
		resp.SetError(err.Error())
		return nil
	}
	var idList, errMessage []string
	var finish, progress []uint64
	for _, m := range result.MigrationStatusList{
		idList = append(idList, m.ID)
		if m.Finished{
			finish = append(finish, 1)
		}else{
			finish = append(finish, 0)
		}
		progress = append(progress, uint64(m.Progress))
		if m.Error != nil{
			errMessage = append(errMessage, m.Error.Error())
		}else{
			errMessage = append(errMessage, "")
		}

	}
	resp.SetSuccess(true)
	resp.SetStringArray(framework.ParamKeyMigration, idList)
	resp.SetUIntArray(framework.ParamKeyStatus, finish)
	resp.SetUIntArray(framework.ParamKeyProgress, progress)
	resp.SetStringArray(framework.ParamKeyError, errMessage)
	log.Printf("[%08X] %d migrations available", id, len(idList))
	return executor.Sender.SendMessage(resp, request.GetSender())
}