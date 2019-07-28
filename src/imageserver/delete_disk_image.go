package imageserver

import (
	"github.com/project-nano/framework"
	"log"
)

type DeleteDiskImageExecutor struct {
	Sender      framework.MessageSender
	ImageServer *ImageManager
}


func (executor *DeleteDiskImageExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	imageID, err := request.GetString(framework.ParamKeyImage)
	if err != nil{
		return err
	}
	resp, _ := framework.CreateJsonMessage(framework.DeleteDiskImageResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	var respChan = make(chan error, 1)
	executor.ImageServer.DeleteDiskImage(imageID, respChan)
	err = <-respChan
	if err != nil {
		resp.SetError(err.Error())
		log.Printf("[%08X] delete disk image fail: %s", id, err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	resp.SetSuccess(true)
	log.Printf("[%08X] disk image '%s' deleted", id, imageID)
	return executor.Sender.SendMessage(resp, request.GetSender())
}