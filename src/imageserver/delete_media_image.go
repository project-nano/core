package imageserver

import (
	"github.com/project-nano/framework"
	"log"
)

type DeleteMediaImageExecutor struct {
	Sender      framework.MessageSender
	ImageServer *ImageManager
}


func (executor *DeleteMediaImageExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	imageID, err := request.GetString(framework.ParamKeyImage)
	if err != nil{
		return err
	}
	resp, _ := framework.CreateJsonMessage(framework.DeleteMediaImageResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	var respChan = make(chan error, 1)
	executor.ImageServer.DeleteMediaImage(imageID, respChan)
	err = <- respChan
	if err != nil{
		resp.SetError(err.Error())
		log.Printf("[%08X] delete media image fail: %s", id, err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	//deallocated
	resp.SetSuccess(true)
	log.Printf("[%08X] media image '%s' deleted", id, imageID)
	return executor.Sender.SendMessage(resp, request.GetSender())
}
