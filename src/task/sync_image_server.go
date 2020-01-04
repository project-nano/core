package task

import (
	"github.com/project-nano/framework"
	"github.com/project-nano/core/modules"
	"net/http"
	"log"
)

type SyncImageServerExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
	Client         *http.Client
}

func (executor *SyncImageServerExecutor) Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	const (
		Protocol = "https"
	)
	var serverName, mediaHost string
	var mediaPort uint
	if serverName, err = request.GetString(framework.ParamKeyName);err != nil{
		log.Printf("[%08X] sync image server fail: %s", id, err.Error())
		return nil
	}
	if mediaHost, err = request.GetString(framework.ParamKeyHost);err != nil{
		log.Printf("[%08X] sync image server fail: %s", id, err.Error())
		return nil
	}
	if mediaPort, err = request.GetUInt(framework.ParamKeyPort);err != nil{
		log.Printf("[%08X] sync image server fail: %s", id, err.Error())
		return nil
	}
	executor.ResourceModule.AddImageServer(serverName, mediaHost, int(mediaPort))
	log.Printf("[%08X] new imager server '%s' available (%s:%d)", id, serverName, mediaHost, mediaPort)

	return nil
}
