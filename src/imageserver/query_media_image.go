package imageserver

import (
	"github.com/project-nano/framework"
	"log"
)

type QueryMediaImageExecutor struct {
	Sender      framework.MessageSender
	ImageServer *ImageManager
}


func (executor *QueryMediaImageExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var filterOwner, filterGroup string
	filterOwner, _ = request.GetString(framework.ParamKeyUser)
	filterGroup, _ = request.GetString(framework.ParamKeyGroup)

	var respChan = make(chan ImageResult, 1)
	executor.ImageServer.QueryMediaImage(filterOwner, filterGroup, respChan)

	var result = <- respChan

	resp, _ := framework.CreateJsonMessage(framework.QueryMediaImageResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	if result.Error != nil{
		err = result.Error
		resp.SetError(err.Error())
		log.Printf("[%08X] query media image fail: %s", id, err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}

	var name, imageID, description, tags, createTime, modifyTime []string
	var size, tagCount[]uint64
	for _, image := range result.MediaList {
		name = append(name, image.Name)
		imageID = append(imageID, image.ID)
		description = append(description, image.Description)
		size = append(size, uint64(image.Size))
		count := uint64(len(image.Tags))
		tagCount = append(tagCount, count)
		for _, tag := range image.Tags{
			tags = append(tags, tag)
		}
		createTime = append(createTime, image.CreateTime)
		modifyTime = append(modifyTime, image.ModifyTime)
	}

	resp.SetSuccess(true)
	resp.SetStringArray(framework.ParamKeyName, name)
	resp.SetStringArray(framework.ParamKeyImage, imageID)
	resp.SetStringArray(framework.ParamKeyDescription, description)
	resp.SetStringArray(framework.ParamKeyTag, tags)
	resp.SetStringArray(framework.ParamKeyCreate, createTime)
	resp.SetStringArray(framework.ParamKeyModify, modifyTime)

	resp.SetUIntArray(framework.ParamKeySize, size)
	resp.SetUIntArray(framework.ParamKeyCount, tagCount)
	//log.Printf("[%08X] query media image success, %d image(s) available", id, len(result.MediaList))
	return executor.Sender.SendMessage(resp, request.GetSender())

}