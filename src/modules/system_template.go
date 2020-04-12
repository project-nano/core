package modules

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
	OperatingSystem string `json:"operating_system"`
	Disk            string `json:"disk"`
	Network         string `json:"network"`
	Display         string `json:"display"`
	Control         string `json:"control"`
	USB             string `json:"usb,omitempty"`
	Tablet          string `json:"tablet,omitempty"`
}

type SystemTemplate struct {
	ID string `json:"id"`
	SystemTemplateConfig
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
