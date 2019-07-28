package imageserver

import (
	"github.com/project-nano/framework"
	"log"
)

type QueryDiskImageExecutor struct {
	Sender      framework.MessageSender
	ImageServer *ImageManager
}


func (executor *QueryDiskImageExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {

	filterOwner, _ := request.GetString(framework.ParamKeyUser)
	filterGroup, _ := request.GetString(framework.ParamKeyGroup)
	filterTags, _ := request.GetStringArray(framework.ParamKeyTag)

	var respChan = make(chan ImageResult, 1)
	executor.ImageServer.QueryDiskImage(filterOwner, filterGroup, filterTags, respChan)

	var result = <- respChan

	resp, _ := framework.CreateJsonMessage(framework.QueryDiskImageResponse)
	resp.SetSuccess(false)
	resp.SetFromSession(id)
	resp.SetToSession(request.GetFromSession())

	if result.Error != nil{
		err = result.Error
		resp.SetError(err.Error())
		log.Printf("[%08X] query disk image fail: %s", id, err.Error())
		return executor.Sender.SendMessage(resp, request.GetSender())
	}

	var name, imageID, description, tags, createTime, modifyTime []string
	var size, tagCount, created, progress []uint64
	for _, image := range result.DiskList {
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
		if image.Created{
			created = append(created, 1)
		}else{
			created = append(created, 0)
		}
		progress = append(progress, uint64(image.Progress))
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
	resp.SetUIntArray(framework.ParamKeyStatus, created)
	resp.SetUIntArray(framework.ParamKeyProgress, progress)
	log.Printf("[%08X] query disk image success, %d image(s) available", id, len(result.DiskList))
	return executor.Sender.SendMessage(resp, request.GetSender())

}
