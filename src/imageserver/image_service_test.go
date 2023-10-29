package imageserver

import (
	"fmt"
	"github.com/project-nano/framework"
	"net"
	"testing"
)

// dummyService implement ServiceHandler
type dummyService struct {
}

func (dummy *dummyService) OnMessageReceived(msg framework.Message) {
	fmt.Printf("dummy service receive message: %s\n", msg)
}

func (dummy *dummyService) OnServiceConnected(name string, t framework.ServiceType, address string) {
	fmt.Printf("dummy service connected: %s, %d, %s\n", name, t, address)
}

func (dummy *dummyService) OnServiceDisconnected(name string, t framework.ServiceType, gracefully bool) {
	fmt.Printf("dummy service disconnected: %s, %d, %t\n", name, t, gracefully)
}

// OnDependencyReady
func (dummy *dummyService) OnDependencyReady() {
	fmt.Println("dummy service dependency ready")
}

// InitialEndpoint
func (dummy *dummyService) InitialEndpoint() error {
	fmt.Println("dummy service initial endpoint")
	return nil
}

func (dummy *dummyService) OnEndpointStarted() error {
	fmt.Println("dummy service endpoint started")
	return nil
}

func (dummy *dummyService) OnEndpointStopped() {
	fmt.Println("dummy service endpoint stopped")
}

func getImageServiceForTest() (image *ImageService, stub framework.EndpointService, err error) {
	const (
		domainName    = "nano"
		groupAddress  = "224.0.0.226"
		groupPort     = 5599
		listenAddress = "192.168.1.167"
		configPath    = "../../config"
		dataPath      = "../../data"
	)
	var inf *net.Interface
	if inf, err = framework.InterfaceByAddress(listenAddress); nil != err {
		return
	}
	if stub, err = framework.CreateStubEndpoint(groupAddress, groupPort, domainName, listenAddress); err != nil {
		return
	}
	var dummy dummyService
	stub.RegisterHandler(&dummy)
	var endpoint framework.EndpointService
	if endpoint, err = framework.CreatePeerEndpoint(groupAddress, groupPort, domainName); err != nil {
		return
	}
	image = &ImageService{EndpointService: endpoint, ConfigPath: configPath, DataPath: dataPath}
	image.RegisterHandler(image)
	err = image.GenerateName(framework.ServiceTypeImage, inf)
	return
}

func TestImageService_StartAndStop(t *testing.T) {
	imageService, stub, err := getImageServiceForTest()
	if err != nil {
		t.Fatalf("load service fail: %s", err.Error())
		return
	}
	if err = stub.Start(); err != nil {
		t.Fatalf("start stub fail: %s", err.Error())
		return
	}
	defer func() {
		_ = stub.Stop()
	}()
	if err = imageService.Start(); err != nil {
		t.Fatalf("start service fail: %s", err.Error())
		return
	}
	if err = imageService.Stop(); err != nil {
		t.Fatalf("stop service fail: %s", err.Error())
	}
	t.Log("test image service start and stop success")
}
