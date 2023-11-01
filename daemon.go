package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/project-nano/core/imageserver"
	"github.com/project-nano/core/modules"
	"github.com/project-nano/framework"
	"github.com/project-nano/sonar"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

type DomainConfig struct {
	Domain        string `json:"domain"`
	GroupAddress  string `json:"group_address"`
	GroupPort     int    `json:"group_port"`
	ListenAddress string `json:"listen_address"`
	Timeout       int    `json:"timeout,omitempty"`
}

type MainService struct {
	core  *CoreService
	image *imageserver.ImageService
}

const (
	ProjectName           = "nano"
	ExecuteName           = "core"
	DomainConfigFileName  = "domain.cfg"
	APIConfigFilename     = "api.cfg"
	ImageConfigFilename   = "image.cfg"
	ConfigPathName        = "config"
	DataPathName          = "data"
	DefaultPathPerm       = 0740
	DefaultFilePerm       = 0640
	defaultOperateTimeout = 10 //10 seconds
)

func (service *MainService) Start() (output string, err error) {
	if nil == service.core {
		err = errors.New("invalid service")
		return
	}
	if err = service.core.Start(); err != nil {
		return
	}
	if err = service.image.Start(); err != nil {
		return
	}
	cert, key := service.image.GetTLSFilePath()
	output = fmt.Sprintf("\nCore Module %s\nservice %s listen at '%s:%d'\ngroup '%s:%d', domain '%s'\nAPI address '%s', image service '%s'\nImage TLS Cert '%s', Key '%s'",
		service.core.GetVersion(),
		service.core.GetName(), service.core.GetListenAddress(), service.core.GetListenPort(),
		service.core.GetGroupAddress(), service.core.GetGroupPort(), service.core.GetDomain(),
		service.core.GetAPIServiceAddress(), service.image.GetImageServiceAddress(),
		cert, key)
	return
}

func (service *MainService) Stop() (output string, err error) {
	if nil == service.core {
		err = errors.New("invalid service")
		return
	}
	if err = service.image.Stop(); err != nil {
		return
	}
	if err = service.core.Stop(); err != nil {
		return
	}
	return
}

func (service *MainService) Snapshot() (output string, err error) {
	output = "hello, this is stub for snapshot"
	return
}

func createDaemon(workingPath string) (service framework.DaemonizedService, err error) {
	var configPath = filepath.Join(workingPath, ConfigPathName)
	var configFile = filepath.Join(configPath, DomainConfigFileName)
	var data []byte
	data, err = os.ReadFile(configFile)
	if err != nil {
		return
	}
	var config DomainConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return
	}
	var dataPath = filepath.Join(workingPath, DataPathName)
	if _, err = os.Stat(dataPath); os.IsNotExist(err) {
		if err = os.Mkdir(dataPath, DefaultPathPerm); err != nil {
			return
		} else {
			log.Printf("data path '%s' created", dataPath)
		}
	}
	var inf *net.Interface
	inf, err = framework.InterfaceByAddress(config.ListenAddress)
	if err != nil {
		return
	}

	endpointCore, err := framework.CreateStubEndpoint(config.GroupAddress, config.GroupPort, config.Domain, config.ListenAddress)
	if err != nil {
		return
	}
	//set timeout
	if config.Timeout > 0 {
		modules.GetConfigurator().SetOperateTimeout(config.Timeout)
	}

	var s = MainService{}
	s.core = &CoreService{EndpointService: endpointCore, ConfigPath: configPath, DataPath: dataPath}
	s.core.RegisterHandler(s.core)
	err = s.core.GenerateName(framework.ServiceTypeCore, inf)
	if err != nil {
		return
	}
	endpointImage, err := framework.CreatePeerEndpoint(config.GroupAddress, config.GroupPort, config.Domain)
	if err != nil {
		return
	}
	s.image = &imageserver.ImageService{EndpointService: endpointImage, ConfigPath: configPath, DataPath: dataPath}
	s.image.RegisterHandler(s.image)
	if err = s.image.GenerateName(framework.ServiceTypeImage, inf); err != nil {
		return
	}
	return &s, nil
}

func main() {
	framework.ProcessDaemon(ExecuteName, generateConfigure, createDaemon)
}

func generateConfigure(workingPath string) (err error) {
	var configPath = filepath.Join(workingPath, ConfigPathName)
	if _, err = os.Stat(configPath); os.IsNotExist(err) {
		//create path
		err = os.Mkdir(configPath, DefaultPathPerm)
		if err != nil {
			return
		}
		fmt.Printf("config path %s created\n", configPath)
	}
	if err = generateDomainConfig(configPath); err != nil {
		return
	}
	if err = generateAPIConfig(configPath); err != nil {
		return
	}
	if err = generateImageConfig(workingPath, configPath); err != nil {
		return
	}
	return
}

func generateDomainConfig(configPath string) (err error) {
	var configFile = filepath.Join(configPath, DomainConfigFileName)
	if _, err = os.Stat(configFile); os.IsNotExist(err) {
		fmt.Println("No domain config available, following instructions to generate a new one.")

		var config = DomainConfig{
			Timeout: defaultOperateTimeout,
		}
		if config.Domain, err = framework.InputString("Group Domain Name", sonar.DefaultDomain); err != nil {
			return
		}
		if config.GroupAddress, err = framework.InputMultiCastAddress("Group MultiCast Address", sonar.DefaultMulticastAddress); err != nil {
			return
		}
		if config.GroupPort, err = framework.InputNetworkPort("Group MultiCast Port", sonar.DefaultMulticastPort); err != nil {
			return
		}
		if config.ListenAddress, err = framework.ChooseIPV4Address("Listen Address"); err != nil {
			return
		}
		//write
		var data []byte
		data, err = json.MarshalIndent(config, "", " ")
		if err != nil {
			return
		}
		if err = os.WriteFile(configFile, data, DefaultFilePerm); err != nil {
			return
		}
		fmt.Printf("domain configure '%s' generated\n", configFile)
	}
	return
}

func generateAPIConfig(configPath string) (err error) {
	const (
		DefaultAPIServePort = 5850
	)
	var configFile = filepath.Join(configPath, APIConfigFilename)
	if _, err = os.Stat(configFile); os.IsNotExist(err) {
		fmt.Println("No API config available, following instructions to generate a new one.")

		var config = modules.APIConfig{}
		if config.Port, err = framework.InputInteger("API Serve Port", DefaultAPIServePort); err != nil {
			return
		}
		//write
		var data []byte
		data, err = json.MarshalIndent(config, "", " ")
		if err != nil {
			return
		}
		if err = os.WriteFile(configFile, data, DefaultFilePerm); err != nil {
			return
		}
		fmt.Printf("api configure '%s' generated\n", configFile)
	}
	return
}

func generateImageConfig(workingPath, configPath string) (err error) {
	const (
		RootPath     = "/opt"
		CertPathName = "cert"
	)
	//cert file
	var certFileName = fmt.Sprintf("%s_image.crt.pem", ProjectName)
	var keyFileName = fmt.Sprintf("%s_image.key.pem", ProjectName)

	var generatedCertFile = filepath.Join(workingPath, CertPathName, certFileName)
	var generatedKeyFile = filepath.Join(workingPath, CertPathName, keyFileName)
	if _, err = os.Stat(generatedCertFile); os.IsNotExist(err) {
		fmt.Println("No cert file available, following instructions to generate a new one.")
		var certPath = filepath.Join(workingPath, CertPathName)
		//generate new cert & key pair
		if _, err = os.Stat(certPath); os.IsNotExist(err) {
			if err = os.Mkdir(certPath, DefaultPathPerm); err != nil {
				return
			} else {
				fmt.Printf("cert path '%s' created\n", certPath)
			}
		}
		var defaultRootCertPath = filepath.Join(RootPath, ProjectName, CertPathName)
		var rootCertPath string
		if rootCertPath, err = framework.InputString("Root Cert File Location", defaultRootCertPath); err != nil {
			return
		}
		var rootCertFile = filepath.Join(rootCertPath, fmt.Sprintf("%s_ca.crt.pem", ProjectName))
		var rootKeyFile = filepath.Join(rootCertPath, fmt.Sprintf("%s_ca.key.pem", ProjectName))
		if _, err = os.Stat(rootCertFile); os.IsNotExist(err) {
			return
		}
		if _, err = os.Stat(rootKeyFile); os.IsNotExist(err) {
			return
		}
		var localAddress string
		if localAddress, err = framework.ChooseIPV4Address("Image Server Address"); err != nil {
			return
		}
		if err = signImageCertificate(rootCertFile, rootKeyFile, localAddress, generatedCertFile, generatedKeyFile); err != nil {
			return
		}
	}

	var configFile = filepath.Join(configPath, ImageConfigFilename)
	if _, err = os.Stat(configFile); os.IsNotExist(err) {

		var config = imageserver.ImageServiceConfig{generatedCertFile, generatedKeyFile}
		//write
		var data []byte
		data, err = json.MarshalIndent(config, "", " ")
		if err != nil {
			return
		}
		if err = os.WriteFile(configFile, data, DefaultFilePerm); err != nil {
			return
		}
		fmt.Printf("image configure '%s' generated\n", configFile)
	}
	return
}

func signImageCertificate(caCert, caKey, localAddress, certPath, keyPath string) (err error) {
	const (
		RSAKeyBits           = 2048
		DefaultDurationYears = 99
	)
	rootPair, err := tls.LoadX509KeyPair(caCert, caKey)
	if err != nil {
		return
	}
	rootCA, err := x509.ParseCertificate(rootPair.Certificate[0])
	if err != nil {
		return err
	}
	var serialNumber = big.NewInt(1700)
	var imageCert = x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   fmt.Sprintf("%s ImageServer", ProjectName),
			Organization: []string{ProjectName},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(DefaultDurationYears, 0, 0),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment,
		IPAddresses: []net.IP{net.ParseIP(localAddress)},
	}
	var imagePrivate *rsa.PrivateKey
	imagePrivate, err = rsa.GenerateKey(rand.Reader, RSAKeyBits)
	if err != nil {
		return
	}
	fmt.Printf("private key with %d bits generated\n", RSAKeyBits)
	var imagePublic = imagePrivate.PublicKey
	var certContent []byte
	certContent, err = x509.CreateCertificate(rand.Reader, &imageCert, rootCA, &imagePublic, rootPair.PrivateKey)
	if err != nil {
		return
	}
	// Public key
	var certFile *os.File
	certFile, err = os.Create(certPath)
	if err != nil {
		return
	}
	if err = pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certContent}); err != nil {
		return
	}
	if err = certFile.Close(); err != nil {
		return
	}
	fmt.Printf("cert file '%s' generated\n", certPath)

	// Private key
	var keyFile *os.File
	keyFile, err = os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, DefaultFilePerm)
	if err != nil {
		os.Remove(certPath)
		return
	}
	if err = pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(imagePrivate)}); err != nil {
		os.Remove(certPath)
		return
	}
	if err = keyFile.Close(); err != nil {
		os.Remove(certPath)
		return
	}
	fmt.Printf("key file '%s' generated\n", keyPath)
	return nil
}
