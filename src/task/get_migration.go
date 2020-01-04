package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
)

type GetMigrationExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *GetMigrationExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error){
	var migrationID string
	migrationID, err = request.GetString(framework.ParamKeyMigration)
	if err != nil{
		return
	}
	resp, _ := framework.CreateJsonMessage(framework.GetMigrationResponse)
	resp.SetSuccess(false)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)

	var respChan = make(chan modules.ResourceResult, 1)
	executor.ResourceModule.GetMigration(migrationID, respChan)
	var result = <- respChan
	if result.Error != nil{
		err = result.Error
		log.Printf("[%08X] get migration fail: %s", id, err.Error())
		resp.SetError(err.Error())
		return nil
	}
	var migration = result.MigrationStatus
	resp.SetSuccess(true)
	resp.SetString(framework.ParamKeyMigration, migrationID)
	resp.SetBoolean(framework.ParamKeyStatus, migration.Finished)
	resp.SetUInt(framework.ParamKeyProgress, migration.Progress)
	if migration.Error != nil{
		resp.SetString(framework.ParamKeyError, migration.Error.Error())
	}else{
		resp.SetString(framework.ParamKeyError, "")
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}