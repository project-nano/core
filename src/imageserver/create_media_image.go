package imageserver

import (
	"github.com/project-nano/framework"
	"log"
)

type CreateMediaImageExecutor struct {
	Sender      framework.MessageSender
	ImageServer *ImageManager
}


func (executor *CreateMediaImageExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var config MediaConfig
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
	var respChan = make(chan ImageResult, 1)
	executor.ImageServer.CreateMediaImage(config, respChan)
	result := <- respChan

	resp, _ := framework.CreateJsonMessage(framework.CreateMediaImageResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	if result.Error != nil{
		err = result.Error
		resp.SetError(err.Error())
		log.Printf("[%08X] create media image fail: %s", id, err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	log.Printf("[%08X] new media image '%s' created(id '%s')", id, config.Name, result.ID)
	resp.SetString(framework.ParamKeyImage, result.ID)
	resp.SetSuccess(true)
	return executor.Sender.SendMessage(resp, request.GetSender())
}
