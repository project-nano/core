package task

import (
	"github.com/project-nano/framework"
	"modules"
	"net/http"
	"strconv"
	"fmt"
	"log"
	"time"
)

type ResetGuestSystemExecutor struct {
	Sender         framework.MessageSender
	ResourceModule modules.ResourceModule
	Client         *http.Client
}

func (executor *ResetGuestSystemExecutor)Execute(id framework.SessionID, request framework.Message,
	incoming chan framework.Message, terminate chan bool) (err error) {
	var guestID, imageID string
	if guestID, err = request.GetString(framework.ParamKeyGuest); err != nil{
		return 
	}
	if imageID, err = request.GetString(framework.ParamKeyImage); err != nil{
		return 
	}
	 
	resp, _ := framework.CreateJsonMessage(framework.ResetSystemResponse)
	resp.SetToSession(request.GetFromSession())
	resp.SetFromSession(id)
	resp.SetSuccess(false)
	log.Printf("[%08X] recv reset system of '%s' to '%s' from %s.[%08X]", 
		id, guestID, imageID, request.GetSender(), request.GetFromSession())

	var fromSession = request.GetFromSession()
	var respChan = make(chan modules.ResourceResult, 1)
	var instanceName, targetCell string
	{
		//check instance status
		executor.ResourceModule.GetInstanceStatus(guestID, respChan)
		var result = <- respChan
		if result.Error != nil{
			log.Printf("[%08X] check instance status fail: %s", id, result.Error.Error())
			return executor.ResponseFail(resp, result.Error.Error(), request.GetSender())
		}
		var instanceStatus = result.InstanceStatus
		if instanceStatus.Running{
			err = fmt.Errorf("guest '%s' still running", instanceStatus.Name)
			log.Printf("[%08X] check instance status fail: %s", id, result.Error.Error())
			return executor.ResponseFail(resp, result.Error.Error(), request.GetSender())			
		}
		instanceName = instanceStatus.Name
		targetCell = instanceStatus.Cell
	}
	
	{
		//get image server
		var imageServer, mediaHost string
		var mediaPort int
		{
			executor.ResourceModule.GetImageServer(respChan)
			var result = <- respChan
			if result.Error != nil{
				log.Printf("[%08X] get image server fail: %s", id, result.Error.Error())
				return executor.ResponseFail(resp, result.Error.Error(), request.GetSender())
			}
			imageServer = result.Name
			mediaHost = result.Host
			mediaPort = result.Port
		}
		{
			query, _ := framework.CreateJsonMessage(framework.GetDiskImageRequest)
			query.SetFromSession(id)
			query.SetString(framework.ParamKeyImage, imageID)
			if err = executor.Sender.SendMessage(query, imageServer); err != nil{
				log.Printf("[%08X] get image info fail: %s", id, err.Error())
				resp.SetError(err.Error())
				return executor.ResponseFail(resp, err.Error(), request.GetSender())
			}

			var imageName string
			var imageSize uint
			var imageCreated bool

			timer := time.NewTimer(modules.DefaultOperateTimeout)
			select{
			case queryResp := <- incoming:
				if !queryResp.IsSuccess(){
					log.Printf("[%08X] get image info fail: %s", id, queryResp.GetError())
					resp.SetError(queryResp.GetError())
					return executor.ResponseFail(resp, queryResp.GetError(), request.GetSender())
				}
				imageName, _ = queryResp.GetString(framework.ParamKeyName)
				imageSize, _ = queryResp.GetUInt(framework.ParamKeySize)
				imageCreated, _ = queryResp.GetBoolean(framework.ParamKeyEnable)

			case <- timer.C:
				//timeout
				log.Printf("[%08X] get image info timeout", id)
				resp.SetError("time out")
				return executor.ResponseFail(resp, err.Error(), request.GetSender())
			}

			if !imageCreated{
				err = fmt.Errorf("disk image '%s' not created", imageID)
				log.Printf("[%08X] check disk image status fail: %s", id, err.Error())
				return executor.ResponseFail(resp, err.Error(), request.GetSender())
			}

			log.Printf("[%08X] using disk image '%s'(%d MB) at server '%s'(%s:%d) to reset guest '%s'", id, imageName, imageSize >> 20,
				imageServer, mediaHost, mediaPort, instanceName)
			request.SetString(framework.ParamKeyHost, mediaHost)
			request.SetUInt(framework.ParamKeyPort, uint(mediaPort))
			request.SetUInt(framework.ParamKeySize, imageSize)
		}
		//forward request
		request.SetFromSession(id)
		if err = executor.Sender.SendMessage(request, targetCell); err != nil{
			log.Printf("[%08X] forward request to cell '%s' fail: %s", id, targetCell, err.Error())
			return executor.ResponseFail(resp, err.Error(), request.GetSender())
		}
	}
	{
		//wait reset start
		timer := time.NewTimer(modules.DefaultOperateTimeout)
		select{
		case cellResp := <- incoming:
			if cellResp.IsSuccess(){
				log.Printf("[%08X] reset guest system started", id)
				var errorChan = make(chan error, 1)
				executor.ResourceModule.BeginResetSystem(guestID, errorChan)
				err = <- errorChan
				if err != nil{
					log.Printf("[%08X] update reset status fail: %s", id, err.Error())
					return executor.ResponseFail(resp, err.Error(), request.GetSender())					
				}
			}else{
				log.Printf("[%08X] cell reset guest system fail: %s", id, cellResp.GetError())
			}
			cellResp.SetFromSession(id)
			cellResp.SetToSession(fromSession)
			//forward
			return executor.Sender.SendMessage(cellResp, request.GetSender())
		case <- timer.C:
			//timeout
			log.Printf("[%08X] wait reset response timeout", id)
			return executor.ResponseFail(resp, "timeout", request.GetSender())
		}
		
	}
	return nil
}

func (executor *ResetGuestSystemExecutor) getImageSize(id, host string, port int) (size uint64, err error){
	const (
		Protocol = "https"
		Resource = "disk_image_files"
		LengthHeaderName = "Content-Length"
	)
	var fileURL = fmt.Sprintf("%s://%s:%d/%s/%s", Protocol, host, port, Resource, id)
	resp, err := executor.Client.Head(fileURL)
	if err != nil{
		return
	}
	defer resp.Body.Close()
	intValue, err := strconv.Atoi(resp.Header.Get(LengthHeaderName))
	if err != nil{
		err = fmt.Errorf("invalid length '%s'", resp.Header.Get(LengthHeaderName))
		return
	}
	return uint64(intValue), nil
}

func (executor *ResetGuestSystemExecutor)ResponseFail(resp framework.Message, err , target string) error{
	resp.SetSuccess(false)
	resp.SetError(err)
	return executor.Sender.SendMessage(resp, target)
}