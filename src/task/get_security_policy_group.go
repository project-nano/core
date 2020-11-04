package task

import (
	"fmt"
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"log"
)

type GetSecurityPolicyGroupExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *GetSecurityPolicyGroupExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var policyID string
	if policyID, err = request.GetString(framework.ParamKeyPolicy);err != nil{
		err = fmt.Errorf("get policy group ID fail: %s", err.Error())
		return
	}

	resp, _ := framework.CreateJsonMessage(framework.GetPolicyGroupResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetTransactionID(request.GetTransactionID())
	resp.SetSuccess(false)
	var respChan = make(chan modules.ResourceResult, 1)
	executor.ResourceModule.GetSecurityPolicyGroup(policyID, respChan)
	var result = <- respChan
	if result.Error != nil{
		err = result.Error
		log.Printf("[%08X] get security policy group '%s' fail: %s",
			id, policyID, err.Error())
		resp.SetError(err.Error())
	}else{
		var policy = result.PolicyGroup
		resp.SetString(framework.ParamKeyPolicy, policy.ID)
		resp.SetString(framework.ParamKeyName, policy.Name)
		resp.SetString(framework.ParamKeyDescription, policy.Description)
		resp.SetString(framework.ParamKeyUser, policy.User)
		resp.SetString(framework.ParamKeyGroup, policy.Group)
		resp.SetBoolean(framework.ParamKeyAction, policy.Accept)
		resp.SetBoolean(framework.ParamKeyEnable, policy.Enabled)
		resp.SetBoolean(framework.ParamKeyLimit, policy.Global)
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
