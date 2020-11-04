package task

import (
	"fmt"
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"log"
)

type QuerySecurityPolicyGroupsExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
}

func (executor *QuerySecurityPolicyGroupsExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var condition modules.SecurityPolicyGroupQueryCondition
	if condition.User, err = request.GetString(framework.ParamKeyUser); err != nil{
		err = fmt.Errorf("get user fail: %s", err.Error())
		return
	}
	if condition.Group, err = request.GetString(framework.ParamKeyGroup); err != nil{
		err = fmt.Errorf("get group fail: %s", err.Error())
		return
	}
	if condition.EnabledOnly, err = request.GetBoolean(framework.ParamKeyEnable); err != nil{
		err = fmt.Errorf("get enable flag fail: %s", err.Error())
		return
	}
	if condition.GlobalOnly, err = request.GetBoolean(framework.ParamKeyLimit); err != nil{
		err = fmt.Errorf("get global flag fail: %s", err.Error())
		return
	}

	resp, _ := framework.CreateJsonMessage(framework.QueryPolicyGroupResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetTransactionID(request.GetTransactionID())
	resp.SetSuccess(false)
	var respChan = make(chan modules.ResourceResult, 1)
	executor.ResourceModule.QuerySecurityPolicyGroups(condition, respChan)
	var result = <- respChan
	if result.Error != nil{
		err = result.Error
		log.Printf("[%08X] query security policy groups fail: %s",
			id, err.Error())
		resp.SetError(err.Error())
	}else{
		var id, name, description, user, group []string
		var accept, enabled, global []uint64
		const (
			flagFalse = iota
			flagTrue
		)
		for _, policy := range result.PolicyGroupList{
			id = append(id, policy.ID)
			name = append(name, policy.Name)
			description = append(description, policy.Description)
			user = append(user, policy.User)
			group = append(group, policy.Group)
			if policy.Accept{
				accept = append(accept, flagTrue)
			}else{
				accept = append(accept, flagFalse)
			}
			if policy.Enabled{
				enabled = append(enabled, flagTrue)
			}else{
				enabled = append(enabled, flagFalse)
			}
			if policy.Global{
				global = append(global, flagTrue)
			}else{
				global = append(global, flagFalse)
			}
		}
		resp.SetStringArray(framework.ParamKeyPolicy, id)
		resp.SetStringArray(framework.ParamKeyName, name)
		resp.SetStringArray(framework.ParamKeyDescription, description)
		resp.SetStringArray(framework.ParamKeyUser, user)
		resp.SetStringArray(framework.ParamKeyGroup, group)
		resp.SetUIntArray(framework.ParamKeyAction, accept)
		resp.SetUIntArray(framework.ParamKeyEnable, enabled)
		resp.SetUIntArray(framework.ParamKeyLimit, global)
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
