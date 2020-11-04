package task

import (
	"fmt"
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"log"
)

type CreateSecurityPolicyGroupExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *CreateSecurityPolicyGroupExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var config modules.SecurityPolicyGroup
	if config.Name, err = request.GetString(framework.ParamKeyName); err != nil{
		err = fmt.Errorf("get name fail: %s", err.Error())
		return
	}
	if config.Description, err = request.GetString(framework.ParamKeyDescription); err != nil{
		err = fmt.Errorf("get description fail: %s", err.Error())
		return
	}
	if config.User, err = request.GetString(framework.ParamKeyUser); err != nil{
		err = fmt.Errorf("get user fail: %s", err.Error())
		return
	}
	if config.Group, err = request.GetString(framework.ParamKeyGroup); err != nil{
		err = fmt.Errorf("get group fail: %s", err.Error())
		return
	}
	if config.Accept, err = request.GetBoolean(framework.ParamKeyAction); err != nil{
		err = fmt.Errorf("get accpet flag fail: %s", err.Error())
		return
	}
	if config.Enabled, err = request.GetBoolean(framework.ParamKeyEnable); err != nil{
		err = fmt.Errorf("get enabled flag fail: %s", err.Error())
		return
	}
	if config.Global, err = request.GetBoolean(framework.ParamKeyLimit); err != nil{
		err = fmt.Errorf("get global flag fail: %s", err.Error())
		return
	}

	resp, _ := framework.CreateJsonMessage(framework.CreatePolicyGroupResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetTransactionID(request.GetTransactionID())
	resp.SetSuccess(false)
	var respChan = make(chan modules.ResourceResult, 1)
	executor.ResourceModule.CreateSecurityPolicyGroup(config, respChan)
	var result = <- respChan
	if result.Error != nil{
		err = result.Error
		log.Printf("[%08X] create security policy group '%s' fail: %s",
			id, config.Name, err.Error())
		resp.SetError(err.Error())
	}else{
		var policy = result.PolicyGroup
		log.Printf("[%08X] new security policy group '%s'('%s') created",
			id, config.Name, policy.ID)
		resp.SetString(framework.ParamKeyPolicy, policy.ID)
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
