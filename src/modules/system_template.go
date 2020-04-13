package modules

import (
	"fmt"
	"github.com/rs/xid"
	"time"
)

type TemplateOperatingSystem int

const (
	TemplateOperatingSystemLinux = iota
	TemplateOperatingSystemWindows
	TemplateOperatingSystemInvalid
)

func (value TemplateOperatingSystem) ToString() string{
	switch value{
	case TemplateOperatingSystemLinux:
		return SystemNameLinux
	case TemplateOperatingSystemWindows:
		return SystemNameWindows
	default:
		return "invalid"
	}
}

type TemplateDiskDriver int

const (
	TemplateDiskDriverSCSI = iota
	TemplateDiskDriverSATA
	TemplateDiskDriverIDE
	TemplateDiskDriverInvalid
)

func (value TemplateDiskDriver) ToString() string{
	switch value{
	case TemplateDiskDriverSCSI:
		return DiskBusSCSI
	case TemplateDiskDriverSATA:
		return DiskBusSATA
	case TemplateDiskDriverIDE:
		return DiskBusIDE
	default:
		return "invalid"
	}
}

type TemplateNetworkModel int

const (
	TemplateNetworkModelVirtIO = iota
	TemplateNetworkModelE1000
	TemplateNetworkModelRTL18139
	TemplateNetworkModelInvalid
)

func (value TemplateNetworkModel) ToString() string{
	switch value{
	case TemplateNetworkModelVirtIO:
		return NetworkModelVIRTIO
	case TemplateNetworkModelE1000:
		return NetworkModelE1000
	case TemplateNetworkModelRTL18139:
		return NetworkModelRTL8139
	default:
		return "invalid"
	}
}

type TemplateDisplayDriver int

const (
	TemplateDisplayDriverVGA = iota
	TemplateDisplayDriverCirrus
	TemplateDisplayDriverVirtIO
	TemplateDisplayDriverQXL
	TemplateDisplayDriverNone
	TemplateDisplayDriverInvalid
)

func (value TemplateDisplayDriver) ToString() string{
	switch value{
	case TemplateDisplayDriverVGA:
		return DisplayDriverVGA
	case TemplateDisplayDriverCirrus:
		return DisplayDriverCirrus
	case TemplateDisplayDriverVirtIO:
		return DisplayDriverVirtIO
	case TemplateDisplayDriverQXL:
		return DisplayDriverQXL
	case TemplateDisplayDriverNone:
		return DisplayDriverNone
	default:
		return "invalid"
	}
}

type TemplateRemoteControl int

const (
	TemplateRemoteControlVNC = iota
	TemplateRemoteControlSPICE
	TemplateRemoteControlInvalid
)

func (value TemplateRemoteControl) ToString() string{
	switch value{
	case TemplateRemoteControlVNC:
		return RemoteControlVNC
	case TemplateRemoteControlSPICE:
		return RemoteControlSPICE
	default:
		return "invalid"
	}
}

type TemplateUSBModel int

const (
	TemplateUSBModelNone = iota
	TemplateUSBModelXHCI
	TemplateUSBModelInvalid
)

func (value TemplateUSBModel) ToString() string{
	switch value{
	case TemplateUSBModelNone:
		return USBModelNone
	case TemplateUSBModelXHCI:
		return USBModelXHCI
	default:
		return "invalid"
	}
}

type TemplateTabletModel int

const (
	TemplateTabletModelNone = iota
	TemplateTabletModelUSB
	TemplateTabletModelVirtIO
	TemplateTabletModelInvalid
)

func (value TemplateTabletModel) ToString() string{
	switch value{
	case TemplateTabletModelNone:
		return TabletBusNone
	case TemplateTabletModelUSB:
		return TabletBusUSB
	case TemplateTabletModelVirtIO:
		return TabletBusVIRTIO
	default:
		return "invalid"
	}
}

type SystemTemplateConfig struct {
	Name            string `json:"name"`
	Admin           string `json:"admin"`
	OperatingSystem string `json:"operating_system"`
	Disk            string `json:"disk"`
	Network         string `json:"network"`
	Display         string `json:"display"`
	Control         string `json:"control"`
	USB             string `json:"usb,omitempty"`
	Tablet          string `json:"tablet,omitempty"`
}

type SystemTemplate struct {
	ID           string `json:"id"`
	SystemTemplateConfig
	CreatedTime  string `json:"created_time"`
	ModifiedTime string `json:"modified_time"`
}

const (
	SystemNameLinux     = "linux"
	SystemNameWindows   = "windows"
	DiskBusIDE          = "ide"
	DiskBusSCSI         = "scsi"
	DiskBusSATA         = "sata"
	NetworkModelRTL8139 = "rtl8139"
	NetworkModelE1000   = "e1000"
	NetworkModelVIRTIO  = "virtio"
	DisplayDriverVGA    = "vga"
	DisplayDriverCirrus = "cirrus"
	DisplayDriverQXL    = "qxl"
	DisplayDriverVirtIO = "virtio"
	DisplayDriverNone   = "none"
	RemoteControlVNC    = "vnc"
	RemoteControlSPICE  = "spice"
	USBModelXHCI        = "nec-xhci"
	USBModelNone        = ""
	TabletBusNone       = ""
	TabletBusVIRTIO     = "virtio"
	TabletBusUSB        = "usb"
)

func CreateSystemTemplate(config SystemTemplateConfig) SystemTemplate {
	var now = time.Now().Format(TimeFormatLayout)
	var t = SystemTemplate{
		ID:                   xid.New().String(),
		SystemTemplateConfig: config,
		CreatedTime:          now,
		ModifiedTime:         now,
	}
	return t
}

func (config SystemTemplateConfig) toOptions() (options []uint64, err error){
	switch config.Disk {
	case DiskBusSCSI:
		options = append(options, TemplateDiskDriverSCSI)
	case DiskBusSATA:
		options = append(options, TemplateDiskDriverSATA)
	case DiskBusIDE:
		options = append(options, TemplateDiskDriverIDE)
	default:
		err = fmt.Errorf("invalid disk option '%s'", config.Disk)
		return
	}
	//network
	switch config.Network {
	case NetworkModelVIRTIO:
		options = append(options, TemplateNetworkModelVirtIO)
	case NetworkModelE1000:
		options = append(options, TemplateNetworkModelE1000)
	case NetworkModelRTL8139:
		options = append(options, TemplateNetworkModelRTL18139)
	default:
		err = fmt.Errorf("invalid network option '%s'", config.Network)
		return
	}
	//display
	switch config.Display {
	case DisplayDriverVGA:
		options = append(options, TemplateDisplayDriverVGA)
	case DisplayDriverCirrus:
		options = append(options, TemplateDisplayDriverCirrus)
	case DisplayDriverQXL:
		options = append(options, TemplateDisplayDriverQXL)
	case DisplayDriverVirtIO:
		options = append(options, TemplateDisplayDriverVirtIO)
	case DisplayDriverNone:
		options = append(options, TemplateDisplayDriverNone)
	default:
		err = fmt.Errorf("invalid display option '%s'", config.Display)
		return
	}
	//remote control
	switch config.Control {
	case RemoteControlVNC:
		options = append(options, TemplateRemoteControlVNC)
	case RemoteControlSPICE:
		options = append(options, TemplateRemoteControlSPICE)
	default:
		err = fmt.Errorf("invalid remote control option '%s'", config.Control)
		return
	}
	//use device
	switch config.USB {
	case USBModelNone:
		options = append(options, TemplateUSBModelNone)
	case USBModelXHCI:
		options = append(options, TemplateUSBModelXHCI)
	default:
		err = fmt.Errorf("invalid usb option '%s'", config.USB)
		return
	}
	//tablet
	switch config.Tablet {
	case TabletBusNone:
		options = append(options, TemplateTabletModelNone)
	case TabletBusUSB:
		options = append(options, TemplateTabletModelUSB)
	case TabletBusVIRTIO:
		options = append(options, TemplateTabletModelVirtIO)
	default:
		err = fmt.Errorf("invalid tablet option '%s'", config.Tablet)
		return
	}
	return
}
