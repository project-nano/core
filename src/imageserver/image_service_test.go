package imageserver

import (
	"github.com/project-nano/framework"
	"net"
	"testing"
)

func getImageServiceForTest() (image *ImageService, err error) {
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
	core, err := getImageServiceForTest()
	if err != nil {
		t.Fatalf("load service fail: %s", err.Error())
		return
	}
	if err = core.Start(); err != nil {
		t.Fatalf("start service fail: %s", err.Error())
		return
	}
	if err = core.Stop(); err != nil {
		t.Fatalf("stop service fail: %s", err.Error())
	}
	t.Log("test image service start and stop success")
}
