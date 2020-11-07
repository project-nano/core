package imageserver

import (
	"fmt"
	"github.com/project-nano/framework"
	"log"
)

type SyncMediaImagesExecutor struct {
	Sender      framework.MessageSender
	ImageServer *ImageManager
}

func (executor *SyncMediaImagesExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var owner, group string
	if owner, err = request.GetString(framework.ParamKeyUser); err != nil {
		err = fmt.Errorf("get owner fail: %s", err.Error())
		return err
	}
	if group, err = request.GetString(framework.ParamKeyGroup); err != nil {
		err = fmt.Errorf("get group fail: %s", err.Error())
		return err
	}
	log.Printf("[%08X] %s.[%08X] request synchronize media images...",
		id, request.GetSender(), request.GetFromSession())
	var respChan = make(chan error, 1)
	executor.ImageServer.SyncMediaImages(owner, group, respChan)
	err = <- respChan

	resp, _ := framework.CreateJsonMessage(framework.SynchronizeMediaImageResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	if err != nil{
		resp.SetError(err.Error())
		log.Printf("[%08X] sync media images fail: %s", id, err.Error())
	}else{
		log.Printf("[%08X] media images synchronized", id)
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
