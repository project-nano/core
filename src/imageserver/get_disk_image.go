package imageserver

import (
	"github.com/project-nano/framework"
	"log"
)

type GetDiskImageExecutor struct {
	Sender      framework.MessageSender
	ImageServer *ImageManager
}


func (executor *GetDiskImageExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	imageID, err := request.GetString(framework.ParamKeyImage)
	if err != nil{
		return err
	}
	var respChan = make(chan ImageResult, 1)
	executor.ImageServer.GetDiskImage(imageID, respChan)
	var result = <- respChan
	resp, _ := framework.CreateJsonMessage(framework.GetDiskImageResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	if result.Error != nil{
		err := result.Error
		log.Printf("[%08X] get disk image fail: %s", id, err.Error())
		resp.SetError(err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}
	var image = result.DiskImage
	resp.SetSuccess(true)
	resp.SetString(framework.ParamKeyName, image.Name)
	resp.SetString(framework.ParamKeyDescription, image.Description)
	resp.SetStringArray(framework.ParamKeyTag, image.Tags)
	resp.SetString(framework.ParamKeyUser, image.Owner)
	resp.SetString(framework.ParamKeyGroup, image.Group)

	resp.SetUInt(framework.ParamKeySize, uint(image.Size))
	resp.SetUInt(framework.ParamKeyProgress, image.Progress)

	resp.SetBoolean(framework.ParamKeyEnable, image.Created)
	return executor.Sender.SendMessage(resp, request.GetSender())
}
