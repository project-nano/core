package modules

import (
	"fmt"
	"github.com/project-nano/framework"
	"time"
)

const (
	DefaultConfigPerm     = 0640
	DefaultOperateTimeout = 5 * time.Second
)

type GuestQueryCondition struct {
	Pool           string
	InCell         bool
	WithOwner      bool
	WithGroup      bool
	WithStatus     bool
	WithCreateFlag bool
	Cell           string
	Owner          string
	Group          string
	Status         int
	Created        bool
}

type ComputePoolInfo struct {
	Name      string
	Enabled   bool
	Network   string
	Storage   string
	Failover  bool
	CellCount uint64
}

type StoragePoolInfo struct {
	Name   string
	Type   string
	Host   string
	Target string
}

type ComputeCellInfo struct {
	Name           string `json:"name"`
	Address        string `json:"address"`
	Enabled        bool   `json:"enabled"`
	Alive          bool   `json:"alive"`
	PurgeAppending bool
}

type ResourceUsage struct {
	Cores           uint
	CpuUsage        float64
	Memory          uint64
	MemoryAvailable uint64
	Disk            uint64
	DiskAvailable   uint64
	BytesWritten    uint64
	BytesRead       uint64
	BytesSent       uint64
	BytesReceived   uint64
	WriteSpeed      uint64
	ReadSpeed       uint64
	SendSpeed       uint64
	ReceiveSpeed    uint64
}

type PoolStatistic struct {
	EnabledPools  uint64
	DisabledPools uint64
}

type CellStatistic struct {
	OfflineCells uint64
	OnlineCells  uint64
}

type InstanceStatistic struct {
	StoppedInstances   uint64
	RunningInstances   uint64
	LostInstances      uint64
	MigratingInstances uint64
}

type ZoneStatus struct {
	Name string
	PoolStatistic
	CellStatistic
	InstanceStatistic
	ResourceUsage
	StartTime time.Time
}

type ComputePoolStatus struct {
	Name    string
	Enabled bool
	CellStatistic
	InstanceStatistic
	ResourceUsage
}

type ComputeCellStatus struct {
	ComputeCellInfo
	InstanceStatistic
	ResourceUsage
}

type ResourceResult struct {
	Error               error
	Pool                string
	Cell                string
	Name                string
	Host                string
	Port                int
	Batch               string
	DiskImageList       []DiskImageStatus
	ComputeCellInfoList []ComputeCellInfo
	ComputePoolInfoList []ComputePoolInfo
	ComputePoolConfig   ComputePoolInfo
	Instance            InstanceStatus
	DiskImage           DiskImageStatus
	Zone                ZoneStatus
	ComputePool         ComputePoolStatus
	ComputePoolList     []ComputePoolStatus
	ComputeCell         ComputeCellStatus
	ComputeCellList     []ComputeCellStatus
	InstanceList        []InstanceStatus
	StoragePool         StoragePoolInfo
	StoragePoolList     []StoragePoolInfo
	Migration           MigrationStatus
	MigrationList       []MigrationStatus
	FailoverPlan        map[string][]string
	AddressPool         AddressPoolStatus
	AddressPoolList     []AddressPoolStatus
	AddressRange        AddressRangeStatus
	AddressRangeList    []AddressRangeStatus
	BatchCreate         []CreateGuestStatus
	BatchDelete         []DeleteGuestStatus
	BatchStop           []StopGuestStatus
	Template            SystemTemplate
	TemplateList        []SystemTemplate
	ID                  string
	PolicyGroup         SecurityPolicyGroupStatus
	PolicyGroupList     []SecurityPolicyGroupStatus
	PolicyRuleList      []SecurityPolicyRule
	Total               int
	Offset              int
	Limit               int
}

type SearchGuestsCondition struct {
	Limit   int
	Offset  int
	Pool    string
	Cell    string
	Keyword string
	Owner   string
	Group   string
}

type DiskImageConfig struct {
	Name        string
	Owner       string
	Group       string
	Description string
	Tags        []string
}

type DiskImageStatus struct {
	DiskImageConfig
	ID         string
	Size       uint64
	Created    bool
	Progress   uint
	CreateTime string
	ModifyTime string
}

type MigrationParameter struct {
	ID         string
	SourcePool string
	SourceCell string
	TargetPool string
	TargetCell string
	Instances  []string
}

type MigrationStatus struct {
	MigrationParameter
	Finished bool
	Progress uint
	Error    error
}

type CellStatusReport struct {
	Name string
	ResourceUsage
}

//batch task

type BatchCreateNameRule int

const (
	NameRuleByOrder = iota
	NameRuleByMAC
	NameRuleByAddress
)

type BatchCreateRequest struct {
	Rule   BatchCreateNameRule
	Prefix string
	Pool   string
	Count  int
}

type BatchTaskStatus int

const (
	BatchTaskStatusProcess = iota
	BatchTaskStatusSuccess
	BatchTaskStatusFail
)

type CreateGuestStatus struct {
	Name     string
	ID       string
	Progress uint
	Status   BatchTaskStatus
	Error    string
}

type DeleteGuestStatus struct {
	Name   string
	ID     string
	Status BatchTaskStatus
	Error  string
}

type StopGuestStatus struct {
	Name   string
	ID     string
	Status BatchTaskStatus
	Error  string
}

//address pool&range

type AllocatedAddress struct {
	Address  string `json:"address"`
	Instance string `json:"instance"`
}

type AddressRangeConfig struct {
	Start    string `json:"start"`
	End      string `json:"end"`
	Netmask  string `json:"netmask"`
	Capacity uint32 `json:"capacity"`
}

type AddressRangeStatus struct {
	AddressRangeConfig
	Allocated []AllocatedAddress `json:"allocated,omitempty"`
}

const (
	AddressProviderDHCP       = "dhcp"
	AddressProviderCloudInit  = "cloudinit"
	AddressAllocationInternal = "internal"
	AddressAllocationExternal = "external"
	AddressAllocationBoth     = "both"
)

type AddressPoolConfig struct {
	Name     string   `json:"name,omitempty"`
	Gateway  string   `json:"gateway"`
	DNS      []string `json:"dns,omitempty"`
	Provider string   `json:"provider"`
	Mode     string   `json:"mode,omitempty"`
}

type AddressPoolStatus struct {
	AddressPoolConfig
	Ranges    []AddressRangeConfig `json:"ranges,omitempty"`
	Allocated []AllocatedAddress   `json:"allocated,omitempty"`
}

//Security Policy Group

type PolicyRuleProtocol string

const (
	PolicyRuleProtocolTCP  = "tcp"
	PolicyRuleProtocolUDP  = "udp"
	PolicyRuleProtocolICMP = "icmp"
)

const (
	PolicyRuleProtocolIndexTCP = iota
	PolicyRuleProtocolIndexUDP
	PolicyRuleProtocolIndexICMP
	PolicyRuleProtocolIndexInvalid
)

const (
	PolicyRuleActionAccept = iota
	PolicyRuleActionReject
)

type SecurityPolicyRule struct {
	Accept        bool               `json:"accept"`
	Protocol      PolicyRuleProtocol `json:"protocol"`
	SourceAddress string             `json:"source_address,omitempty"`
	TargetAddress string             `json:"target_address,omitempty"`
	TargetPort    uint               `json:"target_port"`
}

type SecurityPolicyGroup struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	User        string `json:"user"`
	Group       string `json:"group"`
	Enabled     bool   `json:"enabled"`
	Global      bool   `json:"global"`
	Accept      bool   `json:"accept"`
}

type SecurityPolicyGroupStatus struct {
	ID string `json:"id"`
	SecurityPolicyGroup
}

type SecurityPolicyGroupQueryCondition struct {
	User        string
	Group       string
	EnabledOnly bool
	GlobalOnly  bool
}

type ResourceModule interface {
	//zone
	QueryZoneStatus(resp chan ResourceResult)

	//pools
	CreatePool(name, storage, address string, failover bool, resultChan chan error)
	ModifyPool(name, storage, address string, failover bool, resultChan chan error)
	DeletePool(name string, resultChan chan error)
	GetAllComputePool(resp chan ResourceResult)
	GetComputePool(pool string, resp chan ResourceResult)
	QueryComputePoolStatus(resp chan ResourceResult)
	GetComputePoolStatus(pool string, resp chan ResourceResult)

	//storage pools
	CreateStoragePool(name, storageType, host, target string, respChan chan error)
	ModifyStoragePool(name, storageType, host, target string, respChan chan error)
	DeleteStoragePool(name string, respChan chan error)
	GetStoragePool(name string, respChan chan ResourceResult)
	QueryStoragePool(respChan chan ResourceResult)

	//address pool&range
	QueryAddressPool(respChan chan ResourceResult)
	GetAddressPool(name string, respChan chan ResourceResult)
	CreateAddressPool(config AddressPoolConfig, respChan chan error)
	ModifyAddressPool(config AddressPoolConfig, respChan chan ResourceResult)
	DeleteAddressPool(name string, respChan chan error)

	QueryAddressRange(poolName, rangeType string, respChan chan ResourceResult)
	GetAddressRange(poolName, rangeType, startAddress string, respChan chan ResourceResult)
	AddAddressRange(poolName, rangeType string, config AddressRangeConfig, respChan chan error)
	RemoveAddressRange(poolName, rangeType, startAddress string, respChan chan error)

	//cells
	QueryCellsInPool(pool string, resp chan ResourceResult)
	QueryComputeCellStatus(pool string, resp chan ResourceResult)
	GetComputeCellStatus(pool, cell string, resp chan ResourceResult)
	GetUnallocatedCells(result chan ResourceResult)
	AddCell(pool, cell string, resultChan chan error)
	RemoveCell(pool, cell string, resultChan chan error)
	UpdateCellInfo(name, address string, respChan chan error)
	GetCellStatus(name string, respChan chan ResourceResult)
	UpdateCellStatus(report CellStatusReport)
	SetCellDead(cellName string, respChan chan error)
	EnableCell(poolName, cellName string, respChan chan error)
	DisableCell(poolName, cellName string, purge bool, respChan chan error)
	FinishPurgeCell(cellName string, respChan chan error)

	//instances
	QueryGuestsByCondition(condition GuestQueryCondition, respChan chan ResourceResult)
	BatchUpdateInstanceStatus(pool, cell string, instances []InstanceStatus, respChan chan error)
	SearchGuests(condition SearchGuestsCondition, respChan chan ResourceResult)

	AllocateInstance(pool string, status InstanceStatus, respChan chan ResourceResult)
	//running/create/progress/media only
	UpdateInstanceStatus(status InstanceStatus, respChan chan error)
	ConfirmInstance(id string, monitorPort uint, monitorSecret, ethernetAddress string, respChan chan error)
	DeallocateInstance(id string, err error, respChan chan error)

	GetInstanceStatus(id string, respChan chan ResourceResult)
	QueryInstanceStatusInPool(poolName string, respChan chan ResourceResult)
	QueryInstanceStatusInCell(poolName, cellName string, respChan chan ResourceResult)
	UpdateInstanceAddress(id, ip string, respChan chan error)
	RenameInstance(id, name string, respChan chan error)
	GetInstanceByName(poolName, instanceName string, respChan chan ResourceResult)
	UpdateInstancePriority(id string, priority PriorityEnum, respChan chan error)
	UpdateInstanceDiskThreshold(id string, readSpeed, readIOPS, writeSpeed, writeIOPS uint64, respChan chan error)
	UpdateInstanceNetworkThreshold(id string, receive, send uint64, respChan chan error)
	UpdateInstanceMonitorSecret(id, secret string, respChan chan error)
	UpdateGuestAutoStart(guestID string, enabled bool, respChan chan error)

	//image server
	AddImageServer(name, host string, port int)
	RemoveImageServer(name string)
	GetImageServer(respChan chan ResourceResult)

	//migration
	QueryMigration(respChan chan ResourceResult)
	GetMigration(id string, respChan chan ResourceResult)
	CreateMigration(params MigrationParameter, respChan chan ResourceResult)
	FinishMigration(migration string, instances []string, ports []uint64, respChan chan error)
	CancelMigration(migration string, err error, respChan chan error)
	UpdateMigration(migration string, progress uint, respChan chan error)

	//failover
	BuildFailoverPlan(cellName string, respChan chan ResourceResult)
	MigrateInstance(oldCell, newCell string, instances []string, ports []uint64, respChan chan error)
	PurgeInstance(cellName string, respChan chan error)

	//reset system
	BeginResetSystem(instanceID string, respChan chan error)
	FinishResetSystem(instanceID string, err error, respChan chan error)

	//batch
	StartBatchCreateGuest(request BatchCreateRequest, respChan chan ResourceResult)
	SetBatchCreateGuestStart(batchID, guestName, guestID string, respChan chan error)
	SetBatchCreateGuestFail(batchID, guestName string, err error, respChan chan error)
	GetBatchCreateGuestStatus(batchID string, respChan chan ResourceResult)

	StartBatchDeleteGuest(id []string, respChan chan ResourceResult)
	SetBatchDeleteGuestSuccess(batchID, guestID string, respChan chan error)
	SetBatchDeleteGuestFail(batchID, guestID string, err error, respChan chan error)
	GetBatchDeleteGuestStatus(batchID string, respChan chan ResourceResult)

	StartBatchStopGuest(id []string, respChan chan ResourceResult)
	SetBatchStopGuestSuccess(batchID, guestID string, respChan chan error)
	SetBatchStopGuestFail(batchID, guestID string, err error, respChan chan error)
	GetBatchStopGuestStatus(batchID string, respChan chan ResourceResult)

	QuerySystemTemplates(respChan chan ResourceResult)
	GetSystemTemplate(id string, respChan chan ResourceResult)
	CreateSystemTemplate(config SystemTemplateConfig, respChan chan ResourceResult)
	ModifySystemTemplate(id string, config SystemTemplateConfig, respChan chan error)
	DeleteSystemTemplate(id string, respChan chan error)

	//Security Policy Group
	QuerySecurityPolicyGroups(condition SecurityPolicyGroupQueryCondition, respChan chan ResourceResult)
	GetSecurityPolicyGroup(groupID string, respChan chan ResourceResult)
	CreateSecurityPolicyGroup(config SecurityPolicyGroup, respChan chan ResourceResult)
	ModifySecurityPolicyGroup(groupID string, config SecurityPolicyGroup, respChan chan error)
	DeleteSecurityPolicyGroup(groupID string, respChan chan error)

	GetSecurityPolicyRules(groupID string, respChan chan ResourceResult)
	AddSecurityPolicyRule(groupID string, rule SecurityPolicyRule, respChan chan error)
	ModifySecurityPolicyRule(groupID string, index int, rule SecurityPolicyRule, respChan chan error)
	RemoveSecurityPolicyRule(groupID string, index int, respChan chan error)
	MoveSecurityPolicyRule(groupID string, index int, up bool, respChan chan error)
}

func (report *CellStatusReport) FromMessage(msg framework.Message) (err error) {
	report.Name = msg.GetSender()
	if report.Cores, err = msg.GetUInt(framework.ParamKeyCore); err != nil {
		return err
	}
	if report.CpuUsage, err = msg.GetFloat(framework.ParamKeyUsage); err != nil {
		return err
	}
	memory, err := msg.GetUIntArray(framework.ParamKeyMemory)
	if err != nil {
		return err
	}
	disk, err := msg.GetUIntArray(framework.ParamKeyDisk)
	if err != nil {
		return err
	}
	io, err := msg.GetUIntArray(framework.ParamKeyIO)
	if err != nil {
		return err
	}
	speed, err := msg.GetUIntArray(framework.ParamKeySpeed)
	if err != nil {
		return err
	}
	const (
		validArraySize  = 4
		validStatusSize = 2
	)
	if validStatusSize != len(memory) {
		return fmt.Errorf("invalid memory array size %d", len(memory))
	}
	if validStatusSize != len(disk) {
		return fmt.Errorf("invalid disk array size %d", len(disk))
	}
	if validArraySize != len(io) {
		return fmt.Errorf("invalid io array size %d", len(io))
	}
	if validArraySize != len(speed) {
		return fmt.Errorf("invalid speed array size %d", len(speed))
	}
	report.MemoryAvailable = memory[0]
	report.Memory = memory[1]
	report.DiskAvailable = disk[0]
	report.Disk = disk[1]
	report.BytesRead = io[0]
	report.BytesWritten = io[1]
	report.BytesReceived = io[2]
	report.BytesSent = io[3]
	report.ReadSpeed = speed[0]
	report.WriteSpeed = speed[1]
	report.ReceiveSpeed = speed[2]
	report.SendSpeed = speed[3]
	return nil
}

func CellsToMessage(message framework.Message, cells []ComputeCellInfo) {
	var name, address []string
	var enabled, alive []uint64
	for _, cell := range cells {
		name = append(name, cell.Name)
		address = append(address, cell.Address)
		if cell.Enabled {
			enabled = append(enabled, 1)
		} else {
			enabled = append(enabled, 0)
		}
		if cell.Alive {
			alive = append(alive, 1)
		} else {
			alive = append(alive, 0)
		}
	}
	message.SetStringArray(framework.ParamKeyName, name)
	message.SetStringArray(framework.ParamKeyAddress, address)
	message.SetUIntArray(framework.ParamKeyEnable, enabled)
	message.SetUIntArray(framework.ParamKeyStatus, alive)
}

func CellsFromMessage(message framework.Message) (cells []ComputeCellInfo, err error) {
	cells = make([]ComputeCellInfo, 0)

	name, err := message.GetStringArray(framework.ParamKeyName)
	if err != nil {
		return
	}
	addresses, err := message.GetStringArray(framework.ParamKeyAddress)
	if err != nil {
		return
	}
	enabled, err := message.GetUIntArray(framework.ParamKeyEnable)
	if err != nil {
		return
	}
	alive, err := message.GetUIntArray(framework.ParamKeyStatus)
	if err != nil {
		return
	}
	var count = len(name)
	if len(addresses) != count {
		err = fmt.Errorf("unexpected address count %d / %d", len(addresses), count)
		return
	}
	if len(enabled) != count {
		err = fmt.Errorf("unexpected enabled count %d / %d", len(enabled), count)
		return
	}
	if len(alive) != count {
		err = fmt.Errorf("unexpected alive count %d / %d", len(alive), count)
		return
	}
	for i := 0; i < count; i++ {
		var cell = ComputeCellInfo{Name: name[i], Address: addresses[i]}
		cell.Enabled = 1 == enabled[i]
		cell.Alive = 1 == alive[i]
		cells = append(cells, cell)
	}
	return
}
