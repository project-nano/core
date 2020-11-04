package task

import (
	"fmt"
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"log"
)

type AddSecurityPolicyRuleExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *AddSecurityPolicyRuleExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var policyID string
	if policyID, err = request.GetString(framework.ParamKeyPolicy);err != nil{
		err = fmt.Errorf("get policy group ID fail: %s", err.Error())
		return
	}
	var rule modules.SecurityPolicyRule
	if rule.Accept, err = request.GetBoolean(framework.ParamKeyAction); err != nil{
		err = fmt.Errorf("get action fail: %s", err.Error())
		return
	}
	var protocol string
	if protocol, err = request.GetString(framework.ParamKeyProtocol); err != nil{
		err = fmt.Errorf("get protocol fail: %s", err.Error())
		return
	}
	switch protocol {
	case modules.PolicyRuleProtocolTCP:
	case modules.PolicyRuleProtocolUDP:
	case modules.PolicyRuleProtocolICMP:
	default:
		err = fmt.Errorf("invalid protocol: %s", protocol)
		return
	}
	rule.Protocol = modules.PolicyRuleProtocol(protocol)
	if rule.SourceAddress, err = request.GetString(framework.ParamKeyFrom); err != nil{
		err = fmt.Errorf("get source address fail: %s", err.Error())
		return
	}
	if rule.TargetAddress, err = request.GetString(framework.ParamKeyTo); err != nil{
		err = fmt.Errorf("get target address fail: %s", err.Error())
		return
	}
	if rule.TargetPort, err = request.GetUInt(framework.ParamKeyPort); err != nil{
		err = fmt.Errorf("get target port fail: %s", err.Error())
		return
	}

	resp, _ := framework.CreateJsonMessage(framework.AddPolicyRuleResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetTransactionID(request.GetTransactionID())
	resp.SetSuccess(false)
	var respChan = make(chan error, 1)
	executor.ResourceModule.AddSecurityPolicyRule(policyID, rule, respChan)
	err = <- respChan
	if err != nil{
		log.Printf("[%08X] add new rule to security policy '%s' fail: %s",
			id, policyID, err.Error())
		resp.SetError(err.Error())
	}else{
		log.Printf("[%08X] new rule of security policy '%s' added",
			id, policyID)
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
