package imageserver

import (
	"github.com/project-nano/framework"
	"log"
)

type ModifyDiskImageExecutor struct {
	Sender      framework.MessageSender
	ImageServer *ImageManager
}


func (executor *ModifyDiskImageExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var imageID string
	if imageID, err = request.GetString(framework.ParamKeyImage); err != nil{
		return
	}
	var config ImageConfig
	if config.Name, err = request.GetString(framework.ParamKeyName); err != nil {
		return err
	}
	if config.Owner, err = request.GetString(framework.ParamKeyUser); err != nil {
		return err
	}
	if config.Group, err = request.GetString(framework.ParamKeyGroup); err != nil {
		return err
	}
	if config.Description, err = request.GetString(framework.ParamKeyDescription); err != nil {
		return err
	}
	if config.Tags, err = request.GetStringArray(framework.ParamKeyTag); err != nil {
		return err
	}
	var respChan = make(chan error, 1)
	executor.ImageServer.ModifyDiskImage(imageID, config, respChan)
	err = <- respChan

	resp, _ := framework.CreateJsonMessage(framework.ModifyMediaImageResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	if err != nil{
		resp.SetError(err.Error())
		log.Printf("[%08X] modify disk image fail: %s", id, err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	log.Printf("[%08X] disk image '%s' modified", id, imageID)
	resp.SetSuccess(true)
	return executor.Sender.SendMessage(resp, request.GetSender())
}
