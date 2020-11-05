package task

import (
	"fmt"
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"log"
)

type GetSecurityPolicyRulesExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *GetSecurityPolicyRulesExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var policyID string
	if policyID, err = request.GetString(framework.ParamKeyPolicy);err != nil{
		err = fmt.Errorf("get policy group ID fail: %s", err.Error())
		return
	}

	resp, _ := framework.CreateJsonMessage(framework.QueryPolicyRuleResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetTransactionID(request.GetTransactionID())
	resp.SetSuccess(false)
	var respChan = make(chan modules.ResourceResult, 1)
	executor.ResourceModule.GetSecurityPolicyRules(policyID, respChan)
	var result = <- respChan
	if result.Error != nil{
		err = result.Error
		log.Printf("[%08X] get all rules of security policy '%s' fail: %s",
			id, policyID, err.Error())
		resp.SetError(err.Error())
	}else{
		var actions, targetPorts []uint64
		var protocols, sourceAddresses, targetAddresses []string
		for _, rule := range result.PolicyRuleList{
			if rule.Accept{
				actions = append(actions, modules.PolicyRuleActionAccept)
			}else{
				actions = append(actions, modules.PolicyRuleActionReject)
			}
			targetPorts = append(targetPorts, uint64(rule.TargetPort))
			protocols = append(protocols, string(rule.Protocol))
			targetAddresses = append(targetAddresses, rule.TargetAddress)
			sourceAddresses = append(sourceAddresses, rule.SourceAddress)
		}
		resp.SetUIntArray(framework.ParamKeyAction, actions)
		resp.SetUIntArray(framework.ParamKeyPort, targetPorts)
		resp.SetStringArray(framework.ParamKeyProtocol, protocols)
		resp.SetStringArray(framework.ParamKeyFrom, sourceAddresses)
		resp.SetStringArray(framework.ParamKeyTo, targetAddresses)
		log.Printf("[%08X] %d rules of security policy '%s' available",
			id, len(actions), policyID)
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}