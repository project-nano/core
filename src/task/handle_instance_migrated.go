package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"log"
)

type HandleInstanceMigratedExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *HandleInstanceMigratedExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error){
	var failover = false
	var migrationID string
	var instances []string
	var monitorPorts []uint64
	failover, err = request.GetBoolean(framework.ParamKeyImmediate)
	if err != nil{
		return
	}
	if instances, err = request.GetStringArray(framework.ParamKeyInstance);err != nil{
		return
	}
	if monitorPorts, err = request.GetUIntArray(framework.ParamKeyMonitor); err != nil{
		return
	}
	if !failover{
		//active migration
		migrationID, err = request.GetString(framework.ParamKeyMigration)
		if err != nil{
			return
		}

		var respChan = make(chan error, 1)
		executor.ResourceModule.FinishMigration(migrationID, instances, monitorPorts, respChan)
		err = <- respChan
		if err != nil{
			log.Printf("[%08X] finish migration fail: %s", id, err.Error())
		}else{
			log.Printf("[%08X] migration '%s' finished from %s.[%08X]", id, migrationID, request.GetSender(), request.GetFromSession())
		}
		return nil
	}else{
		//failover
		sourceCell, err := request.GetString(framework.ParamKeyCell)
		if err != nil{
			return err
		}
		var respChan = make(chan error, 1)
		executor.ResourceModule.MigrateInstance(sourceCell, request.GetSender(), instances, monitorPorts, respChan)
		err = <- respChan
		if err != nil{
			log.Printf("[%08X] migrate instance fail: %s", id, err.Error())
		}else{
			log.Printf("[%08X] %d instance(s) migrated from '%s' to '%s'", id, len(instances), sourceCell, request.GetSender())
		}
		return nil
	}

}