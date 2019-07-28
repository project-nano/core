package imageserver

import (
	"github.com/project-nano/framework"
)

type TaskManager struct {
	*framework.TransactionEngine
}

func CreateTaskManager(sender framework.MessageSender, imageManager *ImageManager) (*TaskManager, error) {
	engine, err := framework.CreateTransactionEngine()
	if err != nil {
		return nil, err
	}

	var manager= TaskManager{engine}

	if err = manager.RegisterExecutor(framework.QueryMediaImageRequest,
		&QueryMediaImageExecutor{sender, imageManager}); err != nil{
			return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetMediaImageRequest,
		&GetMediaImageExecutor{sender, imageManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.CreateMediaImageRequest,
		&CreateMediaImageExecutor{sender, imageManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ModifyMediaImageRequest,
		&ModifyMediaImageExecutor{sender, imageManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.DeleteMediaImageRequest,
		&DeleteMediaImageExecutor{sender, imageManager}); err != nil{
		return nil, err
	}

	if err = manager.RegisterExecutor(framework.QueryDiskImageRequest,
		&QueryDiskImageExecutor{sender, imageManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.GetDiskImageRequest,
		&GetDiskImageExecutor{sender, imageManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.CreateDiskImageRequest,
		&CreateDiskImageExecutor{sender, imageManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.ModifyDiskImageRequest,
		&ModifyDiskImageExecutor{sender, imageManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.DeleteDiskImageRequest,
		&DeleteDiskImageExecutor{sender, imageManager}); err != nil{
		return nil, err
	}
	if err = manager.RegisterExecutor(framework.DiskImageUpdatedEvent,
		&DiskImageUpdateExecutor{sender, imageManager}); err != nil{
		return nil, err
	}

	return &manager, nil
}
