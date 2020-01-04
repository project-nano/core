package modules

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/project-nano/framework"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type APIModule struct {
	server           http.Server
	exitChan         chan bool
	currentImageHost string
	currentImageURL  string
	currentImageProxy *httputil.ReverseProxy
	apiCredentials map[string]string
	proxy             *RequestProxy
	resource          ResourceModule
}

type ApiCredential struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

type APIConfig struct {
	Port        int             `json:"port"`
	Credentials []ApiCredential `json:"credentials"`
}

const (
	HeaderNameHost          = "Host"
	//HeaderNameContentType   = "Content-Type"
	//HeaderNameSession       = "Nano-Session"
	HeaderNameDate          = "Nano-Date"
	HeaderNameScope         = "Nano-Scope"
	HeaderNameAuthorization = "Nano-Authorization"
	APIRoot                 = "/api"
	APIVersion              = 1
)

func CreateAPIModule(configPath string, sender framework.MessageSender, resourceModule ResourceModule) (module *APIModule, err error) {
	//load config
	const (
		configFilename = "api.cfg"
	)
	var configFile = filepath.Join(configPath, configFilename)
	var config APIConfig
	var data []byte
	if data, err = ioutil.ReadFile(configFile); err != nil {
		return
	}
	if err = json.Unmarshal(data, &config); err != nil {
		return
	}
	if 0 == len(config.Credentials){
		const (
			dummyID  = "dummyID"
			dummyKey = "ThisIsAKeyPlaceHolder_ChangeToYourContent"
		)
		config.Credentials = []ApiCredential{{ID: dummyID, Key: dummyKey}}
		if data, err = json.MarshalIndent(config, "", " "); err != nil{
			err = fmt.Errorf("marshal new config fail: %s", err.Error())
			return
		}
		var file *os.File
		if file, err = os.Create(configFile); err != nil{
			err = fmt.Errorf("create new config fail: %s", err.Error())
			return
		}
		defer file.Close()
		if _, err = file.Write(data); err != nil{
			err = fmt.Errorf("write new config fail: %s", err.Error())
			return
		}
		log.Printf("<api> warning: dummy API credential '%s' created", dummyID)
	}
	var proxy *RequestProxy

	if proxy, err = CreateRequestProxy(sender); err != nil {
		return
	}
	var listenAddress = fmt.Sprintf(":%d", config.Port)
	module = &APIModule{}
	module.apiCredentials = map[string]string{}

	for _, credential := range config.Credentials{
		if 0 == len(credential.ID){
			err = errors.New("empty API ID")
			return
		}
		if 0 == len(credential.Key){
			err = fmt.Errorf("empty API key for '%s'", credential.ID)
			return
		}
		module.apiCredentials[credential.ID] = credential.Key
	}
	module.exitChan = make(chan bool)
	module.proxy = proxy
	module.server.Addr = listenAddress
	var router = httprouter.New()
	module.RegisterAPIHandler(router)
	log.Println("register finish")
	module.server.Handler = router
	module.resource = resourceModule
	log.Printf("<api> config loaded from %s, listen port %d, %d API credentials available ",
		configFile, config.Port, len(module.apiCredentials))
	return
}

func (module *APIModule) GetServiceAddress() string{
	return module.server.Addr
}

func (module *APIModule) GetModuleName() string {
	return module.proxy.Module
}
func (module *APIModule) GetResponseChannel() chan framework.Message {
	return module.proxy.ResponseChan
}

func (module *APIModule) Start() error {
	if err := module.proxy.Start(); err != nil {
		return err
	}
	go module.routine()
	return nil
}

func (module *APIModule) Stop() error {
	module.server.Close()
	<-module.exitChan
	return module.proxy.Stop()
}

func (module *APIModule) routine() {
	log.Println("<api> module started")
	if err := module.server.ListenAndServe(); err != nil {
		log.Printf("<api> module stopped: %s", err.Error())
	}
	module.exitChan <- true
}

func apiPath(path string) string{
	return fmt.Sprintf("%s/v%d%s", APIRoot, APIVersion, path)
}

func (module *APIModule) verifyRequestSignature(r *http.Request) (err error){
	const (
		SignatureMethodHMAC256 = "Nano-HMAC-SHA256"
		ShortDateFormat        = "20060102"
	)
	r.Header.Set(HeaderNameHost, r.Host)
	var apiID, apiKey, requestScope, signedHeaders, signature, signatureMethod string
	{
		//Method Credential=id/scope, SignedHeaders=headers, Signature=signatures
		//check authorization
		var authorization = r.Header.Get(HeaderNameAuthorization)
		var length = len(authorization)
		if 0 == length{
			err = errors.New("authorization required")
			return
		}
		if length <= len(SignatureMethodHMAC256){
			err = fmt.Errorf("insufficent authorization: %s", authorization)
			return
		}
		signatureMethod = authorization[:len(SignatureMethodHMAC256)]
		if SignatureMethodHMAC256 != signatureMethod {
			err = fmt.Errorf("invalid signature method: %s", signatureMethod)
			return
		}
		var names, values []string
		for _, token := range strings.Split(authorization[len(SignatureMethodHMAC256) + 1:], ","){
			var split = strings.SplitN(token, "=", 2)
			if 2 != len(split){
				err = fmt.Errorf("invalid authorization token: %s", token)
				return
			}
			names = append(names, strings.Trim(split[0], " "))
			values = append(values, strings.Trim(split[1], " "))
		}
		const (
			TokenCount         = 3
			TokenCredential    = "Credential"
			TokenSignedHeaders = "SignedHeaders"
			TokenSignature     = "Signature"
		)
		if TokenCount != len(names) || TokenCount != len(values){
			err = fmt.Errorf("unexpected token count %d/%d", len(names), len(values))
			return
		}
		if TokenCredential != names[0]{
			err = fmt.Errorf("invalid first token %s", names[0])
			return
		}
		if TokenSignedHeaders != names[1]{
			err = fmt.Errorf("invalid second token %s", names[1])
			return
		}
		if TokenSignature != names[2]{
			err = fmt.Errorf("invalid third token %s", names[2])
			return
		}
		var idTail = strings.IndexByte(values[0], '/')
		if -1 == idTail{
			err = fmt.Errorf("no API ID in credential: %s", values[0])
			return
		}
		apiID = values[0][:idTail]
		var exists bool
		if apiKey, exists = module.apiCredentials[apiID]; !exists{
			err = fmt.Errorf("invalid API ID: %s", apiID)
			return
		}
		requestScope = values[0][idTail + 1:]
		signedHeaders = values[1]
		signature = values[2]
	}
	var canonicalRequest, requestDate, stringToSign string
	{
		//canonicalRequest
		var canonicalURI = url.QueryEscape(url.QueryEscape(r.URL.Path))
		var canonicalQueryString string
		if 0 != len(r.URL.Query()){
			var paramNames []string
			for key := range r.URL.Query(){
				paramNames = append(paramNames, key)
			}
			sort.Sort(sort.StringSlice(paramNames))
			var queryParams []string
			for _, name := range paramNames{
				queryParams = append(queryParams,
					fmt.Sprintf("%s=%s", url.QueryEscape(name), url.QueryEscape(r.URL.Query().Get(name))))
			}
			canonicalQueryString = strings.Join(queryParams, "&")
		}else{
			canonicalQueryString = ""
		}
		var headerIndexes = map[string]int{}
		var targetHeaders = strings.Split(signedHeaders, ";")
		for index, name := range targetHeaders{
			headerIndexes[name] = index
		}
		var signedHeaderToken = make([]string, len(signedHeaders))

		//extract signed headers
		for name, _ := range r.Header{
			var lowerName = strings.ToLower(name)
			if index, exists := headerIndexes[lowerName]; exists{
				//signed header
				signedHeaderToken[index] = fmt.Sprintf("%s:%s\n",
					lowerName, strings.Trim(r.Header.Get(name), " "))
			}
			if HeaderNameDate == name{
				requestDate = r.Header.Get(name)
				var requestTime time.Time
				if requestTime, err = time.Parse(time.RFC3339, requestDate); err != nil{
					err = fmt.Errorf("invalid request date: %s", requestDate)
				}
				//must on same day
				if time.Now().Format(ShortDateFormat) != requestTime.Format(ShortDateFormat){
					err = fmt.Errorf("expired request with date %s", requestDate)
					return
				}
			}
			if HeaderNameScope == name{
				var scope = r.Header.Get(name)
				if scope != requestScope{
					err = fmt.Errorf("request scope mismatch: %s => %s", scope, requestScope)
					return
				}
			}
		}
		var canonicalHeaders string
		var headersBuilder strings.Builder
		for _, token := range signedHeaderToken{
			headersBuilder.WriteString(token)
		}
		canonicalHeaders = headersBuilder.String()
		//hash with sha256
		var hash = sha256.New()
		if http.MethodGet == r.Method || http.MethodHead == r.Method || http.MethodOptions == r.Method{
			hash.Write([]byte(""))
		}else {
			//clone request payload
			var payload []byte
			if payload, err = ioutil.ReadAll(r.Body); err != nil{
				return
			}
			hash.Write(payload)
			r.Body = ioutil.NopCloser(bytes.NewBuffer(payload))
		}
		var hashedPayload = strings.ToLower(hex.EncodeToString(hash.Sum(nil)))
		var canonicalRequestContent = strings.Join([]string{
			canonicalURI,
			canonicalQueryString,
			canonicalHeaders,
			signedHeaders,
			hashedPayload,
		}, "\n")
		hash.Reset()
		hash.Write([]byte(canonicalRequestContent))
		canonicalRequest = hex.EncodeToString(hash.Sum(nil))
	}
	{
		stringToSign = strings.Join([]string{
			signatureMethod,
			requestDate,
			requestScope,
			canonicalRequest,
		}, "\n")
	}
	var signKey []byte
	{
		var builder strings.Builder
		builder.WriteString("nano")
		builder.WriteString(apiKey)

		var key = []byte(builder.String())
		var data = []byte(requestScope)
		if signKey, err = computeHMACSha256(key, data); err != nil{
			err = fmt.Errorf("compute signature key fail: %s", err.Error())
			return
		}
		var hmacSignature []byte
		if hmacSignature, err = computeHMACSha256(signKey, []byte(stringToSign)); err != nil{
			err = fmt.Errorf("compute signature fail: %s", err.Error())
			return
		}
		var expectedSignature = hex.EncodeToString(hmacSignature)
		if signature != expectedSignature{
			err = errors.New("signature corrupted")
			return
		}
	}

	return nil
}

func computeHMACSha256(key, data []byte) (hash []byte, err error){
	var h = hmac.New(sha256.New, key)
	if _, err = h.Write(data); err != nil{
		return
	}
	hash = h.Sum(nil)
	return
}

func (module *APIModule) RegisterAPIHandler(router *httprouter.Router) {
	router.GET(apiPath("/compute_pools/"), module.handleQueryAllPools)
	router.GET(apiPath("/compute_pools/:pool"), module.handleGetComputePool)
	router.POST(apiPath("/compute_pools/:pool"), module.handleCreateComputePool)
	router.PUT(apiPath("/compute_pools/:pool"), module.handleModifyComputePool)
	router.DELETE(apiPath("/compute_pools/:pool"), module.handleDeleteComputePool)

	router.POST(apiPath("/compute_pool_cells/:pool/:cell"), module.handleAddComputeCell)
	router.DELETE(apiPath("/compute_pool_cells/:pool/:cell"), module.handleRemoveComputeCell)
	router.PUT(apiPath("/compute_pool_cells/:pool/:cell"), module.handleModifyComputeCell)
	router.GET(apiPath("/compute_pool_cells/"), module.handleQueryUnallocatedCell)
	router.GET(apiPath("/compute_pool_cells/:pool"), module.handleQueryCellsInPool)
	router.GET(apiPath("/compute_pool_cells/:pool/:cell"), module.handleGetComputeCell)

	router.GET(apiPath("/storage_pools/"), module.handleQueryStoragePool)
	router.GET(apiPath("/storage_pools/:pool"), module.handleGetStoragePool)
	router.POST(apiPath("/storage_pools/:pool"), module.handleCreateStoragePool)
	router.PUT(apiPath("/storage_pools/:pool"), module.handleModifyStoragePool)
	router.DELETE(apiPath("/storage_pools/:pool"), module.handleDeleteStoragePool)

	router.GET(apiPath("/compute_zone_status/"), module.queryZoneStatistic)
	router.GET(apiPath("/compute_pool_status/"), module.queryComputePoolsStatus)
	router.GET(apiPath("/compute_pool_status/:pool"), module.getComputePoolStatus)
	router.GET(apiPath("/compute_cell_status/:pool"), module.queryComputeCellStatus)
	router.GET(apiPath("/compute_cell_status/:pool/:cell"), module.getComputeCellStatus)

	router.GET(apiPath("/instance_status/:pool"), module.handleQueryInstanceStatusInPool)
	router.GET(apiPath("/instance_status/:pool/:cell"), module.handleQueryInstanceStatusInCell)

	router.GET(apiPath("/guests/:id"), module.handleGetGuestConfig)
	router.POST(apiPath("/guests/"), module.handleCreateGuest)
	router.DELETE(apiPath("/guests/:id"), module.handleDeleteGuest)

	router.PUT(apiPath("/guests/:id/name/"), module.handleModifyGuestName)
	router.PUT(apiPath("/guests/:id/cores"), module.handleModifyGuestCores)
	router.PUT(apiPath("/guests/:id/memory"), module.handleModifyGuestMemory)
	router.PUT(apiPath("/guests/:id/qos/cpu"), module.handleModifyGuestPriority)
	router.PUT(apiPath("/guests/:id/qos/disk"), module.handleModifyDiskThreshold)
	router.PUT(apiPath("/guests/:id/qos/network"), module.handleModifyNetworkThreshold)
	router.PUT(apiPath("/guests/:id/system/"), module.handleResetGuestSystem)
	router.PUT(apiPath("/guests/:id/auth"), module.handleModifyGuestPassword)
	router.GET(apiPath("/guests/:id/auth"), module.handleGetGuestPassword)
	router.PUT(apiPath("/guests/:id/disks/resize/:index"), module.handleResizeDisk)
	router.PUT(apiPath("/guests/:id/disks/shrink/:index"), module.handleShrinkDisk)

	router.GET(apiPath("/instances/:id"), module.handleGetInstanceStatus)
	router.POST(apiPath("/instances/:id"), module.handleStartInstance)
	router.DELETE(apiPath("/instances/:id"), module.handleStopInstance)

	router.GET(apiPath("/guest_search/*filepath"), module.handleQueryGuestConfig)

	//media image
	router.GET(apiPath("/media_image_search/*filepath"), module.searchMediaImage)
	router.GET(apiPath("/media_images/"), module.queryAllMediaImage)
	router.GET(apiPath("/media_images/:id"), module.getMediaImage)
	router.POST(apiPath("/media_images/"), module.createMediaImage)
	router.PUT(apiPath("/media_images/:id"), module.modifyMediaImage)
	router.DELETE(apiPath("/media_images/:id"), module.deleteMediaImage)


	router.POST(apiPath("/media_images/:id/file/"), module.redirectToImageServer)

	//disk image
	router.GET(apiPath("/disk_image_search/*filepath"), module.queryDiskImage)

	router.GET(apiPath("/disk_images/:id"), module.getDiskImage)
	router.POST(apiPath("/disk_images/"), module.createDiskImage)
	router.PUT(apiPath("/disk_images/:id"), module.modifyDiskImage)
	router.DELETE(apiPath("/disk_images/:id"), module.deleteDiskImage)

	router.GET(apiPath("/disk_images/:id/file/"), module.redirectToImageServer)
	router.POST(apiPath("/disk_images/:id/file/"), module.redirectToImageServer)//upload from web

	router.POST(apiPath("/instances/:id/media"), module.handleInsertMedia)
	router.DELETE(apiPath("/instances/:id/media"), module.handleEjectMedia)

	//snapshots
	router.GET(apiPath("/instances/:id/snapshots/"), module.handleQueryInstanceSnapshots)
	router.POST(apiPath("/instances/:id/snapshots/"), module.handleCreateInstanceSnapshot)
	router.PUT(apiPath("/instances/:id/snapshots/"), module.handleRestoreInstanceSnapshot)
	router.GET(apiPath("/instances/:id/snapshots/:name"), module.handleGetInstanceSnapshot)
	router.DELETE(apiPath("/instances/:id/snapshots/:name"), module.handleDeleteInstanceSnapshot)

	//migrations
	router.GET(apiPath("/migrations/"), module.handleQueryMigrations)
	router.GET(apiPath("/migrations/:id"), module.handleGetMigration)
	router.POST(apiPath("/migrations/"), module.handleCreateMigration)

	//address pool
	router.GET(apiPath("/address_pools/"), module.handleQueryAddressPool)
	router.GET(apiPath("/address_pools/:pool"), module.handleGetAddressPool)
	router.POST(apiPath("/address_pools/:pool"), module.handleCreateAddressPool)
	router.PUT(apiPath("/address_pools/:pool"), module.handleModifyAddressPool)
	router.DELETE(apiPath("/address_pools/:pool"), module.handleDeleteAddressPool)

	//address range
	router.GET(apiPath("/address_pools/:pool/:type/ranges/"), module.handleQueryAddressRange)
	router.GET(apiPath("/address_pools/:pool/:type/ranges/:start"), module.handleGetAddressRange)
	router.POST(apiPath("/address_pools/:pool/:type/ranges/:start"), module.handleAddAddressRange)
	router.DELETE(apiPath("/address_pools/:pool/:type/ranges/:start"), module.handleRemoveAddressRange)

	//batch
	router.GET(apiPath("/batch/create_guest/:id"), module.handleGetBatchCreateGuest)
	router.POST(apiPath("/batch/create_guest/"), module.handleStartBatchCreateGuest)
	router.GET(apiPath("/batch/delete_guest/:id"), module.handleGetBatchDeleteGuest)
	router.POST(apiPath("/batch/delete_guest/"), module.handleStartBatchDeleteGuest)
	router.GET(apiPath("/batch/stop_guest/:id"), module.handleGetBatchStopGuest)
	router.POST(apiPath("/batch/stop_guest/"), module.handleStartBatchStopGuest)

}

func (module *APIModule) queryZoneStatistic(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg, _ := framework.CreateJsonMessage(framework.QueryZoneStatusRequest)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> query zone status fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> query zone status fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	type respData struct {
		Name            string   `json:"name"`
		Pools           []uint64 `json:"pools"`
		Cells           []uint64 `json:"cells"`
		Instances       []uint64 `json:"instances"`
		CpuUsage        float64  `json:"cpu_usage"`
		MaxCpu          uint     `json:"max_cpu"`
		AvailableMemory uint64   `json:"available_memory"`
		MaxMemory       uint64   `json:"max_memory"`
		AvailableDisk   uint64   `json:"available_disk"`
		MaxDisk         uint64   `json:"max_disk"`
		ReadSpeed       uint64   `json:"read_speed"`
		WriteSpeed      uint64   `json:"write_speed"`
		ReceiveSpeed    uint64   `json:"receive_speed"`
		SendSpeed       uint64   `json:"send_speed"`
		StartTime       string   `json:"start_time"`
	}

	parser := func(msg framework.Message) (data respData, err error) {
		if data.Name, err = msg.GetString(framework.ParamKeyName); err != nil {
			return data, err
		}
		if data.StartTime, err = msg.GetString(framework.ParamKeyStart); err != nil {
			return data, err
		}
		if data.Pools, err = msg.GetUIntArray(framework.ParamKeyPool); err != nil {
			return data, err
		}
		if data.Cells, err = msg.GetUIntArray(framework.ParamKeyCell); err != nil {
			return data, err
		}
		if data.Instances, err = msg.GetUIntArray(framework.ParamKeyInstance); err != nil {
			return data, err
		}
		if data.CpuUsage, err = msg.GetFloat(framework.ParamKeyUsage); err != nil {
			return data, err
		}
		if data.MaxCpu, err = msg.GetUInt(framework.ParamKeyCore); err != nil {
			return data, err
		}
		const (
			memoryParamCount = 2
			diskParamCount   = 2
			speedParamCount  = 4
		)
		memory, err := msg.GetUIntArray(framework.ParamKeyMemory)
		if err != nil {
			return data, err
		}
		if memoryParamCount != len(memory) {
			return data, fmt.Errorf("invalid memory param count %d", len(memory))
		}
		disk, err := msg.GetUIntArray(framework.ParamKeyDisk)
		if err != nil {
			return data, err
		}
		if diskParamCount != len(disk) {
			return data, fmt.Errorf("invalid disk param count %d", len(disk))
		}
		speed, err := msg.GetUIntArray(framework.ParamKeySpeed)
		if err != nil {
			return data, err
		}
		if speedParamCount != len(speed) {
			return data, fmt.Errorf("invalid speed param count %d", len(speed))
		}
		data.AvailableMemory = memory[0]
		data.MaxMemory = memory[1]
		data.AvailableDisk = disk[0]
		data.MaxDisk = disk[1]

		data.WriteSpeed = speed[0]
		data.ReadSpeed = speed[1]
		data.SendSpeed = speed[2]
		data.ReceiveSpeed = speed[3]
		return data, nil
	}

	data, err := parser(resp)
	if err != nil {
		log.Printf("<api> parse query zone status fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	if err = ResponseOK(data, w); err != nil{
		log.Printf("<api> marshal zone status fail: %s", err.Error())
	}
}

func (module *APIModule) queryComputePoolsStatus(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg, _ := framework.CreateJsonMessage(framework.QueryComputePoolStatusRequest)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> query compute pool status fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> query compute pool status fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	type poolStatus struct {
		Name            string   `json:"name"`
		Enabled         bool     `json:"enabled"`
		Cells           []uint64 `json:"cells"`
		Instances       []uint64 `json:"instances"`
		CpuUsage        float64  `json:"cpu_usage"`
		MaxCpu          uint     `json:"max_cpu"`
		AvailableMemory uint64   `json:"available_memory"`
		MaxMemory       uint64   `json:"max_memory"`
		AvailableDisk   uint64   `json:"available_disk"`
		MaxDisk         uint64   `json:"max_disk"`
		ReadSpeed       uint64   `json:"read_speed"`
		WriteSpeed      uint64   `json:"write_speed"`
		ReceiveSpeed    uint64   `json:"receive_speed"`
		SendSpeed       uint64   `json:"send_speed"`
	}

	parser := func(msg framework.Message) (status []poolStatus, err error) {
		var name []string
		var enabled, cells, instances, usage, cores, memory, disk, speed []uint64
		if name, err = msg.GetStringArray(framework.ParamKeyName); err != nil {
			return
		}
		if enabled, err = msg.GetUIntArray(framework.ParamKeyEnable); err != nil {
			return
		}
		if cells, err = msg.GetUIntArray(framework.ParamKeyCell); err != nil {
			return
		}
		if instances, err = msg.GetUIntArray(framework.ParamKeyInstance); err != nil {
			return
		}
		if usage, err = msg.GetUIntArray(framework.ParamKeyUsage); err != nil {
			return
		}
		if cores, err = msg.GetUIntArray(framework.ParamKeyCore); err != nil {
			return
		}
		if memory, err = msg.GetUIntArray(framework.ParamKeyMemory); err != nil {
			return
		}
		if disk, err = msg.GetUIntArray(framework.ParamKeyDisk); err != nil {
			return
		}
		if speed, err = msg.GetUIntArray(framework.ParamKeySpeed); err != nil {
			return
		}
		const (
			cellParamCount     = 2
			instanceParamCount = 4
			memoryParamCount   = 2
			diskParamCount     = 2
			speedParamCount    = 4
		)
		var count = len(name)
		var cellCount = len(cells)
		var instanceCount = len(instances)
		var memoryCount = len(memory)
		var diskCount = len(disk)
		var speedCount = len(speed)
		if cellCount != count * cellParamCount{
			err = fmt.Errorf("unexpected cell params %d / %d", cellCount, count * cellParamCount)
			return
		}
		if instanceCount != count * instanceParamCount{
			err = fmt.Errorf("unexpected instance params %d / %d", instanceCount, count * instanceParamCount)
			return
		}
		if memoryCount != count * memoryParamCount{
			err = fmt.Errorf("unexpected memory params %d / %d", memoryCount, count * memoryParamCount)
			return
		}
		if diskCount != count * diskParamCount{
			err = fmt.Errorf("unexpected disk params %d / %d", diskCount, count * diskParamCount)
			return
		}
		if speedCount != count * speedParamCount{
			err = fmt.Errorf("unexpected speed params %d / %d", memoryCount, count * speedParamCount)
			return
		}
		for i := 0; i < count; i++{
			var p = poolStatus{Name:name[i]}
			if 1 == enabled[i]{
				p.Enabled = true
			}else{
				p.Enabled = false
			}
			p.Cells = []uint64{cells[i*cellParamCount], cells[i*cellParamCount + 1]}
			p.Instances = []uint64{instances[i*instanceParamCount], instances[i*instanceParamCount + 1], instances[i*instanceParamCount+ 2], instances[i*instanceParamCount + 3]}
			p.CpuUsage = float64(usage[i])
			p.MaxCpu = uint(cores[i])
			p.AvailableMemory = memory[i*memoryParamCount]
			p.MaxMemory = memory[i*memoryParamCount + 1]
			p.AvailableDisk = disk[i*diskParamCount]
			p.MaxDisk = disk[i*diskParamCount + 1]
			p.ReadSpeed = speed[i*speedParamCount]
			p.WriteSpeed = speed[i*speedParamCount + 1]
			p.ReceiveSpeed = speed[i*speedParamCount + 2]
			p.SendSpeed = speed[i*speedParamCount + 3]
			status = append(status, p)
		}
		return
	}

	data, err := parser(resp)
	if err != nil {
		log.Printf("<api> parse query compute pool status fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	ResponseOK(data, w)
}

func (module *APIModule) getComputePoolStatus(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	pool := params.ByName("pool")

	msg, _ := framework.CreateJsonMessage(framework.GetComputePoolStatusRequest)
	msg.SetString(framework.ParamKeyPool, pool)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> query compute pool status fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> query compute pool status fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	type respData struct {
		Name            string   `json:"name"`
		Enabled         bool     `json:"enabled"`
		Cells           []uint64 `json:"cells"`
		Instances       []uint64 `json:"instances"`
		CpuUsage        float64  `json:"cpu_usage"`
		MaxCpu          uint     `json:"max_cpu"`
		AvailableMemory uint64   `json:"available_memory"`
		MaxMemory       uint64   `json:"max_memory"`
		AvailableDisk   uint64   `json:"available_disk"`
		MaxDisk         uint64   `json:"max_disk"`
		ReadSpeed       uint64   `json:"read_speed"`
		WriteSpeed      uint64   `json:"write_speed"`
		ReceiveSpeed    uint64   `json:"receive_speed"`
		SendSpeed       uint64   `json:"send_speed"`
	}

	parser := func(msg framework.Message) (data respData, err error) {
		if data.Name, err = msg.GetString(framework.ParamKeyName); err != nil {
			return data, err
		}
		if data.Enabled, err = msg.GetBoolean(framework.ParamKeyEnable); err != nil {
			return data, err
		}
		if data.Cells, err = msg.GetUIntArray(framework.ParamKeyCell); err != nil {
			return data, err
		}
		if data.Instances, err = msg.GetUIntArray(framework.ParamKeyInstance); err != nil {
			return data, err
		}
		if data.CpuUsage, err = msg.GetFloat(framework.ParamKeyUsage); err != nil {
			return data, err
		}
		if data.MaxCpu, err = msg.GetUInt(framework.ParamKeyCore); err != nil {
			return data, err
		}
		const (
			memoryParamCount = 2
			diskParamCount   = 2
			speedParamCount  = 4
		)
		memory, err := msg.GetUIntArray(framework.ParamKeyMemory)
		if err != nil {
			return data, err
		}
		if memoryParamCount != len(memory) {
			return data, fmt.Errorf("invalid memory param count %d", len(memory))
		}
		disk, err := msg.GetUIntArray(framework.ParamKeyDisk)
		if err != nil {
			return data, err
		}
		if diskParamCount != len(disk) {
			return data, fmt.Errorf("invalid disk param count %d", len(disk))
		}
		speed, err := msg.GetUIntArray(framework.ParamKeySpeed)
		if err != nil {
			return data, err
		}
		if speedParamCount != len(speed) {
			return data, fmt.Errorf("invalid speed param count %d", len(speed))
		}
		data.AvailableMemory = memory[0]
		data.MaxMemory = memory[1]
		data.AvailableDisk = disk[0]
		data.MaxDisk = disk[1]

		data.ReadSpeed = speed[0]
		data.WriteSpeed = speed[1]
		data.ReceiveSpeed = speed[2]
		data.SendSpeed = speed[3]
		return data, nil
	}

	data, err := parser(resp)
	if err != nil {
		log.Printf("<api> parse get compute pool status fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	ResponseOK(data, w)
}

func (module *APIModule) queryComputeCellStatus(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	pool := params.ByName("pool")
	msg, _ := framework.CreateJsonMessage(framework.QueryComputePoolCellStatusRequest)
	msg.SetString(framework.ParamKeyPool, pool)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> query compute cell status fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> query compute cell status fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	type cellStatus struct {
		Name            string   `json:"name"`
		Address         string   `json:"address"`
		Enabled         bool     `json:"enabled"`
		Alive           bool     `json:"alive"`
		Instances       []uint64 `json:"instances"`
		CpuUsage        float64  `json:"cpu_usage"`
		MaxCpu          uint     `json:"max_cpu"`
		AvailableMemory uint64   `json:"available_memory"`
		MaxMemory       uint64   `json:"max_memory"`
		AvailableDisk   uint64   `json:"available_disk"`
		MaxDisk         uint64   `json:"max_disk"`
		ReadSpeed       uint64   `json:"read_speed"`
		WriteSpeed      uint64   `json:"write_speed"`
		ReceiveSpeed    uint64   `json:"receive_speed"`
		SendSpeed       uint64   `json:"send_speed"`
	}

	parser := func(msg framework.Message) (status []cellStatus, err error) {
		var name, address []string
		var enabled, alive, instances, usage, cores, memory, disk, speed []uint64
		if name, err = msg.GetStringArray(framework.ParamKeyName); err != nil {
			return
		}
		if address, err = msg.GetStringArray(framework.ParamKeyAddress); err != nil {
			return
		}
		if enabled, err = msg.GetUIntArray(framework.ParamKeyEnable); err != nil {
			return
		}
		if alive, err = msg.GetUIntArray(framework.ParamKeyStatus); err != nil {
			return
		}
		if instances, err = msg.GetUIntArray(framework.ParamKeyInstance); err != nil {
			return
		}
		if usage, err = msg.GetUIntArray(framework.ParamKeyUsage); err != nil {
			return
		}
		if cores, err = msg.GetUIntArray(framework.ParamKeyCore); err != nil {
			return
		}
		if memory, err = msg.GetUIntArray(framework.ParamKeyMemory); err != nil {
			return
		}
		if disk, err = msg.GetUIntArray(framework.ParamKeyDisk); err != nil {
			return
		}
		if speed, err = msg.GetUIntArray(framework.ParamKeySpeed); err != nil {
			return
		}
		const (
			instanceParamCount = 4
			memoryParamCount   = 2
			diskParamCount     = 2
			speedParamCount    = 4
		)
		var count = len(name)
		var instanceCount = len(instances)
		var memoryCount = len(memory)
		var diskCount = len(disk)
		var speedCount = len(speed)
		if instanceCount != count * instanceParamCount{
			err = fmt.Errorf("unexpected instance params %d / %d", instanceCount, count * instanceParamCount)
			return
		}
		if memoryCount != count * memoryParamCount{
			err = fmt.Errorf("unexpected memory params %d / %d", memoryCount, count * memoryParamCount)
			return
		}
		if diskCount != count * diskParamCount{
			err = fmt.Errorf("unexpected disk params %d / %d", diskCount, count * diskParamCount)
			return
		}
		if speedCount != count * speedParamCount{
			err = fmt.Errorf("unexpected speed params %d / %d", memoryCount, count * speedParamCount)
			return
		}
		for i := 0; i < count; i++{
			var p = cellStatus{Name:name[i], Address:address[i]}
			if 1 == enabled[i]{
				p.Enabled = true
			}else{
				p.Enabled = false
			}
			if 1 == alive[i]{
				p.Alive = true
			}else{
				p.Alive = false
			}
			p.Instances = []uint64{instances[i*instanceParamCount], instances[i*instanceParamCount + 1], instances[i*instanceParamCount+ 2], instances[i*instanceParamCount + 3]}
			p.CpuUsage = float64(usage[i])
			p.MaxCpu = uint(cores[i])
			p.AvailableMemory = memory[i*memoryParamCount]
			p.MaxMemory = memory[i*memoryParamCount + 1]
			p.AvailableDisk = disk[i*diskParamCount]
			p.MaxDisk = disk[i*diskParamCount + 1]
			p.ReadSpeed = speed[i*speedParamCount]
			p.WriteSpeed = speed[i*speedParamCount + 1]
			p.ReceiveSpeed = speed[i*speedParamCount + 2]
			p.SendSpeed = speed[i*speedParamCount + 3]
			status = append(status, p)
		}
		return
	}

	data, err := parser(resp)
	if err != nil {
		log.Printf("<api> parse query compute cell status fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	ResponseOK(data, w)
}

func (module *APIModule) getComputeCellStatus(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	pool := params.ByName("pool")
	cell := params.ByName("cell")

	msg, _ := framework.CreateJsonMessage(framework.GetComputePoolCellStatusRequest)
	msg.SetString(framework.ParamKeyPool, pool)
	msg.SetString(framework.ParamKeyCell, cell)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> get compute cell status fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> get compute cell status fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	type respData struct {
		Name            string   `json:"name"`
		Address         string   `json:"address"`
		Enabled         bool     `json:"enabled"`
		Alive           bool     `json:"alive"`
		Instances       []uint64 `json:"instances"`
		CpuUsage        float64  `json:"cpu_usage"`
		MaxCpu          uint     `json:"max_cpu"`
		AvailableMemory uint64   `json:"available_memory"`
		MaxMemory       uint64   `json:"max_memory"`
		AvailableDisk   uint64   `json:"available_disk"`
		MaxDisk         uint64   `json:"max_disk"`
		ReadSpeed       uint64   `json:"read_speed"`
		WriteSpeed      uint64   `json:"write_speed"`
		ReceiveSpeed    uint64   `json:"receive_speed"`
		SendSpeed       uint64   `json:"send_speed"`
	}

	parser := func(msg framework.Message) (data respData, err error) {
		if data.Name, err = msg.GetString(framework.ParamKeyName); err != nil {
			return data, err
		}
		if data.Address, err = msg.GetString(framework.ParamKeyAddress); err != nil {
			return data, err
		}
		if data.Enabled, err = msg.GetBoolean(framework.ParamKeyEnable); err != nil {
			return data, err
		}
		if data.Alive, err = msg.GetBoolean(framework.ParamKeyStatus); err != nil {
			return data, err
		}
		if data.Instances, err = msg.GetUIntArray(framework.ParamKeyInstance); err != nil {
			return data, err
		}
		if data.CpuUsage, err = msg.GetFloat(framework.ParamKeyUsage); err != nil {
			return data, err
		}
		if data.MaxCpu, err = msg.GetUInt(framework.ParamKeyCore); err != nil {
			return data, err
		}
		const (
			memoryParamCount = 2
			diskParamCount   = 2
			speedParamCount  = 4
		)
		memory, err := msg.GetUIntArray(framework.ParamKeyMemory)
		if err != nil {
			return data, err
		}
		if memoryParamCount != len(memory) {
			return data, fmt.Errorf("invalid memory param count %d", len(memory))
		}
		disk, err := msg.GetUIntArray(framework.ParamKeyDisk)
		if err != nil {
			return data, err
		}
		if diskParamCount != len(disk) {
			return data, fmt.Errorf("invalid disk param count %d", len(disk))
		}
		speed, err := msg.GetUIntArray(framework.ParamKeySpeed)
		if err != nil {
			return data, err
		}
		if speedParamCount != len(speed) {
			return data, fmt.Errorf("invalid speed param count %d", len(speed))
		}
		data.AvailableMemory = memory[0]
		data.MaxMemory = memory[1]
		data.AvailableDisk = disk[0]
		data.MaxDisk = disk[1]

		data.ReadSpeed = speed[0]
		data.WriteSpeed = speed[1]
		data.ReceiveSpeed = speed[2]
		data.SendSpeed = speed[3]
		return data, nil
	}

	data, err := parser(resp)
	if err != nil {
		log.Printf("<api> parse get compute cell status fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	ResponseOK(data, w)

}

func (module *APIModule) handleQueryInstanceStatusInPool(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var poolName = params.ByName("pool")
	if "" == poolName{
		err := errors.New("must specify target pool")
		log.Printf("<api> query instance in pool fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.QueryInstanceStatusRequest)
	msg.SetString(framework.ParamKeyPool, poolName)

	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send query instance in pool fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> query instance in pool fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	result, err := UnmarshalGuestConfigListFromMessage(resp)
	if err != nil{
		log.Printf("<api> parse query instance in pool result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	ResponseOK(result, w)
}

func (module *APIModule) handleQueryInstanceStatusInCell(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var poolName = params.ByName("pool")
	if "" == poolName{
		err := errors.New("must specify target pool")
		log.Printf("<api> query instance in cell fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	var cellName = params.ByName("cell")
	if "" == cellName{
		err := errors.New("must specify target cell")
		log.Printf("<api> query instance in cell fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.QueryInstanceStatusRequest)
	msg.SetString(framework.ParamKeyPool, poolName)
	msg.SetString(framework.ParamKeyCell, cellName)

	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send query instance in cell fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> query instance in cell fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	result, err := UnmarshalGuestConfigListFromMessage(resp)
	if err != nil{
		log.Printf("<api> parse query instance in cell result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	ResponseOK(result, w)
}

func (module *APIModule) handleCreateComputePool(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	pool := params.ByName("pool")
	rawData, err := ioutil.ReadAll(r.Body)
	if err != nil{
		log.Printf("<api> read create compute pool param fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	type UserRequest struct {
		Storage  string `json:"storage,omitempty"`
		Network  string `json:"network,omitempty"`
		Failover bool   `json:"failover,omitempty"`
	}
	var requestData UserRequest
	if 0 != len(rawData){
		//body available
		if err = json.Unmarshal(rawData, &requestData); err != nil{
			log.Printf("<api> parse create compute pool request fail: %s", err.Error())
			ResponseFail(ResponseDefaultError, err.Error(), w)
			return
		}
	}
	msg, _ := framework.CreateJsonMessage(framework.CreateComputePoolRequest)
	msg.SetString(framework.ParamKeyPool, pool)
	msg.SetString(framework.ParamKeyStorage, requestData.Storage)
	msg.SetString(framework.ParamKeyNetwork, requestData.Network)
	msg.SetBoolean(framework.ParamKeyOption, requestData.Failover)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> request create compute pool fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> create compute pool fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleDeleteComputePool(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	pool := params.ByName("pool")
	msg, _ := framework.CreateJsonMessage(framework.DeleteComputePoolRequest)
	msg.SetString(framework.ParamKeyPool, pool)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> request delete compute pool fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> delete compute pool fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleModifyComputePool(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	pool := params.ByName("pool")
	type UserRequest struct {
		Enable   bool   `json:"enable,omitempty"`
		Storage  string `json:"storage,omitempty"`
		Network  string `json:"network,omitempty"`
		Failover bool   `json:"failover,omitempty"`
	}
	var requestData UserRequest
	var decoder = json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestData); err != nil{
		log.Printf("<api> parse delete compute pool request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg, _ := framework.CreateJsonMessage(framework.ModifyComputePoolRequest)
	msg.SetString(framework.ParamKeyPool, pool)
	msg.SetString(framework.ParamKeyStorage, requestData.Storage)
	msg.SetString(framework.ParamKeyNetwork, requestData.Network)
	msg.SetBoolean(framework.ParamKeyOption, requestData.Failover)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> request modify compute pool fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> modify compute pool fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleQueryStoragePool(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg, _ := framework.CreateJsonMessage(framework.QueryStoragePoolRequest)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> request query storage pool fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> query storage pool fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	type Pool struct {
		Name   string `json:"name"`
		Type   string `json:"type"`
		Host   string `json:"host"`
		Target string `json:"target"`
	}

	parser := func(msg framework.Message) (pools []Pool, err error) {
		pools = make([]Pool, 0)
		var nameArray, typeArray, hostArray, targetArray []string
		if nameArray, err = msg.GetStringArray(framework.ParamKeyName); err != nil {
			return pools, err
		}
		if typeArray, err = msg.GetStringArray(framework.ParamKeyType); err != nil {
			return pools, err
		}
		if hostArray, err = msg.GetStringArray(framework.ParamKeyHost); err != nil {
			return pools, err
		}
		if targetArray, err = msg.GetStringArray(framework.ParamKeyTarget); err != nil{
			return
		}
		poolCount := len(nameArray)
		if poolCount != len(targetArray) {
			return pools, fmt.Errorf("unexpected target array %d / %d", len(targetArray), poolCount)
		}
		if poolCount != len(typeArray) {
			return pools, fmt.Errorf("unexpected type array %d / %d", len(typeArray), poolCount)
		}
		if poolCount != len(hostArray) {
			return pools, fmt.Errorf("unexpected host array %d / %d", len(hostArray), poolCount)
		}

		for i := 0; i < poolCount; i++ {
			pools = append(pools, Pool{nameArray[i], typeArray[i], hostArray[i], targetArray[i]})
		}
		return pools, nil

	}
	//success
	pools, err := parser(resp)
	if err != nil {
		log.Printf("<api> parse query storage pool result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	ResponseOK(pools, w)
}

func (module *APIModule) handleGetStoragePool(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	poolName := params.ByName("pool")
	if "" == poolName{
		err := errors.New("must specify pool name")
		log.Printf("<api> get storage pool fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg, _ := framework.CreateJsonMessage(framework.GetStoragePoolRequest)
	msg.SetString(framework.ParamKeyStorage, poolName)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> get storage pool fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> get storage pool fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	type Pool struct {
		Type   string `json:"type"`
		Host   string `json:"host"`
		Target string `json:"target"`
	}

	parser := func(msg framework.Message) (pool Pool, err error) {
		if pool.Type, err = msg.GetString(framework.ParamKeyType); err != nil {
			return
		}
		if pool.Host, err = msg.GetString(framework.ParamKeyHost); err != nil{
			return
		}
		if pool.Target, err = msg.GetString(framework.ParamKeyTarget); err != nil{
			return
		}
		return
	}
	//success
	pool, err := parser(resp)
	if err != nil {
		log.Printf("<api> parse get storage pool result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	ResponseOK(pool, w)
}

func (module *APIModule) handleCreateStoragePool(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	pool := params.ByName("pool")
	type UserRequest struct {
		Type string `json:"type"`
		Host string `json:"host,omitempty"`
		Target string `json:"target,omitempty"`
	}
	var requestData UserRequest
	var decoder = json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestData); err != nil{
		log.Printf("<api> parse create storage pool request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg, _ := framework.CreateJsonMessage(framework.CreateStoragePoolRequest)
	msg.SetString(framework.ParamKeyStorage, pool)
	msg.SetString(framework.ParamKeyType, requestData.Type)
	msg.SetString(framework.ParamKeyHost, requestData.Host)
	msg.SetString(framework.ParamKeyTarget, requestData.Target)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> request create storage pool fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> create storage pool fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleModifyStoragePool(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	pool := params.ByName("pool")
	type UserRequest struct {
		Type string `json:"type"`
		Host string `json:"host,omitempty"`
		Target string `json:"target,omitempty"`
	}
	var requestData UserRequest
	var decoder = json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestData); err != nil{
		log.Printf("<api> parse modify storage pool request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg, _ := framework.CreateJsonMessage(framework.ModifyStoragePoolRequest)
	msg.SetString(framework.ParamKeyStorage, pool)
	msg.SetString(framework.ParamKeyType, requestData.Type)
	msg.SetString(framework.ParamKeyHost, requestData.Host)
	msg.SetString(framework.ParamKeyTarget, requestData.Target)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> request modify storage pool fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> modify storage pool fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleDeleteStoragePool(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	pool := params.ByName("pool")
	msg, _ := framework.CreateJsonMessage(framework.DeleteStoragePoolRequest)
	msg.SetString(framework.ParamKeyStorage, pool)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> request delete storage pool fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> delete storage pool fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleAddComputeCell(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	pool := params.ByName("pool")
	cell := params.ByName("cell")
	msg, _ := framework.CreateJsonMessage(framework.AddComputePoolCellRequest)
	msg.SetString(framework.ParamKeyPool, pool)
	msg.SetString(framework.ParamKeyCell, cell)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequestTimeout(msg, 10*time.Second, respChan); err != nil {
		log.Printf("<api> add compute cell fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> add compute cell fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleRemoveComputeCell(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	pool := params.ByName("pool")
	cell := params.ByName("cell")
	msg, _ := framework.CreateJsonMessage(framework.RemoveComputePoolCellRequest)
	msg.SetString(framework.ParamKeyPool, pool)
	msg.SetString(framework.ParamKeyCell, cell)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> remove compute cell fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> remove compute cell fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleModifyComputeCell(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	pool := params.ByName("pool")
	cell := params.ByName("cell")
	type UserRequest struct {
		Enable bool `json:"enable"`
	}
	var requestData UserRequest
	var decoder = json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestData); err != nil{
		log.Printf("<api> parse modify cell request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var operateName string
	var msg framework.Message
	if requestData.Enable {
		operateName = "enable"
		msg, _ = framework.CreateJsonMessage(framework.EnableComputePoolCellRequest)
	}else{
		operateName = "disable"
		msg, _ = framework.CreateJsonMessage(framework.DisableComputePoolCellRequest)
	}

	msg.SetString(framework.ParamKeyPool, pool)
	msg.SetString(framework.ParamKeyCell, cell)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> request %s cell fail: %s", operateName, err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> %s cell fail: %s", operateName, errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleGetComputeCell(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	pool := params.ByName("pool")
	cell := params.ByName("cell")

	msg, _ := framework.CreateJsonMessage(framework.GetComputePoolCellRequest)
	msg.SetString(framework.ParamKeyPool, pool)
	msg.SetString(framework.ParamKeyCell, cell)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> request get compute cell fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> get compute cell status fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	type StorageStatus struct {
		Name     string `json:"name"`
		Attached bool   `json:"attached"`
		Error    string `json:"error,omitempty"`
	}
	type ResponseData struct {
		Name    string          `json:"name"`
		Address string          `json:"address"`
		Enabled bool            `json:"enabled"`
		Alive   bool            `json:"alive"`
		Storage []StorageStatus `json:"storage,omitempty"`
	}

	parser := func(msg framework.Message) (data ResponseData, err error) {
		if data.Name, err = msg.GetString(framework.ParamKeyName); err != nil {
			return data, err
		}
		if data.Address, err = msg.GetString(framework.ParamKeyAddress); err != nil {
			return data, err
		}
		if data.Enabled, err = msg.GetBoolean(framework.ParamKeyEnable); err != nil {
			return data, err
		}
		if data.Alive, err = msg.GetBoolean(framework.ParamKeyStatus); err != nil {
			return data, err
		}
		if data.Alive && data.Enabled{
			var storage, errors []string
			var attached []uint64
			if storage, err = msg.GetStringArray(framework.ParamKeyStorage); err != nil{
				return
			}
			if errors, err = msg.GetStringArray(framework.ParamKeyError);err != nil{
				return
			}
			if attached, err = msg.GetUIntArray(framework.ParamKeyAttach); err != nil{
				return
			}
			var storageCount = len(storage)
			if 0 == storageCount{
				return
			}
			var index = 0
			for ; index < storageCount; index++{
				var pool = StorageStatus{Name: storage[index]}
				if 1 == attached[index]{
					pool.Attached = true
				}else{
					pool.Attached = false
					pool.Error = errors[index]
				}
				data.Storage = append(data.Storage, pool)
			}
		}
		return data, nil
	}

	data, err := parser(resp)
	if err != nil {
		log.Printf("<api> parse get compute cell fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	ResponseOK(data, w)

}

func (module *APIModule) handleQueryUnallocatedCell(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg, _ := framework.CreateJsonMessage(framework.QueryUnallocatedComputePoolCellRequest)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> query unallocated cells fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> query unallocated cells fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	cells, err := CellsFromMessage(resp)
	if err != nil {
		log.Printf("<api> get unallocated cells fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	ResponseOK(cells, w)
}

func (module *APIModule) handleQueryCellsInPool(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	pool := params.ByName("pool")
	msg, _ := framework.CreateJsonMessage(framework.QueryComputePoolCellRequest)
	msg.SetString(framework.ParamKeyPool, pool)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> query cells in pool fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> query cells in pool fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	cells, err := CellsFromMessage(resp)
	if err != nil {
		log.Printf("<api> get cells fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	ResponseOK(cells, w)
}

func (module *APIModule) handleQueryAllPools(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg, _ := framework.CreateJsonMessage(framework.QueryComputePoolRequest)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> query pool fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> query pool fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	type Pool struct {
		Name     string `json:"name"`
		Enabled  bool   `json:"enabled"`
		Cells    uint64 `json:"cells"`
		Network  string `json:"network"`
		Storage  string `json:"storage"`
		Failover bool   `json:"failover"`
	}

	parser := func(msg framework.Message) (pools []Pool, err error) {
		pools = make([]Pool, 0)
		var nameArray, networkArray, storageArray []string
		var cellArray, statusArray, failoverArray []uint64
		if nameArray, err = msg.GetStringArray(framework.ParamKeyName); err != nil {
			return pools, err
		}
		if networkArray, err = msg.GetStringArray(framework.ParamKeyNetwork); err != nil {
			return pools, err
		}
		if storageArray, err = msg.GetStringArray(framework.ParamKeyStorage); err != nil {
			return pools, err
		}
		if cellArray, err = msg.GetUIntArray(framework.ParamKeyCell); err != nil {
			return pools, err
		}
		if statusArray, err = msg.GetUIntArray(framework.ParamKeyStatus); err != nil {
			return pools, err
		}
		if failoverArray, err = msg.GetUIntArray(framework.ParamKeyOption); err != nil {
			return pools, err
		}
		poolCount := len(nameArray)
		if poolCount != len(cellArray) {
			return pools, fmt.Errorf("unexpected cell array %d / %d", len(cellArray), poolCount)
		}
		if poolCount != len(networkArray) {
			return pools, fmt.Errorf("unexpected network array %d / %d", len(networkArray), poolCount)
		}
		if poolCount != len(storageArray) {
			return pools, fmt.Errorf("unexpected storage array %d / %d", len(storageArray), poolCount)
		}
		if poolCount != len(statusArray) {
			return pools, fmt.Errorf("unexpected status array %d / %d", len(statusArray), poolCount)
		}
		for i := 0; i < poolCount; i++ {
			var enable, failover = false, false
			if 1 == statusArray[i] {
				enable = true
			}
			if 1 == failoverArray[i] {
				failover = true
			}
			pools = append(pools, Pool{nameArray[i], enable, cellArray[i], networkArray[i], storageArray[i], failover})

		}
		return pools, nil

	}
	//success
	pools, err := parser(resp)
	if err != nil {
		log.Printf("<api> parse query pool result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	ResponseOK(pools, w)
}

func (module *APIModule) handleGetComputePool(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	poolName := params.ByName("pool")
	if "" == poolName{
		err := errors.New("must specify pool name")
		log.Printf("<api> get compute pool fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg, _ := framework.CreateJsonMessage(framework.GetComputePoolRequest)
	msg.SetString(framework.ParamKeyPool, poolName)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> get compute pool fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> get compute pool fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	type Pool struct {
		Name     string `json:"name"`
		Enabled  bool   `json:"enabled"`
		Cells    uint   `json:"cells"`
		Network  string `json:"network"`
		Storage  string `json:"storage"`
		Failover bool   `json:"failover"`
	}

	parser := func(msg framework.Message) (pool Pool, err error) {
		if pool.Name, err = msg.GetString(framework.ParamKeyName); err != nil {
			return
		}
		if pool.Enabled, err = msg.GetBoolean(framework.ParamKeyEnable); err != nil{
			return
		}
		if pool.Cells, err = msg.GetUInt(framework.ParamKeyCell);err != nil{
			return
		}
		if pool.Network, err = msg.GetString(framework.ParamKeyNetwork); err != nil{
			return
		}
		if pool.Storage, err = msg.GetString(framework.ParamKeyStorage); err != nil{
			return
		}
		if pool.Failover, err = msg.GetBoolean(framework.ParamKeyOption); err != nil{
			return
		}
		return
	}
	//success
	pool, err := parser(resp)
	if err != nil {
		log.Printf("<api> parse get compute pool result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	ResponseOK(pool, w)
}



func (module *APIModule) handleQueryGuestConfig(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	type userRequest struct {
		Pool           string `json:"pool"`
		InCell         bool   `json:"in_cell"`
		WithOwner      bool   `json:"with_owner"`
		WithGroup      bool   `json:"with_group"`
		WithStatus     bool   `json:"with_status"`
		WithCreateFlag bool   `json:"with_create_flag"`
		Cell           string `json:"cell,omitemtpy"`
		Owner          string `json:"owner,omitemtpy"`
		Group          string `json:"group,omitemtpy"`
		Status         int    `json:"status,omitemtpy"`
		Created        bool   `json:"created,omitemtpy"`
	}
	var request userRequest
	var err error
	query := r.URL.Query()
	request.Pool = query.Get("pool")
	if request.Cell = query.Get("cell");request.Cell != ""{
		request.InCell = true
	}
	if request.Owner = query.Get("owner");request.Owner != ""{
		request.WithOwner = true
	}
	if request.Group = query.Get("group");request.Group != ""{
		request.WithGroup = true
	}
	if status := query.Get("status");status != ""{
		request.WithStatus = true
		request.Status, err = strconv.Atoi(status)
		if err != nil{
			log.Printf("<api> parse query guest request fail: %s", err.Error())
			ResponseFail(ResponseDefaultError, err.Error(), w)
			return
		}
	}
	if created := query.Get("created");created != ""{
		request.WithCreateFlag = true
		request.Created, err = strconv.ParseBool(created)
		if err != nil{
			log.Printf("<api> parse query guest request fail: %s", err.Error())
			ResponseFail(ResponseDefaultError, err.Error(), w)
			return
		}
	}

	msg, _ := framework.CreateJsonMessage(framework.QueryGuestRequest)
	if "" == request.Pool{
		err := errors.New("must specify target pool")
		log.Printf("<api> build query guest request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg.SetString(framework.ParamKeyPool, request.Pool)
	var options []uint64
	if request.InCell{
		options = append(options, 1)
		if "" == request.Cell{
			err := errors.New("must specify target cell")
			log.Printf("<api> build query guest request fail: %s", err.Error())
			ResponseFail(ResponseDefaultError, err.Error(), w)
			return
		}
		msg.SetString(framework.ParamKeyCell, request.Cell)
	}else{
		options = append(options, 0)
	}

	if request.WithOwner{
		options = append(options, 1)
		if "" == request.Owner{
			err := errors.New("must specify instance owner")
			log.Printf("<api> build query guest request fail: %s", err.Error())
			ResponseFail(ResponseDefaultError, err.Error(), w)
			return
		}
		msg.SetString(framework.ParamKeyUser, request.Owner)
	}else{
		options = append(options, 0)
	}

	if request.WithGroup{
		options = append(options, 1)
		if "" == request.Group{
			err := errors.New("must specify instance group")
			log.Printf("<api> build query guest request fail: %s", err.Error())
			ResponseFail(ResponseDefaultError, err.Error(), w)
			return
		}
		msg.SetString(framework.ParamKeyGroup, request.Group)
	}else{
		options = append(options, 0)
	}

	if request.WithStatus{
		options = append(options, 1)
		msg.SetUInt(framework.ParamKeyStatus, uint(request.Status))
	}else{
		options = append(options, 0)
	}

	if request.WithCreateFlag{
		options = append(options, 1)
		msg.SetBoolean(framework.ParamKeyEnable, request.Created)
	}else{
		options = append(options, 0)
	}

	msg.SetUIntArray(framework.ParamKeyOption, options)

	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send query guest request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> query guest fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	result, err := UnmarshalGuestConfigListFromMessage(resp)
	if err != nil{
		log.Printf("<api> parse query result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	ResponseOK(result, w)
}

func (module *APIModule) redirectToImageServer(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var respChan = make(chan ResourceResult, 1)
	module.resource.GetImageServer(respChan)
	var result = <- respChan
	if result.Error != nil{
		var err = result.Error
		log.Printf("<api> fetch current image server fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var imageHost = result.Host
	var imagePort = result.Port
	const (
		DefaultProtocol = "https"
	)
	var address = fmt.Sprintf("%s://%s:%d",DefaultProtocol, imageHost, imagePort)
	if address == module.currentImageURL{
		//not changed
		r.Host = module.currentImageHost
		module.currentImageProxy.ServeHTTP(w, r)
		return
	}
	//update new URL
	if url, err := url.Parse(address);err != nil{
		log.Printf("<api> parse image proxy url fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}else{
		module.currentImageHost = imageHost
		module.currentImageURL = address
		module.currentImageProxy = httputil.NewSingleHostReverseProxy(url)
		module.currentImageProxy.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		log.Printf("<api> new image proxy established: %s", url)

		r.Host = module.currentImageHost
		module.currentImageProxy.ServeHTTP(w, r)
	}
}

func (module *APIModule) handleGetGuestConfig(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	id := params.ByName("id")
	msg, _ := framework.CreateJsonMessage(framework.GetGuestRequest)
	msg.SetString(framework.ParamKeyInstance, id)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send get guest request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> get guest fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	var config restGuestConfig
	if err := config.Unmarshal(resp);err != nil{
		log.Printf("<api> parse guest config fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	if !config.Created{
		w.WriteHeader(http.StatusCreated)
	}else{
		w.WriteHeader(http.StatusOK)
	}
	ResponseOK(config, w)
}


func (module *APIModule) handleCreateGuest(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	type ciConfig struct {
		RootEnabled bool   `json:"root_enabled,omitempty"`
		AdminName   string `json:"admin_name,omitempty"`
		AdminSecret string `json:"admin_secret,omitempty"`
		DataPath    string `json:"data_path,omitempty"`
	}

	type userRequest struct {
		Name            string           `json:"name"`
		Owner           string           `json:"owner"`
		Group           string           `json:"group"`
		Pool            string           `json:"pool"`
		Cores           uint             `json:"cores"`
		Memory          uint             `json:"memory"`
		Disks           []uint64         `json:"disks"`
		AutoStart       bool             `json:"auto_start,omitempty"`
		System          string           `json:"system,omitempty"`
		NetworkAddress  string           `json:"network_address,omitempty"`
		EthernetAddress string           `json:"ethernet_address,omitempty"`
		FromImage       string           `json:"from_image,omitempty"`
		Port            []uint64         `json:"port,omitempty"`
		SystemVersion   string           `json:"system_version,omitempty"`
		Modules         []string         `json:"modules,omitempty"`
		CloudInit       *ciConfig        `json:"cloud_init,omitempty"`
		QoS             *restInstanceQoS `json:"qos,omitempty"`
	}

	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse create guest request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg, _ := framework.CreateJsonMessage(framework.CreateGuestRequest)
	msg.SetString(framework.ParamKeyName, request.Name)
	msg.SetString(framework.ParamKeyUser, request.Owner)
	msg.SetString(framework.ParamKeyGroup, request.Group)
	msg.SetString(framework.ParamKeyPool, request.Pool)

	msg.SetUInt(framework.ParamKeyCore, request.Cores)
	msg.SetUInt(framework.ParamKeyMemory, request.Memory)
	msg.SetUIntArray(framework.ParamKeyDisk, request.Disks)
	msg.SetBoolean(framework.ParamKeyOption, request.AutoStart)
	msg.SetString(framework.ParamKeySystem, request.System)
	//optional disk image
	if "" != request.FromImage{
		msg.SetString(framework.ParamKeyImage, request.FromImage)
	}
	msg.SetString(framework.ParamKeyVersion, request.SystemVersion)
	msg.SetStringArray(framework.ParamKeyModule, request.Modules)
	const (
		RootLoginDisabled = iota
		RootLoginEnabled
	)
	var flags []uint64
	if nil != request.CloudInit{
		msg.SetString(framework.ParamKeyAdmin, request.CloudInit.AdminName)
		msg.SetString(framework.ParamKeySecret, request.CloudInit.AdminSecret)
		msg.SetString(framework.ParamKeyPath, request.CloudInit.DataPath)
		if request.CloudInit.RootEnabled{
			flags = append(flags, RootLoginEnabled)
		}else{
			flags = append(flags, RootLoginDisabled)
		}

	}else{
		msg.SetString(framework.ParamKeyAdmin, "")
		msg.SetString(framework.ParamKeySecret, "")
		msg.SetString(framework.ParamKeyPath, "")
		flags = append(flags, RootLoginEnabled)
	}

	if nil != request.QoS{
		var qos = request.QoS
		switch qos.CPUPriority {
		case priority_label_high:
			msg.SetUInt(framework.ParamKeyPriority, PriorityHigh)
		case priority_label_medium:
			msg.SetUInt(framework.ParamKeyPriority, PriorityMedium)
		case priority_label_low:
			msg.SetUInt(framework.ParamKeyPriority, PriorityLow)
		default:
			var err = fmt.Errorf("invalid CPU priority %s", qos.CPUPriority)
			log.Printf("<api> invalid create request with CPU priority: %s", qos.CPUPriority)
			ResponseFail(ResponseDefaultError, err.Error(), w)
			return
		}
		msg.SetUIntArray(framework.ParamKeyLimit, []uint64{qos.ReadSpeed, qos.WriteSpeed,
			qos.ReadIOPS, qos.WriteIOPS, qos.ReceiveSpeed, qos.SendSpeed})
	}else{
		msg.SetUInt(framework.ParamKeyPriority, PriorityHigh)
		msg.SetUIntArray(framework.ParamKeyLimit, []uint64{0, 0, 0, 0, 0, 0})
	}
	msg.SetUIntArray(framework.ParamKeyFlag, flags)

	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send create request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> create guest fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	created, err := resp.GetBoolean(framework.ParamKeyEnable)
	if err != nil{
		log.Printf("<api> parse create result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	id, err := resp.GetString(framework.ParamKeyInstance)
	if err != nil{
		log.Printf("<api> parse instance id fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	type userResponse struct {
		ID string `json:"id"`
	}

	var result = userResponse{ID:id}
	if !created{
		w.WriteHeader(http.StatusAccepted)
	}else{
		w.WriteHeader(http.StatusOK)
	}
	ResponseOK(result, w)
}

func (module *APIModule) handleDeleteGuest(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	id := params.ByName("id")
	type userRequest struct {
		Force bool `json:"force,omitempty"`
	}
	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse delete guest request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.DeleteGuestRequest)
	msg.SetString(framework.ParamKeyInstance, id)
	if request.Force{
		msg.SetUInt(framework.ParamKeyOption, 1)
	}else {
		msg.SetUInt(framework.ParamKeyOption, 0)
	}

	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send delete request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> delete guest fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleGetInstanceStatus(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	id := params.ByName("id")
	msg, _ := framework.CreateJsonMessage(framework.GetInstanceStatusRequest)
	msg.SetString(framework.ParamKeyInstance, id)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send get instance request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> get instance fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	var status restInstanceStatus
	if err := status.Unmarshal(resp);err != nil{
		log.Printf("<api> parse status fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	ResponseOK(status, w)
}

func (module *APIModule) handleStartInstance(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	id := params.ByName("id")
	type userRequest struct {
		FromMedia bool `json:"from_media,omitempty"`
		FromNetwork bool `json:"from_network,omitempty"`
		Source string `json:"source,omitempty"`
	}
	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse start instance request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg, _ := framework.CreateJsonMessage(framework.StartInstanceRequest)
	msg.SetString(framework.ParamKeyInstance, id)

	if request.FromMedia{
		msg.SetUInt(framework.ParamKeyOption, InstanceMediaOptionImage)
		msg.SetString(framework.ParamKeySource, request.Source)
	}else if request.FromNetwork{
		msg.SetUInt(framework.ParamKeyOption, InstanceMediaOptionNetwork)
	}else{
		msg.SetUInt(framework.ParamKeyOption, InstanceMediaOptionNone)
	}
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send start instance request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> start instance fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleStopInstance(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	id := params.ByName("id")
	type userRequest struct {
		Reboot bool `json:"reboot,omitempty"`
		Force bool `json:"force,omitempty"`
	}
	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse stop instance request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg, _ := framework.CreateJsonMessage(framework.StopInstanceRequest)
	msg.SetString(framework.ParamKeyInstance, id)
	var options []uint64
	if request.Reboot{
		options = append(options, 1)
	}else{
		options = append(options, 0)
	}
	if request.Force{
		options = append(options, 1)
	}else{
		options = append(options, 0)
	}
	msg.SetUIntArray(framework.ParamKeyOption, options)
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send stop instance request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> stop instance fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}


func (module *APIModule) searchMediaImage(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	var filterOwner = r.URL.Query().Get("owner")
	var filterGroup = r.URL.Query().Get("group")

	msg, _ := framework.CreateJsonMessage(framework.QueryMediaImageRequest)
	msg.SetString(framework.ParamKeyUser, filterOwner)
	msg.SetString(framework.ParamKeyGroup, filterGroup)

	respChan := make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send query media image request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> query media image fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}

	type respImage struct {
		Name        string   `json:"name"`
		ID          string   `json:"id"`
		Description string   `json:"description,omitempty"`
		Size        uint64   `json:"size"`
		Tags        []string `json:"tags,omitempty"`
		CreateTime  string   `json:"create_time,omitempty"`
		ModifyTime  string   `json:"modify_time,omitempty"`
	}

	var parser = func(msg framework.Message) (images []respImage, err error){
		//unmarshal
		var name, id, description, tags, createTime, modifyTime []string
		var size, tagCount []uint64
		if name, err = msg.GetStringArray(framework.ParamKeyName); err != nil{
			return
		}
		if id, err = msg.GetStringArray(framework.ParamKeyImage); err != nil{
			return
		}
		if description, err = msg.GetStringArray(framework.ParamKeyDescription); err != nil{
			return
		}
		if tags, err = msg.GetStringArray(framework.ParamKeyTag); err != nil{
			return
		}
		if createTime, err = msg.GetStringArray(framework.ParamKeyCreate); err != nil{
			return
		}
		if modifyTime, err = msg.GetStringArray(framework.ParamKeyModify); err != nil{
			return
		}
		if size, err = msg.GetUIntArray(framework.ParamKeySize); err != nil{
			return
		}
		if tagCount, err = msg.GetUIntArray(framework.ParamKeyCount); err != nil{
			return
		}

		var totalTags uint64 = 0
		for _, count := range tagCount{
			totalTags += count
		}
		if int(totalTags) != len(tags){
			err = fmt.Errorf("unexpect tag count %d / %d", len(tags), totalTags)
			return
		}
		var imageCount = len(name)
		images = make([]respImage, 0)
		var tagBegin = 0
		for i := 0 ; i < imageCount;i++{
			var image = respImage{}
			image.Name = name[i]
			image.ID = id[i]
			image.Size = size[i]
			image.Description = description[i]
			var tagEnd = tagBegin + int(tagCount[i])
			image.Tags = tags[tagBegin:tagEnd]
			image.CreateTime = createTime[i]
			image.ModifyTime = modifyTime[i]
			tagBegin = tagEnd
			images = append(images, image)
		}
		return images, nil
	}

	payload, err := parser(resp)
	if err != nil{
		log.Printf("<api> parse query media image result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
	}
	ResponseOK(payload, w)
}

func (module *APIModule) queryAllMediaImage(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg, _ := framework.CreateJsonMessage(framework.QueryMediaImageRequest)
	respChan := make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send query media image request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> query media image fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}

	type respImage struct {
		Name        string   `json:"name"`
		ID          string   `json:"id"`
		Description string   `json:"description,omitempty"`
		Size        uint64   `json:"size"`
		Tags        []string `json:"tags,omitempty"`
		CreateTime  string   `json:"create_time,omitempty"`
		ModifyTime  string   `json:"modify_time,omitempty"`
	}

	var parser = func(msg framework.Message) (images []respImage, err error){
		//unmarshal
		var name, id, description, tags, createTime, modifyTime []string
		var size, tagCount []uint64
		if name, err = msg.GetStringArray(framework.ParamKeyName); err != nil{
			return
		}
		if id, err = msg.GetStringArray(framework.ParamKeyImage); err != nil{
			return
		}
		if description, err = msg.GetStringArray(framework.ParamKeyDescription); err != nil{
			return
		}
		if tags, err = msg.GetStringArray(framework.ParamKeyTag); err != nil{
			return
		}
		if createTime, err = msg.GetStringArray(framework.ParamKeyCreate); err != nil{
			return
		}
		if modifyTime, err = msg.GetStringArray(framework.ParamKeyModify); err != nil{
			return
		}
		if size, err = msg.GetUIntArray(framework.ParamKeySize); err != nil{
			return
		}
		if tagCount, err = msg.GetUIntArray(framework.ParamKeyCount); err != nil{
			return
		}

		var totalTags uint64 = 0
		for _, count := range tagCount{
			totalTags += count
		}
		if int(totalTags) != len(tags){
			err = fmt.Errorf("unexpect tag count %d / %d", len(tags), totalTags)
			return
		}
		var imageCount = len(name)
		images = make([]respImage, 0)
		var tagBegin = 0
		for i := 0 ; i < imageCount;i++{
			var image = respImage{}
			image.Name = name[i]
			image.ID = id[i]
			image.Size = size[i]
			image.Description = description[i]
			var tagEnd = tagBegin + int(tagCount[i])
			image.Tags = tags[tagBegin:tagEnd]
			image.CreateTime = createTime[i]
			image.ModifyTime = modifyTime[i]
			tagBegin = tagEnd
			images = append(images, image)
		}
		return images, nil
	}

	payload, err := parser(resp)
	if err != nil{
		log.Printf("<api> parse query media image result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
	}
	ResponseOK(payload, w)
}


func (module *APIModule) getMediaImage(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var id = params.ByName("id")
	msg, _ := framework.CreateJsonMessage(framework.GetMediaImageRequest)
	msg.SetString(framework.ParamKeyImage, id)
	var respChan = make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send get media image request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> get media image fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}

	type userResponse struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Size        uint     `json:"size"`
		Tags        []string `json:"tags"`
	}
	var data userResponse
	var err error
	if data.Name, err = resp.GetString(framework.ParamKeyName); err != nil{
		log.Printf("<api> parse media image name fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	if data.Description, err = resp.GetString(framework.ParamKeyDescription); err != nil{
		log.Printf("<api> parse media image description fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	if data.Tags, err = resp.GetStringArray(framework.ParamKeyTag); err != nil{
		log.Printf("<api> parse media image tags fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	if data.Size, err = resp.GetUInt(framework.ParamKeySize); err != nil{
		log.Printf("<api> parse media image size fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}

	ResponseOK(data, w)
}

func (module *APIModule) createMediaImage(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	type userRequest struct {
		Name        string   `json:"name"`
		Owner       string   `json:"owner"`
		Group       string   `json:"group"`
		Description string   `json:"description,omitempty"`
		Tags        []string `json:"tags,omitempty"`
	}
	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse create media image request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.CreateMediaImageRequest)
	msg.SetString(framework.ParamKeyName, request.Name)
	msg.SetString(framework.ParamKeyUser, request.Owner)
	msg.SetString(framework.ParamKeyGroup, request.Group)
	msg.SetString(framework.ParamKeyDescription, request.Description)
	msg.SetStringArray(framework.ParamKeyTag, request.Tags)

	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send create media image request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> create media image fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	var imageID string
	var err error
	if imageID, err = resp.GetString(framework.ParamKeyImage);err != nil{
		log.Printf("<api> get image from create result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	type userResponse struct {
		ID string `json:"id"`
	}
	var data = userResponse{ID:imageID}
	ResponseOK(data, w)
}


func (module *APIModule) modifyMediaImage(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var imageID = params.ByName("id")

	type userRequest struct {
		Name        string   `json:"name,omitempty"`
		Owner       string   `json:"owner,omitempty"`
		Group       string   `json:"group,omitempty"`
		Description string   `json:"description,omitempty"`
		Tags        []string `json:"tags,omitempty"`
	}
	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse modify media image request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.ModifyMediaImageRequest)
	msg.SetString(framework.ParamKeyImage, imageID)
	msg.SetString(framework.ParamKeyName, request.Name)
	msg.SetString(framework.ParamKeyUser, request.Owner)
	msg.SetString(framework.ParamKeyGroup, request.Group)
	msg.SetString(framework.ParamKeyDescription, request.Description)
	msg.SetStringArray(framework.ParamKeyTag, request.Tags)

	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send modify media image request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> modify media image fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}


func (module *APIModule) deleteMediaImage(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var id = params.ByName("id")
	msg, _ := framework.CreateJsonMessage(framework.DeleteMediaImageRequest)
	msg.SetString(framework.ParamKeyImage, id)
	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send delete media image request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> delete media image fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}


func (module *APIModule) queryDiskImage(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var filterOwner = r.URL.Query().Get("owner")
	var filterGroup = r.URL.Query().Get("group")
	var filterTags = r.URL.Query()["tags"]

	msg, _ := framework.CreateJsonMessage(framework.QueryDiskImageRequest)

	msg.SetString(framework.ParamKeyUser, filterOwner)

	if filterGroup != ""{
		msg.SetString(framework.ParamKeyGroup, filterGroup)
	}
	if 0 != len(filterTags){
		msg.SetStringArray(framework.ParamKeyTag, filterTags)
	}
	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send query disk image request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> query disk image fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}

	type respImage struct {
		Name        string   `json:"name"`
		ID          string   `json:"id"`
		Description string   `json:"description,omitempty"`
		Size        uint64   `json:"size"`
		Tags        []string `json:"tags,omitempty"`
		CreateTime  string   `json:"create_time,omitempty"`
		ModifyTime  string   `json:"modify_time,omitempty"`
	}

	var parser = func(msg framework.Message) (images []respImage, err error){
		//unmarshal
		var name, id, description, tags, createTime, modifyTime []string
		var size, tagCount []uint64
		if name, err = msg.GetStringArray(framework.ParamKeyName); err != nil{
			return
		}
		if id, err = msg.GetStringArray(framework.ParamKeyImage); err != nil{
			return
		}
		if description, err = msg.GetStringArray(framework.ParamKeyDescription); err != nil{
			return
		}
		if tags, err = msg.GetStringArray(framework.ParamKeyTag); err != nil{
			return
		}
		if createTime, err = msg.GetStringArray(framework.ParamKeyCreate); err != nil{
			return
		}
		if modifyTime, err = msg.GetStringArray(framework.ParamKeyModify); err != nil{
			return
		}
		if size, err = msg.GetUIntArray(framework.ParamKeySize); err != nil{
			return
		}
		if tagCount, err = msg.GetUIntArray(framework.ParamKeyCount); err != nil{
			return
		}

		var totalTags uint64 = 0
		for _, count := range tagCount{
			totalTags += count
		}
		if int(totalTags) != len(tags){
			err = fmt.Errorf("unexpect tag count %d / %d", len(tags), totalTags)
			return
		}
		var imageCount = len(name)
		images = make([]respImage, 0)
		var tagBegin = 0
		for i := 0 ; i < imageCount;i++{
			var image = respImage{}
			image.Name = name[i]
			image.ID = id[i]
			image.Size = size[i]
			image.Description = description[i]
			var tagEnd = tagBegin + int(tagCount[i])
			image.Tags = tags[tagBegin:tagEnd]
			image.CreateTime = createTime[i]
			image.ModifyTime = modifyTime[i]
			tagBegin = tagEnd
			images = append(images, image)
		}
		return images, nil
	}

	payload, err := parser(resp)
	if err != nil{
		log.Printf("<api> parse query disk image result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
	}
	ResponseOK(payload, w)
}

func (module *APIModule) getDiskImage(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var id = params.ByName("id")
	msg, _ := framework.CreateJsonMessage(framework.GetDiskImageRequest)
	msg.SetString(framework.ParamKeyImage, id)
	var respChan = make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send get disk image request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> get disk image fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}

	type userResponse struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Size        uint     `json:"size"`
		Created     bool     `json:"created"`
		Progress    uint     `json:"progress"`
		Tags        []string `json:"tags"`
	}
	var data userResponse
	var err error
	if data.Name, err = resp.GetString(framework.ParamKeyName); err != nil{
		log.Printf("<api> parse disk image name fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	if data.Description, err = resp.GetString(framework.ParamKeyDescription); err != nil{
		log.Printf("<api> parse disk image description fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	if data.Tags, err = resp.GetStringArray(framework.ParamKeyTag); err != nil{
		log.Printf("<api> parse disk image tags fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	if data.Size, err = resp.GetUInt(framework.ParamKeySize); err != nil{
		log.Printf("<api> parse disk image size fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	if data.Progress, err = resp.GetUInt(framework.ParamKeyProgress); err != nil{
		log.Printf("<api> parse disk image progress fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	if data.Created, err = resp.GetBoolean(framework.ParamKeyEnable); err != nil{
		log.Printf("<api> parse disk image status fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}

	ResponseOK(data, w)
}

func (module *APIModule) createDiskImage(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	type userRequest struct {
		Name        string   `json:"name"`
		Guest       string   `json:"guest,omitempty"`
		Owner       string   `json:"owner"`
		Group       string   `json:"group"`
		Description string   `json:"description,omitempty"`
		Tags        []string `json:"tags,omitempty"`
	}
	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse create disk image request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.CreateDiskImageRequest)
	msg.SetString(framework.ParamKeyName, request.Name)
	msg.SetString(framework.ParamKeyGuest, request.Guest)
	msg.SetString(framework.ParamKeyUser, request.Owner)
	msg.SetString(framework.ParamKeyGroup, request.Group)
	msg.SetString(framework.ParamKeyDescription, request.Description)
	msg.SetStringArray(framework.ParamKeyTag, request.Tags)

	var respChan = make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send create disk image request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> create disk image fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	var imageID string
	var err error
	if imageID, err = resp.GetString(framework.ParamKeyImage);err != nil{
		log.Printf("<api> get image from create result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	type userResponse struct {
		ID string `json:"id"`
	}
	var data = userResponse{ID:imageID}
	if "" != request.Guest{
		w.WriteHeader(http.StatusAccepted)
	}
	ResponseOK(data, w)
}


func (module *APIModule) modifyDiskImage(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var imageID = params.ByName("id")

	type userRequest struct {
		Name        string   `json:"name,omitempty"`
		Owner       string   `json:"owner,omitempty"`
		Group       string   `json:"group,omitempty"`
		Description string   `json:"description,omitempty"`
		Tags        []string `json:"tags,omitempty"`
	}
	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse modify media image request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.ModifyDiskImageRequest)
	msg.SetString(framework.ParamKeyImage, imageID)
	msg.SetString(framework.ParamKeyName, request.Name)
	msg.SetString(framework.ParamKeyUser, request.Owner)
	msg.SetString(framework.ParamKeyGroup, request.Group)
	msg.SetString(framework.ParamKeyDescription, request.Description)
	msg.SetStringArray(framework.ParamKeyTag, request.Tags)

	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send modify disk image request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> modify disk image fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}


func (module *APIModule) deleteDiskImage(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var id = params.ByName("id")
	msg, _ := framework.CreateJsonMessage(framework.DeleteDiskImageRequest)
	msg.SetString(framework.ParamKeyImage, id)
	var respChan = make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send delete disk image request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> delete disk image fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleModifyGuestName(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var id= params.ByName("id")
	type userRequest struct {
		Name string `json:"name"`
	}
	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse modify name request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.ModifyGuestNameRequest)
	msg.SetString(framework.ParamKeyGuest, id)
	msg.SetString(framework.ParamKeyName, request.Name)

	var respChan = make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send modify guest name request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> modify guest name fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleModifyGuestCores(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var id= params.ByName("id")
	type userRequest struct {
		Cores     uint `json:"cores"`
		Immediate bool `json:"immediate,omitempty"`
	}
	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse modify cores request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.ModifyCoreRequest)
	msg.SetString(framework.ParamKeyGuest, id)
	msg.SetUInt(framework.ParamKeyCore, request.Cores)
	msg.SetBoolean(framework.ParamKeyImmediate, request.Immediate)

	var respChan = make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send modify cores request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> modify cores fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleModifyGuestMemory(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var id= params.ByName("id")
	type userRequest struct {
		Memory    uint `json:"memory"`
		Immediate bool `json:"immediate,omitempty"`
	}
	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse modify memory request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.ModifyMemoryRequest)
	msg.SetString(framework.ParamKeyGuest, id)
	msg.SetUInt(framework.ParamKeyMemory, request.Memory)
	msg.SetBoolean(framework.ParamKeyImmediate, request.Immediate)

	var respChan = make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send modify memory request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> modify memory fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleModifyGuestPriority(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var id= params.ByName("id")
	type userRequest struct {
		Priority string `json:"priority"`
	}
	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse modify priority request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.ModifyPriorityRequest)
	msg.SetString(framework.ParamKeyGuest, id)

	switch request.Priority {
	case priority_label_high:
		msg.SetUInt(framework.ParamKeyPriority, PriorityHigh)
	case priority_label_medium:
		msg.SetUInt(framework.ParamKeyPriority, PriorityMedium)
	case priority_label_low:
		msg.SetUInt(framework.ParamKeyPriority, PriorityLow)
	default:
		var err = fmt.Errorf("invalid CPU priority %s", request.Priority)
		log.Printf("<api> modify with invalid CPU prioirty %s", request.Priority)
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send modify CPU priority request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> modify CPU priority fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleModifyDiskThreshold(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var id= params.ByName("id")
	type userRequest struct {
		ReadSpeed  uint64 `json:"read_speed,omitempty"`
		ReadIOPS   uint64 `json:"read_iops,omitempty"`
		WriteSpeed uint64 `json:"write_speed,omitempty"`
		WriteIOPS  uint64 `json:"write_iops,omitempty"`
	}
	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse modify disk threshold request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.ModifyDiskThresholdRequest)
	msg.SetString(framework.ParamKeyGuest, id)
	msg.SetUIntArray(framework.ParamKeyLimit, []uint64{request.ReadSpeed, request.WriteSpeed, request.ReadIOPS, request.WriteIOPS})

	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send modify nil threshold request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> modify nil threshold fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}


func (module *APIModule) handleModifyNetworkThreshold(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var id= params.ByName("id")
	type userRequest struct {
		ReceiveSpeed uint64 `json:"receive_speed,omitempty"`
		SendSpeed    uint64 `json:"send_speed,omitempty"`
	}
	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse modify network threshold request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.ModifyNetworkThresholdRequest)
	msg.SetString(framework.ParamKeyGuest, id)
	msg.SetUIntArray(framework.ParamKeyLimit, []uint64{request.ReceiveSpeed, request.SendSpeed})

	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send modify network threshold request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> modify network threshold fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleResetGuestSystem(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var guestID = params.ByName("id")
	type userRequest struct {
		FromImage string `json:"from_image"`
	}
	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse reset system request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.ResetSystemRequest)
	msg.SetString(framework.ParamKeyGuest, guestID)
	msg.SetString(framework.ParamKeyImage, request.FromImage)

	var respChan = make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send reset system request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> reset system fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}


func (module *APIModule) handleModifyGuestPassword(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var id= params.ByName("id")
	type userRequest struct {
		Password string `json:"password,omitempty"`
		User     string `json:"user,omitempty"`
	}
	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse modify password request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.ModifyAuthRequest)
	msg.SetString(framework.ParamKeyGuest, id)
	msg.SetString(framework.ParamKeySecret, request.Password)
	msg.SetString(framework.ParamKeyUser, request.User)

	var respChan = make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send modify password request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> modify password fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	type RespData struct {
		Password string `json:"password,omitempty"`
		User     string `json:"user,omitempty"`
	}
	var data RespData
	data.Password, _ = resp.GetString(framework.ParamKeySecret)
	data.User, _ = resp.GetString(framework.ParamKeyUser)
	ResponseOK(data, w)
}

func (module *APIModule) handleGetGuestPassword(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var id= params.ByName("id")
	msg, _ := framework.CreateJsonMessage(framework.GetAuthRequest)
	msg.SetString(framework.ParamKeyGuest, id)

	var respChan = make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send get password request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> get password fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	type RespData struct {
		Password string `json:"password,omitempty"`
		User     string `json:"user,omitempty"`
	}
	var data RespData
	var err error
	if data.Password, err = resp.GetString(framework.ParamKeySecret); err != nil{
		log.Printf("<api> get password fail when parse password: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	if data.User, err = resp.GetString(framework.ParamKeyUser); err != nil{
		log.Printf("<api> get password fail when parse user: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return

	}
	ResponseOK(data, w)
}

func (module *APIModule) handleResizeDisk(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var id = params.ByName("id")
	var index = params.ByName("index")
	diskOffset, err := strconv.Atoi(index)
	if err != nil{
		log.Printf("<api> try resize disk with invalid index %s", index)
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	type userRequest struct {
		Size      uint `json:"size"`
		Immediate bool `json:"immediate,omitempty"`
	}
	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse resize disk request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.ResizeDiskRequest)
	msg.SetString(framework.ParamKeyGuest, id)
	msg.SetUInt(framework.ParamKeyDisk, uint(diskOffset))
	msg.SetUInt(framework.ParamKeySize, request.Size)
	msg.SetBoolean(framework.ParamKeyImmediate, request.Immediate)

	var respChan = make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send resize disk request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> resize disk fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleShrinkDisk(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var id = params.ByName("id")
	var index = params.ByName("index")
	diskOffset, err := strconv.Atoi(index)
	if err != nil{
		log.Printf("<api> try shrink disk with invalid index %s", index)
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	type userRequest struct {
		Immediate bool `json:"immediate,omitempty"`
	}
	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse shrink disk request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.ShrinkDiskRequest)
	msg.SetString(framework.ParamKeyGuest, id)
	msg.SetUInt(framework.ParamKeyDisk, uint(diskOffset))
	msg.SetBoolean(framework.ParamKeyImmediate, request.Immediate)

	var respChan = make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send shrink disk request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> shrink disk fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleInsertMedia(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var id = params.ByName("id")

	type userRequest struct {
		Source string `json:"source"`
		Type   uint   `json:"type,omitempty"`
	}

	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse insert media request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.InsertMediaRequest)
	msg.SetString(framework.ParamKeyInstance, id)
	msg.SetString(framework.ParamKeyMedia, request.Source)
	msg.SetUInt(framework.ParamKeyType, request.Type)

	var respChan = make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send insert media request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> insert media fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleEjectMedia(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var id = params.ByName("id")

	msg, _ := framework.CreateJsonMessage(framework.EjectMediaRequest)
	msg.SetString(framework.ParamKeyInstance, id)

	var respChan = make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send eject media request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> eject media fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleQueryInstanceSnapshots(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var instanceID = params.ByName("id")
	msg, _ := framework.CreateJsonMessage(framework.QuerySnapshotRequest)
	msg.SetString(framework.ParamKeyInstance, instanceID)

	var respChan = make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send create snapshot request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> create snapshot fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	type Snapshot struct {
		IsRoot      bool   `json:"is_root,omitempty"`
		IsCurrent   bool   `json:"is_current,omitempty"`
		Backing     string `json:"backing,omitempty"`
	}

	var data = map[string]Snapshot{}
	var rootFlags, currentFlags []uint64
	var backings, names []string
	var parser = func () (err error) {
		if names, err = resp.GetStringArray(framework.ParamKeyName); err != nil{
			return
		}
		if backings, err = resp.GetStringArray(framework.ParamKeyPrevious); err != nil{
			return
		}
		if rootFlags, err = resp.GetUIntArray(framework.ParamKeySource); err != nil{
			return
		}
		if currentFlags, err = resp.GetUIntArray(framework.ParamKeyCurrent); err != nil{
			return
		}
		return nil
	}
	if err := parser(); err != nil{
		log.Printf("<api> parse snapshots fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var count = len(names)
	for i := 0; i < count; i++{
		var snapshot = Snapshot{}
		snapshot.Backing = backings[i]
		if 1 == rootFlags[i]{
			snapshot.IsRoot = true
		}
		if 1 == currentFlags[i]{
			snapshot.IsCurrent = true
		}
		data[names[i]] = snapshot
	}

	ResponseOK(data, w)
}

func (module *APIModule) handleCreateInstanceSnapshot(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var instanceID = params.ByName("id")

	type UserRequest struct {
		Name string `json:"name"`
		Description string `json:"description,omitempty"`
	}
	var err error
	var decoder = json.NewDecoder(r.Body)
	var requestData UserRequest
	if err = decoder.Decode(&requestData); err != nil{
		log.Printf("<api> decode create snapshot request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.CreateSnapshotRequest)
	msg.SetString(framework.ParamKeyInstance, instanceID)
	msg.SetString(framework.ParamKeyName, requestData.Name)
	msg.SetString(framework.ParamKeyDescription, requestData.Description)

	var respChan = make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send create snapshot request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> create snapshot fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleDeleteInstanceSnapshot(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var instanceID = params.ByName("id")
	var snapshotName = params.ByName("name")

	msg, _ := framework.CreateJsonMessage(framework.DeleteSnapshotRequest)
	msg.SetString(framework.ParamKeyInstance, instanceID)
	msg.SetString(framework.ParamKeyName, snapshotName)

	var respChan = make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send delete snapshot request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> delete snapshot fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleRestoreInstanceSnapshot(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var instanceID = params.ByName("id")
	type UserRequest struct {
		Target string `json:"target"`
	}
	var request UserRequest
	var decoder = json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil{
		log.Printf("<api> parse restore snapshot request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	var snapshotName = request.Target
	msg, _ := framework.CreateJsonMessage(framework.RestoreSnapshotRequest)
	msg.SetString(framework.ParamKeyInstance, instanceID)
	msg.SetString(framework.ParamKeyName, snapshotName)

	var respChan = make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send restore snapshot request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> restore snapshot fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleGetInstanceSnapshot(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var instanceID = params.ByName("id")
	var snapshotName = params.ByName("name")

	msg, _ := framework.CreateJsonMessage(framework.GetSnapshotRequest)
	msg.SetString(framework.ParamKeyInstance, instanceID)
	msg.SetString(framework.ParamKeyName, snapshotName)

	var respChan = make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send get snapshot request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> get snapshot fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	type ResponseData struct {
		Running bool `json:"running"`
		Description string `json:"description,omitempty"`
		CreateTime string `json:"create_time"`
	}
	var data ResponseData
	var err error
	if data.Running, err = resp.GetBoolean(framework.ParamKeyStatus); err != nil{
		log.Printf("<api> parse snapshot status fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	if data.CreateTime, err = resp.GetString(framework.ParamKeyCreate); err != nil{
		log.Printf("<api> parse snapshot create time fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	data.Description, _ = resp.GetString(framework.ParamKeyDescription)
	ResponseOK(data, w)
}

func (module *APIModule) handleQueryMigrations(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg, _ := framework.CreateJsonMessage(framework.QueryMigrationRequest)
	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send query migration request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> get migration fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	type Migration struct {
		ID       string `json:"id"`
		Finished bool   `json:"finished"`
		Progress uint   `json:"progress,omitempty"`
		Error    string `json:"error,omitempty"`
	}
	var id, errMessage []string
	var finished, progress []uint64
	var err error

	var respPayload = make([]Migration, 0)
	if id, err = resp.GetStringArray(framework.ParamKeyMigration); err != nil{
		log.Printf("<api> parse id fail when query migration: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	if finished, err = resp.GetUIntArray(framework.ParamKeyStatus); err != nil{
		log.Printf("<api> parse status fail when query migration: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	if progress, err = resp.GetUIntArray(framework.ParamKeyProgress); err != nil {
		log.Printf("<api> parse progress fail when query migration: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	if errMessage, err = resp.GetStringArray(framework.ParamKeyError); err != nil{
		log.Printf("<api> parse message fail when query migration: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var count = len(id)
	if len(finished) != count{
		var err = fmt.Errorf("unexpect status array size %d", len(finished))
		log.Printf("<api> verify status fail when query migration: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	if len(progress) != count{
		var err = fmt.Errorf("unexpect progress array size %d", len(finished))
		log.Printf("<api> verify progress fail when query migration: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	if len(errMessage) != count{
		var err = fmt.Errorf("unexpect message array size %d", len(finished))
		log.Printf("<api> verify message fail when query migration: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	for i := 0; i < count;i++{
		var m = Migration{ID: id[i]}
		if 1 == finished[i]{
			m.Finished = true
		}
		if 0 != progress[i]{
			m.Progress = uint(progress[i])
		}
		m.Error = errMessage[i]
		respPayload = append(respPayload, m)
	}
	ResponseOK(respPayload, w)
}

func (module *APIModule) handleGetMigration(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var migrationID = params.ByName("id")
	msg, _ := framework.CreateJsonMessage(framework.GetMigrationRequest)
	msg.SetString(framework.ParamKeyMigration, migrationID)
	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send get migration request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> get migration fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	type UserResponse struct {
		Finished bool   `json:"finished"`
		Progress uint   `json:"progress,omitempty"`
		Error    string `json:"error,omitempty"`
	}
	var respPayload = UserResponse{}
	respPayload.Finished, _ = resp.GetBoolean(framework.ParamKeyStatus)
	respPayload.Progress, _ = resp.GetUInt(framework.ParamKeyProgress)
	respPayload.Error, _ = resp.GetString(framework.ParamKeyError)
	ResponseOK(respPayload, w)
}

func (module *APIModule) handleCreateMigration(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	type UserRequest struct {
		SourcePool string   `json:"source_pool"`
		SourceCell string   `json:"source_cell"`
		TargetPool string   `json:"target_pool,omitempty"`
		TargetCell string   `json:"target_cell,omitempty"`
		Instances  []string `json:"instances,omitempty"`
	}
	var err error
	var decoder = json.NewDecoder(r.Body)
	var requestData UserRequest
	if err = decoder.Decode(&requestData); err != nil{
		log.Printf("<api> decode create migration request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	if "" != requestData.TargetPool{
		err = errors.New("migration between pools not support")
		log.Printf("<api> verify migration request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.CreateMigrationRequest)
	msg.SetStringArray(framework.ParamKeyPool, []string{requestData.SourcePool})
	msg.SetStringArray(framework.ParamKeyCell, []string{requestData.SourceCell, requestData.TargetCell})
	msg.SetStringArray(framework.ParamKeyInstance, requestData.Instances)

	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send create migration request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> create migration fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	type UserResponse struct {
		ID string `json:"id"`
	}
	migrationID, _ := resp.GetString(framework.ParamKeyMigration)
	var respPayload = UserResponse{migrationID}
	ResponseOK(respPayload, w)
}

func (module *APIModule) handleQueryAddressPool(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg, _ := framework.CreateJsonMessage(framework.QueryAddressPoolRequest)
	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send query address pool request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> query address pool fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}

	type Pool struct {
		Name      string   `json:"name"`
		Gateway   string   `json:"gateway"`
		DNS       []string `json:"dns,omitempty"`
		Addresses uint64   `json:"addresses"`
		Allocated uint64   `json:"allocated"`
	}

	var parser = func(msg framework.Message) (payload []Pool, err error) {
		payload = make([]Pool, 0)
		var nameArray, gatewayArray, dnsArray []string
		var addressArray, allocateArray, dnsCountArray []uint64
		if nameArray, err = msg.GetStringArray(framework.ParamKeyName); err != nil{
			return
		}
		if gatewayArray, err = msg.GetStringArray(framework.ParamKeyGateway); err != nil{
			return
		}

		if dnsArray, err = msg.GetStringArray(framework.ParamKeyServer); err != nil{
			return
		}
		if addressArray, err = msg.GetUIntArray(framework.ParamKeyAddress); err != nil{
			return
		}
		if allocateArray, err = msg.GetUIntArray(framework.ParamKeyAllocate); err != nil{
			return
		}
		if dnsCountArray, err = msg.GetUIntArray(framework.ParamKeyCount); err != nil{
			return
		}
		var count = len(nameArray)
		if count != len(gatewayArray){
			err = fmt.Errorf("unmatched gateway array size %d", len(gatewayArray))
			return
		}
		if count != len(addressArray){
			err = fmt.Errorf("unmatched address array size %d", len(addressArray))
			return
		}
		if count != len(allocateArray){
			err = fmt.Errorf("unmatched allocate array size %d", len(allocateArray))
			return
		}
		var start = 0
		for i := 0; i < count;i++{
			var dnsCount = int(dnsCountArray[i])
			var end = start + dnsCount
			var dns = dnsArray[start : end]
			payload = append(payload, Pool{nameArray[i], gatewayArray[i], dns, addressArray[i], allocateArray[i]})
			start = end
		}
		return
	}
	payload, err := parser(resp)
	if err != nil{
		log.Printf("<api> parse query address pool result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	ResponseOK(payload, w)
}

func (module *APIModule) handleGetAddressPool(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var poolName = params.ByName("pool")

	msg, _ := framework.CreateJsonMessage(framework.GetAddressPoolRequest)
	msg.SetString(framework.ParamKeyAddress, poolName)

	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send get address pool request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> get address pool fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}

	var parser = func(msg framework.Message) (payload AddressPoolStatus, err error) {
		payload.Allocated = make([]AllocatedAddress, 0)
		payload.Ranges = make([]AddressRangeConfig, 0)
		var addressArray, instanceArray, startArray, endArray, maskArray []string
		var capacityArray []uint64
		if startArray, err = msg.GetStringArray(framework.ParamKeyStart); err != nil{
			return
		}
		if endArray, err = msg.GetStringArray(framework.ParamKeyEnd); err != nil{
			return
		}
		if maskArray, err = msg.GetStringArray(framework.ParamKeyMask); err != nil{
			return
		}
		if capacityArray, err = msg.GetUIntArray(framework.ParamKeyCount); err != nil{
			return
		}
		if payload.Gateway, err = msg.GetString(framework.ParamKeyGateway); err != nil{
			return
		}
		if payload.DNS, err = msg.GetStringArray(framework.ParamKeyServer); err != nil{
			return
		}

		if addressArray, err = msg.GetStringArray(framework.ParamKeyAddress); err != nil{
			return
		}
		if instanceArray, err = msg.GetStringArray(framework.ParamKeyInstance); err != nil{
			return
		}


		var rangeCount = len(startArray)
		if rangeCount != len(endArray) {
			err = fmt.Errorf("unmatched end array size %d", len(endArray))
			return
		}
		if rangeCount != len(maskArray) {
			err = fmt.Errorf("unmatched netmask array size %d", len(maskArray))
			return
		}

		var allocatedCount = len(addressArray)
		if allocatedCount != len(instanceArray){
			err = fmt.Errorf("unmatched instance array size %d", len(instanceArray))
			return
		}

		for i := 0; i < rangeCount; i++{
			payload.Ranges = append(payload.Ranges, AddressRangeConfig{Start:startArray[i], End: endArray[i], Netmask: maskArray[i], Capacity: uint32(capacityArray[i])})
		}

		for i := 0; i < allocatedCount; i++{
			payload.Allocated = append(payload.Allocated, AllocatedAddress{addressArray[i], instanceArray[i]})
		}

		return payload, nil
	}
	payload, err := parser(resp)
	if err != nil{
		log.Printf("<api> parse get address pool result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	payload.Name = poolName
	ResponseOK(payload, w)
}

func (module *APIModule) handleCreateAddressPool(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var poolName = params.ByName("pool")
	type PoolConfig struct {
		Gateway string `json:"gateway"`
		DNS []string `json:"dns,omitempty"`
	}
	var err error
	var decoder = json.NewDecoder(r.Body)
	var request PoolConfig
	if err = decoder.Decode(&request); err != nil{
		log.Printf("<api> parse create address pool request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg, _ := framework.CreateJsonMessage(framework.CreateAddressPoolRequest)
	msg.SetString(framework.ParamKeyAddress, poolName)
	msg.SetString(framework.ParamKeyGateway, request.Gateway)
	msg.SetStringArray(framework.ParamKeyServer, request.DNS)
	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send create address pool request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> create address pool fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleModifyAddressPool(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var poolName = params.ByName("pool")
	type PoolConfig struct {
		Gateway string `json:"gateway"`
		DNS []string `json:"dns,omitempty"`
	}
	var err error
	var decoder = json.NewDecoder(r.Body)
	var request PoolConfig
	if err = decoder.Decode(&request); err != nil{
		log.Printf("<api> parse modify address pool request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg, _ := framework.CreateJsonMessage(framework.ModifyAddressPoolRequest)
	msg.SetString(framework.ParamKeyAddress, poolName)
	msg.SetString(framework.ParamKeyGateway, request.Gateway)
	msg.SetStringArray(framework.ParamKeyServer, request.DNS)
	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send modify address pool request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> modify address pool fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleDeleteAddressPool(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var poolName = params.ByName("pool")
	msg, _ := framework.CreateJsonMessage(framework.DeleteAddressPoolRequest)
	msg.SetString(framework.ParamKeyAddress, poolName)
	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send delete address pool request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> delete address pool fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleQueryAddressRange(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var poolName = params.ByName("pool")
	var rangeType = params.ByName("type")
	msg, _ := framework.CreateJsonMessage(framework.QueryAddressRangeRequest)
	msg.SetString(framework.ParamKeyAddress, poolName)
	msg.SetString(framework.ParamKeyType, rangeType)
	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send query address range request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> query address range fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	type Range struct {
		Start   string `json:"start"`
		End     string `json:"end"`
		Netmask string `json:"netmask"`
	}

	var parser = func(msg framework.Message) (payload []Range, err error) {
		payload = make([]Range, 0)
		var startArray, endArray, maskArray []string
		if startArray, err = msg.GetStringArray(framework.ParamKeyStart); err != nil{
			return
		}
		if endArray, err = msg.GetStringArray(framework.ParamKeyEnd); err != nil{
			return
		}
		if maskArray, err = msg.GetStringArray(framework.ParamKeyMask); err != nil{
			return
		}
		var count = len(startArray)
		if count != len(endArray){
			err = fmt.Errorf("unmatched end array size %d", len(endArray))
			return
		}
		if count != len(maskArray){
			err = fmt.Errorf("unmatched netmask array size %d", len(maskArray))
			return
		}
		for i := 0; i < count;i++{
			payload = append(payload, Range{startArray[i], endArray[i], maskArray[i]})
		}
		return
	}
	payload, err := parser(resp)
	if err != nil{
		log.Printf("<api> parse query address range result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	ResponseOK(payload, w)
}

func (module *APIModule) handleGetAddressRange(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var poolName = params.ByName("pool")
	var rangeType = params.ByName("type")
	var startAddress = params.ByName("start")

	msg, _ := framework.CreateJsonMessage(framework.GetAddressRangeRequest)
	msg.SetString(framework.ParamKeyAddress, poolName)
	msg.SetString(framework.ParamKeyType, rangeType)
	msg.SetString(framework.ParamKeyStart, startAddress)

	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send get address range request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> get address range fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}

	var parser = func(msg framework.Message) (payload AddressRangeStatus, err error) {
		payload.Allocated = make([]AllocatedAddress, 0)
		var addressArray, instanceArray []string
		if payload.Start, err = msg.GetString(framework.ParamKeyStart); err != nil{
			return
		}
		if payload.End, err = msg.GetString(framework.ParamKeyEnd); err != nil{
			return
		}
		if payload.Netmask, err = msg.GetString(framework.ParamKeyMask); err != nil{
			return
		}
		capacity, err := msg.GetUInt(framework.ParamKeyCount)
		if err != nil{
			return
		}
		payload.Capacity = uint32(capacity)

		if addressArray, err = msg.GetStringArray(framework.ParamKeyAddress); err != nil{
			return
		}
		if instanceArray, err = msg.GetStringArray(framework.ParamKeyInstance); err != nil{
			return
		}
		var count = len(addressArray)
		if count != len(instanceArray){
			err = fmt.Errorf("unmatched instance array size %d", len(instanceArray))
			return
		}
		payload.Allocated = make([]AllocatedAddress, 0)
		for i := 0; i< count ; i++{
			payload.Allocated = append(payload.Allocated, AllocatedAddress{addressArray[i], instanceArray[i]})
		}
		return payload, nil
	}
	payload, err := parser(resp)
	if err != nil{
		log.Printf("<api> parse get address range result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	ResponseOK(payload, w)
}

func (module *APIModule) handleAddAddressRange(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var poolName = params.ByName("pool")
	var rangeType = params.ByName("type")
	var startAddress = params.ByName("start")
	type UserRequest struct {
		End     string `json:"end"`
		Netmask string `json:"netmask"`
	}
	var err error
	var decoder = json.NewDecoder(r.Body)
	var requestData UserRequest
	if err = decoder.Decode(&requestData); err != nil{
		log.Printf("<api> parse add address range request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.AddAddressRangeRequest)
	msg.SetString(framework.ParamKeyAddress, poolName)
	msg.SetString(framework.ParamKeyType, rangeType)
	msg.SetString(framework.ParamKeyStart, startAddress)
	msg.SetString(framework.ParamKeyEnd, requestData.End)
	msg.SetString(framework.ParamKeyMask, requestData.Netmask)

	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send add address range request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> add address range fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleRemoveAddressRange(w http.ResponseWriter, r *http.Request, params httprouter.Params){
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var poolName = params.ByName("pool")
	var rangeType = params.ByName("type")
	var startAddress = params.ByName("start")

	msg, _ := framework.CreateJsonMessage(framework.RemoveAddressRangeRequest)
	msg.SetString(framework.ParamKeyAddress, poolName)
	msg.SetString(framework.ParamKeyType, rangeType)
	msg.SetString(framework.ParamKeyStart, startAddress)

	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send remove address range request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	_, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> remove address range fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	ResponseOK("", w)
}

func (module *APIModule) handleGetBatchCreateGuest(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var batchID = params.ByName("id")
	msg, _ := framework.CreateJsonMessage(framework.GetBatchCreateGuestRequest)
	msg.SetString(framework.ParamKeyID, batchID)

	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send get batch create guest request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> get batch create guest fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}

	type GuestStatus struct {
		Name     string `json:"name,omitempty"`
		ID       string `json:"id"`
		Status   string `json:"status"`
		Error    string `json:"error,omitempty"`
		Progress uint64 `json:"progress"`
	}

	const (
		StatusProcess = "creating"
		StatusSuccess = "created"
		StatusFail = "fail"
	)
	var allFinished = true
	var parser = func(msg framework.Message) (payload []GuestStatus, err error) {
		var nameArray, idArray, errArray []string
		var statusArray, progressArray []uint64
		if nameArray, err = msg.GetStringArray(framework.ParamKeyName); err != nil{
			return
		}
		if idArray, err = msg.GetStringArray(framework.ParamKeyGuest); err != nil{
			return
		}
		if errArray, err = msg.GetStringArray(framework.ParamKeyError); err != nil{
			return
		}
		if statusArray, err = msg.GetUIntArray(framework.ParamKeyStatus); err != nil{
			return
		}
		if progressArray, err = msg.GetUIntArray(framework.ParamKeyProgress); err != nil{
			return
		}
		var count = len(nameArray)
		if count != len(idArray){
			err = fmt.Errorf("unmatched id array size %d", len(idArray))
			return
		}
		if count != len(errArray){
			err = fmt.Errorf("unmatched error array size %d", len(errArray))
			return
		}
		if count != len(statusArray){
			err = fmt.Errorf("unmatched status array size %d", len(statusArray))
			return
		}
		if count != len(progressArray){
			err = fmt.Errorf("unmatched progress array size %d", len(progressArray))
			return
		}
		for i := 0; i < count;i++{
			switch statusArray[i] {
			case BatchTaskStatusProcess:
				allFinished = false
				payload = append(payload, GuestStatus{nameArray[i], idArray[i], StatusProcess,errArray[i], progressArray[i]})
			case BatchTaskStatusSuccess:
				payload = append(payload, GuestStatus{nameArray[i], idArray[i], StatusSuccess,errArray[i], progressArray[i]})
			case BatchTaskStatusFail:
				payload = append(payload, GuestStatus{nameArray[i], idArray[i], StatusFail,errArray[i], progressArray[i]})
			default:
				err = fmt.Errorf("invalid status %d for guest '%s'", statusArray[i], nameArray[i])
				return
			}
		}
		return payload, nil
	}
	payload, err := parser(resp)
	if err != nil{
		log.Printf("<api> parse batch create guest result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	if allFinished{
		w.WriteHeader(http.StatusOK)
	}else{
		w.WriteHeader(http.StatusAccepted)
	}
	ResponseOK(payload, w)
}

func (module *APIModule) handleStartBatchCreateGuest(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	type ciConfig struct {
		RootEnabled bool   `json:"root_enabled,omitempty"`
		AdminName   string `json:"admin_name,omitempty"`
		AdminSecret string `json:"admin_secret,omitempty"`
		DataPath    string `json:"data_path,omitempty"`
	}

	type userRequest struct {
		NameRule        string           `json:"name_rule"`
		NamePrefix      string           `json:"name_prefix"`
		Count           uint             `json:"count"`
		Owner           string           `json:"owner"`
		Group           string           `json:"group"`
		Pool            string           `json:"pool"`
		Cores           uint             `json:"cores"`
		Memory          uint             `json:"memory"`
		Disks           []uint64         `json:"disks"`
		AutoStart       bool             `json:"auto_start,omitempty"`
		System          string           `json:"system,omitempty"`
		NetworkAddress  string           `json:"network_address,omitempty"`
		EthernetAddress string           `json:"ethernet_address,omitempty"`
		FromImage       string           `json:"from_image,omitempty"`
		Port            []uint64         `json:"port,omitempty"`
		SystemVersion   string           `json:"system_version,omitempty"`
		Modules         []string         `json:"modules,omitempty"`
		CloudInit       *ciConfig        `json:"cloud_init,omitempty"`
		QoS             *restInstanceQoS `json:"qos,omitempty"`
	}

	decoder := json.NewDecoder(r.Body)
	var request userRequest
	if err := decoder.Decode(&request);err != nil{
		log.Printf("<api> parse batch create guest request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg, _ := framework.CreateJsonMessage(framework.StartBatchCreateGuestRequest)
	const (
		NameRuleOrder   = "order"
		NameRuleMAC     = "MAC"
		NameRuleAddress = "address"
	)
	switch request.NameRule {
	case NameRuleOrder:
		msg.SetUInt(framework.ParamKeyMode, NameRuleByOrder)
	case NameRuleMAC:
		msg.SetUInt(framework.ParamKeyMode, NameRuleByMAC)
	case NameRuleAddress:
		msg.SetUInt(framework.ParamKeyMode, NameRuleByAddress)
	default:
		var err = fmt.Errorf("invalid name rule '%s'", request.NameRule)
		log.Printf("<api> validate batch create guest request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	msg.SetString(framework.ParamKeyName, request.NamePrefix)
	msg.SetString(framework.ParamKeyUser, request.Owner)
	msg.SetString(framework.ParamKeyGroup, request.Group)
	msg.SetString(framework.ParamKeyPool, request.Pool)

	msg.SetUInt(framework.ParamKeyCount, request.Count)
	msg.SetUInt(framework.ParamKeyCore, request.Cores)
	msg.SetUInt(framework.ParamKeyMemory, request.Memory)
	msg.SetUIntArray(framework.ParamKeyDisk, request.Disks)
	msg.SetBoolean(framework.ParamKeyOption, request.AutoStart)
	msg.SetString(framework.ParamKeySystem, request.System)
	//optional disk image
	if "" != request.FromImage{
		msg.SetString(framework.ParamKeyImage, request.FromImage)
	}
	msg.SetString(framework.ParamKeyVersion, request.SystemVersion)
	msg.SetStringArray(framework.ParamKeyModule, request.Modules)
	const (
		RootLoginDisabled = iota
		RootLoginEnabled
	)
	var flags []uint64
	if nil != request.CloudInit{
		msg.SetString(framework.ParamKeyAdmin, request.CloudInit.AdminName)
		msg.SetString(framework.ParamKeySecret, request.CloudInit.AdminSecret)
		msg.SetString(framework.ParamKeyPath, request.CloudInit.DataPath)
		if request.CloudInit.RootEnabled{
			flags = append(flags, RootLoginEnabled)
		}else{
			flags = append(flags, RootLoginDisabled)
		}

	}else{
		msg.SetString(framework.ParamKeyAdmin, "")
		msg.SetString(framework.ParamKeySecret, "")
		msg.SetString(framework.ParamKeyPath, "")
		flags = append(flags, RootLoginEnabled)
	}

	if nil != request.QoS{
		var qos = request.QoS
		switch qos.CPUPriority {
		case priority_label_high:
			msg.SetUInt(framework.ParamKeyPriority, PriorityHigh)
		case priority_label_medium:
			msg.SetUInt(framework.ParamKeyPriority, PriorityMedium)
		case priority_label_low:
			msg.SetUInt(framework.ParamKeyPriority, PriorityLow)
		default:
			var err = fmt.Errorf("invalid CPU priority %s", qos.CPUPriority)
			log.Printf("<api> invalid batch create request with CPU priority: %s", qos.CPUPriority)
			ResponseFail(ResponseDefaultError, err.Error(), w)
			return
		}
		msg.SetUIntArray(framework.ParamKeyLimit, []uint64{qos.ReadSpeed, qos.WriteSpeed,
			qos.ReadIOPS, qos.WriteIOPS, qos.ReceiveSpeed, qos.SendSpeed})
	}else{
		msg.SetUInt(framework.ParamKeyPriority, PriorityHigh)
		msg.SetUIntArray(framework.ParamKeyLimit, []uint64{0, 0, 0, 0, 0, 0})
	}

	msg.SetUIntArray(framework.ParamKeyFlag, flags)

	respChan := make(chan ProxyResult)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send batch create request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> batch create guest fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	batchID, err := resp.GetString(framework.ParamKeyID)
	if err != nil{
		log.Printf("<api> parse batch id fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	type userResponse struct {
		ID string `json:"id"`
	}

	var result = userResponse{ID: batchID}
	w.WriteHeader(http.StatusAccepted)
	ResponseOK(result, w)
}

func (module *APIModule) handleGetBatchDeleteGuest(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var batchID = params.ByName("id")
	msg, _ := framework.CreateJsonMessage(framework.GetBatchDeleteGuestRequest)
	msg.SetString(framework.ParamKeyID, batchID)

	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send get batch delete guest request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> get batch delete guest fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}

	type GuestStatus struct {
		Name   string `json:"name,omitempty"`
		ID     string `json:"id"`
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}

	const (
		StatusProcess = "deleting"
		StatusSuccess = "deleted"
		StatusFail = "fail"
	)

	var allFinished = true
	var parser = func(msg framework.Message) (payload []GuestStatus, err error) {
		var nameArray, idArray, errArray []string
		var statusArray []uint64
		if nameArray, err = msg.GetStringArray(framework.ParamKeyName); err != nil{
			return
		}
		if idArray, err = msg.GetStringArray(framework.ParamKeyGuest); err != nil{
			return
		}
		if errArray, err = msg.GetStringArray(framework.ParamKeyError); err != nil{
			return
		}
		if statusArray, err = msg.GetUIntArray(framework.ParamKeyStatus); err != nil{
			return
		}
		var count = len(nameArray)
		if count != len(idArray){
			err = fmt.Errorf("unmatched id array size %d", len(idArray))
			return
		}
		if count != len(errArray){
			err = fmt.Errorf("unmatched error array size %d", len(errArray))
			return
		}
		if count != len(statusArray){
			err = fmt.Errorf("unmatched status array size %d", len(statusArray))
			return
		}
		for i := 0; i < count;i++{
			switch statusArray[i] {
			case BatchTaskStatusProcess:
				payload = append(payload, GuestStatus{nameArray[i], idArray[i], StatusProcess,errArray[i] })
				allFinished = false
			case BatchTaskStatusSuccess:
				payload = append(payload, GuestStatus{nameArray[i], idArray[i], StatusSuccess,errArray[i] })
			case BatchTaskStatusFail:
				payload = append(payload, GuestStatus{nameArray[i], idArray[i], StatusFail,errArray[i] })
			default:
				err = fmt.Errorf("invalid status %d for guest '%s'", statusArray[i], nameArray[i])
				return
			}
		}
		return payload, nil
	}
	payload, err := parser(resp)
	if err != nil{
		log.Printf("<api> parse batch delete guest result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	if allFinished{
		w.WriteHeader(http.StatusOK)
	}else{
		w.WriteHeader(http.StatusAccepted)
	}
	ResponseOK(payload, w)
}

func (module *APIModule) handleStartBatchDeleteGuest(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	type UserRequest struct {
		Guest []string `json:"guest"`
	}
	var err error
	var decoder = json.NewDecoder(r.Body)
	var requestData UserRequest
	if err = decoder.Decode(&requestData); err != nil{
		log.Printf("<api> parse start batch delete request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	msg, _ := framework.CreateJsonMessage(framework.StartBatchDeleteGuestRequest)
	msg.SetStringArray(framework.ParamKeyGuest, requestData.Guest)

	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send start batch delete request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> start batch delete fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	batchID, err := resp.GetString(framework.ParamKeyID)
	if err != nil{
		log.Printf("<api> parse batch id fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	type userResponse struct {
		ID string `json:"id"`
	}

	var result = userResponse{ID: batchID}
	w.WriteHeader(http.StatusAccepted)
	ResponseOK(result, w)
}


func (module *APIModule) handleGetBatchStopGuest(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var batchID = params.ByName("id")
	msg, _ := framework.CreateJsonMessage(framework.GetBatchStopGuestRequest)
	msg.SetString(framework.ParamKeyID, batchID)

	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send get batch stop guest request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> get batch stop guest fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}

	type GuestStatus struct {
		Name   string `json:"name,omitempty"`
		ID     string `json:"id"`
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}

	const (
		StatusProcess = "stopping"
		StatusSuccess = "stopped"
		StatusFail = "fail"
	)

	var allFinished = true
	var parser = func(msg framework.Message) (payload []GuestStatus, err error) {
		var nameArray, idArray, errArray []string
		var statusArray []uint64
		if nameArray, err = msg.GetStringArray(framework.ParamKeyName); err != nil{
			return
		}
		if idArray, err = msg.GetStringArray(framework.ParamKeyGuest); err != nil{
			return
		}
		if errArray, err = msg.GetStringArray(framework.ParamKeyError); err != nil{
			return
		}
		if statusArray, err = msg.GetUIntArray(framework.ParamKeyStatus); err != nil{
			return
		}
		var count = len(nameArray)
		if count != len(idArray){
			err = fmt.Errorf("unmatched id array size %d", len(idArray))
			return
		}
		if count != len(errArray){
			err = fmt.Errorf("unmatched error array size %d", len(errArray))
			return
		}
		if count != len(statusArray){
			err = fmt.Errorf("unmatched status array size %d", len(statusArray))
			return
		}
		for i := 0; i < count;i++{
			switch statusArray[i] {
			case BatchTaskStatusProcess:
				payload = append(payload, GuestStatus{nameArray[i], idArray[i], StatusProcess,errArray[i] })
				allFinished = false
			case BatchTaskStatusSuccess:
				payload = append(payload, GuestStatus{nameArray[i], idArray[i], StatusSuccess,errArray[i] })
			case BatchTaskStatusFail:
				payload = append(payload, GuestStatus{nameArray[i], idArray[i], StatusFail,errArray[i] })
			default:
				err = fmt.Errorf("invalid status %d for guest '%s'", statusArray[i], nameArray[i])
				return
			}
		}
		return payload, nil
	}
	payload, err := parser(resp)
	if err != nil{
		log.Printf("<api> parse batch stop guest result fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	if allFinished{
		w.WriteHeader(http.StatusOK)
	}else{
		w.WriteHeader(http.StatusAccepted)
	}
	ResponseOK(payload, w)
}

func (module *APIModule) handleStartBatchStopGuest(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	if err := module.verifyRequestSignature(r); err != nil{
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	type UserRequest struct {
		Guest []string `json:"guest"`
		Force bool     `json:"force,omitempty"`
	}
	var err error
	var decoder = json.NewDecoder(r.Body)
	var requestData UserRequest
	if err = decoder.Decode(&requestData); err != nil{
		log.Printf("<api> parse start batch stop request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}

	const (
		ForceStopEnable  = 1
		ForceStopDisable = 1
		RebootDisable    = 0
	)

	var options []uint64
	if requestData.Force{
		options = []uint64{RebootDisable, ForceStopEnable}
	}else{
		options = []uint64{RebootDisable, ForceStopDisable}
	}

	msg, _ := framework.CreateJsonMessage(framework.StartBatchStopGuestRequest)
	msg.SetStringArray(framework.ParamKeyGuest, requestData.Guest)
	msg.SetUIntArray(framework.ParamKeyOption, options)

	var respChan = make(chan ProxyResult, 1)
	if err := module.proxy.SendRequest(msg, respChan); err != nil {
		log.Printf("<api> send start batch stop request fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	resp, errMsg, success := IsResponseSuccess(respChan)
	if !success {
		log.Printf("<api> start batch stop fail: %s", errMsg)
		ResponseFail(ResponseDefaultError, errMsg, w)
		return
	}
	batchID, err := resp.GetString(framework.ParamKeyID)
	if err != nil{
		log.Printf("<api> parse batch id fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	type userResponse struct {
		ID string `json:"id"`
	}

	var result = userResponse{ID: batchID}
	w.WriteHeader(http.StatusAccepted)
	ResponseOK(result, w)
}


func IsResponseSuccess(respChan chan ProxyResult) (resp framework.Message, errMessage string, success bool) {
	result, ok := <-respChan
	if !ok {
		return resp, "channel closed", false
	}
	if result.Error != nil {
		return resp, result.Error.Error(), false
	}
	if !result.Response.IsSuccess() {
		return resp, result.Response.GetError(), false
	}
	return result.Response, "", true
}

type Response struct {
	ErrorCode int         `json:"error_code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
}

const (
	ResponseDefaultError = 500
)

func ResponseFail(code int, message string, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(Response{code, message, struct{}{}})
}

func ResponseOK(data interface{}, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(Response{0, "", data})
}
