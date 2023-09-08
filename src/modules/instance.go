package modules

import (
	"fmt"
	"github.com/project-nano/framework"
)

type InstanceResource struct {
	Cores  uint
	Memory uint
	Disks  []uint64
}

type PriorityEnum uint

const (
	PriorityHigh = iota
	PriorityMedium
	PriorityLow
)

type InstanceNetworkInfo struct {
	InstanceAddress string
	MonitorAddress  string
	AssignedAddress string
	MonitorPort     uint
	MappedPorts     map[int]int
}

type InstanceStatus struct {
	InstanceResource
	Name            string
	ID              string
	Pool            string
	Cell            string
	Host            string //hosting cell ip
	User            string
	Group           string
	AutoStart       bool
	System          string
	Created         bool
	Progress        uint //limit to 100
	Running         bool
	Lost            bool
	Migrating       bool
	InternalNetwork InstanceNetworkInfo
	ExternalNetwork InstanceNetworkInfo
	MediaAttached   bool
	MediaSource     string
	MediaName       string
	MonitorProtocol string
	MonitorSecret   string
	CreateTime      string
	HardwareAddress string
	CPUPriority     PriorityEnum
	WriteSpeed      uint64
	WriteIOPS       uint64
	ReadSpeed       uint64
	ReadIOPS        uint64
	ReceiveSpeed    uint64
	SendSpeed       uint64
}

const (
	InstanceStatusStopped = iota
	InstanceStatusRunning
)

const (
	//bit 0~1 for running/stopped
	InstanceStatusLostBit    = 2
	InstanceStatusMigrateBit = 3
)

const (
	InstanceMediaOptionNone uint = iota
	InstanceMediaOptionImage
	InstanceMediaOptionNetwork
)

const (
	NetworkModePrivate = iota
	NetworkModePlain
	NetworkModeMono
	NetworkModeShare
	NetworkModeVPC
)

const (
	StorageModeLocal = iota
)

func (instance *InstanceStatus) IsVisible(userID, groupID string) (visible bool) {
	if userID == instance.User {
		return true
	} else if "" != groupID {
		//by group
		return groupID == instance.Group
	}
	return false
}

func MarshalInstanceStatusListToMessage(list []InstanceStatus, msg framework.Message) error {
	var count = uint(len(list))
	msg.SetUInt(framework.ParamKeyCount, count)
	var names, ids, pools, cells, hosts, users, monitors, addresses, groups, secrets, systems,
		createTime, internal, external, hardware []string
	var cores, options, enables, progress, status, memories, disks, diskCounts, mediaAttached, cpuPriorities, ioLimits []uint64
	for _, ins := range list {
		names = append(names, ins.Name)
		ids = append(ids, ins.ID)
		pools = append(pools, ins.Pool)
		cells = append(cells, ins.Cell)
		hosts = append(hosts, ins.Host)
		users = append(users, ins.User)
		groups = append(groups, ins.Group)
		cores = append(cores, uint64(ins.Cores))
		if ins.AutoStart {
			options = append(options, 1)
		} else {
			options = append(options, 0)
		}
		if ins.MediaAttached {
			mediaAttached = append(mediaAttached, 1)
		} else {
			mediaAttached = append(mediaAttached, 0)
		}
		if ins.Created {
			enables = append(enables, 1)
			progress = append(progress, 0)
		} else {
			enables = append(enables, 0)
			progress = append(progress, uint64(ins.Progress))
		}
		var insStatus uint64
		if ins.Running {
			insStatus = InstanceStatusRunning
		} else {
			insStatus = InstanceStatusStopped
		}
		if ins.Lost {
			insStatus |= 1 << InstanceStatusLostBit
		}
		status = append(status, insStatus)

		secrets = append(secrets, ins.MonitorSecret)
		var internalMonitor = fmt.Sprintf("%s:%d", ins.InternalNetwork.MonitorAddress, ins.InternalNetwork.MonitorPort)
		var externalMonitor = fmt.Sprintf("%s:%d", ins.ExternalNetwork.MonitorAddress, ins.ExternalNetwork.MonitorPort)
		monitors = append(monitors, internalMonitor)
		monitors = append(monitors, externalMonitor)
		addresses = append(addresses, ins.InternalNetwork.InstanceAddress)
		addresses = append(addresses, ins.ExternalNetwork.InstanceAddress)

		internal = append(internal, ins.InternalNetwork.AssignedAddress)
		external = append(external, ins.ExternalNetwork.AssignedAddress)

		systems = append(systems, ins.System)
		createTime = append(createTime, ins.CreateTime)
		hardware = append(hardware, ins.HardwareAddress)
		memories = append(memories, uint64(ins.Memory))
		var diskCount = len(ins.Disks)
		diskCounts = append(diskCounts, uint64(diskCount))
		for _, diskSize := range ins.Disks {
			disks = append(disks, diskSize)
		}
		//QoS
		cpuPriorities = append(cpuPriorities, uint64(ins.CPUPriority))
		ioLimits = append(ioLimits, []uint64{ins.ReadSpeed, ins.WriteSpeed,
			ins.ReadIOPS, ins.WriteIOPS, ins.ReceiveSpeed, ins.SendSpeed}...)
	}

	msg.SetStringArray(framework.ParamKeyName, names)
	msg.SetStringArray(framework.ParamKeyInstance, ids)
	msg.SetStringArray(framework.ParamKeyPool, pools)
	msg.SetStringArray(framework.ParamKeyCell, cells)
	msg.SetStringArray(framework.ParamKeyHost, hosts)
	msg.SetStringArray(framework.ParamKeyUser, users)

	msg.SetStringArray(framework.ParamKeyMonitor, monitors)
	msg.SetStringArray(framework.ParamKeySecret, secrets)
	msg.SetStringArray(framework.ParamKeyAddress, addresses)
	msg.SetStringArray(framework.ParamKeySystem, systems)
	msg.SetStringArray(framework.ParamKeyCreate, createTime)
	msg.SetStringArray(framework.ParamKeyInternal, internal)
	msg.SetStringArray(framework.ParamKeyExternal, external)
	msg.SetStringArray(framework.ParamKeyHardware, hardware)

	msg.SetStringArray(framework.ParamKeyGroup, groups)
	msg.SetUIntArray(framework.ParamKeyCore, cores)
	msg.SetUIntArray(framework.ParamKeyOption, options)
	msg.SetUIntArray(framework.ParamKeyEnable, enables)
	msg.SetUIntArray(framework.ParamKeyProgress, progress)
	msg.SetUIntArray(framework.ParamKeyStatus, status)
	msg.SetUIntArray(framework.ParamKeyMemory, memories)
	msg.SetUIntArray(framework.ParamKeyCount, diskCounts)
	msg.SetUIntArray(framework.ParamKeyDisk, disks)
	msg.SetUIntArray(framework.ParamKeyMedia, mediaAttached)
	msg.SetUIntArray(framework.ParamKeyPriority, cpuPriorities)
	msg.SetUIntArray(framework.ParamKeyLimit, ioLimits)
	return nil
}

func (instance *InstanceStatus) Marshal(msg framework.Message) error {
	msg.SetUInt(framework.ParamKeyCore, instance.Cores)
	msg.SetUInt(framework.ParamKeyMemory, instance.Memory)
	msg.SetUIntArray(framework.ParamKeyDisk, instance.Disks)

	msg.SetString(framework.ParamKeyName, instance.Name)
	msg.SetString(framework.ParamKeyUser, instance.User)
	msg.SetString(framework.ParamKeyGroup, instance.Group)
	msg.SetString(framework.ParamKeyPool, instance.Pool)
	msg.SetString(framework.ParamKeyCell, instance.Cell)
	msg.SetString(framework.ParamKeyHost, instance.Host)
	if instance.ID != "" {
		msg.SetString(framework.ParamKeyInstance, instance.ID)
	}
	msg.SetBoolean(framework.ParamKeyEnable, instance.Created)
	msg.SetUInt(framework.ParamKeyProgress, instance.Progress)

	if instance.AutoStart {
		msg.SetUIntArray(framework.ParamKeyOption, []uint64{1})
	} else {
		msg.SetUIntArray(framework.ParamKeyOption, []uint64{0})
	}
	msg.SetBoolean(framework.ParamKeyMedia, instance.MediaAttached)
	var insStatus uint
	if instance.Running {
		insStatus = InstanceStatusRunning
	} else {
		insStatus = InstanceStatusStopped
	}
	if instance.Lost {
		insStatus |= 1 << InstanceStatusLostBit
	}

	msg.SetUInt(framework.ParamKeyStatus, insStatus)
	msg.SetString(framework.ParamKeySecret, instance.MonitorSecret)
	msg.SetString(framework.ParamKeySystem, instance.System)
	msg.SetString(framework.ParamKeyCreate, instance.CreateTime)
	msg.SetString(framework.ParamKeyHardware, instance.HardwareAddress)
	var internalMonitor = fmt.Sprintf("%s:%d", instance.InternalNetwork.MonitorAddress, instance.InternalNetwork.MonitorPort)
	var externalMonitor = fmt.Sprintf("%s:%d", instance.ExternalNetwork.MonitorAddress, instance.ExternalNetwork.MonitorPort)
	msg.SetStringArray(framework.ParamKeyMonitor, []string{internalMonitor, externalMonitor})
	msg.SetStringArray(framework.ParamKeyAddress, []string{instance.InternalNetwork.InstanceAddress, instance.ExternalNetwork.InstanceAddress})
	msg.SetString(framework.ParamKeyInternal, instance.InternalNetwork.AssignedAddress)
	msg.SetString(framework.ParamKeyExternal, instance.ExternalNetwork.AssignedAddress)
	//QoS
	msg.SetUInt(framework.ParamKeyPriority, uint(instance.CPUPriority))
	msg.SetUIntArray(framework.ParamKeyLimit, []uint64{instance.ReadSpeed, instance.WriteSpeed, instance.ReadIOPS,
		instance.WriteIOPS, instance.ReceiveSpeed, instance.SendSpeed})
	return nil
}
