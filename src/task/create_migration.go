package task

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"log"
	"time"
)

type CreateMigrationExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *CreateMigrationExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var poolName, sourceCell, targetCell string
	var instances []string
	{
		const (
			SourcePoolOffset = 0
			SourceCellOffset = 0
			TargetCellOffset = 1
			ValidPoolCount   = 1
			ValidCellCount   = 2
		)
		var pools, cells []string
		if pools, err = request.GetStringArray(framework.ParamKeyPool); err != nil {
			return
		}
		if cells, err = request.GetStringArray(framework.ParamKeyCell); err != nil {
			return
		}
		if instances, err = request.GetStringArray(framework.ParamKeyInstance); err != nil {
			return
		}
		if ValidPoolCount != len(pools) {
			err = fmt.Errorf("invalid migration pool count %d", len(pools))
			return
		}
		if ValidCellCount != len(cells) {
			err = fmt.Errorf("invalid migration cell count %d", len(cells))
			return
		}
		poolName = pools[SourcePoolOffset]
		sourceCell = cells[SourceCellOffset]
		targetCell = cells[TargetCellOffset]
	}
	if 0 == len(instances) {
		log.Printf("[%08X] request migrate all instance in '%s.%s' to '%s.%s' from %s.[%08X]",
			id, poolName, sourceCell, poolName, targetCell, request.GetSender(), request.GetFromSession())
	} else {
		log.Printf("[%08X] request migrate %d instance(s) in '%s.%s' to '%s.%s' from %s.[%08X]",
			id, len(instances), poolName, sourceCell, poolName, targetCell, request.GetSender(), request.GetFromSession())
	}

	var migrationID string
	var params = modules.MigrationParameter{SourcePool: poolName, SourceCell: sourceCell, TargetPool: poolName, TargetCell: targetCell, Instances: instances}
	{
		resp, _ := framework.CreateJsonMessage(framework.CreateMigrationResponse)
		resp.SetSuccess(false)
		resp.SetFromSession(id)
		resp.SetToSession(request.GetFromSession())

		//allocate migration
		var respChan = make(chan modules.ResourceResult, 1)
		executor.ResourceModule.CreateMigration(params, respChan)
		var result = <-respChan
		if result.Error != nil {
			err = result.Error
			resp.SetError(err.Error())
			log.Printf("[%08X] allocate migration task fail: %s", id, err.Error())
			return executor.Sender.SendMessage(resp, request.GetSender())
		}
		migrationID = result.Migration.ID
		instances = result.Migration.Instances
		resp.SetSuccess(true)
		resp.SetString(framework.ParamKeyMigration, migrationID)
		log.Printf("[%08X] migration '%s' allocated", id, migrationID)
		if err = executor.Sender.SendMessage(resp, request.GetSender()); err != nil {
			log.Printf("[%08X] warning: notify migration id fail: %s", id, err.Error())
		}
	}
	var targetSession framework.SessionID
	{
		//attach instance
		attach, _ := framework.CreateJsonMessage(framework.AttachInstanceRequest)
		attach.SetFromSession(id)
		attach.SetBoolean(framework.ParamKeyImmediate, false)
		attach.SetStringArray(framework.ParamKeyInstance, instances)
		if err = executor.Sender.SendMessage(attach, targetCell); err != nil {
			log.Printf("[%08X] request attach instance fail: %s", id, err.Error())
			executor.releaseMigration(id, migrationID, err)
			return nil
		}
		timer := time.NewTimer(modules.GetConfigurator().GetOperateTimeout())
		select {
		case cellResp := <-incoming:
			if !cellResp.IsSuccess() {
				//attach fail
				log.Printf("[%08X] attach instances fail: %s", id, cellResp.GetError())
				executor.releaseMigration(id, migrationID, fmt.Errorf("attach instance fail: %s", cellResp.GetError()))
				return nil
			}
			log.Printf("[%08X] instances attached to '%s.%s'", id, poolName, targetCell)
			targetSession = cellResp.GetFromSession()
		case <-timer.C:
			//timeout
			log.Printf("[%08X] attach instances timeout", id)
			executor.releaseMigration(id, migrationID, errors.New("attach instance timeout"))
			return nil
		}
	}
	{
		//detach
		detach, _ := framework.CreateJsonMessage(framework.DetachInstanceRequest)
		detach.SetFromSession(id)
		detach.SetStringArray(framework.ParamKeyInstance, instances)
		if err = executor.Sender.SendMessage(detach, sourceCell); err != nil {
			log.Printf("[%08X] request detach instance fail: %s", id, err.Error())
			executor.releaseMigration(id, migrationID, err)
			return nil
		}
		timer := time.NewTimer(modules.GetConfigurator().GetOperateTimeout())
		select {
		case cellResp := <-incoming:
			if !cellResp.IsSuccess() {
				//detach fail
				log.Printf("[%08X] detach instances fail: %s", id, cellResp.GetError())
				executor.releaseMigration(id, migrationID, fmt.Errorf("detach instance fail: %s", cellResp.GetError()))
				return nil
			}
			log.Printf("[%08X] instances detached from '%s.%s'", id, poolName, sourceCell)
		case <-timer.C:
			//timeout
			log.Printf("[%08X] detach instances timeout", id)
			executor.releaseMigration(id, migrationID, errors.New("detach instance timeout"))
			return nil
		}
	}
	{
		//migrate
		migrate, _ := framework.CreateJsonMessage(framework.MigrateInstanceRequest)
		migrate.SetFromSession(id)
		migrate.SetToSession(targetSession)
		migrate.SetString(framework.ParamKeyMigration, migrationID)
		migrate.SetStringArray(framework.ParamKeyInstance, instances)
		if err = executor.Sender.SendMessage(migrate, targetCell); err != nil {
			log.Printf("[%08X] warning: notify migrate fail: %s", id, err.Error())
			executor.releaseMigration(id, migrationID, err)
		} else {
			log.Printf("[%08X] notify '%s.%s' start migrate", id, poolName, targetCell)
		}
	}
	return nil
}

func (executor *CreateMigrationExecutor) releaseMigration(id framework.SessionID, migration string, reason error) {
	var respChan = make(chan error, 1)
	executor.ResourceModule.CancelMigration(migration, reason, respChan)
	var err = <-respChan
	if err != nil {
		log.Printf("[%08X] warning: release migration fail: %s", id, migration)
	} else {
		log.Printf("[%08X] migration %s released", id, migration)
	}
}
