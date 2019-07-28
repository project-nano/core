package modules

import (
	"github.com/project-nano/framework"
	"fmt"
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
	InstanceStatusLostBit    = 2;
	InstanceStatusMigrateBit = 3;
)

const (
	InstanceMediaOptionNone    uint = iota
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

func MarshalInstanceStatusListToMessage(list []InstanceStatus, msg framework.Message) error {
	var count = uint(len(list))
	msg.SetUInt(framework.ParamKeyCount, count)
	var names, ids, pools, cells, users, monitors, addresses, groups, secrets, systems, createTime, internal, external []string
	var cores, options, enables, progress, status, memories, disks, diskCounts, mediaAttached, cpuPriorities, ioLimits []uint64
	for _, ins := range list {
		names = append(names, ins.Name)
		ids = append(ids, ins.ID)
		pools = append(pools, ins.Pool)
		cells = append(cells, ins.Cell)
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
	msg.SetStringArray(framework.ParamKeyUser, users)

	msg.SetStringArray(framework.ParamKeyMonitor, monitors)
	msg.SetStringArray(framework.ParamKeySecret, secrets)
	msg.SetStringArray(framework.ParamKeyAddress, addresses)
	msg.SetStringArray(framework.ParamKeySystem, systems)
	msg.SetStringArray(framework.ParamKeyCreate, createTime)
	msg.SetStringArray(framework.ParamKeyInternal, internal)
	msg.SetStringArray(framework.ParamKeyExternal, external)

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

func (config *InstanceStatus) Marshal(msg framework.Message) error {
	msg.SetUInt(framework.ParamKeyCore, config.Cores)
	msg.SetUInt(framework.ParamKeyMemory, config.Memory)
	msg.SetUIntArray(framework.ParamKeyDisk, config.Disks)

	msg.SetString(framework.ParamKeyName, config.Name)
	msg.SetString(framework.ParamKeyUser, config.User)
	msg.SetString(framework.ParamKeyGroup, config.Group)
	msg.SetString(framework.ParamKeyPool, config.Pool)
	msg.SetString(framework.ParamKeyCell, config.Cell)
	if config.ID != "" {
		msg.SetString(framework.ParamKeyInstance, config.ID)
	}
	msg.SetBoolean(framework.ParamKeyEnable, config.Created)
	msg.SetUInt(framework.ParamKeyProgress, config.Progress)

	if config.AutoStart {
		msg.SetUIntArray(framework.ParamKeyOption, []uint64{1})
	} else {
		msg.SetUIntArray(framework.ParamKeyOption, []uint64{0})
	}
	msg.SetBoolean(framework.ParamKeyMedia, config.MediaAttached)
	var insStatus uint
	if config.Running {
		insStatus = InstanceStatusRunning
	} else {
		insStatus = InstanceStatusStopped
	}
	if config.Lost {
		insStatus |= 1 << InstanceStatusLostBit
	}

	msg.SetUInt(framework.ParamKeyStatus, insStatus)
	msg.SetString(framework.ParamKeySecret, config.MonitorSecret)
	msg.SetString(framework.ParamKeySystem, config.System)
	msg.SetString(framework.ParamKeyCreate, config.CreateTime)
	var internalMonitor = fmt.Sprintf("%s:%d", config.InternalNetwork.MonitorAddress, config.InternalNetwork.MonitorPort)
	var externalMonitor = fmt.Sprintf("%s:%d", config.ExternalNetwork.MonitorAddress, config.ExternalNetwork.MonitorPort)
	msg.SetStringArray(framework.ParamKeyMonitor, []string{internalMonitor, externalMonitor})
	msg.SetStringArray(framework.ParamKeyAddress, []string{config.InternalNetwork.InstanceAddress, config.ExternalNetwork.InstanceAddress})
	msg.SetString(framework.ParamKeyInternal, config.InternalNetwork.AssignedAddress)
	msg.SetString(framework.ParamKeyExternal, config.ExternalNetwork.AssignedAddress)
	//QoS
	msg.SetUInt(framework.ParamKeyPriority, uint(config.CPUPriority))
	msg.SetUIntArray(framework.ParamKeyLimit, []uint64{config.ReadSpeed, config.WriteSpeed, config.ReadIOPS,
		config.WriteIOPS, config.ReceiveSpeed, config.SendSpeed})
	return nil
}
