package task

import (
	"fmt"
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"log"
)

type DeleteSecurityPolicyGroupExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *DeleteSecurityPolicyGroupExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var policyID string
	if policyID, err = request.GetString(framework.ParamKeyPolicy);err != nil{
		err = fmt.Errorf("get policy group ID fail: %s", err.Error())
		return
	}

	resp, _ := framework.CreateJsonMessage(framework.DeletePolicyGroupResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetTransactionID(request.GetTransactionID())
	resp.SetSuccess(false)
	var respChan = make(chan error, 1)
	executor.ResourceModule.DeleteSecurityPolicyGroup(policyID, respChan)
	err = <- respChan
	if err != nil{
		log.Printf("[%08X] delete security policy group '%s' fail: %s",
			id, policyID, err.Error())
		resp.SetError(err.Error())
	}else{
		log.Printf("[%08X] security policy group '%s' deleted",
			id, policyID)
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
