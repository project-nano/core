package main

import (
	"github.com/project-nano/framework"
	"net"
	"testing"
)

func getCoreServiceForTest() (core *CoreService, err error) {
	const (
		domainName    = "nano"
		groupAddress  = "224.0.0.226"
		groupPort     = 5599
		listenAddress = "192.168.1.167"
		configPath    = "config"
		dataPath      = "data"
	)
	var inf *net.Interface
	if inf, err = framework.InterfaceByAddress(listenAddress); nil != err {
		return
	}
	var endpoint framework.EndpointService
	if endpoint, err = framework.CreateStubEndpoint(groupAddress, groupPort, domainName, listenAddress); err != nil {
		return
	}
	core = &CoreService{EndpointService: endpoint, ConfigPath: configPath, DataPath: dataPath}
	core.RegisterHandler(core)
	err = core.GenerateName(framework.ServiceTypeCore, inf)
	return
}

func TestCoreService_StartAndStop(t *testing.T) {
	core, err := getCoreServiceForTest()
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
	t.Log("test core service start and stop success")
}
