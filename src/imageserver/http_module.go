package imageserver

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/project-nano/framework"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type HttpModule struct {
	listenHost   string
	listenPort   int
	listener     net.Listener
	certFile     string
	keyFile      string
	server       http.Server
	imageManager *ImageManager
	runner       *framework.SimpleRunner
}

type ImageServiceConfig struct {
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`
}

const (
	APIRoot    = "/api"
	APIVersion = 1
)

func CreateHttpModule(configPath, dataPath, host string, image *ImageManager) (module *HttpModule, err error) {
	const (
		ListenPortRangeBegin = 5801
		ListenPortRange      = 100
		ListenPortRangeEnd   = ListenPortRangeBegin + ListenPortRange
		ConfigFilename       = "image.cfg"
	)
	var configFile = filepath.Join(configPath, ConfigFilename)
	var config ImageServiceConfig

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if _, err := os.Stat(config.CertFile); os.IsNotExist(err) {
		return nil, err
	}
	if _, err := os.Stat(config.KeyFile); os.IsNotExist(err) {
		return nil, err
	}
	if err = syncCertificateAddress(configPath, config.CertFile, config.KeyFile, host); err != nil {
		return
	}
	module = &HttpModule{}
	module.runner = framework.CreateSimpleRunner(module.Routine)
	var found = false
	for port := ListenPortRangeBegin; port < ListenPortRangeEnd; port++ {
		var address = fmt.Sprintf("%s:%d", host, port)
		listener, err := net.Listen("tcp", address)
		if err != nil {
			continue
		}
		found = true
		module.listener = listener
		module.listenHost = host
		module.listenPort = port
		module.server.Addr = address
		break
	}
	if !found {
		return nil, errors.New("no port available")
	}

	//bind handler
	var router = httprouter.New()
	module.RegisterHandler(router)
	module.server.Handler = router
	module.imageManager = image
	module.certFile = config.CertFile
	module.keyFile = config.KeyFile
	return module, nil
}

func syncCertificateAddress(configPath, certPath, keyPath, host string) (err error) {
	currentPair, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return err
	}
	if 0 == len(currentPair.Certificate) {
		err = fmt.Errorf("no certificate data in %s", certPath)
		return
	}
	currentCertificate, err := x509.ParseCertificate(currentPair.Certificate[0])
	if err != nil {
		return err
	}
	if 0 == len(currentCertificate.IPAddresses) {
		err = fmt.Errorf("no IP address in %s", certPath)
		return
	}
	var hostAddress = net.ParseIP(host)
	if nil == hostAddress {
		err = fmt.Errorf("invalid host address '%s'", host)
		return
	}
	var currentAddress = currentCertificate.IPAddresses[0]
	if currentAddress.Equal(hostAddress) {
		//Equal
		return nil
	}
	//generate from RootCA
	var pathTree = strings.Split(configPath, string(os.PathSeparator))
	var pathDepth = len(pathTree)
	var projectPath string
	if pathDepth > 2 {
		var pathNames = []string{"/"}
		pathNames = append(pathNames, pathTree[:pathDepth-2]...)
		projectPath = path.Join(pathNames...)
	} else {
		projectPath = "/"
	}

	const (
		CertPathName         = "cert"
		ProjectName          = "nano"
		DefaultDurationYears = 99
		RSAKeyBits           = 2048
		CertSerialNumber     = 1700
		TypeCertificate      = "CERTIFICATE"
		TypePrivateKey       = "RSA PRIVATE KEY"
		DefaultFilePerm      = 0640
	)
	var rootInstallPath = path.Join(projectPath, CertPathName)
	var rootCertPath = path.Join(rootInstallPath, "nano_ca.crt.pem")
	var rootKeyPath = path.Join(rootInstallPath, "nano_ca.key.pem")
	if _, err = os.Stat(rootCertPath); os.IsNotExist(err) {
		err = fmt.Errorf("can not find root cert file %s", rootCertPath)
		return
	}
	if _, err = os.Stat(rootKeyPath); os.IsNotExist(err) {
		err = fmt.Errorf("can not find root key file %s", rootKeyPath)
		return
	}

	rootPair, err := tls.LoadX509KeyPair(rootCertPath, rootKeyPath)
	if err != nil {
		return
	}
	rootCA, err := x509.ParseCertificate(rootPair.Certificate[0])
	if err != nil {
		return err
	}
	var serialNumber = big.NewInt(CertSerialNumber)
	var imageCertificate = x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   fmt.Sprintf("%s ImageServer", ProjectName),
			Organization: []string{ProjectName},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(DefaultDurationYears, 0, 0),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment,
		IPAddresses: []net.IP{hostAddress},
	}
	var privateKey *rsa.PrivateKey
	privateKey, err = rsa.GenerateKey(rand.Reader, RSAKeyBits)
	if err != nil {
		return
	}

	var publicKey = privateKey.PublicKey
	var keyData []byte
	keyData, err = x509.CreateCertificate(rand.Reader, &imageCertificate, rootCA, &publicKey, rootPair.PrivateKey)
	if err != nil {
		return
	}
	// Public key
	var publicKeyFile *os.File
	publicKeyFile, err = os.Create(certPath)
	if err != nil {
		return
	}
	if err = pem.Encode(publicKeyFile, &pem.Block{Type: TypeCertificate, Bytes: keyData}); err != nil {
		return
	}
	if err = publicKeyFile.Close(); err != nil {
		return
	}

	var privateKeyFile *os.File
	privateKeyFile, err = os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, DefaultFilePerm)
	if err != nil {
		os.Remove(certPath)
		return
	}
	if err = pem.Encode(privateKeyFile, &pem.Block{Type: TypePrivateKey, Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}); err != nil {
		os.Remove(certPath)
		return
	}
	if err = privateKeyFile.Close(); err != nil {
		os.Remove(certPath)
		return
	}
	log.Printf("<img_http> synchronize IP address of certificate '%s' to '%s'", certPath, host)
	return nil
}

func (module *HttpModule) Start() error {
	return module.runner.Start()
}

func (module *HttpModule) Stop() error {
	return module.runner.Stop()
}

func (module *HttpModule) Routine(c framework.RoutineController) {
	var finished = make(chan bool)
	go module.HttpRoutine(finished)
	log.Println("<img_http> started")
	for !c.IsStopping() {
		select {
		case <-c.GetNotifyChannel():
			c.SetStopping()
			break

		}
	}
	log.Println("<img_http> stopping http server...")
	ctx, _ := context.WithCancel(context.TODO())
	module.server.Shutdown(ctx)
	<-finished
	log.Println("<img_http> stopped")
	c.NotifyExit()
}

func (module *HttpModule) HttpRoutine(finished chan bool) {
	if err := module.server.ServeTLS(module.listener, module.certFile, module.keyFile); err != nil {
		log.Printf("<img_http> serve finished: %s", err.Error())
	}
	finished <- true
}

func (module *HttpModule) GetHost() string {
	return module.listenHost
}

func (module *HttpModule) GetPort() int {
	return module.listenPort
}

func (module *HttpModule) GetCertFilePath() string {
	return module.certFile
}

func (module *HttpModule) GetKeyFilePath() string {
	return module.keyFile
}

func apiPath(path string) string {
	return fmt.Sprintf("%s/v%d%s", APIRoot, APIVersion, path)
}

func (module *HttpModule) RegisterHandler(router *httprouter.Router) {

	router.HEAD(apiPath("/media_images/:id/file/"), module.CheckMediaImageFile)
	router.GET(apiPath("/media_images/:id/file/"), module.DownloadMediaImageFile)
	router.POST(apiPath("/media_images/:id/file/"), module.UploadMediaImageFile)

	router.HEAD(apiPath("/disk_images/:id/file/"), module.CheckDiskImageFile)
	router.GET(apiPath("/disk_images/:id/file/"), module.ReadDiskImageFile)
	router.PUT(apiPath("/disk_images/:id/file/"), module.WriteDiskImageFile)
	router.POST(apiPath("/disk_images/:id/file/"), module.uploadDiskImageFile) //upload form
}

func (module *HttpModule) UploadMediaImageFile(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var id = params.ByName("id")
	var targetFile string
	{
		//lock for update
		var respChan = make(chan ImageResult)
		module.imageManager.LockMediaImageForUpdate(id, respChan)
		result := <-respChan
		if result.Error != nil {
			err := result.Error
			log.Printf("<img_http> lock media image fail: %s", err.Error())
			ResponseFail(ResponseDefaultError, err.Error(), w)
			return
		}
		targetFile = result.Path
	}
	multiReader, err := r.MultipartReader()
	if err != nil {
		module.UnlockMediaImage(id)
		log.Printf("<img_http> prepare multi fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var sourceFile io.ReadCloser
	for {
		part, err := multiReader.NextPart()
		if err == io.EOF {
			module.UnlockMediaImage(id)
			err = errors.New("no image part available")
			log.Printf("<img_http> parse image part fail: %s", err.Error())
			ResponseFail(ResponseDefaultError, err.Error(), w)
			return
		}
		if "image" == part.FormName() {
			sourceFile = part
			break
		}
	}
	if err != nil {
		module.UnlockMediaImage(id)
		log.Printf("<img_http> read upload media image fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	log.Printf("<img_http> upload stream ready for media image '%s'", id)
	imageWriter, err := os.Create(targetFile)
	if err != nil {
		module.UnlockMediaImage(id)
		log.Printf("<img_http> create file for upload media image fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	log.Printf("<img_http> target file '%s' ready for upload media image '%s'", targetFile, id)
	if _, err = io.Copy(imageWriter, sourceFile); err != nil {
		module.UnlockMediaImage(id)
		os.Remove(targetFile)
		log.Printf("<img_http> upload media image fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	if err = imageWriter.Close(); err != nil {
		module.UnlockMediaImage(id)
		os.Remove(targetFile)
		log.Printf("<img_http> close uploaded media image fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	log.Printf("<img_http> media image '%s' all data uploaded", id)
	{
		//update&unlock
		var respChan = make(chan error)
		module.imageManager.FinishMediaImage(id, respChan)
		err = <-respChan
		if err != nil {
			module.CancelLockedDiskImage(id)
			os.Remove(targetFile)
			log.Printf("<img_http> update media image fail: %s", err.Error())
			ResponseFail(ResponseDefaultError, err.Error(), w)
			return
		}
	}
	ResponseOK("", w)
}

func (module *HttpModule) DownloadMediaImageFile(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var id = params.ByName("id")
	var respChan = make(chan ImageResult)
	module.imageManager.GetMediaImageFile(id, respChan)
	var result = <-respChan
	if result.Error != nil {
		err := result.Error
		log.Printf("<img_http> get media image fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Transfer-Encoding", "binary")
	w.Header().Set("Expires", "0")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	http.ServeFile(w, r, result.Path)
}

func (module *HttpModule) CheckMediaImageFile(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var id = params.ByName("id")
	{
		var respChan = make(chan ImageResult)
		module.imageManager.GetMediaImageFile(id, respChan)
		var result = <-respChan
		if result.Error != nil {
			err := result.Error
			log.Printf("<img_http> check media image fail: %s", err.Error())
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Transfer-Encoding", "binary")
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", result.Size))
		w.Header().Set("Expires", "0")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		log.Printf("<img_http> media image '%s' available, %d MB in size", id, result.Size>>20)
	}
}

func (module *HttpModule) UnlockMediaImage(id string) {
	var respChan = make(chan error)
	module.imageManager.UnlockMediaImage(id, respChan)
	var err = <-respChan
	if err != nil {
		log.Printf("<img_http> unlock media image ('%s') fail: %s", id, err.Error())
	}
}

func (module *HttpModule) CheckDiskImageFile(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var id = params.ByName("id")

	var respChan = make(chan ImageResult)
	module.imageManager.GetDiskImageFile(id, respChan)
	var result = <-respChan
	if result.Error != nil {
		err := result.Error
		log.Printf("<img_http> check disk image fail: %s", err.Error())
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Transfer-Encoding", "binary")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", result.Size))
	w.Header().Set("Expires", "0")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	log.Printf("<img_http> disk image '%s' available, %d MB in size", id, result.Size>>20)

}

func (module *HttpModule) ReadDiskImageFile(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var id = params.ByName("id")
	var respChan = make(chan ImageResult)
	module.imageManager.GetDiskImageFile(id, respChan)
	var result = <-respChan
	if result.Error != nil {
		err := result.Error
		log.Printf("<img_http> get disk image fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Transfer-Encoding", "binary")
	w.Header().Set("Expires", "0")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Signature", result.CheckSum)
	http.ServeFile(w, r, result.Path)
}

func (module *HttpModule) WriteDiskImageFile(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	const (
		ImageFieldName    = "image"
		CheckSumFieldName = "checksum"
	)
	var id = params.ByName("id")

	log.Printf("<img_http> recv write disk image '%s', content-length %d", id, r.ContentLength)
	var targetFile string
	{
		//lock for update
		var respChan = make(chan ImageResult)
		module.imageManager.LockDiskImageForUpdate(id, respChan)
		result := <-respChan
		if result.Error != nil {
			err := result.Error
			log.Printf("<img_http> lock disk image fail: %s", err.Error())
			ResponseFail(ResponseDefaultError, err.Error(), w)
			return
		}
		log.Printf("<img_http> disk image '%s' locked", id)
		targetFile = result.Path
	}

	multiReader, err := r.MultipartReader()
	if err != nil {
		module.CancelLockedDiskImage(id)
		log.Printf("<img_http> prepare multi fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var sourceFile io.ReadCloser
	var checksum string
	var streamReady, checkSumReady = false, false
	for {
		part, err := multiReader.NextPart()
		if err == io.EOF {
			err = errors.New("no more part available")
			module.CancelLockedDiskImage(id)
			log.Printf("<img_http> parse upload part fail: %s", err.Error())
			ResponseFail(ResponseDefaultError, err.Error(), w)
			return
		}
		if CheckSumFieldName == part.FormName() {
			data, err := ioutil.ReadAll(part)
			if err != nil {
				module.CancelLockedDiskImage(id)
				log.Printf("<img_http> read check sum fail: %s", err.Error())
				ResponseFail(ResponseDefaultError, err.Error(), w)
				return
			}
			checksum = string(data)
			checkSumReady = true
		}
		if ImageFieldName == part.FormName() {
			sourceFile = part
			streamReady = true
		}
		if checkSumReady && streamReady {
			break
		}
	}

	log.Printf("<img_http> upload stream ready for writing disk image '%s'", id)
	imageWriter, err := os.Create(targetFile)
	if err != nil {
		module.CancelLockedDiskImage(id)
		log.Printf("<img_http> create file for writing disk image fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	log.Printf("<img_http> target file '%s' ready for writing disk image '%s'", targetFile, id)
	if _, err = io.Copy(imageWriter, sourceFile); err != nil {
		module.CancelLockedDiskImage(id)
		os.Remove(targetFile)
		log.Printf("<img_http> upload disk image fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	if err = imageWriter.Close(); err != nil {
		module.CancelLockedDiskImage(id)
		os.Remove(targetFile)
		log.Printf("<img_http> close uploaded disk image fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	log.Printf("<img_http> disk image '%s' all data uploaded, checking integrity...", id)
	//checksum
	if err = checkFileIntegrity(targetFile, checksum); err != nil {
		module.CancelLockedDiskImage(id)
		os.Remove(targetFile)
		log.Printf("<img_http> check file integrity fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	{
		//update
		var respChan = make(chan error)
		module.imageManager.FinishDiskImage(id, checksum, respChan)
		err = <-respChan
		if err != nil {
			os.Remove(targetFile)
			module.imageManager.UnlockDiskImage(id, respChan)
			<-respChan
			log.Printf("<img_http> update disk image fail: %s", err.Error())
			ResponseFail(ResponseDefaultError, err.Error(), w)
			return
		}
	}
	ResponseOK("", w)
}

func (module *HttpModule) uploadDiskImageFile(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	const (
		ImageFieldName = "image"
	)
	var id = params.ByName("id")

	log.Printf("<img_http> recv upload disk image '%s', content-length %d", id, r.ContentLength)
	var targetFile string
	{
		//lock for update
		var respChan = make(chan ImageResult)
		module.imageManager.LockDiskImageForUpdate(id, respChan)
		result := <-respChan
		if result.Error != nil {
			err := result.Error
			log.Printf("<img_http> lock disk image fail: %s", err.Error())
			ResponseFail(ResponseDefaultError, err.Error(), w)
			return
		}
		log.Printf("<img_http> disk image '%s' locked", id)
		targetFile = result.Path
	}

	multiReader, err := r.MultipartReader()
	if err != nil {
		log.Printf("<img_http> prepare multi fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	var sourceFile io.ReadCloser
	for {
		part, err := multiReader.NextPart()
		if err == io.EOF {
			err = errors.New("no more part available")
			module.CancelLockedDiskImage(id)
			log.Printf("<img_http> parse upload part fail: %s", err.Error())
			ResponseFail(ResponseDefaultError, err.Error(), w)
			return
		}
		if ImageFieldName == part.FormName() {
			sourceFile = part
			break
		}
	}

	log.Printf("<img_http> upload stream ready for uploading disk image '%s'", id)
	imageWriter, err := os.Create(targetFile)
	if err != nil {
		module.CancelLockedDiskImage(id)
		log.Printf("<img_http> create file for uploading disk image fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	log.Printf("<img_http> target file '%s' ready for uploading disk image '%s'", targetFile, id)
	if _, err = io.Copy(imageWriter, sourceFile); err != nil {
		module.CancelLockedDiskImage(id)
		os.Remove(targetFile)
		log.Printf("<img_http> upload disk image fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	if err = imageWriter.Close(); err != nil {
		module.CancelLockedDiskImage(id)
		os.Remove(targetFile)
		log.Printf("<img_http> close uploaded disk image fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	log.Printf("<img_http> disk image '%s' all data uploaded, checking integrity...", id)
	//checksum
	checksum, err := computeCheckSum(targetFile)
	if err != nil {
		module.CancelLockedDiskImage(id)
		os.Remove(targetFile)
		log.Printf("<img_http> compute check sum fail: %s", err.Error())
		ResponseFail(ResponseDefaultError, err.Error(), w)
		return
	}
	{
		//update
		var respChan = make(chan error)
		module.imageManager.FinishDiskImage(id, checksum, respChan)
		err = <-respChan
		if err != nil {
			os.Remove(targetFile)
			module.imageManager.UnlockDiskImage(id, respChan)
			<-respChan
			log.Printf("<img_http> update disk image fail: %s", err.Error())
			ResponseFail(ResponseDefaultError, err.Error(), w)
			return
		}
	}
	ResponseOK("", w)
}

func (module *HttpModule) CancelLockedDiskImage(id string) {
	var respChan = make(chan error)
	module.imageManager.UnlockDiskImage(id, respChan)
	var err = <-respChan
	if err != nil {
		log.Printf("<img_http> cancel locked disk image ('%s') fail: %s", id, err.Error())
	}
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

func checkFileIntegrity(path, expect string) (err error) {
	checkSum, err := computeCheckSum(path)
	if err != nil {
		return
	}
	if checkSum != expect {
		err = fmt.Errorf("checksum is '%s', but '%s' expected", checkSum, expect)
	}
	return nil
}

func computeCheckSum(path string) (checkSum string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	var checkBuffer = make([]byte, 4<<20) //4M buffer
	var hash = sha1.New()
	if _, err = io.CopyBuffer(hash, file, checkBuffer); err != nil {
		return
	}
	checkSum = hex.EncodeToString(hash.Sum(nil))
	return checkSum, nil
}
