package imageserver

import (
	"github.com/project-nano/framework"
	"log"
	"errors"
)

type DiskImageUpdateExecutor struct {
	Sender      framework.MessageSender
	ImageServer *ImageManager
}

func (executor *DiskImageUpdateExecutor)Execute(id framework.SessionID, event framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	imageID, err := event.GetString(framework.ParamKeyImage)
	if err != nil{
		return
	}
	if !event.IsSuccess(){
		err = errors.New(event.GetError())
		log.Printf("[%08X] update disk image progress fail: %s", id, err.Error())
		executor.releaseImage(id, imageID)
		return err
	}
	var created bool
	var progress uint
	if created, err = event.GetBoolean(framework.ParamKeyEnable);err != nil{
		log.Printf("[%08X] parse created status from disk image updated fail: %s", id, err.Error())
		executor.releaseImage(id, imageID)
		return err
	}
	if progress, err = event.GetUInt(framework.ParamKeyProgress);err != nil{
		log.Printf("[%08X] parse progress from disk image updated fail: %s", id, err.Error())
		executor.releaseImage(id, imageID)
		return err
	}
	if created{
		//finished
		var imageSize uint
		if imageSize, err = event.GetUInt(framework.ParamKeySize); err != nil{
			log.Printf("[%08X] parse image size from disk image updated fail: %s", id, err.Error())
			executor.releaseImage(id, imageID)
			return err
		}
		log.Printf("[%08X] disk image creation finished, %d MB in size", id, imageSize >> 20)

	}else{
		var respChan = make(chan error, 1)
		executor.ImageServer.UpdateDiskImageProgress(imageID, progress, respChan)
		err = <- respChan
		if err != nil{
			log.Printf("[%08X] update disk image progress fail: %s", id, err.Error())
			return err
		}
		//log.Printf("[%08X] progress %d %%", id, progress)
	}
	return nil
}

func (executor *DiskImageUpdateExecutor) releaseImage(id framework.SessionID, imageID string){
	var respChan = make(chan error, 1)
	executor.ImageServer.DeleteDiskImage(imageID, respChan)
	var err = <- respChan
	if err != nil{
		log.Printf("[%08X] delete disk image fail: %s", id, imageID)
	}else {
		log.Printf("[%08X] disk image '%s' deleted", id, imageID)
	}
}