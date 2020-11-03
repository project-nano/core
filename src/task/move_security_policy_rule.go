package task

import (
	"fmt"
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"log"
)

type MoveSecurityPolicyRuleExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *MoveSecurityPolicyRuleExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var policyID string
	if policyID, err = request.GetString(framework.ParamKeyPolicy);err != nil{
		err = fmt.Errorf("get policy group ID fail: %s", err.Error())
		return
	}
	var index int
	if index, err = request.GetInt(framework.ParamKeyIndex);err != nil{
		err = fmt.Errorf("get index fail: %s", err.Error())
		return
	}
	var moveUp bool
	if moveUp, err = request.GetBoolean(framework.ParamKeyFlag); err != nil{
		err = fmt.Errorf("get move flag fail: %s", err.Error())
		return
	}

	resp, _ := framework.CreateJsonMessage(framework.ChangePolicyRuleOrderResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetTransactionID(request.GetTransactionID())
	resp.SetSuccess(false)
	var respChan = make(chan error, 1)
	executor.ResourceModule.MoveSecurityPolicyRule(policyID, index, moveUp, respChan)
	err = <- respChan
	if err != nil{
		log.Printf("[%08X] move %dth rule of security policy '%s' fail: %s",
			id, index, policyID, err.Error())
		resp.SetError(err.Error())
	}else{
		if moveUp{
			log.Printf("[%08X] %dth rule of security policy '%s' moved up",
				id, index, policyID)
		}else{
			log.Printf("[%08X] %dth rule of security policy '%s' moved down",
				id, index, policyID)
		}
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
