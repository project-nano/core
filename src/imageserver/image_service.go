package imageserver

import (
	"github.com/project-nano/framework"
	"fmt"
	"log"
)

type ImageService struct {
	framework.EndpointService //base class
	ConfigPath   string
	DataPath     string
	httpModule   *HttpModule
	imageManager *ImageManager
	taskManager  *TaskManager
}

func (service *ImageService) GetImageServiceAddress() string{
	if nil == service.httpModule{
		return ""
	}
	return fmt.Sprintf("%s:%d", service.httpModule.GetHost(), service.httpModule.GetPort())
}

func (service *ImageService) GetTLSFilePath() (cert, key string){
	if nil == service.httpModule{
		return "", ""
	}
	return service.httpModule.GetCertFilePath(), service.httpModule.GetKeyFilePath()
}

func (service *ImageService) OnMessageReceived(msg framework.Message){
	if targetSession := msg.GetToSession(); targetSession != 0{
		if err := service.taskManager.PushMessage(msg);err != nil{
			log.Printf("<image> push message [%08X] from %s to session [%08X] fail: %s", msg.GetID(), msg.GetSender(), targetSession, err.Error())
		}
		return
	}

	var err error
	switch msg.GetID() {
	case framework.QueryDiskImageRequest:
	case framework.GetDiskImageRequest:
	case framework.CreateDiskImageRequest:
	case framework.DeleteDiskImageRequest:
	case framework.ModifyDiskImageRequest:

	case framework.QueryMediaImageRequest:
	case framework.GetMediaImageRequest:
	case framework.CreateMediaImageRequest:
	case framework.DeleteMediaImageRequest:
	case framework.ModifyMediaImageRequest:
	case framework.DiskImageUpdatedEvent:

	default:
		log.Printf("<image> message [%08X] from %s.[%08X] ignored", msg.GetID(), msg.GetSender(), msg.GetFromSession())
		return
	}
	//Invoke transaction
	err = service.taskManager.InvokeTask(msg)
	if err != nil{
		log.Printf("<image> invoke transaction with message [%08X] fail: %s", msg.GetID(), err.Error())
	}
}

func (service *ImageService) OnServiceConnected(nodeName string, t framework.ServiceType, address string){
	switch t {
	case framework.ServiceTypeCore:
		event, _ := framework.CreateJsonMessage(framework.ImageServerAvailableEvent)
		event.SetString(framework.ParamKeyName, service.GetName())
		event.SetString(framework.ParamKeyHost, service.httpModule.GetHost())
		event.SetUInt(framework.ParamKeyPort, uint(service.httpModule.GetPort()))
		if err := service.SendMessage(event, nodeName); err != nil{
			log.Printf("<image> warning: notify image available fail: %s", err.Error())
		}else{
			log.Printf("<image> notify image address '%s:%d' to %s", service.httpModule.GetHost(), service.httpModule.GetPort(), nodeName)
		}
	default:
		break
	}
}

func (service *ImageService) OnServiceDisconnected(name string, t framework.ServiceType, gracefully bool){

}

func (service *ImageService) OnDependencyReady(){

}

func (service *ImageService) InitialEndpoint() (err error){
	service.imageManager, err = CreateImageManager(service.DataPath)
	if err != nil{
		return
	}
	service.taskManager, err = CreateTaskManager(service, service.imageManager)
	if err != nil{
		return
	}
	return nil
}

func (service *ImageService) OnEndpointStarted() (err error){
	service.httpModule, err = CreateHttpModule(service.ConfigPath, service.DataPath, service.GetListenAddress(), service.imageManager)
	if err != nil{
		return
	}

	if err = service.imageManager.Start(); err != nil{
		return
	}
	if err = service.taskManager.Start(); err != nil{
		return
	}
	if err = service.httpModule.Start(); err != nil{
		return
	}
	log.Print("<image> all service started")
	return nil
}

func (service *ImageService) OnEndpointStopped(){
	var err error
	if err = service.httpModule.Stop(); err != nil{
		log.Printf("<image> warning: stop http module fail: %s", err.Error())
	}
	if err = service.taskManager.Stop(); err != nil{
		log.Printf("<image> warning: stop task manager fail: %s", err.Error())
	}
	if err = service.imageManager.Stop(); err != nil{
		log.Printf("<image> warning: stop image manager fail: %s", err.Error())
	}
	log.Print("<image> all service stopped")
}
