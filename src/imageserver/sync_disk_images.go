package imageserver

import (
	"fmt"
	"github.com/project-nano/framework"
	"log"
)

type SyncDiskImagesExecutor struct {
	Sender      framework.MessageSender
	ImageServer *ImageManager
}

func (executor *SyncDiskImagesExecutor)Execute(id framework.SessionID, request framework.Message,
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
	log.Printf("[%08X] %s.[%08X] request synchronize disk images...",
		id, request.GetSender(), request.GetFromSession())
	var respChan = make(chan error, 1)
	executor.ImageServer.SyncDiskImages(owner, group, respChan)
	err = <- respChan

	resp, _ := framework.CreateJsonMessage(framework.SynchronizeDiskImageResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	if err != nil{
		resp.SetError(err.Error())
		log.Printf("[%08X] sync disk images fail: %s", id, err.Error())
	}else{
		log.Printf("[%08X] disk images synchronized", id)
		resp.SetSuccess(true)
	}
	return executor.Sender.SendMessage(resp, request.GetSender())
}
