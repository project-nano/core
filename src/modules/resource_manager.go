package modules

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/project-nano/framework"
	uuid "github.com/satori/go.uuid"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

//config file define

type cellDefine struct {
	Enabled        bool     `json:"enabled,omitempty"`
	PurgeAppending bool     `json:"purge_appending,omitempty"`
	Instances      []string `json:"-"`
}

type poolDefine struct {
	Name     string                `json:"name"`
	Enabled  bool                  `json:"enabled,omitempty"`
	Network  string                `json:"network,omitempty"`
	Storage  string                `json:"storage,omitempty"`
	Failover bool                  `json:"failover,omitempty"`
	Cells    map[string]cellDefine `json:"cells,omitempty"`
}

type storageDefine struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Host   string `json:"host"`
	Target string `json:"target"`
}

type addressPoolDefine struct {
	AddressPoolConfig
	Ranges []AddressRangeStatus `json:"ranges,omitempty"`
}

type ResourceData struct {
	Zone            string              `json:"zone"`
	Pools           []poolDefine        `json:"pools"`
	StoragePools    []storageDefine     `json:"storage_pools,omitempty"`
	AddressPools    []addressPoolDefine `json:"address_pools,omitempty"`
	SystemTemplates []SystemTemplate    `json:"system_templates,omitempty"`
}

//memory status define
type ManagedZone struct {
	Name string
	PoolStatistic
	CellStatistic
	InstanceStatistic
	ResourceUsage
}

type ManagedComputePool struct {
	ComputePoolInfo
	Cells         map[string]bool
	InstanceNames map[string]string //name => id
	CellStatistic
	InstanceStatistic
	ResourceUsage
}

type ManagedComputeCell struct {
	ComputeCellInfo
	Pool           string
	LatestUpdate   time.Time
	Instances      map[string]bool
	Pending        map[string]bool
	InstanceStatistic
	ResourceUsage
}

type ManagedIPV4AddressRange struct {
	startAddress net.IP
	endAddress   net.IP
	netmask      net.IPMask
	capacity     uint32
	allocated    map[string]string
}

type ManagedAddressPool struct {
	name                string
	gateway             string
	dns                 []string
	ranges              map[string]ManagedIPV4AddressRange
	rangeStartAddressed []string
}

type imageServer struct {
	Host string
	Port int
}

type BatchCreateGuestTask struct {
	StartTime    time.Time
	LatestUpdate time.Time
	Finished     bool
	Guests       []CreateGuestStatus
	GuestName    map[string]int //name => index
}

type BatchDeleteGuestTask struct {
	StartTime    time.Time
	LatestUpdate time.Time
	Finished     bool
	Guests       []DeleteGuestStatus
	GuestID      map[string]int //id => index
}

type BatchStopGuestTask struct {
	StartTime    time.Time
	LatestUpdate time.Time
	Finished     bool
	Guests       []StopGuestStatus
	GuestID      map[string]int //id => index
}

type ResourceManager struct {
	reportChan       chan CellStatusReport
	commands         chan resourceCommand
	pools            map[string]ManagedComputePool
	cells            map[string]ManagedComputeCell
	unallocatedCells map[string]bool
	instances        map[string]InstanceStatus
	imageServers     map[string]imageServer //key = server name
	pendingError     map[string]error       //pending create error
	storagePools     map[string]StoragePoolInfo
	addressPools     map[string]ManagedAddressPool
	migrations       map[string]MigrationStatus
	batchCreateTasks map[string]BatchCreateGuestTask
	batchDeleteTasks map[string]BatchDeleteGuestTask
	batchStopTasks   map[string]BatchStopGuestTask
	templates        map[string]SystemTemplate
	allTemplateID    []string
	generator        *rand.Rand
	zone             ManagedZone
	startTime        time.Time
	dataFile         string
	runner           *framework.SimpleRunner
}

type resourceCommand struct {
	Type           commandType
	DiskImage      DiskImageConfig
	Migration      MigrationParameter
	Pool           string
	Cell           string
	Address        string
	Range          string
	Start          string
	InstanceID     string
	Hardware       string
	InstanceList   []InstanceStatus
	Instance       InstanceStatus
	InstanceQuery  GuestQueryCondition
	MonitorPort    uint
	Name           string
	Host           string
	Port           int
	Group          string
	Tags           []string
	Progress       uint
	Size           uint64
	Image          string
	Secret         string
	Storage        string
	StorageType    string
	Target         string
	MigrationID    string
	Error          error
	Failover       bool
	IDList         []string
	PortList       []uint64
	DiskImages     []DiskImageStatus
	AddressPool    AddressPoolConfig
	AddressRange   AddressRangeConfig
	BatchID        string
	BatchCreating  BatchCreateRequest
	Priority       PriorityEnum
	ReadSpeed      uint64
	WriteSpeed     uint64
	ReadIOPS       uint64
	WriteIOPS      uint64
	ReceiveSpeed   uint64
	SendSpeed      uint64
	TemplateID     string
	TemplateConfig SystemTemplateConfig
	ErrorChan      chan error
	ResultChan     chan ResourceResult
}

type ResourceStatistic struct {
	Name    string
	Error   error
	Enabled bool
	Alive   bool
	PoolStatistic
	CellStatistic
	InstanceStatistic
	ResourceUsage
}

type commandType int

const (
	cmdQueryAllComputePoolInfo     = iota
	cmdGetComputePoolInfo
	cmdCreateComputePool
	cmdDeleteComputePool
	cmdModifyComputePool
	cmdQueryZoneStatus
	cmdQueryComputePoolStatus
	cmdGetComputePoolStatus
	cmdQueryUnallocatedComputeCell
	cmdQueryComputeCells
	cmdQueryComputeCellStatus
	cmdGetComputeCellStatus
	cmdAddComputeCell
	cmdRemoveComputeCell
	cmdEnableComputeCell
	cmdDisableComputeCell
	cmdFinishPurgeCell
	cmdGetCellStatus
	cmdUpdateCellInfo
	cmdSetCellDead
	cmdBatchUpdateInstanceStatus
	cmdUpdateInstanceStatus
	cmdAllocateInstance
	cmdConfirmInstance
	cmdDeallocateInstance
	cmdGetInstanceStatus
	cmdQueryInstanceStatusInPool
	cmdQueryInstanceStatusInCell
	cmdUpdateInstanceAddress
	cmdUpdateInstancePriority
	cmdUpdateInstanceDiskThreshold
	cmdUpdateInstanceNetworkThreshold
	cmdUpdateInstanceMonitorSecret
	cmdRenameInstance
	cmdGetInstanceByName
	cmdSearchGuests
	cmdAddImageServer
	cmdRemoveImageServer
	cmdGetImageServer
	cmdCreateStoragePool
	cmdDeleteStoragePool
	cmdModifyStoragePool
	cmdQueryStoragePool
	cmdGetStoragePool
	cmdQueryAddressPool
	cmdGetAddressPool
	cmdCreateAddressPool
	cmdModifyAddressPool
	cmdDeleteAddressPool
	cmdQueryAddressRange
	cmdGetAddressRange
	cmdAddAddressRange
	cmdRemoveAddressRange
	cmdQueryMigration
	cmdGetMigration
	cmdCreateMigration
	cmdFinishMigration
	cmdCancelMigration
	cmdUpdateMigration
	cmdBuildFailoverPlan
	cmdMigrationInstance
	cmdPurgeInstance
	cmdBeginResetSystem
	cmdFinishResetSystem	
	cmdStartBatchCreateGuest
	cmdSetBatchCreateGuestStart
	cmdSetBatchCreateGuestFail
	cmdGetBatchCreateGuest
	cmdStartBatchDeleteGuest
	cmdSetBatchDeleteGuestSuccess
	cmdSetBatchDeleteGuestFail
	cmdGetBatchDeleteGuest
	cmdStartBatchStopGuest
	cmdSetBatchStopGuestSuccess
	cmdSetBatchStopGuestFail
	cmdGetBatchStopGuest
	cmdQuerySystemTemplates
	cmdGetSystemTemplate
	cmdCreateSystemTemplate
	cmdModifySystemTemplate
	cmdDeleteSystemTemplate
	cmdInvalid
)

var commandNames = []string{
	"QueryAllComputePoolInfo",
	"GetComputePoolInfo",
	"CreateComputePool",
	"DeleteComputePool",
	"ModifyComputePool",
	"QueryZoneStatus",
	"QueryComputePoolStatus",
	"GetComputePoolStatus",
	"QueryUnallocatedComputeCell",
	"QueryComputeCells",
	"QueryComputeCellStatus",
	"GetComputeCellStatus",
	"AddComputeCell",
	"RemoveComputeCell",
	"EnableComputeCell",
	"DisableComputeCell",
	"FinishPurgeCell",
	"GetCellStatus",
	"UpdateCellInfo",
	"SetCellDead",
	"BatchUpdateInstanceStatus",
	"UpdateInstanceStatus",
	"AllocateInstance",
	"ConfirmInstance",
	"DeallocateInstance",
	"GetInstanceStatus",
	"QueryInstanceStatusInPool",
	"QueryInstanceStatusInCell",
	"UpdateInstanceAddress",
	"UpdateInstancePriority",
	"UpdateInstanceDiskThreshold",
	"UpdateInstanceNetworkThreshold",
	"UpdateInstanceMonitorSecret",
	"RenameInstance",
	"GetInstanceByName",
	"SearchGuests",
	"AddImageServer",
	"RemoveImageServer",
	"GetImageServer",
	"CreateStoragePool",
	"DeleteStoragePool",
	"ModifyStoragePool",
	"QueryStoragePool",
	"GetStoragePool",
	"QueryAddressPool",
	"GetAddressPool",
	"CreateAddressPool",
	"ModifyAddressPool",
	"DeleteAddressPool",
	"QueryAddressRange",
	"GetAddressRange",
	"AddAddressRange",
	"RemoveAddressRange",
	"QueryMigration",
	"GetMigration",
	"CreateMigration",
	"FinishMigration",
	"CancelMigration",
	"UpdateMigration",
	"BuildFailoverPlan",
	"MigrationInstance",
	"PurgeInstance",
	"BeginResetSystem",
	"FinishResetSystem",
	"StartBatchCreateGuest",
	"SetBatchCreateGuestStart",
	"SetBatchCreateGuestFail",
	"GetBatchCreateGuest",
	"StartBatchDeleteGuest",
	"SetBatchDeleteGuestSuccess",
	"SetBatchDeleteGuestFail",
	"GetBatchDeleteGuest",
	"StartBatchStopGuest",
	"SetBatchStopGuestSuccess",
	"SetBatchStopGuestFail",
	"GetBatchStopGuest",
	"QuerySystemTemplates",
	"GetSystemTemplate",
	"CreateSystemTemplate",
	"ModifySystemTemplate",
	"DeleteSystemTemplate",
}

func (c commandType) toString() string {
	if c >= cmdInvalid{
		return  "invalid"
	}
	return commandNames[c]
}

const (
	TimeFormatLayout = "2006-01-02 15:04:05"
	StorageTypeNFS = "nfs"
	RangeTypeExternal = "external"
	RangeTypeInternal = "internal"
)


func CreateResourceManager(dataPath string) (manager *ResourceManager, err error) {
	if cmdInvalid != len(commandNames){
		err = fmt.Errorf("insufficient command names %d/%d", len(commandNames), cmdInvalid)
		return
	}

	const (
		DefaultQueueLength  = 1 << 10
		DefaultDataFilename = "resource.data"
	)
	manager = &ResourceManager{}
	manager.runner = framework.CreateSimpleRunner(manager.mainRoutine)
	manager.reportChan = make(chan CellStatusReport, DefaultQueueLength)
	manager.commands = make(chan resourceCommand, DefaultQueueLength)
	manager.dataFile = filepath.Join(dataPath, DefaultDataFilename)
	manager.pools = map[string]ManagedComputePool{}
	manager.cells = map[string]ManagedComputeCell{}
	manager.instances = map[string]InstanceStatus{}
	manager.unallocatedCells = map[string]bool{}
	manager.imageServers = map[string]imageServer{}
	manager.pendingError = map[string]error{}
	manager.storagePools = map[string]StoragePoolInfo{}
	manager.addressPools = map[string]ManagedAddressPool{}
	manager.templates = map[string]SystemTemplate{}
	manager.migrations = map[string]MigrationStatus{}
	manager.batchCreateTasks = map[string]BatchCreateGuestTask{}
	manager.batchDeleteTasks = map[string]BatchDeleteGuestTask{}
	manager.batchStopTasks = map[string]BatchStopGuestTask{}
	manager.generator = rand.New(rand.NewSource(time.Now().UnixNano()))
	manager.startTime = time.Now()
	if err := manager.loadConfig(); err != nil {
		return nil, err
	}
	return manager, nil
}

func (manager *ResourceManager) Start() error {
	return manager.runner.Start()
}

func (manager *ResourceManager) Stop() error {
	return manager.runner.Stop()
}

func (manager *ResourceManager) UpdateCellStatus(report CellStatusReport) {
	manager.reportChan <- report
}

func (manager *ResourceManager) CreatePool(name, storage, address string, failover bool, resultChan chan error) {
	req := resourceCommand{Type: cmdCreateComputePool, Pool: name, Storage:storage, Address: address, Failover: failover, ErrorChan: resultChan}
	manager.commands <- req
}

func (manager *ResourceManager) ModifyPool(name, storage, address string, failover bool, resultChan chan error){
	manager.commands <- resourceCommand{Type:cmdModifyComputePool, Pool:name, Storage:storage, Address: address, Failover: failover, ErrorChan:resultChan}
}

func (manager *ResourceManager) DeletePool(name string, resultChan chan error) {
	req := resourceCommand{Type: cmdDeleteComputePool, Pool: name, ErrorChan: resultChan}
	manager.commands <- req
}

//storage pools
func (manager *ResourceManager) CreateStoragePool(name, storageType, host, target string, respChan chan error){
	manager.commands <- resourceCommand{Type: cmdCreateStoragePool, Storage:name, StorageType:storageType, Host:host, Target: target, ErrorChan:respChan}
}
func (manager *ResourceManager) ModifyStoragePool(name, storageType, host, target string, respChan chan error){
	manager.commands <- resourceCommand{Type: cmdModifyStoragePool, Storage:name, StorageType:storageType, Host:host, Target: target, ErrorChan:respChan}
}

func (manager *ResourceManager) DeleteStoragePool(name string, respChan chan error){
	manager.commands <- resourceCommand{Type: cmdDeleteStoragePool, Storage:name, ErrorChan:respChan}
}
func (manager *ResourceManager) GetStoragePool(name string, respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type: cmdGetStoragePool, Storage:name, ResultChan: respChan}
}
func (manager *ResourceManager) QueryStoragePool(respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type: cmdQueryStoragePool, ResultChan: respChan}
}

func (manager *ResourceManager) QueryCellsInPool(pool string, resp chan ResourceResult){
	cmd := resourceCommand{Type:cmdQueryComputeCells, Pool:pool, ResultChan:resp}
	manager.commands <- cmd
}
func (manager *ResourceManager) AddCell(pool, cell string, resultChan chan error) {
	req := resourceCommand{Type: cmdAddComputeCell, Pool: pool, Cell: cell, ErrorChan: resultChan}
	manager.commands <- req
}

func (manager *ResourceManager) RemoveCell(pool, cell string, resultChan chan error) {
	req := resourceCommand{Type: cmdRemoveComputeCell, Pool: pool, Cell: cell, ErrorChan: resultChan}
	manager.commands <- req
}

func (manager *ResourceManager) EnableCell(poolName, cellName string, respChan chan error){
	manager.commands <- resourceCommand{Type:cmdEnableComputeCell, Pool:poolName, Cell:cellName, ErrorChan:respChan}
}
func (manager *ResourceManager) DisableCell(poolName, cellName string, purge bool, respChan chan error){
	manager.commands <- resourceCommand{Type:cmdDisableComputeCell, Pool:poolName, Cell:cellName, ErrorChan:respChan}
}

func (manager *ResourceManager) FinishPurgeCell(cellName string, respChan chan error){
	manager.commands <- resourceCommand{Type:cmdFinishPurgeCell, Cell:cellName, ErrorChan:respChan}
}

func (manager *ResourceManager) GetUnallocatedCells(resp chan ResourceResult) {
	req := resourceCommand{Type: cmdQueryUnallocatedComputeCell, ResultChan:resp}
	manager.commands <- req
}

func (manager *ResourceManager) QueryZoneStatus(resp chan ResourceResult) {
	req := resourceCommand{Type: cmdQueryZoneStatus, ResultChan: resp}
	manager.commands <- req
}

func (manager *ResourceManager) QueryComputePoolStatus(resp chan ResourceResult) {
	req := resourceCommand{Type: cmdQueryComputePoolStatus, ResultChan: resp}
	manager.commands <- req
}
func (manager *ResourceManager) GetComputePoolStatus(pool string, resp chan ResourceResult) {
	req := resourceCommand{Type: cmdGetComputePoolStatus, Pool: pool, ResultChan: resp}
	manager.commands <- req
}

func (manager *ResourceManager) QueryComputeCellStatus(pool string, resp chan ResourceResult) {
	req := resourceCommand{Type: cmdQueryComputeCellStatus, Pool: pool, ResultChan: resp}
	manager.commands <- req
}

func (manager *ResourceManager) GetComputeCellStatus(pool, cell string, resp chan ResourceResult) {
	req := resourceCommand{Type: cmdGetComputeCellStatus, Pool: pool, Cell:cell, ResultChan: resp}
	manager.commands <- req
}

func (manager *ResourceManager) GetAllComputePool(resp chan ResourceResult) {
	req := resourceCommand{Type: cmdQueryAllComputePoolInfo, ResultChan: resp}
	manager.commands <- req
}


func (manager *ResourceManager) GetComputePool(pool string, resp chan ResourceResult){
	cmd := resourceCommand{Type: cmdGetComputePoolInfo, Pool:pool, ResultChan:resp}
	manager.commands <- cmd
}


func (manager *ResourceManager) UpdateCellInfo(name, address string, respChan chan error){
	manager.commands <- resourceCommand{Type: cmdUpdateCellInfo, Cell:name, Address:address, ErrorChan:respChan}
}

func (manager *ResourceManager) GetCellStatus(cell string, respChan chan ResourceResult) {
	req := resourceCommand{Type: cmdGetCellStatus, Cell: cell, ResultChan: respChan}
	manager.commands <- req
}

func (manager *ResourceManager) SetCellDead(cellName string, respChan chan error){
	manager.commands <- resourceCommand{Type: cmdSetCellDead, Cell:cellName, ErrorChan:respChan}
}

func (manager *ResourceManager) SearchGuestConfig(condition GuestQueryCondition, respChan chan ResourceResult) {
	cmd := resourceCommand{Type: cmdSearchGuests, InstanceQuery: condition, ResultChan: respChan}
	manager.commands <- cmd
}

func (manager *ResourceManager) BatchUpdateInstanceStatus(pool, cell string, instances []InstanceStatus, respChan chan error) {
	cmd := resourceCommand{Type: cmdBatchUpdateInstanceStatus, Pool: pool, Cell: cell, InstanceList: instances, ErrorChan: respChan}
	manager.commands <- cmd
}

func (manager *ResourceManager) AllocateInstance(pool string, config InstanceStatus, respChan chan ResourceResult) {
	cmd := resourceCommand{Type: cmdAllocateInstance, Pool: pool, Instance: config, ResultChan: respChan}
	manager.commands <- cmd
}
func (manager *ResourceManager) UpdateInstanceStatus(status InstanceStatus, respChan chan error) {
	cmd := resourceCommand{Type: cmdUpdateInstanceStatus, Instance: status, ErrorChan: respChan}
	manager.commands <- cmd
}
func (manager *ResourceManager) ConfirmInstance(id string, monitor uint, secret, ethernetAddress string, respChan chan error) {
	cmd := resourceCommand{Type: cmdConfirmInstance, InstanceID: id, MonitorPort: monitor, Secret:secret, Hardware: ethernetAddress, ErrorChan: respChan}
	manager.commands <- cmd
}

func (manager *ResourceManager) DeallocateInstance(id string, err error, respChan chan error) {
	cmd := resourceCommand{Type: cmdDeallocateInstance, InstanceID: id, Error: err, ErrorChan: respChan}
	manager.commands <- cmd
}

func (manager *ResourceManager) GetInstanceStatus(id string, respChan chan ResourceResult) {
	cmd := resourceCommand{Type: cmdGetInstanceStatus, InstanceID: id, ResultChan: respChan}
	manager.commands <- cmd
}

func (manager *ResourceManager) QueryInstanceStatusInPool(poolName string, respChan chan ResourceResult){
	cmd := resourceCommand{Type: cmdQueryInstanceStatusInPool, Pool: poolName, ResultChan: respChan}
	manager.commands <- cmd
}
func (manager *ResourceManager) QueryInstanceStatusInCell(poolName, cellName string, respChan chan ResourceResult){
	cmd := resourceCommand{Type: cmdQueryInstanceStatusInCell, Pool: poolName, Cell:cellName, ResultChan: respChan}
	manager.commands <- cmd
}

func (manager *ResourceManager) UpdateInstanceAddress(id, ip string, respChan chan error){
	manager.commands <- resourceCommand{Type:cmdUpdateInstanceAddress, InstanceID:id, Address:ip, ErrorChan:respChan}
}

func (manager *ResourceManager) UpdateInstancePriority(id string, priority PriorityEnum, respChan chan error) {
	manager.commands <- resourceCommand{Type: cmdUpdateInstancePriority, InstanceID: id, Priority: priority, ErrorChan:respChan}
}

func (manager *ResourceManager) UpdateInstanceMonitorSecret(id, secret string, respChan chan error){
	manager.commands <- resourceCommand{Type: cmdUpdateInstanceMonitorSecret, InstanceID: id, Secret: secret, ErrorChan: respChan}
}

func (manager *ResourceManager) UpdateInstanceDiskThreshold(id string, readSpeed, readIOPS, writeSpeed, writeIOPS uint64, respChan chan error) {
	manager.commands <- resourceCommand{Type: cmdUpdateInstanceDiskThreshold, InstanceID: id, ReadSpeed: readSpeed, ReadIOPS: readIOPS, WriteSpeed:writeSpeed, WriteIOPS: writeIOPS, ErrorChan: respChan}
}

func (manager *ResourceManager) UpdateInstanceNetworkThreshold(id string, receive, send uint64, respChan chan error) {
	manager.commands <- resourceCommand{Type: cmdUpdateInstanceNetworkThreshold, InstanceID: id, ReceiveSpeed:receive, SendSpeed: send, ErrorChan: respChan}
}

func (manager *ResourceManager) RenameInstance(id, name string, respChan chan error){
	manager.commands <- resourceCommand{Type:cmdRenameInstance, InstanceID:id, Name:name, ErrorChan:respChan}
}

func (manager *ResourceManager) GetInstanceByName(poolName, instanceName string, respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type:cmdGetInstanceByName, Pool:poolName, Name:instanceName, ResultChan:respChan}
}

func (manager *ResourceManager) AddImageServer(name, host string, port int) {
	cmd := resourceCommand{Type: cmdAddImageServer, Name: name, Host: host, Port: port}
	manager.commands <- cmd
}

func (manager *ResourceManager) RemoveImageServer(name string) {
	cmd := resourceCommand{Type: cmdRemoveImageServer, Name: name}
	manager.commands <- cmd
}

func (manager *ResourceManager) GetImageServer(respChan chan ResourceResult) {
	cmd := resourceCommand{Type: cmdGetImageServer, ResultChan: respChan}
	manager.commands <- cmd
}

//migration
func (manager *ResourceManager) QueryMigration(respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type:cmdQueryMigration, ResultChan:respChan}
}

func (manager *ResourceManager) GetMigration(id string, respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type:cmdGetMigration, MigrationID:id, ResultChan:respChan}
}

func (manager *ResourceManager) CreateMigration(params MigrationParameter, respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type:cmdCreateMigration, Migration: params,  ResultChan:respChan}
}

func (manager *ResourceManager) FinishMigration(migration string, instances []string, ports []uint64, respChan chan error){
	manager.commands <- resourceCommand{Type:cmdFinishMigration, MigrationID:migration, IDList:instances, PortList:ports, ErrorChan:respChan}
}

func (manager *ResourceManager) CancelMigration(migration string, err error, respChan chan error){
	manager.commands <- resourceCommand{Type:cmdCancelMigration, MigrationID:migration, Error: err, ErrorChan:respChan}
}
func (manager *ResourceManager) UpdateMigration(migration string, progress uint, respChan chan error){
	manager.commands <- resourceCommand{Type:cmdUpdateMigration, MigrationID:migration, Progress:progress, ErrorChan:respChan}
}

func (manager *ResourceManager) BuildFailoverPlan(cellName string, respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type: cmdBuildFailoverPlan, Cell:cellName, ResultChan:respChan}
}

func (manager *ResourceManager) MigrateInstance(oldCell, newCell string, instances []string, ports []uint64, respChan chan error){
	manager.commands <- resourceCommand{Type: cmdMigrationInstance, Cell:oldCell, Target:newCell, IDList:instances, PortList:ports, ErrorChan:respChan}
}

func (manager *ResourceManager) PurgeInstance(cellName string, respChan chan error){
	manager.commands <- resourceCommand{Type: cmdPurgeInstance, Cell:cellName, ErrorChan:respChan}
}

func (manager *ResourceManager) QueryAddressPool(respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type: cmdQueryAddressPool, ResultChan:respChan}
}
func (manager *ResourceManager) GetAddressPool(name string, respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type: cmdGetAddressPool, Address:name, ResultChan:respChan}
}
func (manager *ResourceManager) CreateAddressPool(config AddressPoolConfig, respChan chan error){
	manager.commands <- resourceCommand{Type:cmdCreateAddressPool, AddressPool:config, ErrorChan:respChan}
}
func (manager *ResourceManager) ModifyAddressPool(config AddressPoolConfig, respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type:cmdModifyAddressPool, AddressPool: config, ResultChan:respChan}
}
func (manager *ResourceManager) DeleteAddressPool(name string, respChan chan error){
	manager.commands <- resourceCommand{Type:cmdDeleteAddressPool, Address:name, ErrorChan:respChan}
}

func (manager *ResourceManager) QueryAddressRange(poolName, rangeType string, respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type:cmdQueryAddressRange, Address:poolName, Range:rangeType, ResultChan:respChan}
}
func (manager *ResourceManager) GetAddressRange(poolName, rangeType, startAddress string, respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type:cmdGetAddressRange, Address:poolName, Range:rangeType, Start:startAddress, ResultChan:respChan}
}
func (manager *ResourceManager) AddAddressRange(poolName, rangeType string, config AddressRangeConfig, respChan chan error){
	manager.commands <- resourceCommand{Type:cmdAddAddressRange, Address:poolName, Range:rangeType, AddressRange: config, ErrorChan:respChan}
}
func (manager *ResourceManager) RemoveAddressRange(poolName, rangeType, startAddress string, respChan chan error){
	manager.commands <- resourceCommand{Type:cmdRemoveAddressRange, Address:poolName, Range:rangeType, Start:startAddress, ErrorChan:respChan}
}

func (manager *ResourceManager) BeginResetSystem(instanceID string, respChan chan error){
	manager.commands <- resourceCommand{Type: cmdBeginResetSystem, InstanceID: instanceID, ErrorChan:respChan}
}

func (manager *ResourceManager) FinishResetSystem(instanceID string, err error,  respChan chan error){
	manager.commands <- resourceCommand{Type: cmdFinishResetSystem, InstanceID: instanceID, Error:err, ErrorChan:respChan}
}

//batch
func (manager *ResourceManager) StartBatchCreateGuest(request BatchCreateRequest, respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type: cmdStartBatchCreateGuest, BatchCreating: request, ResultChan:respChan}
}
func (manager *ResourceManager) SetBatchCreateGuestStart(batchID, guestName, guestID string, respChan chan error){
	manager.commands <- resourceCommand{Type: cmdSetBatchCreateGuestStart, BatchID: batchID, Name: guestName, InstanceID: guestID, ErrorChan:respChan}
}

func (manager *ResourceManager) SetBatchCreateGuestFail(batchID, guestName string, err error, respChan chan error){
	manager.commands <- resourceCommand{Type: cmdSetBatchCreateGuestFail, BatchID: batchID, Name: guestName, Error:err, ErrorChan:respChan}
}

func (manager *ResourceManager) GetBatchCreateGuestStatus(batchID string, respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type: cmdGetBatchCreateGuest, BatchID: batchID, ResultChan:respChan}
}

func (manager *ResourceManager) StartBatchDeleteGuest(id []string, respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type: cmdStartBatchDeleteGuest, IDList: id, ResultChan:respChan}
}

func (manager *ResourceManager) SetBatchDeleteGuestSuccess(batchID, guestID string, respChan chan error){
	manager.commands <- resourceCommand{Type: cmdSetBatchDeleteGuestSuccess, BatchID:batchID, InstanceID:guestID, ErrorChan:respChan}
}

func (manager *ResourceManager) SetBatchDeleteGuestFail(batchID, guestID string, err error, respChan chan error){
	manager.commands <- resourceCommand{Type: cmdSetBatchDeleteGuestFail, BatchID:batchID, InstanceID: guestID, Error:err, ErrorChan:respChan}
}

func (manager *ResourceManager) GetBatchDeleteGuestStatus(batchID string, respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type: cmdGetBatchDeleteGuest, BatchID:batchID, ResultChan:respChan}
}

func (manager *ResourceManager) StartBatchStopGuest(id []string, respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type: cmdStartBatchStopGuest, IDList: id, ResultChan:respChan}
}

func (manager *ResourceManager) SetBatchStopGuestSuccess(batchID, guestID string, respChan chan error){
	manager.commands <- resourceCommand{Type: cmdSetBatchStopGuestSuccess, BatchID:batchID, InstanceID:guestID, ErrorChan:respChan}
}

func (manager *ResourceManager) SetBatchStopGuestFail(batchID, guestID string, err error, respChan chan error){
	manager.commands <- resourceCommand{Type: cmdSetBatchStopGuestFail, BatchID:batchID, InstanceID: guestID, Error:err, ErrorChan:respChan}
}

func (manager *ResourceManager) GetBatchStopGuestStatus(batchID string, respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type: cmdGetBatchStopGuest, BatchID:batchID, ResultChan:respChan}
}

func (manager *ResourceManager) QuerySystemTemplates(respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type: cmdQuerySystemTemplates, ResultChan: respChan}
}

func (manager *ResourceManager) GetSystemTemplate(id string, respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type: cmdGetSystemTemplate, TemplateID: id, ResultChan: respChan}
}

func (manager *ResourceManager) CreateSystemTemplate(config SystemTemplateConfig, respChan chan ResourceResult){
	manager.commands <- resourceCommand{Type: cmdCreateSystemTemplate, TemplateConfig: config, ResultChan: respChan}
}

func (manager *ResourceManager) ModifySystemTemplate(id string, config SystemTemplateConfig, respChan chan error){
	manager.commands <- resourceCommand{Type: cmdModifySystemTemplate, TemplateID: id, TemplateConfig: config, ErrorChan: respChan}
}

func (manager *ResourceManager) DeleteSystemTemplate(id string, respChan chan error){
	manager.commands <- resourceCommand{Type: cmdDeleteSystemTemplate, TemplateID: id, ErrorChan: respChan}
}

func (manager *ResourceManager) mainRoutine(c framework.RoutineController) {
	const (
		summaryInterval     = time.Second * 5
		batchUpdateInterval = time.Second * 2
	)
	var summaryTicker = time.NewTicker(summaryInterval)
	var batchUpdateTicker = time.NewTicker(batchUpdateInterval)
	for !c.IsStopping() {
		select {
		case <- c.GetNotifyChannel():
			c.SetStopping()
		case report := <-manager.reportChan:
			manager.onCellStatusUpdate(report)
		case <-summaryTicker.C:
			manager.onUpdateSystemStatus()
		case <- batchUpdateTicker.C:
			manager.updateBatchStatus()
		case cmd := <-manager.commands:
			manager.handleCommand(cmd)
		}
	}
	c.NotifyExit()
}

func (manager *ResourceManager) updateBatchStatus(){
	const (
		TaskExpire = time.Second * 30
	)
	var expireTime = time.Now().Add(-TaskExpire)
	if 0 != len(manager.batchCreateTasks){
		//create batch
		var expired []string
		for taskID, task := range manager.batchCreateTasks{
			if task.Finished {
				if task.LatestUpdate.Before(expireTime){
					//expired
					expired = append(expired, taskID)
				}
			}else{
				//unfinished
				if task.LatestUpdate.Before(expireTime){
					//expired
					task.Finished = true
					manager.batchCreateTasks[taskID] = task
					log.Printf("<resource_manager> mark batch create task '%s' finished due to expire", taskID)
					continue
				}
				//check all guest
				var unfinishedGuestCount = 0
				var taskUpdated = false
				for guestIndex, guest := range task.Guests{
					if guest.Status == BatchTaskStatusProcess{
						if createError, exists := manager.pendingError[guest.ID]; exists{
							//create fail
							guest.Status = BatchTaskStatusFail
							guest.Error = createError.Error()
							log.Printf("<resource_manager> batch create guest '%s' fail: %s", guest.Name, createError.Error())
						}else if 0 == len(guest.ID){
							//not id allocated
							unfinishedGuestCount++
							continue
						}else{
							ins, exists := manager.instances[guest.ID]
							if !exists{
								unfinishedGuestCount++
								log.Printf("<resource_manager> warning: invalid guest '%s' in batch '%s'", guest.ID, taskID)
								continue
							}
							if ins.Created{
								guest.Status = BatchTaskStatusSuccess
								log.Printf("<resource_manager> update guest '%s' as created in batch '%s'", guest.Name, taskID)
							}else{
								guest.Progress = ins.Progress
								unfinishedGuestCount++
							}
						}
						task.Guests[guestIndex] = guest
						taskUpdated = true
					}
				}
				if taskUpdated{
					task.LatestUpdate = time.Now()
					manager.batchCreateTasks[taskID] = task
				}
				if 0 == unfinishedGuestCount{
					//all guest processed
					task.Finished = true
					manager.batchCreateTasks[taskID] = task
					log.Printf("<resource_manager> batch create task '%s' finished", taskID)
				}
			}
		}
		if 0 != len(expired){
			for _, taskID := range expired{
				delete(manager.batchCreateTasks, taskID)
				log.Printf("<resource_manager> release expired batch create task '%s'", taskID)
			}
		}
	}
	if 0 != len(manager.batchDeleteTasks){
		//delete batch
		var expired []string
		for taskID, task := range manager.batchDeleteTasks{
			if task.Finished {
				if task.LatestUpdate.Before(expireTime){
					//expired
					expired = append(expired, taskID)
				}
			}else{
				//unfinished
				if task.LatestUpdate.Before(expireTime){
					//expired
					task.Finished = true
					manager.batchDeleteTasks[taskID] = task
					log.Printf("<resource_manager> mark batch delete task '%s' finished due to expire", taskID)
					continue
				}
				//check all guest
				var unfinishedGuestCount = 0
				for guestIndex, guest := range task.Guests{
					if guest.Status == BatchTaskStatusProcess{
						if _, exists := manager.instances[guest.ID];!exists{
							//already deleted
							guest.Status = BatchTaskStatusSuccess
							task.Guests[guestIndex] = guest
							log.Printf("<resource_manager> update guest '%s' as deleted in batch '%s'", guest.Name, taskID)
						}else{
							unfinishedGuestCount++
						}
					}
				}
				if 0 == unfinishedGuestCount{
					//all guest processed
					task.Finished = true
					manager.batchDeleteTasks[taskID] = task
					log.Printf("<resource_manager> batch delete task '%s' finished", taskID)
				}
			}
		}
		if 0 != len(expired){
			for _, taskID := range expired{
				delete(manager.batchDeleteTasks, taskID)
				log.Printf("<resource_manager> release expired batch delete task '%s'", taskID)
			}
		}
	}

	if 0 != len(manager.batchStopTasks){
		//stop batch
		var expired []string
		for taskID, task := range manager.batchStopTasks{
			if task.Finished {
				if task.LatestUpdate.Before(expireTime){
					//expired
					expired = append(expired, taskID)
				}
			}else{
				//unfinished
				if task.LatestUpdate.Before(expireTime){
					//expired
					task.Finished = true
					manager.batchStopTasks[taskID] = task
					log.Printf("<resource_manager> mark batch stop task '%s' finished due to expire", taskID)
					continue
				}
				//check all guest
				var unfinishedGuestCount = 0
				for guestIndex, guest := range task.Guests{
					if guest.Status == BatchTaskStatusProcess{
						if _, exists := manager.instances[guest.ID];!exists{
							//already stopd
							guest.Status = BatchTaskStatusSuccess
							task.Guests[guestIndex] = guest
							log.Printf("<resource_manager> update guest '%s' as stopped in batch '%s'", guest.Name, taskID)
						}else{
							unfinishedGuestCount++
						}
					}
				}
				if 0 == unfinishedGuestCount{
					//all guest processed
					task.Finished = true
					manager.batchStopTasks[taskID] = task
					log.Printf("<resource_manager> batch stop task '%s' finished", taskID)
				}
			}
		}
		if 0 != len(expired){
			for _, taskID := range expired{
				delete(manager.batchStopTasks, taskID)
				log.Printf("<resource_manager> release expired batch stop task '%s'", taskID)
			}
		}
	}
}

func (manager *ResourceManager) handleCommand(cmd resourceCommand) {
	var err error
	switch cmd.Type {
	case cmdQueryAllComputePoolInfo:
		err = manager.handleQueryAllPools(cmd.ResultChan)
	case cmdGetComputePoolInfo:
		err = manager.handleGetComputePool(cmd.Pool, cmd.ResultChan)
	case cmdCreateComputePool:
		err = manager.handleCreatePool(cmd.Pool, cmd.Storage, cmd.Address, cmd.Failover, cmd.ErrorChan)
	case cmdModifyComputePool:
		err = manager.handleModifyPool(cmd.Pool, cmd.Storage, cmd.Address, cmd.Failover, cmd.ErrorChan)
	case cmdDeleteComputePool:
		err = manager.handleDeletePool(cmd.Pool, cmd.ErrorChan)
	case cmdQueryStoragePool:
		err = manager.handleQueryStoragePool(cmd.ResultChan)
	case cmdGetStoragePool:
		err = manager.handleGetStoragePool(cmd.Storage, cmd.ResultChan)
	case cmdCreateStoragePool:
		err = manager.handleCreateStoragePool(cmd.Storage, cmd.StorageType, cmd.Host, cmd.Target, cmd.ErrorChan)
	case cmdModifyStoragePool:
		err = manager.handleModifyStoragePool(cmd.Storage, cmd.StorageType, cmd.Host, cmd.Target, cmd.ErrorChan)
	case cmdDeleteStoragePool:
		err = manager.handleDeleteStoragePool(cmd.Storage, cmd.ErrorChan)
		
	case cmdQueryComputeCells:
		err = manager.handleQueryCellsInPool(cmd.Pool, cmd.ResultChan)
	case cmdAddComputeCell:
		err = manager.handleAddCell(cmd.Pool, cmd.Cell, cmd.ErrorChan)
	case cmdRemoveComputeCell:
		err = manager.handleRemoveCell(cmd.Pool, cmd.Cell, cmd.ErrorChan)
	case cmdEnableComputeCell:
		err = manager.handleEnableCell(cmd.Pool, cmd.Cell, cmd.ErrorChan)
	case cmdDisableComputeCell:
		err = manager.handleDisableCell(cmd.Pool, cmd.Cell, cmd.Failover, cmd.ErrorChan)
	case cmdFinishPurgeCell:
		err = manager.handleFinishPurgeCell(cmd.Cell, cmd.ErrorChan)
	case cmdQueryUnallocatedComputeCell:
		err = manager.handleGetUnallocatedCells(cmd.ResultChan)
	case cmdQueryZoneStatus:
		err = manager.handleQueryZoneStatus(cmd.ResultChan)
	case cmdQueryComputePoolStatus:
		err = manager.handleQueryComputePoolStatus(cmd.ResultChan)
	case cmdGetComputePoolStatus:
		err = manager.handleGetComputePoolStatus(cmd.Pool, cmd.ResultChan)
	case cmdQueryComputeCellStatus:
		err = manager.handleQueryComputeCellStatus(cmd.Pool, cmd.ResultChan)
	case cmdGetComputeCellStatus:
		err = manager.handleGetComputeCellStatus(cmd.Pool, cmd.Cell, cmd.ResultChan)
	case cmdUpdateCellInfo:
		err = manager.handleUpdateCellInfo(cmd.Cell, cmd.Address, cmd.ErrorChan)
	case cmdGetCellStatus:
		err = manager.handleGetCellStatus(cmd.Cell, cmd.ResultChan)
	case cmdBatchUpdateInstanceStatus:
		err = manager.handleBatchUpdateInstanceStatus(cmd.Pool, cmd.Cell, cmd.InstanceList, cmd.ErrorChan)
	case cmdAllocateInstance:
		err = manager.handleAllocateInstance(cmd.Pool, cmd.Instance, cmd.ResultChan)
	case cmdConfirmInstance:
		err = manager.handleConfirmInstance(cmd.InstanceID, cmd.MonitorPort, cmd.Secret, cmd.Hardware, cmd.ErrorChan)
	case cmdDeallocateInstance:
		err = manager.handleDeallocateInstance(cmd.InstanceID, cmd.Error, cmd.ErrorChan)
	case cmdUpdateInstanceStatus:
		err = manager.handleUpdateInstanceStatus(cmd.Instance, cmd.ErrorChan)
	case cmdGetInstanceStatus:
		err = manager.handleGetInstanceStatus(cmd.InstanceID, cmd.ResultChan)
	case cmdQueryInstanceStatusInPool:
		err = manager.handleQueryInstanceStatusInPool(cmd.Pool, cmd.ResultChan)
	case cmdQueryInstanceStatusInCell:
		err = manager.handleQueryInstanceStatusInCell(cmd.Pool, cmd.Cell, cmd.ResultChan)
	case cmdUpdateInstanceAddress:
		err = manager.handleUpdateInstanceAddress(cmd.InstanceID, cmd.Address, cmd.ErrorChan)
	case cmdRenameInstance:
		err = manager.handleRenameInstance(cmd.InstanceID, cmd.Name, cmd.ErrorChan)
	case cmdUpdateInstancePriority:
		err = manager.handleUpdateInstancePriority(cmd.InstanceID, cmd.Priority, cmd.ErrorChan)
	case cmdUpdateInstanceMonitorSecret:
		err = manager.handleUpdateInstanceMonitorSecret(cmd.InstanceID, cmd.Secret, cmd.ErrorChan)
	case cmdUpdateInstanceNetworkThreshold:
		err = manager.handleUpdateInstanceNetworkThreshold(cmd.InstanceID, cmd.ReceiveSpeed, cmd.SendSpeed, cmd.ErrorChan)
	case cmdUpdateInstanceDiskThreshold:
		err = manager.handleUpdateInstanceDiskThreshold(cmd.InstanceID, cmd.ReadSpeed, cmd.ReadIOPS, cmd.WriteSpeed, cmd.WriteIOPS, cmd.ErrorChan)
	case cmdGetInstanceByName:
		err = manager.handleGetInstanceByName(cmd.Pool, cmd.Name, cmd.ResultChan)
	case cmdSearchGuests:
		err = manager.handleSearchGuestConfig(cmd.InstanceQuery, cmd.ResultChan)
	case cmdAddImageServer:
		err = manager.handleAddImageServer(cmd.Name, cmd.Host, cmd.Port)
	case cmdGetImageServer:
		err = manager.handleGetImageServer(cmd.ResultChan)
	case cmdSetCellDead:
		err = manager.handleSetCellStopped(cmd.Cell, cmd.ErrorChan)
	case cmdQueryMigration:
		err = manager.handleQueryMigration(cmd.ResultChan)
	case cmdGetMigration:
		err = manager.handleGetMigration(cmd.MigrationID, cmd.ResultChan)
	case cmdCreateMigration:
		err = manager.handleCreateMigration(cmd.Migration, cmd.ResultChan)
	case cmdFinishMigration:
		err = manager.handleFinishMigration(cmd.MigrationID, cmd.IDList, cmd.PortList, cmd.ErrorChan)
	case cmdCancelMigration:
		err = manager.handleCancelMigration(cmd.MigrationID, cmd.Error, cmd.ErrorChan)
	case cmdBuildFailoverPlan:
		err = manager.handleBuildFailoverPlan(cmd.Cell, cmd.ResultChan)
	case cmdMigrationInstance:
		err = manager.handleMigrateInstance(cmd.Cell, cmd.Target, cmd.IDList, cmd.PortList, cmd.ErrorChan)
	case cmdPurgeInstance:
		err = manager.handlePurgeInstance(cmd.Cell, cmd.ErrorChan)
	case cmdQueryAddressPool:
		err = manager.handleQueryAddressPool(cmd.ResultChan)
	case cmdGetAddressPool:
		err = manager.handleGetAddressPool(cmd.Address, cmd.ResultChan)
	case cmdCreateAddressPool:
		err = manager.handleCreateAddressPool(cmd.AddressPool, cmd.ErrorChan)
	case cmdModifyAddressPool:
		err = manager.handleModifyAddressPool(cmd.AddressPool, cmd.ResultChan)
	case cmdDeleteAddressPool:
		err = manager.handleDeleteAddressPool(cmd.Address, cmd.ErrorChan)
	case cmdQueryAddressRange:
		err = manager.handleQueryAddressRange(cmd.Address, cmd.Range, cmd.ResultChan)
	case cmdGetAddressRange:
		err = manager.handleGetAddressRange(cmd.Address, cmd.Range, cmd.Start, cmd.ResultChan)
	case cmdAddAddressRange:
		err = manager.handleAddAddressRange(cmd.Address, cmd.Range, cmd.AddressRange, cmd.ErrorChan)
	case cmdRemoveAddressRange:
		err = manager.handleRemoveAddressRange(cmd.Address, cmd.Range, cmd.Start, cmd.ErrorChan)
	case cmdBeginResetSystem:
		err = manager.handleBeginResetSystem(cmd.InstanceID, cmd.ErrorChan)
	case cmdFinishResetSystem:
		err = manager.handleFinishResetSystem(cmd.InstanceID, cmd.Error, cmd.ErrorChan)
	case cmdStartBatchCreateGuest:
		err = manager.handleStartBatchCreateGuest(cmd.BatchCreating, cmd.ResultChan)
	case cmdSetBatchCreateGuestStart:
		err = manager.handleSetBatchCreateGuestStart(cmd.BatchID, cmd.Name, cmd.InstanceID, cmd.ErrorChan)
	case cmdSetBatchCreateGuestFail:
		err = manager.handleSetBatchCreateGuestFail(cmd.BatchID, cmd.Name, cmd.Error, cmd.ErrorChan)
	case cmdGetBatchCreateGuest:
		err = manager.handleGetBatchCreateGuestStatus(cmd.BatchID, cmd.ResultChan)
	case cmdStartBatchDeleteGuest:
		err = manager.handleStartBatchDeleteGuest(cmd.IDList, cmd.ResultChan)
	case cmdSetBatchDeleteGuestSuccess:
		err = manager.handleSetBatchDeleteGuestSuccess(cmd.BatchID, cmd.InstanceID, cmd.ErrorChan)
	case cmdSetBatchDeleteGuestFail:
		err = manager.handleSetBatchDeleteGuestFail(cmd.BatchID, cmd.InstanceID, cmd.Error, cmd.ErrorChan)
	case cmdGetBatchDeleteGuest:
		err = manager.handleGetBatchDeleteGuestStatus(cmd.BatchID, cmd.ResultChan)
	case cmdStartBatchStopGuest:
		err = manager.handleStartBatchStopGuest(cmd.IDList, cmd.ResultChan)
	case cmdSetBatchStopGuestSuccess:
		err = manager.handleSetBatchStopGuestSuccess(cmd.BatchID, cmd.InstanceID, cmd.ErrorChan)
	case cmdSetBatchStopGuestFail:
		err = manager.handleSetBatchStopGuestFail(cmd.BatchID, cmd.InstanceID, cmd.Error, cmd.ErrorChan)
	case cmdGetBatchStopGuest:
		err = manager.handleGetBatchStopGuestStatus(cmd.BatchID, cmd.ResultChan)
	case cmdQuerySystemTemplates:
		err = manager.handleQuerySystemTemplates(cmd.ResultChan)
	case cmdGetSystemTemplate:
		err = manager.handleGetSystemTemplate(cmd.TemplateID, cmd.ResultChan)
	case cmdCreateSystemTemplate:
		err = manager.handleCreateSystemTemplate(cmd.TemplateConfig, cmd.ResultChan)
	case cmdModifySystemTemplate:
		err = manager.handleModifySystemTemplate(cmd.TemplateID, cmd.TemplateConfig, cmd.ErrorChan)
	case cmdDeleteSystemTemplate:
		err = manager.handleDeleteSystemTemplate(cmd.TemplateID, cmd.ErrorChan)
	default:
		log.Printf("<resource_manager> unsupported command type %d", cmd.Type)
		break
	}
	if err != nil {
		log.Printf("<resource_manager> handle command %s fail: %s", cmd.Type.toString(), err.Error())
	}
}

func (manager *ResourceManager) onCellStatusUpdate(report CellStatusReport) {
	var name = report.Name
	cell, exists := manager.cells[name]
	if !exists {
		log.Printf("<resource_manager> ignore status update for invalid cell '%s' ", name)
		return
	}
	//update status
	cell.ResourceUsage = report.ResourceUsage
	cell.LatestUpdate = time.Now()
	cell.Alive = true
	manager.cells[name] = cell
}

func (manager *ResourceManager) onUpdateSystemStatus() {
	const (
		LostThreshold = 10 * time.Second
	)
	//begin := time.Now()

	lostTime := time.Now().Add(-LostThreshold)

	manager.zone.PoolStatistic.Reset()
	manager.zone.CellStatistic.Reset()
	manager.zone.InstanceStatistic.Reset()
	manager.zone.ResourceUsage.Reset()

	var modifiedCells []ManagedComputeCell
	var modifiedPools []ManagedComputePool
	for _, pool := range manager.pools {
		pool.CellStatistic.Reset()
		pool.InstanceStatistic.Reset()
		pool.ResourceUsage.Reset()
		for cellName, _ := range pool.Cells {
			cell, exists := manager.cells[cellName]
			if !exists {
				log.Printf("<resource_manager> can not update status with invalid cell '%s'", cellName)
				continue
			}
			if cell.Alive && cell.LatestUpdate.Before(lostTime) {
				//cell lost
				log.Printf("<resource_manager> cell '%s' lost", cellName)
				cell.Alive = false
				pool.OfflineCells++
				modifiedCells = append(modifiedCells, cell)
				continue
			}
			pool.ResourceUsage.Accumulate(cell.ResourceUsage)
			pool.InstanceStatistic.Accumulate(cell.InstanceStatistic)
			pool.OnlineCells++
		}

		modifiedPools = append(modifiedPools, pool)

		//update zone status
		if pool.Enabled {
			manager.zone.EnabledPools++
		} else {
			manager.zone.DisabledPools++
		}
		manager.zone.CellStatistic.Accumulate(pool.CellStatistic)
		manager.zone.InstanceStatistic.Accumulate(pool.InstanceStatistic)
		manager.zone.ResourceUsage.Accumulate(pool.ResourceUsage)
	}
	for _, cell := range modifiedCells {
		manager.cells[cell.Name] = cell
	}
	for _, pool := range modifiedPools {
		manager.pools[pool.Name] = pool
	}

	//elapsed := time.Now().Sub(begin)/time.Millisecond
	//log.Printf("<resource_manager> resource usage accumulated in %d milliseconds", elapsed)
}

func (manager *ResourceManager) handleQueryAllPools(resp chan ResourceResult) error {
	var result []ComputePoolInfo
	var names []string
	for name, _ := range manager.pools{
		names = append(names, name)
	}
	sort.Stable(sort.StringSlice(names))
	for _, poolName := range names {
		pool, _ := manager.pools[poolName]
		var info = ComputePoolInfo{poolName, pool.Enabled, pool.Network, pool.Storage, pool.Failover, uint64(len(pool.Cells))}
		result = append(result, info)
	}
	resp <- ResourceResult{ComputePoolInfoList:result}
	return nil
}

func (manager *ResourceManager) handleGetComputePool(poolName string, resp chan ResourceResult) error{
	pool, exists := manager.pools[poolName]
	if !exists{
		err := fmt.Errorf("invalid pool '%s'", poolName)
		resp <- ResourceResult{Error:err}
		return err
	}
	resp <- ResourceResult{ComputePoolConfig: pool.ComputePoolInfo}
	return nil
}

func (manager *ResourceManager) handleCreatePool(name, storage, addressPool string, failover bool, resp chan error) (err error) {
	if _, exists := manager.pools[name]; exists {
		err = fmt.Errorf("'%s' alrady exists", name)
		resp <- err
		return err
	}
	var newPool = ManagedComputePool{}
	newPool.Enabled = true
	newPool.Name = name
	newPool.Cells = map[string]bool{}
	newPool.InstanceNames = map[string]string{}
	if "" != storage{
		if _, exists := manager.storagePools[storage]; !exists{
			err = fmt.Errorf("invalid storage pool '%s'", storage)
			resp <- err
			return err
		}
		newPool.Storage = storage
		log.Printf("<resource_manager> new compute pool '%s' using storage '%s' created", name, storage)
	}else{
		if failover{
			err = errors.New("using shared storage to enable Failover feature")
			resp <- err
			return err
		}
		log.Printf("<resource_manager> new compute pool '%s' using local storage created", name)
	}
	if "" != addressPool{
		if _, exists := manager.addressPools[addressPool]; !exists{
			err = fmt.Errorf("invalid address pool '%s'", addressPool)
			resp <- err
			return err
		}
		newPool.Network = addressPool
		log.Printf("<resource_manager> address pool '%s' bound to '%s'", addressPool, name)
	}
	newPool.Failover = failover
	manager.pools[name] = newPool
	resp <- nil
	return manager.saveConfig()
}

func (manager *ResourceManager) handleModifyPool(poolName, storage, addressPool string, failover bool, resp chan error) (err error) {
	pool, exists := manager.pools[poolName]
	if !exists {
		err = fmt.Errorf("invalid pool'%s'", poolName)
		resp <- err
		return err
	}
	if (pool.Storage == storage) && (pool.Failover == failover) && (pool.Network == addressPool){
		err = errors.New("no need to change")
		resp <- err
		return err
	}
	var sharedStorage = "" != storage
	if pool.Failover != failover{
		//change failover
		if failover{
			//enable
			if !sharedStorage{
				err = errors.New("using shared storage to enable Failover feature")
				resp <- err
				return err
			}
			log.Printf("<resource_manager> failover enabled on pool '%s'", poolName)
		}else{
			log.Printf("<resource_manager> failover disabled on pool '%s'", poolName)
		}
		pool.Failover = failover
	}

	if pool.Storage != storage{
		if 0 != len(pool.Cells) {
			err = errors.New("must remove all cells before change storage")
			resp <- err
			return err
		}
		if sharedStorage{
			if _, exists = manager.storagePools[storage]; !exists{
				err = fmt.Errorf("invalid storage pool '%s'", storage)
				resp <- err
				return err
			}
			log.Printf("<resource_manager> compute pool '%s' change to storage pool '%s'", poolName, storage)
		}else if pool.Failover{
			err = errors.New("can not using local storage when failover enabled")
			resp <- err
			return err
		}else{
			log.Printf("<resource_manager> compute pool '%s' change to local storage", poolName)
		}
		pool.Storage = storage
	}
	if addressPool != pool.Network{
		if "" != pool.Network{
			//check previous addresses
			if current, exists := manager.addressPools[pool.Network]; !exists{
				err = fmt.Errorf("invalid current address pool '%s'", pool.Network)
				resp <- err
				return err
			}else{
				var allocated = 0
				for _, addressRange := range current.ranges{
					for allocatedAddress, instanceID := range addressRange.allocated{
						ins, exists := manager.instances[instanceID]
						if !exists{
							err = fmt.Errorf("can't find instance '%s' allocated with address '%s' in current pool '%s'",
								instanceID, allocatedAddress, pool.Network)
							resp <- err
							return err
						}
						if ins.Pool == poolName{
							allocated++
						}
					}
				}
				if 0 != allocated{
					err = fmt.Errorf("%d instance address(es) allocated in current pool '%s', remove or detach all address before change address pool",
						allocated, current.name)
					resp <- err
					return err
				}
			}
		}
		if "" != addressPool{
			if _, exists := manager.addressPools[addressPool]; !exists{
				err = fmt.Errorf("invalid address pool '%s'", addressPool)
				resp <- err
				return err
			}
			log.Printf("<resource_manager> address pool of '%s' changed to '%s'", poolName, addressPool)
		}else{
			log.Printf("<resource_manager> address pool '%s' detached from '%s'", pool.Network, poolName)
		}

		pool.Network = addressPool

	}

	manager.pools[poolName] = pool
	resp <- nil
	return manager.saveConfig()
}

func (manager *ResourceManager) handleDeletePool(name string, resp chan error) error {
	pool, exists := manager.pools[name]
	if !exists {
		err := fmt.Errorf("invalid compute pool '%s'", name)
		resp <- err
		return err
	}
	if 0 != len(pool.Cells) {
		err := errors.New("must remove all cells before delete")
		resp <- err
		return err
	}
	delete(manager.pools, name)
	log.Printf("<resource_manager> compute pool '%s' deleted", name)
	resp <- nil
	return manager.saveConfig()
}

//storage pools
func (manager *ResourceManager) handleCreateStoragePool(name, storageType, host, target string, respChan chan error) (err error){
	if _, exists := manager.storagePools[name]; exists{
		err = fmt.Errorf("storage pool '%s' already exists", name)
		respChan <- err
		return err
	}
	switch storageType {
	case StorageTypeNFS:
		break
	default:
		err = fmt.Errorf("invalid storage type '%s'", storageType)
		respChan <- err
		return err
	}
	var newStorage = StoragePoolInfo{name, storageType, host, target}
	manager.storagePools[name] = newStorage
	log.Printf("<resource_manager> new storage pool '%s' created for %s://%s/%s",
		name, storageType, host, target)
	respChan <- nil
	return manager.saveConfig()
}

func (manager *ResourceManager) handleModifyStoragePool(name, storageType, host, target string, respChan chan error) (err error){
	currentStorage, exists := manager.storagePools[name]
	if !exists{
		err = fmt.Errorf("invalid storage pool '%s'", name)
		respChan <- err
		return err
	}
	//check attached compute pool
	for poolName, pool := range manager.pools{
		if pool.Storage == name{
			err = fmt.Errorf("compute pool '%s' still attached to storage '%s'", poolName, name)
			respChan <- err
			return err
		}
	}

	switch storageType {
	case StorageTypeNFS:
		break
	default:
		err = fmt.Errorf("invalid storage type '%s'", storageType)
		respChan <- err
		return err
	}
	var isEqual = func(source, target StoragePoolInfo) bool{
		if source.Type != target.Type{
			return false
		}
		if source.Host != target.Host{
			return false
		}
		if source.Target != target.Target{
			return false
		}
		return true
	}
	var newStorage = StoragePoolInfo{name, storageType, host, target}
	if isEqual(currentStorage, newStorage){
		err = errors.New("no need to change")
		respChan <- err
		return err
	}
	manager.storagePools[name] = newStorage
	log.Printf("<resource_manager> storage pool '%s' changed to %s://%s/%s",
		name, storageType, host, target)
	respChan <- nil
	return manager.saveConfig()
}

func (manager *ResourceManager) handleDeleteStoragePool(name string, respChan chan error) (err error){
	if _, exists := manager.storagePools[name]; !exists{
		err = fmt.Errorf("invalid storage pool '%s'", name)
		respChan <- err
		return err
	}
	//check attached compute pool
	for poolName, pool := range manager.pools{
		if pool.Storage == name{
			err = fmt.Errorf("compute pool '%s' still attached to storage '%s'", poolName, name)
			respChan <- err
			return err
		}
	}
	delete(manager.storagePools, name)
	log.Printf("<resource_manager> storage pool '%s' deleted", name)
	respChan <- nil
	return manager.saveConfig()
}

func (manager *ResourceManager) handleGetStoragePool(name string, respChan chan ResourceResult) (err error){
	pool, exists := manager.storagePools[name]
	if !exists{
		err = fmt.Errorf("invalid storage pool '%s'", name)
		respChan <- ResourceResult{Error:err}
		return err
	}
	respChan <- ResourceResult{StoragePool: pool}
	return nil
}

func (manager *ResourceManager) handleQueryStoragePool(respChan chan ResourceResult) (err error){
	var result []StoragePoolInfo
	var keys = make([]string, len(manager.storagePools))
	var keyIndex = 0
	for name, _ := range manager.storagePools{
		keys[keyIndex] = name
		keyIndex++
	}

	sort.Stable(sort.StringSlice(keys))
	for _, poolName := range keys{
		storage, exists := manager.storagePools[poolName]
		if !exists{
			err = fmt.Errorf("invalid storage pool '%s'", poolName)
			respChan <- ResourceResult{Error:err}
			return err
		}
		result = append(result, storage)
	}
	respChan <- ResourceResult{StoragePoolList: result}
	return nil
}

func (manager *ResourceManager) handleQueryCellsInPool(poolName string, resp chan ResourceResult) error{
	pool, exists := manager.pools[poolName]
	if !exists {
		err := fmt.Errorf("invalid compute pool '%s'", poolName)
		resp <- ResourceResult{Error:err}
		return err
	}
	var names []string
	for name, _ := range pool.Cells{
		names = append(names, name)
	}
	sort.Stable(sort.StringSlice(names))

	var cells []ComputeCellInfo
	for _, cellName:= range names{
		if cell, exists := manager.cells[cellName];!exists{
			err := fmt.Errorf("invalid compute cell '%s'", cellName)
			resp <- ResourceResult{Error:err}
			return err
		}else {
			var info = ComputeCellInfo{cell.Name, cell.Address, cell.Enabled, cell.Alive, cell.PurgeAppending}
			cells = append(cells, info)
		}
	}
	resp <- ResourceResult{ComputeCellInfoList:cells}
	return nil
}

func (manager *ResourceManager) handleAddCell(poolName, cellName string, resp chan error) error {
	pool, exists := manager.pools[poolName]
	if !exists {
		err := fmt.Errorf("invalid compute pool '%s'", poolName)
		resp <- err
		return err
	}
	if _, exists := manager.unallocatedCells[cellName]; !exists {
		err := fmt.Errorf("cell '%s' already allocated", cellName)
		resp <- err
		return err
	}
	cell, exists := manager.cells[cellName]
	if !exists {
		err := fmt.Errorf("invalid compute cell '%s'", cellName)
		resp <- err
		return err
	}
	if cell.Pool != "" {
		err := fmt.Errorf("cell '%s' already in pool '%s'", cellName, cell.Pool)
		resp <- err
		return err
	}
	delete(manager.unallocatedCells, cellName)
	cell.Pool = poolName
	pool.Cells[cellName] = true
	pool.CellCount = uint64(len(pool.Cells))

	manager.cells[cellName] = cell
	manager.pools[poolName] = pool
	log.Printf("<resource_manager> new cell '%s' added into pool '%s'", cellName, poolName)
	resp <- nil
	return manager.saveConfig()
}

func (manager *ResourceManager) handleRemoveCell(poolName, cellName string, resp chan error) error {
	pool, exists := manager.pools[poolName]
	if !exists {
		err := fmt.Errorf("invalid compute pool '%s'", poolName)
		resp <- err
		return err
	}
	cell, exists := manager.cells[cellName]
	if !exists {
		err := fmt.Errorf("invalid compute cell '%s'", cellName)
		resp <- err
		return err
	}
	if cell.Pool != poolName {
		err := fmt.Errorf("cell '%s' not in pool '%s'", cellName, cell.Pool)
		resp <- err
		return err
	}
	var left = len(cell.Instances) + len(cell.Pending)
	if 0 != left{
		err := fmt.Errorf("%d instance(s) left in cell '%s', migrate or delete all instance(s) before remove cell", left, cellName,)
		resp <- err
		return err
	}
	delete(pool.Cells, cellName)
	cell.Pool = ""
	manager.unallocatedCells[cellName] = true
	pool.CellCount = uint64(len(pool.Cells))

	//update
	manager.cells[cellName] = cell
	manager.pools[poolName] = pool

	log.Printf("<resource_manager> cell '%s' removed from pool '%s'", cellName, poolName)
	resp <- nil
	return manager.saveConfig()
}

func (manager *ResourceManager) handleEnableCell(poolName, cellName string, respChan chan error) (err error){
	_, exists := manager.pools[poolName]
	if !exists {
		err = fmt.Errorf("invalid compute pool '%s'", poolName)
		respChan <- err
		return err
	}
	cell, exists := manager.cells[cellName]
	if !exists {
		err = fmt.Errorf("invalid compute cell '%s'", cellName)
		respChan <- err
		return err
	}
	if cell.Pool != poolName {
		err = fmt.Errorf("cell '%s' not in pool '%s'", cellName, cell.Pool)
		respChan <- err
		return err
	}
	if cell.Enabled{
		err = fmt.Errorf("cell '%s' already enabled", cellName)
		respChan <- err
		return err
	}
	if cell.PurgeAppending{
		log.Printf("<resource_manager> warning: appending purge canceled when cell '%s' in pool '%s' enabled", cellName, poolName)
	}else{
		log.Printf("<resource_manager> cell '%s' in pool '%s' enabled", cellName, poolName)
	}
	cell.PurgeAppending = false
	cell.Enabled = true
	manager.cells[cellName] = cell
	respChan <- nil
	return manager.saveConfig()
}

func (manager *ResourceManager) handleDisableCell(poolName, cellName string, purge bool, respChan chan error) (err error){
	_, exists := manager.pools[poolName]
	if !exists {
		err = fmt.Errorf("invalid compute pool '%s'", poolName)
		respChan <- err
		return err
	}
	cell, exists := manager.cells[cellName]
	if !exists {
		err = fmt.Errorf("invalid compute cell '%s'", cellName)
		respChan <- err
		return err
	}
	if cell.Pool != poolName {
		err = fmt.Errorf("cell '%s' not in pool '%s'", cellName, cell.Pool)
		respChan <- err
		return err
	}
	if !cell.Enabled{
		err = fmt.Errorf("cell '%s' already disabled", cellName)
		respChan <- err
		return err
	}
	if purge{
		log.Printf("<resource_manager> cell '%s' in pool '%s' disabled with purge appending", cellName, poolName)
	}else{
		log.Printf("<resource_manager> cell '%s' in pool '%s' disabled", cellName, poolName)
	}
	cell.PurgeAppending = purge
	cell.Enabled = false
	manager.cells[cellName] = cell
	respChan <- nil
	return manager.saveConfig()
}

func (manager *ResourceManager) handleFinishPurgeCell(cellName string, respChan chan error) (err error){
	cell, exists := manager.cells[cellName]
	if !exists {
		err = fmt.Errorf("invalid compute cell '%s'", cellName)
		respChan <- err
		return err
	}
	if !cell.PurgeAppending{
		err = fmt.Errorf("cell '%s' doesn't have purge appending", cellName)
		respChan <- err
		return err
	}
	cell.PurgeAppending = false
	manager.cells[cellName] = cell
	log.Printf("<resource_manager> appending purge in cell '%s' canceled ", cellName)
	respChan <- nil
	return manager.saveConfig()
}

func (manager *ResourceManager) handleGetUnallocatedCells(resp chan ResourceResult) error {
	var cells []ComputeCellInfo
	for name, _ := range manager.unallocatedCells {
		if cell, exists := manager.cells[name];!exists{
			err := fmt.Errorf("invalid cell '%s'", name)
			resp <- ResourceResult{Error:err}
			return err
		}else{
			var info = ComputeCellInfo{cell.Name, cell.Address, cell.Enabled, cell.Alive, cell.PurgeAppending}
			cells = append(cells, info)
		}
	}
	resp <- ResourceResult{ComputeCellInfoList:cells}
	return nil
}

func (manager *ResourceManager) handleQueryZoneStatus(resp chan ResourceResult) error {
	var s = ZoneStatus{Name:manager.zone.Name,
		PoolStatistic: manager.zone.PoolStatistic, CellStatistic: manager.zone.CellStatistic,
		InstanceStatistic: manager.zone.InstanceStatistic, ResourceUsage: manager.zone.ResourceUsage,
		StartTime: manager.startTime}

	resp <- ResourceResult{Zone: s}
	return nil
}

func (manager *ResourceManager) handleQueryComputePoolStatus(resp chan ResourceResult) error {
	var pools []ComputePoolStatus
	var names []string
	for name, _ := range manager.pools{
		names = append(names, name)
	}
	sort.Stable(sort.StringSlice(names))
	for _, poolName := range names {
		pool, _ := manager.pools[poolName]
		var s = ComputePoolStatus{
			Name:pool.Name, Enabled:pool.Enabled,
			CellStatistic:pool.CellStatistic, InstanceStatistic: pool.InstanceStatistic, ResourceUsage: pool.ResourceUsage}
		pools = append(pools, s)
	}
	resp <- ResourceResult{ComputePoolList: pools}
	return nil
}

func (manager *ResourceManager) handleGetComputePoolStatus(name string, resp chan ResourceResult) error {
	pool, exists := manager.pools[name]
	if !exists {
		err := fmt.Errorf("invalid pool '%s'", name)
		resp <- ResourceResult{Error: err}
		return err
	}
	var s = ComputePoolStatus{
		Name:pool.Name, Enabled:pool.Enabled,
		CellStatistic:pool.CellStatistic, InstanceStatistic: pool.InstanceStatistic, ResourceUsage: pool.ResourceUsage}

	resp <- ResourceResult{ComputePool: s}
	return nil
}


func (manager *ResourceManager) handleQueryComputeCellStatus(poolName string, resp chan ResourceResult) error {
	pool, exists := manager.pools[poolName]
	if !exists{
		err := fmt.Errorf("invalid pool '%s'", poolName)
		resp <- ResourceResult{Error:err}
		return err
	}
	var result []ComputeCellStatus
	var names []string
	for name, _ := range pool.Cells{
		names = append(names, name)
	}
	sort.Stable(sort.StringSlice(names))

	for _, cellName:= range names{
		cell, exists := manager.cells[cellName]
		if !exists {
			err := fmt.Errorf("invalid cell '%s' in pool %s", cellName, poolName)
			resp <- ResourceResult{Error: err}
			return err
		}
		var s = ComputeCellStatus{ComputeCellInfo:cell.ComputeCellInfo,
			InstanceStatistic: cell.InstanceStatistic, ResourceUsage: cell.ResourceUsage}
		result = append(result, s)
	}
	resp <- ResourceResult{ComputeCellList: result}
	return nil
}

func (manager *ResourceManager) handleGetComputeCellStatus(poolName, cellName string, resp chan ResourceResult) error{
	cell, exists := manager.cells[cellName]
	if !exists {
		err := fmt.Errorf("invalid cell '%s'", cellName)
		resp <- ResourceResult{Error:err}
		return err
	}
	if cell.Pool != poolName{
		err := fmt.Errorf("cell '%s' not in pool '%s'", cellName, poolName)
		resp <- ResourceResult{Error:err}
		return err
	}
	var s = ComputeCellStatus{ComputeCellInfo:cell.ComputeCellInfo, InstanceStatistic: cell.InstanceStatistic, ResourceUsage: cell.ResourceUsage}
	resp <- ResourceResult{ComputeCell: s}
	return nil
}

func (manager *ResourceManager) handleUpdateCellInfo(cellName, cellAddress string, respChan chan error) (err error){
	cell, exists := manager.cells[cellName]
	if !exists{
		var cellStatus = ManagedComputeCell{}
		cellStatus.LatestUpdate = time.Now()
		cellStatus.Name = cellName
		cellStatus.Address = cellAddress
		cellStatus.Enabled = true
		cellStatus.Alive = true
		cellStatus.Instances = map[string]bool{}
		cellStatus.Pending = map[string]bool{}
		manager.unallocatedCells[cellName] = true
		manager.cells[cellName] = cellStatus
		log.Printf("<resource_manager> new unallocated cell '%s' (address %s) available", cellName, cellAddress)
	}else{
		cell.Alive = true
		cell.LatestUpdate = time.Now()
		if cell.Address != cellAddress{
			log.Printf("<resource_manager> cell '%s' address changed to %s", cellName, cellAddress)
			cell.Address = cellAddress
		}
		manager.cells[cellName] = cell
	}
	respChan <- nil
	return nil

}

func (manager *ResourceManager) handleGetCellStatus(cellName string, respChan chan ResourceResult) error {
	cell, exists := manager.cells[cellName]
	if !exists {
		err := fmt.Errorf("invalid cell '%s'", cellName)
		respChan <- ResourceResult{Error: err}
		return err
	}
	var status = ComputeCellStatus{cell.ComputeCellInfo, cell.InstanceStatistic, cell.ResourceUsage}
	respChan <- ResourceResult{Pool: cell.Pool, ComputeCell: status}
	return nil
}

func (manager *ResourceManager) handleSetCellStopped(cellName string, respChan chan error) (err error){
	cell, exists := manager.cells[cellName]
	if !exists {
		err = fmt.Errorf("invalid cell '%s'", cellName)
		respChan <- err
		return err
	}
	cell.Alive = false
	log.Printf("<resource_manager> remote cell '%s' stopped", cellName)
	if "" != cell.Pool{
		//update resource statistic
		cell.LostInstances = 0
		for instanceID, _ := range cell.Instances{
			if ins, exists := manager.instances[instanceID];exists{
				if ins.Running{
					cell.RunningInstances--
				}else{
					cell.StoppedInstances--
				}
				cell.LostInstances++
				ins.Lost = true
				manager.instances[instanceID] = ins
			}else{
				err = fmt.Errorf("invalid instance '%s' in cell '%s'", instanceID, cellName)
				respChan <- err
				return err
			}
		}
	}
	manager.cells[cellName] = cell
	respChan <- nil
	return nil
}
func (manager *ResourceManager) handleSearchGuestConfig(condition GuestQueryCondition, respChan chan ResourceResult) (err error) {

	var idList []string
	{
		//build filter list
		if condition.InCell {
			if cell, exists := manager.cells[condition.Cell]; !exists {
				err := fmt.Errorf("invalid cell '%s'", condition.Cell)
				respChan <- ResourceResult{Error:err}
				return err
			} else {
				for id, _ := range cell.Instances {
					idList = append(idList, id)
				}
			}
		} else if pool, exists := manager.pools[condition.Pool]; !exists {
			err := fmt.Errorf("invalid pool '%s'", condition.Pool)
			respChan <- ResourceResult{Error:err}
			return err
		} else {
			for cellName, _ := range pool.Cells {
				if cell, exists := manager.cells[cellName]; !exists {
					err := fmt.Errorf("invalid cell '%s' in pool '%s'", condition.Cell, condition.Pool)
					respChan <- ResourceResult{Error:err}
					return err
				} else {
					for id, _ := range cell.Instances {
						idList = append(idList, id)
					}
				}

			}
		}
	}
	var names []string
	var nameToID = map[string]string{}
	{
		//filter by property
		for _, id := range idList {
			if instance, exists := manager.instances[id]; !exists {
				err := fmt.Errorf("invalid instance '%s'", id)
				respChan <- ResourceResult{Error:err}
				return err
			} else {
				if !(condition.WithOwner && instance.User == condition.Owner) && !(condition.WithGroup && instance.Group == condition.Group) {
					continue
				}
				if condition.WithStatus && (instance.Running != (condition.Status == InstanceStatusRunning)) {
					continue
				}
				if condition.WithCreateFlag && (instance.Created != condition.Created) {
					continue
				}

				if _, exists := nameToID[instance.Name];exists{
					err = fmt.Errorf("encounter duplicate instance name '%s'", instance.Name)
					respChan <- ResourceResult{Error:err}
					return
				}

				names = append(names, instance.Name)
				nameToID[instance.Name] = id
			}
		}
	}
	sort.Stable(sort.StringSlice(names))
	var result []InstanceStatus
	for _, name := range names{
		id, exists := nameToID[name]
		if !exists{
			err = fmt.Errorf("no instance mapped with name '%s'", name)
			respChan <- ResourceResult{Error:err}
			return err
		}
		ins, exists := manager.instances[id]
		if !exists{
			err = fmt.Errorf("invalid instance '%s' with name '%s'", id, name)
			respChan <- ResourceResult{Error:err}
			return err
		}
		result = append(result, ins)
	}
	respChan <- ResourceResult{InstanceList: result}
	return nil
}

func (manager *ResourceManager) handleBatchUpdateInstanceStatus(poolName, cellName string, instances []InstanceStatus, respChan chan error) error {
	pool, exists := manager.pools[poolName]
	if !exists{
		err := fmt.Errorf("invalid pool '%s'", poolName)
		respChan <- err
		return err
	}
	cell, exists := manager.cells[cellName]
	if !exists {
		err := fmt.Errorf("invalid cell '%s'", cellName)
		respChan <- err
		return err
	}
	if cell.Pool != poolName {
		err := fmt.Errorf("cell '%s' not belong to '%s'", cellName, poolName)
		respChan <- err
		return err
	}
	if 0 != len(cell.Instances) {
		for id, _ := range cell.Instances {
			log.Printf("<resource_manager> clear expired instance status, id '%s'", id)
			if ins, exists := manager.instances[id];exists{
				delete(pool.InstanceNames, ins.Name)
			}
			delete(manager.instances, id)
		}
	}
	cell.Instances = map[string]bool{}
	cell.Pending = map[string]bool{}
	cell.InstanceStatistic.Reset()
	for _, config := range instances {
		config.InternalNetwork.MonitorAddress = cell.Address
		manager.instances[config.ID] = config
		cell.Instances[config.ID] = true
		//todo: migrating
		if config.Running{
			cell.RunningInstances++
		}else{
			cell.StoppedInstances++
		}
		pool.InstanceNames[config.Name] = config.ID
	}

	manager.cells[cellName] = cell
	manager.pools[poolName] = pool
	log.Printf("<resource_manager> %d instance updated in cell '%s' ", len(cell.Instances), cellName)
	respChan <- nil
	return nil
}

func (manager *ResourceManager) handleAllocateInstance(poolName string, config InstanceStatus, respChan chan ResourceResult) (err error) {
	pool, exists := manager.pools[poolName]
	if !exists{
		err = fmt.Errorf("invalid compute pool '%s'", poolName)
		respChan <- ResourceResult{Error: err}
		return err
	}
	if _, exists = pool.InstanceNames[config.Name];exists{
		err = fmt.Errorf("instance '%s' already exists in pool '%s'", config.Name, poolName)
		respChan <- ResourceResult{Error: err}
		return err
	}
	var newID = uuid.NewV4()
	config.ID = newID.String()
	cellName, err := manager.selectCell(poolName, config.InstanceResource, true)
	if err != nil {
		log.Printf("<resource_manager> select cell fail: %s", err.Error())
		respChan <- ResourceResult{Error: err}
		return err
	}
	config.Cell = cellName
	config.Pool = poolName
	cell, exists := manager.cells[cellName]
	if !exists {
		err = fmt.Errorf("invalid cell '%s'", cellName)
		respChan <- ResourceResult{Error: err}
		return err
	}
	if "" != pool.Network{
		//select address
		internal, external, err := manager.allocateNetworkAddress(pool, config.ID)
		if err != nil{
			respChan <- ResourceResult{Error: err}
			return err
		}
		if "" != external{
			config.ExternalNetwork.AssignedAddress = external
			log.Printf("<resource_manager> address '%s/%s' assigned for instance '%s'", internal, external, config.Name)
		}else{
			log.Printf("<resource_manager> internal address '%s' assigned for instance '%s'", internal, config.Name)
		}
		config.InternalNetwork.AssignedAddress = internal
	}
	config.InternalNetwork.MonitorAddress = cell.Address
	cell.Pending[config.ID] = true
	pool.InstanceNames[config.Name] = config.ID

	manager.instances[config.ID] = config
	manager.cells[cellName] = cell
	manager.pools[poolName] = pool
	respChan <- ResourceResult{Instance: config}
	log.Printf("<resource_manager> allocate cell '%s' for instance '%s'(%s)", cellName, config.Name, config.ID)
	return nil
}

//running/create/progress/media only
func (manager *ResourceManager) handleUpdateInstanceStatus(status InstanceStatus, respChan chan error) error {
	ins, exists := manager.instances[status.ID]
	if !exists {
		err := fmt.Errorf("invalid instance '%s'", status.ID)
		respChan <- err
		return err
	}
	if ins.Running != status.Running {
		cell, exists := manager.cells[ins.Cell]
		if !exists{
			err := fmt.Errorf("invalid cell '%s' for instance '%s'", ins.Cell, status.ID)
			respChan <- err
			return err
		}
		if ins.Running{
			//running => stopped
			cell.StoppedInstances++
			cell.RunningInstances--
		}else{
			//stopped => running
			cell.StoppedInstances--
			cell.RunningInstances++
		}
		ins.Running = status.Running
		manager.cells[ins.Cell] = cell
	}
	if !ins.Created {
		ins.Progress = status.Progress
		ins.Created = status.Created
	}
	if ins.Cores != status.Cores{
		ins.Cores = status.Cores
	}
	if ins.Memory != status.Memory{
		ins.Memory = status.Memory
	}
	if len(ins.Disks) == len(status.Disks){
		for index := 0; index < len(ins.Disks); index++{
			if ins.Disks[index] != status.Disks[index]{
				ins.Disks[index] = status.Disks[index]
			}
		}
	}
	if ins.MediaAttached != status.MediaAttached{
		ins.MediaAttached = status.MediaAttached
		ins.MediaSource = status.MediaSource
	}

	manager.instances[status.ID] = ins
	respChan <- nil
	return nil
}

func (manager *ResourceManager) handleConfirmInstance(id string, monitorPort uint, monitorSecret, ethernetAddress string, respChan chan error) error {
	status, exists := manager.instances[id]
	if !exists {
		err := fmt.Errorf("invalid instance '%s'", id)
		respChan <- err
		return err
	}
	cell, exists := manager.cells[status.Cell]
	if !exists {
		err := fmt.Errorf("invalid cell '%s'", status.Cell)
		respChan <- err
		return err
	}
	if _, exists := cell.Pending[id]; !exists {
		err := fmt.Errorf("instance '%s' already confirmed", id)
		respChan <- err
		return err
	}
	delete(cell.Pending, id)
	cell.Instances[id] = true
	cell.StoppedInstances++

	status.InternalNetwork.MonitorPort = monitorPort
	status.MonitorSecret = monitorSecret
	status.HardwareAddress = ethernetAddress
	status.Created = true
	status.Progress = 0
	status.CreateTime = time.Now().Format(TimeFormatLayout)

	//update
	manager.instances[id] = status
	manager.cells[status.Cell] = cell
	log.Printf("<resource_manager> instance '%s' confirmed, monitor port %d", id, status.InternalNetwork.MonitorPort)
	respChan <- nil
	return nil
}
func (manager *ResourceManager) handleDeallocateInstance(id string, err error, respChan chan error) error {
	ins, exists := manager.instances[id]
	if !exists {
		err := fmt.Errorf("invalid instance '%s'", id)
		respChan <- err
		return err
	}
	if ins.Cell == "" {
		err := fmt.Errorf("instance '%s' not allocated", id)
		respChan <- err
		return err
	}
	var cellName = ins.Cell
	cell, exists := manager.cells[cellName]
	if !exists {
		err := fmt.Errorf("invalid cell '%s'", cellName)
		respChan <- err
		return err
	}
	if _, exists = cell.Pending[id]; exists {
		delete(cell.Pending, id)
		log.Printf("<resource_manager> pending instance '%s' deallocated in cell '%s'", id, cellName)
	} else {
		delete(cell.Instances, id)
		log.Printf("<resource_manager> instance '%s' deallocated in cell '%s'", id, cellName)
	}
	if ins.Running{
		cell.RunningInstances--
	}else{
		cell.StoppedInstances--
	}

	if pool, exists := manager.pools[ins.Pool];exists{
		if "" != pool.Network{
			manager.deallocateNetworkAddress(pool, ins.InternalNetwork.AssignedAddress, ins.ExternalNetwork.AssignedAddress)
		}
		delete(pool.InstanceNames, ins.Name)
		manager.pools[ins.Pool] = pool
	}

	//update instance statistic
	manager.cells[cellName] = cell
	delete(manager.instances, id)
	if err != nil{
		manager.pendingError[id] = err
	}
	respChan <- nil
	return nil
}

func (manager *ResourceManager) handleGetInstanceStatus(id string, respChan chan ResourceResult) (err error) {
	var exists bool
	var status InstanceStatus
	if err, exists = manager.pendingError[id]; exists{
		//fetch pending error
		delete(manager.pendingError, id)
		respChan <- ResourceResult{Error: err}
		log.Printf("<resource_manager> pending error of instance '%s' fetched", id)
		return nil
	}else if status, exists = manager.instances[id]; exists{
		respChan <- ResourceResult{Instance: status}
		return nil
	}else{
		err = fmt.Errorf("invalid instance '%s'", id)
		respChan <- ResourceResult{Error: err}
		return err
	}
}


func (manager *ResourceManager) handleQueryInstanceStatusInPool(poolName string, respChan chan ResourceResult) (err error){

	pool, exists := manager.pools[poolName]
	if !exists{
		err = fmt.Errorf("invalid pool '%s'", poolName)
		respChan <- ResourceResult{Error:err}
		return err
	}
	var idList []string
	for cellName, _ := range pool.Cells{
		if cell, exists := manager.cells[cellName]; !exists{
			err = fmt.Errorf("invalid cell '%s'", cellName)
			respChan <- ResourceResult{Error:err}
			return err
		}else{
			for instanceID, _ := range cell.Instances{
				idList = append(idList, instanceID)
			}
			for instanceID, _ := range cell.Pending{
				idList = append(idList, instanceID)
			}
		}
	}
	result, err := manager.getSortedInstances(idList)
	if err != nil{
		respChan <- ResourceResult{Error:err}
		return err
	}
	respChan <- ResourceResult{InstanceList: result}
	return nil
}

func (manager *ResourceManager) handleQueryInstanceStatusInCell(poolName, cellName string, respChan chan ResourceResult) (err error){
	pool, exists := manager.pools[poolName]
	if !exists{
		err = fmt.Errorf("invalid pool '%s'", poolName)
		respChan <- ResourceResult{Error:err}
		return err
	}
	if _, exists := pool.Cells[cellName];!exists{
		err = fmt.Errorf("cell '%s' not in pool '%s'", cellName, poolName)
		respChan <- ResourceResult{Error:err}
		return err
	}
	cell, exists := manager.cells[cellName]
	if !exists{
		err = fmt.Errorf("invalid cell '%s'", cellName)
		respChan <- ResourceResult{Error:err}
		return err
	}
	var idList []string
	for instanceID, _ := range cell.Instances{
		idList = append(idList, instanceID)
	}
	for instanceID, _ := range cell.Pending{
		idList = append(idList, instanceID)
	}

	result, err := manager.getSortedInstances(idList)
	if err != nil{
		respChan <- ResourceResult{Error:err}
		return err
	}

	respChan <- ResourceResult{InstanceList: result}
	return nil
}

func (manager *ResourceManager) getSortedInstances(idList []string) (result []InstanceStatus, err error){
	var names []string
	var nameToID = map[string]string{}
	for _, id := range idList{
		ins, exists := manager.instances[id]
		if !exists{
			err = fmt.Errorf("invalid instance '%s", id)
			return
		}

		if _, exists := nameToID[ins.Name];exists{
			err = fmt.Errorf("encounter duplicate instance name '%s'", ins.Name)
			return
		}
		nameToID[ins.Name] = id
		names = append(names, ins.Name)
	}
	sort.Stable(sort.StringSlice(names))
	for _, name := range names{
		id, exists := nameToID[name]
		if !exists{
			err = fmt.Errorf("no instance mapped with name '%s'", name)
			return
		}
		ins, exists := manager.instances[id]
		if !exists{
			err = fmt.Errorf("invalid instance '%s' with name '%s'", id, name)
			return
		}
		result = append(result, ins)
	}
	return
}

func (manager *ResourceManager)  handleUpdateInstanceAddress(id, ip string, respChan chan error) error{
	instance, exists := manager.instances[id]
	if !exists{
		err := fmt.Errorf("invalid instance %s", id)
		respChan <- err
		return err
	}
	if instance.InternalNetwork.InstanceAddress != ip{
		instance.InternalNetwork.InstanceAddress = ip
		log.Printf("<resource_manager> update address of instance '%s' to %s", instance.Name, ip)
		manager.instances[id] = instance
	}
	respChan <- nil
	return nil
}

func (manager *ResourceManager) handleRenameInstance(id, name string, respChan chan error) (err error){
	instance, exists := manager.instances[id]
	if !exists{
		err = fmt.Errorf("invalid instance %s", id)
		respChan <- err
		return err
	}
	if instance.Name == name{
		err = errors.New("no need to change")
		respChan <- err
		return err
	}
	var previousName = instance.Name
	pool, exists := manager.pools[instance.Pool]
	if !exists{
		err = fmt.Errorf("invalid pool '%s' attached with instance '%s'", instance.Pool, previousName)
		respChan <- err
		return err
	}
	delete(pool.InstanceNames, previousName)
	pool.InstanceNames[name] = id
	manager.pools[pool.Name] = pool

	instance.Name = name
	manager.instances[id] = instance
	log.Printf("<resource_manager> instance '%s' renamed from '%s' to '%s'", id, previousName, name)
	respChan <- nil
	return nil
}

func (manager *ResourceManager) handleUpdateInstancePriority(instanceID string, priority PriorityEnum, respChan chan error) (err error){
	instance, exists := manager.instances[instanceID]
	if !exists{
		err = fmt.Errorf("invalid instance %s", instanceID)
		respChan <- err
		return err
	}
	instance.CPUPriority = priority
	manager.instances[instanceID] = instance
	log.Printf("<resource_manager> CPU priority of instance '%s' changed to %d", instanceID, priority)
	respChan <- nil
	return nil
}

func (manager *ResourceManager) handleUpdateInstanceDiskThreshold(instanceID string, readSpeed, readIOPS, writeSpeed, writeIOPS uint64, respChan chan error)  (err error){
	instance, exists := manager.instances[instanceID]
	if !exists{
		err = fmt.Errorf("invalid instance %s", instanceID)
		respChan <- err
		return err
	}
	instance.ReadSpeed = readSpeed
	instance.ReadIOPS = readIOPS
	instance.WriteSpeed = writeSpeed
	instance.WriteIOPS = writeIOPS
	manager.instances[instanceID] = instance
	log.Printf("<resource_manager> disk threshold of instance '%s' changed to read: %d MB/s, %d ops, write: %d MB/s, %d ops",
		instanceID, readSpeed >> 20, readIOPS, writeSpeed >> 20, writeIOPS)
	respChan <- nil
	return nil
}

func (manager *ResourceManager) handleUpdateInstanceNetworkThreshold(instanceID string, receive, send uint64, respChan chan error) (err error){
	instance, exists := manager.instances[instanceID]
	if !exists{
		err = fmt.Errorf("invalid instance %s", instanceID)
		respChan <- err
		return err
	}
	instance.ReceiveSpeed = receive
	instance.SendSpeed = send
	manager.instances[instanceID] = instance
	log.Printf("<resource_manager> network threshold of instance '%s' changed to receive %d KB/s, send: %d KB/s",
		instanceID, receive >> 10, send >> 10)
	respChan <- nil
	return nil
}

func (manager *ResourceManager) handleGetInstanceByName(poolName, instanceName string, respChan chan ResourceResult) (err error){
	pool, exists := manager.pools[poolName]
	if !exists{
		err = fmt.Errorf("invalid pool '%s'", poolName)
		respChan <- ResourceResult{Error:err}
		return err
	}
	instanceID, exists := pool.InstanceNames[instanceName]
	if !exists{
		err = fmt.Errorf("no instance named '%s' in pool '%s'", instanceName, poolName)
		respChan <- ResourceResult{Error:err}
		return err
	}
	instance, exists := manager.instances[instanceID]
	if !exists{
		err = fmt.Errorf("invalid instance '%s' with name '%s'", instanceID, instanceName)
		respChan <- ResourceResult{Error:err}
		return err
	}
	respChan <- ResourceResult{Instance: instance}
	return nil
}

func (manager *ResourceManager) handleAddImageServer(name, host string, port int) error {
	if _, exists := manager.imageServers[name]; exists {
		return fmt.Errorf("image server '%s' already exists", name)
	}
	manager.imageServers[name] = imageServer{host, port}
	log.Printf("<resource_manager> new image server '%s' added, serve at '%s:%d'", name, host, port)
	return nil
}

func (manager *ResourceManager) handleRemoveImageServer(name string) error {
	if _, exists := manager.imageServers[name]; !exists {
		return fmt.Errorf("invalid image server '%s'", name)
	}
	delete(manager.imageServers, name)
	log.Printf("<resource_manager> image server '%s' removed", name)
	return nil
}

func (manager *ResourceManager) handleGetImageServer(respChan chan ResourceResult) error {
	if 0 == len(manager.imageServers) {
		err := errors.New("no image server available")
		respChan <- ResourceResult{Error: err}
		return err
	}
	for name, server := range manager.imageServers {
		respChan <- ResourceResult{Name: name, Host: server.Host, Port: server.Port}
		break
	}
	return nil
}

func (manager *ResourceManager) handleQueryMigration(respChan chan ResourceResult) (err error){
	var result []MigrationStatus
	var releaseList []string
	for _, m := range manager.migrations{
		result = append(result, m)
		if (m.Finished) || (m.Error != nil){
			releaseList = append(releaseList, m.ID)
		}
	}
	respChan <- ResourceResult{MigrationList: result}
	if 0 != len(releaseList){
		for _, id := range releaseList{
			delete(manager.migrations, id)
		}
		log.Printf("<resource_manager> %d migration(s) released", len(releaseList))
	}
	return nil
}

func (manager *ResourceManager) handleGetMigration(id string, respChan chan ResourceResult) (err error){
	migration, exists := manager.migrations[id]
	if !exists{
		err = fmt.Errorf("invalid migration '%s'", id)
		respChan <- ResourceResult{Error:err}
		return err
	}
	respChan <- ResourceResult{Migration: migration}
	if (migration.Error != nil) || migration.Finished{
		delete(manager.migrations, id)
	}
	return nil
}

func (manager *ResourceManager) handleCreateMigration(params MigrationParameter, respChan chan ResourceResult) (err error){

	if params.TargetPool != params.SourcePool{
		err = errors.New("migrate between pool not support")
		respChan <- ResourceResult{Error:err}
		return err
	}
	if params.TargetCell == params.SourceCell{
		err = errors.New("migrate in same cell")
		respChan <- ResourceResult{Error:err}
		return err
	}
	pool, exists := manager.pools[params.SourcePool]
	if !exists{
		err = fmt.Errorf("invalid pool '%s'", params.SourcePool)
		respChan <- ResourceResult{Error:err}
		return err
	}
	if "" == pool.Storage{
		err = errors.New("migrate only work on shared storage")
		respChan <- ResourceResult{Error:err}
		return err
	}
	storage, exists := manager.storagePools[pool.Storage]
	if !exists{
		err = fmt.Errorf("invalid storage '%s' attached to pool '%s'", pool.Storage, params.SourcePool)
		respChan <- ResourceResult{Error:err}
		return err
	}
	if storage.Type != StorageTypeNFS{
		err = fmt.Errorf("migrate not work on storage type '%s'(storage '%s' attached to pool '%s')",
			storage.Type, pool.Storage, params.SourcePool)
		respChan <- ResourceResult{Error:err}
		return err
	}
	if _, exists = pool.Cells[params.SourceCell];!exists{
		err = fmt.Errorf("source cell '%s' not in pool '%s'", params.SourceCell, params.SourcePool)
		respChan <- ResourceResult{Error:err}
		return err
	}
	if _, exists = pool.Cells[params.TargetCell];!exists{
		err = fmt.Errorf("target cell '%s' not in pool '%s'", params.TargetCell, params.TargetPool)
		respChan <- ResourceResult{Error:err}
		return err
	}
	targetCell, exists := manager.cells[params.TargetCell]
	if !exists{
		err = fmt.Errorf("invalid target cell '%s'", params.TargetCell)
		respChan <- ResourceResult{Error:err}
		return err
	}
	if !targetCell.Alive{
		err = fmt.Errorf("target cell '%s' already lost", params.TargetCell)
		respChan <- ResourceResult{Error:err}
		return err
	}
	if !targetCell.Enabled{
		err = fmt.Errorf("target cell '%s' already disabled", params.TargetCell)
		respChan <- ResourceResult{Error:err}
		return err
	}
	sourceCell, exists := manager.cells[params.SourceCell]
	if !exists{
		err = fmt.Errorf("invalid source cell '%s'", params.SourceCell)
		respChan <- ResourceResult{Error:err}
		return err
	}
	if !sourceCell.Alive{
		err = fmt.Errorf("source cell '%s' already lost", params.SourceCell)
		respChan <- ResourceResult{Error:err}
		return err
	}
	var newID = uuid.NewV4()
	var M = MigrationStatus{}
	M.ID = newID.String()
	M.SourcePool = params.SourcePool
	M.SourceCell = params.SourceCell
	M.TargetPool = params.TargetPool
	M.TargetCell = params.TargetCell
	M.Finished = false
	M.Progress = 0
	if _, exists = manager.migrations[M.ID];exists{
		err = fmt.Errorf("migration '%s' already exists", M.ID)
		respChan <- ResourceResult{Error:err}
		return err
	}

	if 0 == len(params.Instances){
		//migrate all instance in source
		if 0 == len(sourceCell.Instances){
			err = fmt.Errorf("no instance need migrate in cell '%s.%s'", params.SourcePool, params.SourceCell)
			respChan <- ResourceResult{Error:err}
			return err
		}
		for instanceID, _ := range sourceCell.Instances{
			M.Instances = append(M.Instances, instanceID)
		}
	}else{
		M.Instances = params.Instances
	}
	//verify instance status
	for _, instanceID := range M.Instances{
		ins, exists := manager.instances[instanceID]
		if !exists{
			err = fmt.Errorf("invalid instance '%s'", instanceID)
			respChan <- ResourceResult{Error:err}
			return err
		}
		if ins.Running{
			err = fmt.Errorf("instance '%s'('%s') is still running", instanceID, ins.Name)
			respChan <- ResourceResult{Error:err}
			return err
		}
		if ins.Migrating{
			err = fmt.Errorf("instance '%s'('%s') already in migrating", instanceID, ins.Name)
			respChan <- ResourceResult{Error:err}
			return err
		}
		if ins.Cell == targetCell.Name{
			err = fmt.Errorf("instance '%s'('%s') already in cell '%s'", instanceID, ins.Name, targetCell.Name)
			respChan <- ResourceResult{Error:err}
			return err
		}
	}
	//batch update
	for _, instanceID := range M.Instances{
		if ins, exists := manager.instances[instanceID];exists{
			ins.Migrating = true
			manager.instances[instanceID] = ins
		}
	}
	log.Printf("<resource_manager> %d instance(s) will migrate from '%s.%s' to '%s.%s', migration '%s' created",
		len(M.Instances), M.SourcePool, M.SourceCell, M.TargetPool, M.TargetCell, M.ID)
	manager.migrations[M.ID] = M
	respChan <- ResourceResult{Migration: M}
	return nil
}

func (manager *ResourceManager) handleFinishMigration(migrationID string, instances []string, monitorPorts []uint64, respChan chan error) (err error){
	migration, exists := manager.migrations[migrationID]
	if !exists{
		err = fmt.Errorf("invalid migration '%s'", migrationID)
		respChan <- err
		return err
	}
	if err = manager.transferInstances(migration.SourceCell, migration.TargetCell, instances, monitorPorts); err != nil{
		log.Printf("<resource_manager> migrate instance(s) for migration '%s' fail: %s", migrationID, err.Error())
		respChan <- err
		return err
	}
	migration.Finished = true
	manager.migrations[migrationID] = migration
	log.Printf("<resource_manager> migration '%s' finished", migrationID)
	respChan <- nil
	return nil	
}

func (manager *ResourceManager) handleCancelMigration(migrationID string, reason error, respChan chan error)(err error){
	migration, exists := manager.migrations[migrationID]
	if !exists{
		err = fmt.Errorf("invalid migration '%s'", migrationID)
		respChan <- err
		return err
	}
	for _, instanceID := range migration.Instances{
		instance, exists := manager.instances[instanceID]
		if !exists{
			log.Printf("<resource_manager> warning: invalid migrating instance '%s'", instanceID)
			continue
		}
		if instance.Migrating{
			instance.Migrating = false
			log.Printf("<resource_manager> cancel migrating instance '%s'(%s)", instance.Name, instanceID)
			manager.instances[instanceID] = instance
		}
	}
	if reason != nil{
		migration.Error = reason
		log.Printf("<resource_manager> migration '%s' canceled due to %s", migrationID, reason.Error())
	}else{
		migration.Finished = true
		log.Printf("<resource_manager> migration '%s' canceled without reason", migrationID)
	}
	manager.migrations[migrationID] = migration
	respChan <- nil
	return nil
}

func (manager *ResourceManager) handleBuildFailoverPlan(cellName string, respChan chan ResourceResult)(err error){
	cell, exists := manager.cells[cellName]
	if !exists {
		err = fmt.Errorf("invalid cell '%s'", cellName)
		respChan <- ResourceResult{Error:err}
		return err
	}
	cell.Alive = false
	log.Printf("<resource_manager> cell '%s' lost", cellName)
	if "" == cell.Pool{
		//unallocated
		manager.cells[cellName] = cell
		respChan <- ResourceResult{}
		return nil
	}
	pool, exists := manager.pools[cell.Pool]
	if !exists{
		err = fmt.Errorf("invalid pool '%s'", cell.Pool)
		respChan <- ResourceResult{Error:err}
		return err
	}
	cell.InstanceStatistic.Reset()
	if !pool.Failover{
		//mark lost
		for instanceID, _ := range cell.Instances{
			ins, exists := manager.instances[instanceID]
			if !exists{
				err = fmt.Errorf("invalid instance '%s' in cell '%s'", instanceID, cellName)
				respChan <- ResourceResult{Error:err}
				return err
			}
			ins.Lost = true
			manager.instances[instanceID] = ins
		}
		cell.LostInstances = uint64(len(cell.Instances))
		log.Printf("<resource_manager> %d instance(s) in cell '%s' marked to lost", cell.LostInstances, cellName)
		respChan <- ResourceResult{}
		manager.cells[cellName] = cell
		return nil
	}else{
		cell.Enabled = false
		cell.PurgeAppending = true
		manager.cells[cellName] = cell
		//build plan and migrate
		var plan = map[string][]string{}
		for instanceID, _ := range cell.Instances{
			ins, exists := manager.instances[instanceID]
			if !exists{
				err = fmt.Errorf("invalid instance '%s' in cell '%s'", instanceID, cellName)
				respChan <- ResourceResult{Error:err}
				return err
			}
			targetName, err := manager.selectCell(pool.Name, ins.InstanceResource, false)
			if err != nil{
				respChan <- ResourceResult{Error:err}
				return err
			}
			targetCell, exists := manager.cells[targetName]
			if !exists{
				err = fmt.Errorf("invalid target cell '%s'", targetName)
				respChan <- ResourceResult{Error:err}
				return err
			}
			//mark
			targetCell.Instances[instanceID] = false
			ins.Migrating = true
			if insList, exists := plan[targetName]; exists{
				insList = append(insList, instanceID)
				plan[targetName] = insList
			}else{
				plan[targetName] = []string{instanceID}
			}
			manager.instances[instanceID] = ins
			manager.cells[targetName] = targetCell
		}
		cell.MigratingInstances = uint64(len(cell.Instances))
		log.Printf("<resource_manager> schedule migrate %d instance(s) in cell '%s' to %d new cells",
			cell.MigratingInstances, cellName, len(plan))
		respChan <- ResourceResult{FailoverPlan:plan}
		manager.cells[cellName] = cell
		return nil
	}

}

func (manager *ResourceManager) handleMigrateInstance(sourceCell, targetCell string, instances []string, monitorPorts []uint64, respChan chan error)(err error){
	err = manager.transferInstances(sourceCell, targetCell, instances, monitorPorts)
	if err != nil{
		log.Printf("<resource_manager> migrate instance fail: %s", err.Error())
		respChan <- err
		return
	}
	log.Printf("<resource_manager> %d instance(s) migrated from '%s' to '%s'", len(instances), sourceCell, targetCell)
	respChan <- nil
	return nil
}

func (manager *ResourceManager) handlePurgeInstance(cellName string, respChan chan error) (err error){
	cell, exists := manager.cells[cellName]
	if !exists{
		err = fmt.Errorf("invalid cell '%s'", cellName)
		respChan <- err
		return err
	}
	var count = len(cell.Instances) + len(cell.Pending)
	cell.Instances = map[string]bool{}
	cell.Pending = map[string]bool{}
	cell.PurgeAppending = false
	cell.InstanceStatistic.Reset()
	log.Printf("<resource_manager> %d instance(s) purged in cell '%s'", count, cellName)
	manager.cells[cellName] = cell
	respChan <- nil
	return nil
}

//address pool&range
func (manager *ResourceManager) handleQueryAddressPool(respChan chan ResourceResult) (err error){
	var result = make([]AddressPoolStatus, 0)
	for poolName, pool := range manager.addressPools{
		var status AddressPoolStatus
		status.Name = poolName
		status.Gateway = pool.gateway
		status.DNS = pool.dns
		status.Allocated = make([]AllocatedAddress, 0)
		status.Ranges = make([]AddressRangeConfig, 0)
		for _, addressRange := range pool.ranges{
			var rangeConfig AddressRangeConfig
			rangeConfig.Start = addressRange.startAddress.String()
			rangeConfig.End = addressRange.endAddress.String()
			rangeConfig.Netmask = IPv4MaskToString(addressRange.netmask)
			rangeConfig.Capacity = addressRange.capacity
			for allocatedAddress, allocatedInstance := range addressRange.allocated{
				status.Allocated = append(status.Allocated, AllocatedAddress{allocatedAddress, allocatedInstance})
			}
			status.Ranges = append(status.Ranges, rangeConfig)
		}
		result = append(result, status)
	}
	respChan <- ResourceResult{AddressPoolList: result}
	log.Printf("<resource_manager> %d address pool(s) available", len(result))
	return nil
}

func (manager *ResourceManager) handleGetAddressPool(poolName string, respChan chan ResourceResult) (err error){
	pool, exists := manager.addressPools[poolName]
	if !exists{
		err = fmt.Errorf("invalid address pool '%s", poolName)
		respChan <- ResourceResult{Error:err}
		return err
	}
	var status AddressPoolStatus
	status.Name = poolName
	status.Gateway = pool.gateway
	status.DNS = pool.dns
	status.Allocated = make([]AllocatedAddress, 0)
	status.Ranges = make([]AddressRangeConfig, 0)
	for _, addressRange := range pool.ranges{
		var rangeConfig AddressRangeConfig
		rangeConfig.Start = addressRange.startAddress.String()
		rangeConfig.End = addressRange.endAddress.String()
		rangeConfig.Netmask = IPv4MaskToString(addressRange.netmask)
		rangeConfig.Capacity = addressRange.capacity
		for allocatedAddress, allocatedInstance := range addressRange.allocated{
			status.Allocated = append(status.Allocated, AllocatedAddress{allocatedAddress, allocatedInstance})
		}
		status.Ranges = append(status.Ranges, rangeConfig)
	}
	respChan <- ResourceResult{AddressPool: status}
	return nil
}

func (manager *ResourceManager) handleCreateAddressPool(config AddressPoolConfig, respChan chan error) (err error){
	_, exists := manager.addressPools[config.Name]
	if exists{
		err = fmt.Errorf("address pool '%s' already exists", config.Name)
		respChan <- err
		return err
	}
	var pool ManagedAddressPool
	pool.name = config.Name
	//verify params
	if nil == net.ParseIP(config.Gateway){
		err = fmt.Errorf("invalid gateway '%s'", config.Gateway)
		respChan <- err
		return err
	}
	pool.gateway = config.Gateway
	for _, dns := range config.DNS{
		if nil == net.ParseIP(dns){
			err = fmt.Errorf("invalid DNS '%s'", dns)
			respChan <- err
			return err
		}
	}
	pool.dns = config.DNS
	pool.ranges = map[string]ManagedIPV4AddressRange{}
	pool.rangeStartAddressed = make([]string, 0)
	manager.addressPools[pool.name] = pool
	log.Printf("<resource_manager> address pool '%s' created with gateway '%s' and %d DNS server",
		pool.name, pool.gateway, len(pool.dns))
	respChan <- nil
	return manager.saveConfig()
}

func (manager *ResourceManager) handleModifyAddressPool(config AddressPoolConfig, respChan chan ResourceResult) (err error){
	pool, exists := manager.addressPools[config.Name]
	if !exists{
		err = fmt.Errorf("address pool '%s' not exists", config.Name)
		respChan <- ResourceResult{Error:err}
		return err
	}
	//verify params
	if nil == net.ParseIP(config.Gateway){
		err = fmt.Errorf("invalid gateway '%s'", config.Gateway)
		respChan <- ResourceResult{Error:err}
		return err
	}
	for _, dns := range config.DNS{
		if nil == net.ParseIP(dns){
			err = fmt.Errorf("invalid DNS '%s'", dns)
			respChan <- ResourceResult{Error:err}
			return err
		}
	}
	pool.gateway = config.Gateway
	pool.dns = config.DNS
	manager.addressPools[pool.name] = pool
	//check affected cells
	var affected = make([]ComputeCellInfo, 0)
	for _, pool := range manager.pools{
		if pool.Network != config.Name{
			continue
		}
		for cellName, _ := range pool.Cells{
			cell, exists := manager.cells[cellName]
			if !exists{
				err = fmt.Errorf("invalid cell '%s' in pool '%s'", cellName, pool.Name)
				respChan <- ResourceResult{Error:err}
				return err
			}
			if !cell.Alive{
				continue
			}
			affected = append(affected, cell.ComputeCellInfo)
		}
	}
	log.Printf("<resource_manager> address pool '%s' modified with gateway '%s' and %d DNS server, %d cell(s) affected",
		pool.name, pool.gateway, len(pool.dns), len(affected))
	respChan <- ResourceResult{ComputeCellInfoList:affected}
	return manager.saveConfig()
}

func (manager *ResourceManager) handleDeleteAddressPool(poolName string, respChan chan error) (err error){
	pool, exists := manager.addressPools[poolName]
	if !exists{
		err = fmt.Errorf("address pool '%s' not exists", poolName)
		respChan <- err
		return err
	}
	//check attached compute pool
	for computeName, pool := range manager.pools{
		if pool.Network == poolName{
			err = fmt.Errorf("compute pool '%s' still attached to address pool '%s'", computeName, poolName)
			respChan <- err
			return err
		}
	}
	for _, addressRange := range pool.ranges{
		if 0 != len(addressRange.allocated){
			err = fmt.Errorf("%d address(es) of range '%s' allocated in pool '%s'",
				len(addressRange.allocated), addressRange.startAddress.String(), poolName)
			respChan <- err
			return
		}
	}
	delete(manager.addressPools, poolName)
	log.Printf("<resource_manager> address pool '%s' deleted", poolName)
	respChan <- nil
	return manager.saveConfig()
}

func (manager *ResourceManager) handleQueryAddressRange(poolName, rangeType string, respChan chan ResourceResult) (err error){
	pool, exists := manager.addressPools[poolName]
	if !exists{
		err = fmt.Errorf("address pool '%s' not exists", poolName)
		respChan <- ResourceResult{Error:err}
		return err
	}
	if rangeType != RangeTypeInternal{
		err = fmt.Errorf("unsupported range type '%s'", rangeType)
		respChan <- ResourceResult{Error:err}
		return err
	}
	var result ResourceResult
	result.AddressRangeList = make([]AddressRangeStatus, 0)
	for _, addressRange := range pool.ranges{
		var status AddressRangeStatus
		status.Start = addressRange.startAddress.String()
		status.End = addressRange.endAddress.String()
		status.Netmask = IPv4MaskToString(addressRange.netmask)
		status.Capacity = addressRange.capacity
		status.Allocated = make([]AllocatedAddress, 0)
		for address, instance := range addressRange.allocated{
			status.Allocated = append(status.Allocated, AllocatedAddress{address, instance})
		}
		result.AddressRangeList = append(result.AddressRangeList, status)
	}
	respChan <- result
	log.Printf("<resource_manager> %d range(s) in address pool '%s'", len(result.AddressRangeList), poolName)
	return nil
}

func (manager *ResourceManager) handleGetAddressRange(poolName, rangeType, startAddress string, respChan chan ResourceResult) (err error){
	pool, exists := manager.addressPools[poolName]
	if !exists{
		err = fmt.Errorf("address pool '%s' not exists", poolName)
		respChan <- ResourceResult{Error:err}
		return err
	}
	if rangeType != RangeTypeInternal{
		err = fmt.Errorf("unsupported range type '%s'", rangeType)
		respChan <- ResourceResult{Error:err}
		return err
	}
	addressRange, exists := pool.ranges[startAddress]
	if !exists{
		err = fmt.Errorf("range '%s' not exists in pool '%s'", startAddress, poolName)
		respChan <- ResourceResult{Error:err}
		return err
	}
	var status AddressRangeStatus
	status.Start = addressRange.startAddress.String()
	status.End = addressRange.endAddress.String()
	status.Netmask = IPv4MaskToString(addressRange.netmask)
	status.Capacity = addressRange.capacity
	status.Allocated = make([]AllocatedAddress, 0)
	for address, instance := range addressRange.allocated{
		status.Allocated = append(status.Allocated, AllocatedAddress{address, instance})
	}
	respChan <- ResourceResult{AddressRange: status}
	return nil
}

func (manager *ResourceManager) handleAddAddressRange(poolName, rangeType string, config AddressRangeConfig, respChan chan error) (err error){
	pool, exists := manager.addressPools[poolName]
	if !exists{
		err = fmt.Errorf("address pool '%s' not exists", poolName)
		respChan <- err
		return err
	}
	if rangeType != RangeTypeInternal{
		err = fmt.Errorf("unsupported range type '%s'", rangeType)
		respChan <- err
		return err
	}
	_, exists = pool.ranges[config.Start]
	if exists{
		err = fmt.Errorf("range '%s' already exists in pool '%s'", config.Start, poolName)
		respChan <- err
		return err
	}
	var addressRange ManagedIPV4AddressRange
	if addressRange.startAddress = net.ParseIP(config.Start); nil == addressRange.startAddress{
		err = fmt.Errorf("invalid start address '%s'", config.Start)
		respChan <- err
		return err
	}
	if addressRange.endAddress = net.ParseIP(config.End); nil == addressRange.endAddress{
		err = fmt.Errorf("invalid end address '%s'", config.End)
		respChan <- err
		return err
	}
	addressRange.netmask, err = IPv4ToMask(config.Netmask)
	if err != nil{
		respChan <- err
		return
	}
	//check end range
	{
		if bytes.Compare(addressRange.endAddress, addressRange.startAddress) < 0{
			err = fmt.Errorf("end address '%s' must greater than start address '%s'", config.End, config.Start)
			respChan <- err
			return err
		}
		var rangeNet = net.IPNet{addressRange.startAddress, addressRange.netmask}
		if !rangeNet.Contains(addressRange.endAddress){
			err = fmt.Errorf("end address '%s' not in net '%s/%s'",
				addressRange.endAddress.String(), addressRange.startAddress.String(), IPv4MaskToString(addressRange.netmask))
			respChan <- err
			return err
		}
		addressRange.capacity = IPv4ToNumber(addressRange.endAddress) - IPv4ToNumber(addressRange.startAddress) +1
	}
	//range conflict
	for startAddress, currentRange := range pool.ranges{
		if bytes.Compare(addressRange.endAddress, currentRange.startAddress) < 0 || bytes.Compare(addressRange.startAddress, currentRange.endAddress) > 0{
			continue
		}else{
			err = fmt.Errorf("address range '%s~%s' conflict with exists range '%s~%s'",
				config.Start, config.End, startAddress, currentRange.endAddress.String())
			respChan <- err
			return err
		}
	}
	addressRange.allocated = map[string]string{}
	pool.rangeStartAddressed = append(pool.rangeStartAddressed, config.Start)
	pool.ranges[config.Start] = addressRange
	manager.addressPools[poolName] = pool
	log.Printf("<resource_manager> range '%s~%s/%s' added to address pool '%s'", config.Start, config.End, config.Netmask, poolName)
	respChan <- nil
	return manager.saveConfig()
}

func (manager *ResourceManager) handleRemoveAddressRange(poolName, rangeType, startAddress string, respChan chan error) (err error){
	pool, exists := manager.addressPools[poolName]
	if !exists{
		err = fmt.Errorf("address pool '%s' not exists", poolName)
		respChan <- err
		return err
	}
	if rangeType != RangeTypeInternal{
		err = fmt.Errorf("unsupported range type '%s'", rangeType)
		respChan <- err
		return err
	}
	addressRange, exists := pool.ranges[startAddress]
	if !exists{
		err = fmt.Errorf("range '%s' not exists in pool '%s'", startAddress, poolName)
		respChan <- err
		return err
	}
	if 0 != len(addressRange.allocated){
		err = fmt.Errorf("%d address(es) of range '%s' allocated, release before delete",
			len(addressRange.allocated), addressRange.startAddress.String())
		respChan <- err
		return
	}
	for index, address := range pool.rangeStartAddressed{
		if address == startAddress{
			pool.rangeStartAddressed = append(pool.rangeStartAddressed[:index], pool.rangeStartAddressed[index+1:]...)
			break
		}
	}
	delete(pool.ranges, startAddress)
	manager.addressPools[poolName] = pool
	log.Printf("<resource_manager> range '%s' removed from address pool '%s'", startAddress, poolName)
	respChan <- nil
	return manager.saveConfig()
}

func (manager *ResourceManager) handleBeginResetSystem(instanceID string, respChan chan error) (err error){
	ins, exists := manager.instances[instanceID]
	if !exists{
		err = fmt.Errorf("invalid guest '%s'", instanceID)
		respChan <- err
		return
	}
	if !ins.Created{
		err = fmt.Errorf("guest '%s' not created yet", instanceID)
		respChan <- err
		return
	}
	if ins.Running{
		err = fmt.Errorf("guest '%s' is still running", instanceID)
		respChan <- err
		return
	}
	ins.Created = false
	ins.Progress = 0
	manager.instances[instanceID] = ins
	log.Printf("<resource_manager> begin reset system of guest '%s'", ins.Name)
	respChan <- nil
	return nil
}

func (manager *ResourceManager) handleFinishResetSystem(instanceID string, resetError error,  respChan chan error) (err error){
	ins, exists := manager.instances[instanceID]
	if !exists{
		err = fmt.Errorf("invalid guest '%s'", instanceID)
		respChan <- err
		return
	}
	if ins.Created{
		err = fmt.Errorf("guest '%s' already created", instanceID)
		respChan <- err
		return
	}
	if resetError != nil{
		err = resetError
		manager.pendingError[instanceID] = resetError
		log.Printf("<resource_manager> reset system fail: %s", err.Error())
	}else{
		ins.Created = true
		ins.Progress = 0
		manager.instances[instanceID] = ins
		log.Printf("<resource_manager> reset system of guest '%s' success", ins.Name)
	}
	respChan <- nil
	return nil
}

//batch
func (manager *ResourceManager) handleStartBatchCreateGuest(request BatchCreateRequest, respChan chan ResourceResult) (err error){
	if len(request.Prefix) == 0{
		err = errors.New("name prefix required")
		respChan <- ResourceResult{Error: err}
		return err
	}
	var r = regexp.MustCompile("[^\\w-]")
	if r.MatchString(request.Prefix){
		err = errors.New("only '0~9a~Z_-' allowed in prefix")
		respChan <- ResourceResult{Error: err}
		return err
	}
	if request.Count == 0{
		err = errors.New("guest count required")
		respChan <- ResourceResult{Error: err}
		return err
	}
	if _, exists := manager.pools[request.Pool]; !exists{
		err = fmt.Errorf("invalid compute pool '%s'", request.Pool)
		respChan <- ResourceResult{Error: err}
		return err
	}
	var newID = uuid.NewV4()
	var task BatchCreateGuestTask
	task.Finished = false
	task.StartTime = time.Now()
	task.LatestUpdate = time.Now()
	task.GuestName = map[string]int{}

	switch request.Rule {
	case NameRuleByOrder:
		for index := 0 ; index < request.Count ; index++{
			var guestName = fmt.Sprintf("%s_%d", request.Prefix, index)
			task.GuestName[guestName] = index
			var status = CreateGuestStatus{guestName, "", 0, BatchTaskStatusProcess, ""}
			task.Guests = append(task.Guests, status)
		}
	default:
		err = fmt.Errorf("unsupport name rule %d", request.Rule)
		respChan <- ResourceResult{Error: err}
		return err
	}
	var taskID = newID.String()
	manager.batchCreateTasks[taskID] = task
	respChan <- ResourceResult{Batch:taskID, BatchCreate:task.Guests}
	log.Printf("<resource_manager> new create guest batch allocated, task id '%s'", taskID)
	return nil
}

func (manager *ResourceManager) handleSetBatchCreateGuestStart(batchID, guestName, guestID string, respChan chan error) (err error){
	task, exists := manager.batchCreateTasks[batchID]
	if !exists{
		err = fmt.Errorf("invalid batch task '%s", batchID)
		respChan <- err
		return err
	}
	guestIndex, exists := task.GuestName[guestName]
	if !exists{
		err = fmt.Errorf("no guest named '%s' in batch '%s'", guestName, batchID)
		respChan <- err
		return err
	}
	if guestIndex > len(task.Guests){
		err = fmt.Errorf("invalid index %d with name '%s'", guestIndex,  guestName)
		respChan <- err
		return err
	}
	var guestStatus = task.Guests[guestIndex]
	guestStatus.ID = guestID
	task.Guests[guestIndex] = guestStatus
	task.LatestUpdate = time.Now()
	manager.batchCreateTasks[batchID] = task
	log.Printf("<resource_manager> guest '%s' created with id '%s' in batch '%s'", guestName, guestID, batchID)
	respChan <- nil
	return nil
}

func (manager *ResourceManager) handleSetBatchCreateGuestFail(batchID, guestName string, createError error, respChan chan error) (err error){
	task, exists := manager.batchCreateTasks[batchID]
	if !exists{
		err = fmt.Errorf("invalid batch task '%s", batchID)
		respChan <- err
		return err
	}
	guestIndex, exists := task.GuestName[guestName]
	if !exists{
		err = fmt.Errorf("no guest named '%s' in batch '%s'", guestName, batchID)
		respChan <- err
		return err
	}
	if guestIndex > len(task.Guests){
		err = fmt.Errorf("invalid index %d with name '%s'", guestIndex,  guestName)
		respChan <- err
		return err
	}
	var guestStatus = task.Guests[guestIndex]
	guestStatus.Error = createError.Error()
	guestStatus.Status = BatchTaskStatusFail
	task.Guests[guestIndex] = guestStatus
	task.LatestUpdate = time.Now()
	manager.batchCreateTasks[batchID] = task
	log.Printf("<resource_manager> batch create guest '%s' fail: %s", guestName, createError.Error())
	respChan <- nil
	return nil
}

func (manager *ResourceManager) handleGetBatchCreateGuestStatus(batchID string, respChan chan ResourceResult) (err error){
	task, exists := manager.batchCreateTasks[batchID]
	if !exists{
		err = fmt.Errorf("invalid batch task '%s", batchID)
		respChan <- ResourceResult{Error:err}
		return err
	}
	respChan <- ResourceResult{BatchCreate: task.Guests}
	return nil
}

func (manager *ResourceManager) handleStartBatchDeleteGuest(id []string, respChan chan ResourceResult) (err error){
	if 0 == len(id){
		err = errors.New("guest id required")
		respChan <- ResourceResult{Error:err}
		return err
	}
	var newID = uuid.NewV4()
	var task BatchDeleteGuestTask
	task.Finished = false
	task.StartTime = time.Now()
	task.LatestUpdate = time.Now()
	task.GuestID = map[string]int{}


	for taskIndex, guestID := range id{
		guest, exists := manager.instances[guestID]
		if !exists{
			err = fmt.Errorf("invalid guest '%s'", guestID)
			respChan <- ResourceResult{Error: err}
			return err
		}
		var status = DeleteGuestStatus{guest.Name, guestID, BatchTaskStatusProcess, ""}
		task.GuestID[guestID] = taskIndex
		task.Guests = append(task.Guests, status)
	}
	var taskID = newID.String()
	manager.batchDeleteTasks[taskID] = task
	respChan <- ResourceResult{Batch:taskID, BatchDelete:task.Guests}
	log.Printf("<resource_manager> new delete guest batch allocated, task id '%s'", taskID)
	return nil
}

func (manager *ResourceManager) handleSetBatchDeleteGuestSuccess(batchID, guestID string, respChan chan error) (err error){
	task, exists := manager.batchDeleteTasks[batchID]
	if !exists{
		err = fmt.Errorf("invalid batch task '%s", batchID)
		respChan <- err
		return err
	}
	guestIndex, exists := task.GuestID[guestID]
	if !exists{
		err = fmt.Errorf("no guest with id '%s' in batch '%s'", guestID, batchID)
		respChan <- err
		return err
	}
	if guestIndex > len(task.Guests){
		err = fmt.Errorf("invalid index %d with id '%s'", guestIndex,  guestID)
		respChan <- err
		return err
	}
	var guestStatus = task.Guests[guestIndex]
	guestStatus.Status = BatchTaskStatusSuccess
	task.Guests[guestIndex] = guestStatus
	task.LatestUpdate = time.Now()
	manager.batchDeleteTasks[batchID] = task
	log.Printf("<resource_manager> batch delete guest '%s' success", guestID)
	respChan <- nil
	return nil
}

func (manager *ResourceManager) handleSetBatchDeleteGuestFail(batchID, guestID string, deleteError error, respChan chan error) (err error){
	task, exists := manager.batchDeleteTasks[batchID]
	if !exists{
		err = fmt.Errorf("invalid batch task '%s", batchID)
		respChan <- err
		return err
	}
	guestIndex, exists := task.GuestID[guestID]
	if !exists{
		err = fmt.Errorf("no guest with id '%s' in batch '%s'", guestID, batchID)
		respChan <- err
		return err
	}
	if guestIndex > len(task.Guests){
		err = fmt.Errorf("invalid index %d with id '%s'", guestIndex,  guestID)
		respChan <- err
		return err
	}
	var guestStatus = task.Guests[guestIndex]
	guestStatus.Error = deleteError.Error()
	guestStatus.Status = BatchTaskStatusFail
	task.Guests[guestIndex] = guestStatus
	task.LatestUpdate = time.Now()
	manager.batchDeleteTasks[batchID] = task
	log.Printf("<resource_manager> batch delete guest '%s' fail: %s", guestID, deleteError.Error())
	respChan <- nil
	return nil
}

func (manager *ResourceManager) handleGetBatchDeleteGuestStatus(batchID string, respChan chan ResourceResult) (err error){
	task, exists := manager.batchDeleteTasks[batchID]
	if !exists{
		err = fmt.Errorf("invalid batch task '%s", batchID)
		respChan <- ResourceResult{Error:err}
		return err
	}
	respChan <- ResourceResult{BatchDelete: task.Guests}
	return nil
}

func (manager *ResourceManager) handleStartBatchStopGuest(id []string, respChan chan ResourceResult) (err error){
	if 0 == len(id){
		err = errors.New("guest id required")
		respChan <- ResourceResult{Error:err}
		return err
	}
	var newID = uuid.NewV4()
	var task BatchStopGuestTask
	task.Finished = false
	task.StartTime = time.Now()
	task.LatestUpdate = time.Now()
	task.GuestID = map[string]int{}


	for taskIndex, guestID := range id{
		guest, exists := manager.instances[guestID]
		if !exists{
			err = fmt.Errorf("invalid guest '%s'", guestID)
			respChan <- ResourceResult{Error: err}
			return err
		}
		var status = StopGuestStatus{guest.Name, guestID, BatchTaskStatusProcess, ""}
		task.GuestID[guestID] = taskIndex
		task.Guests = append(task.Guests, status)
	}
	var taskID = newID.String()
	manager.batchStopTasks[taskID] = task
	respChan <- ResourceResult{Batch:taskID, BatchStop:task.Guests}
	log.Printf("<resource_manager> new stop guest batch allocated, task id '%s'", taskID)
	return nil
}

func (manager *ResourceManager) handleSetBatchStopGuestSuccess(batchID, guestID string, respChan chan error) (err error){
	task, exists := manager.batchStopTasks[batchID]
	if !exists{
		err = fmt.Errorf("invalid batch task '%s", batchID)
		respChan <- err
		return err
	}
	guestIndex, exists := task.GuestID[guestID]
	if !exists{
		err = fmt.Errorf("no guest with id '%s' in batch '%s'", guestID, batchID)
		respChan <- err
		return err
	}
	if guestIndex > len(task.Guests){
		err = fmt.Errorf("invalid index %d with id '%s'", guestIndex,  guestID)
		respChan <- err
		return err
	}
	var guestStatus = task.Guests[guestIndex]
	guestStatus.Status = BatchTaskStatusSuccess
	task.Guests[guestIndex] = guestStatus
	task.LatestUpdate = time.Now()
	manager.batchStopTasks[batchID] = task
	log.Printf("<resource_manager> batch stop guest '%s' success", guestID)
	respChan <- nil
	return nil
}

func (manager *ResourceManager) handleSetBatchStopGuestFail(batchID, guestID string, deleteError error, respChan chan error) (err error){
	task, exists := manager.batchStopTasks[batchID]
	if !exists{
		err = fmt.Errorf("invalid batch task '%s", batchID)
		respChan <- err
		return err
	}
	guestIndex, exists := task.GuestID[guestID]
	if !exists{
		err = fmt.Errorf("no guest with id '%s' in batch '%s'", guestID, batchID)
		respChan <- err
		return err
	}
	if guestIndex > len(task.Guests){
		err = fmt.Errorf("invalid index %d with id '%s'", guestIndex,  guestID)
		respChan <- err
		return err
	}
	var guestStatus = task.Guests[guestIndex]
	guestStatus.Error = deleteError.Error()
	guestStatus.Status = BatchTaskStatusFail
	task.Guests[guestIndex] = guestStatus
	task.LatestUpdate = time.Now()
	manager.batchStopTasks[batchID] = task
	log.Printf("<resource_manager> batch stop guest '%s' fail: %s", guestID, deleteError.Error())
	respChan <- nil
	return nil
}

func (manager *ResourceManager) handleGetBatchStopGuestStatus(batchID string, respChan chan ResourceResult) (err error){
	task, exists := manager.batchStopTasks[batchID]
	if !exists{
		err = fmt.Errorf("invalid batch task '%s", batchID)
		respChan <- ResourceResult{Error:err}
		return err
	}
	respChan <- ResourceResult{BatchStop: task.Guests}
	return nil
}

func (manager *ResourceManager) handleUpdateInstanceMonitorSecret(instanceID, secret string, respChan chan error) (err error){
	defer func() {
		respChan <- err
	}()
	var instance InstanceStatus
	var exists bool
	if instance, exists = manager.instances[instanceID]; !exists{
		err = fmt.Errorf("invalid instance '%s'", instanceID)
		return
	}
	instance.MonitorSecret = secret
	manager.instances[instanceID] = instance
	log.Printf("<resource_manager> monitor secret of instance '%s' updated", instance.Name)
	return
}
func (manager *ResourceManager) handleQuerySystemTemplates(respChan chan ResourceResult) (err error){
	defer func() {
		if err != nil{
			respChan <- ResourceResult{Error: err}
		}
	}()
	var result = make([]SystemTemplate, 0)
	for _, id := range manager.allTemplateID{
		template, exists := manager.templates[id]
		if !exists{
			err = fmt.Errorf("invalid template '%s'", id)
			return
		}
		result = append(result, template)
	}
	respChan <- ResourceResult{TemplateList: result}
	return nil
}

func (manager *ResourceManager) handleGetSystemTemplate(id string, respChan chan ResourceResult) (err error){
	defer func() {
		if err != nil{
			respChan <- ResourceResult{Error: err}
		}
	}()
	template, exists := manager.templates[id]
	if !exists{
		err = fmt.Errorf("invalid template '%s'", id)
		return
	}
	respChan <- ResourceResult{Template: template}
	return nil
}

func (manager *ResourceManager) handleCreateSystemTemplate(config SystemTemplateConfig, respChan chan ResourceResult) (err error){
	defer func() {
		if err != nil{
			respChan <- ResourceResult{Error: err}
		}
	}()
	for _, t := range manager.templates{
		if t.Name == config.Name{
			err = fmt.Errorf("system template '%s' already exists", t.Name)
			return
		}
	}
	if _, err = config.ToOptions(); err != nil{
		err = fmt.Errorf("invalid template config: %s", err.Error())
		return
	}
	var template = CreateSystemTemplate(config)
	manager.templates[template.ID] = template
	manager.allTemplateID = append(manager.allTemplateID, template.ID)
	if err = manager.saveConfig(); err != nil{
		err = fmt.Errorf("add new template fail: %s", err.Error())
		return
	}
	respChan <- ResourceResult{Template: template}
	log.Printf("<resource_manager> new template '%s'(%s) created", template.Name, template.ID)
	return nil
}

func (manager *ResourceManager) handleModifySystemTemplate(id string, config SystemTemplateConfig, respChan chan error) (err error){
	defer func() {
		respChan <- err
	}()
	if _, err = config.ToOptions(); err != nil{
		err = fmt.Errorf("invalid template config: %s", err.Error())
		return
	}
	template, exists := manager.templates[id]
	if !exists{
		err = fmt.Errorf("invalid template '%s'", id)
		return
	}
	template.SystemTemplateConfig = config
	template.ModifiedTime = time.Now().Format(TimeFormatLayout)
	manager.templates[id] = template
	log.Printf("<resource_manager> template '%s' modified", template.Name)
	err = manager.saveConfig()
	return
}

func (manager *ResourceManager) handleDeleteSystemTemplate(id string, respChan chan error) (err error){
	defer func() {
		respChan <- err
	}()
	template, exists := manager.templates[id]
	if !exists{
		err = fmt.Errorf("invalid template '%s'", id)
		return
	}
	delete(manager.templates, id)
	var newArray []string
	for _, templateID := range manager.allTemplateID{
		if id != templateID{
			newArray = append(newArray, templateID)
		}
	}
	manager.allTemplateID = newArray
	log.Printf("<resource_manager> template '%s' deleted", template.Name)
	err = manager.saveConfig()
	return
}

func (manager *ResourceManager) transferInstances(sourceName, targetName string, instances []string, monitorPorts []uint64) (err error) {
	sourceCell, exists := manager.cells[sourceName]
	if !exists{
		err = fmt.Errorf("invalid source cell '%s'", sourceName)
		return err
	}
	targetCell, exists := manager.cells[targetName]
	if !exists{
		err = fmt.Errorf("invalid target cell '%s'", targetName)
		return err
	}
	if len(instances) != len(monitorPorts){
		err = fmt.Errorf("unmatched port count %d/%d", len(monitorPorts), len(instances))
		return err
	}
	for i, instanceID := range instances{
		var monitor = monitorPorts[i]
		instance, exists := manager.instances[instanceID]
		if !exists{
			err = fmt.Errorf("invalid migrating instance '%s'", instanceID)
			return err
		}
		if !instance.Migrating{
			err = fmt.Errorf("instance '%s' not in migrating", instance.Name)
			return err
		}
		instance.Migrating = false
		instance.Cell = targetName
		instance.InternalNetwork.MonitorPort = uint(monitor)
		instance.InternalNetwork.MonitorAddress = targetCell.Address
		manager.instances[instanceID] = instance
		delete(sourceCell.Instances, instanceID)
		targetCell.Instances[instanceID] = true
		log.Printf("<resource_manager> instance '%s' migrate from '%s' to '%s', monitor address change to %s:%d",
			instance.Name, sourceName, targetName,
			instance.InternalNetwork.MonitorAddress, instance.InternalNetwork.MonitorPort)
	}
	manager.cells[sourceName] = sourceCell
	manager.cells[targetName] = targetCell
	return nil
}

func (manager *ResourceManager) allocateNetworkAddress(pool ManagedComputePool, instanceID string) (internal, external string, err error){
	addresses, exists := manager.addressPools[pool.Network]
	if !exists{
		err = fmt.Errorf("invalid address pool '%s'", pool.Network)
		return
	}
	{
		//internal only
		for _, startAddress := range addresses.rangeStartAddressed{
			currentRange, exists := addresses.ranges[startAddress]
			if !exists{
				err = fmt.Errorf("invalid range '%s' in address pool '%s'", startAddress, pool.Network)
				return
			}
			if len(currentRange.allocated) == int(currentRange.capacity){
				log.Printf("<resource_manager> debug: ignore depleted range '%s' of address pool '%s'", startAddress, pool.Network)
				continue
			}
			var seekStart = IPv4ToNumber(currentRange.startAddress) + uint32(manager.generator.Intn(int(currentRange.capacity)))
			var seekEnd = IPv4ToNumber(currentRange.endAddress)
			var offset uint32 = 0
			for ; offset < currentRange.capacity; offset++{
				var selected = seekStart + offset
				if selected > seekEnd{
					selected -= currentRange.capacity
				}
				var ip = NumberToIPv4(selected)
				var ipString = ip.String()
				if _, exists := currentRange.allocated[ipString]; !exists{
					var internalNetwork = net.IPNet{ip, currentRange.netmask}
					internal = internalNetwork.String()
					currentRange.allocated[ipString] = instanceID
					addresses.ranges[startAddress] = currentRange
					manager.addressPools[pool.Network] = addresses
					log.Printf("<resource_manager> internal address '%s' allocated in range '%s~%s/%s'",
						internal, currentRange.startAddress.String(), currentRange.endAddress.String(), IPv4MaskToString(currentRange.netmask))
					manager.saveConfig()
					return
				}else{
					log.Printf("<resource_manager> debug: ignore allocated address '%s'", ipString)
				}
			}
		}
	}
	err = fmt.Errorf("no address available in address pool '%s'", pool.Network)
	return
}

func (manager *ResourceManager) deallocateNetworkAddress(pool ManagedComputePool, internalCIDR, external string) (err error){
	addresses, exists := manager.addressPools[pool.Network]
	if !exists{
		err = fmt.Errorf("invalid address pool '%s'", pool.Network)
		return
	}
	if "" != external{
		err = errors.New("external address not supported")
		return
	}
	if "" == internalCIDR {
		err = errors.New("internal address required")
		return
	}
	internalIP, _, err := net.ParseCIDR(internalCIDR)
	if err != nil{
		err = fmt.Errorf("invalid internal address '%s'", internalCIDR)
		return
	}
	var internalString = internalIP.String()
	for _, startAddress := range addresses.rangeStartAddressed{
		currentRange, exists := addresses.ranges[startAddress]
		if !exists{
			err = fmt.Errorf("invalid range '%s' in address pool '%s'", startAddress, pool.Network)
			return
		}
		instanceID, exists := currentRange.allocated[internalString]
		if !exists{
			log.Printf("<resource_manager> debug: internal address '%s' not in range '%s'", internalCIDR, startAddress)
			continue
		}
		log.Printf("<resource_manager> internal address '%s' deallocated for instance '%s'", internalCIDR, instanceID)
		delete(currentRange.allocated, internalString)
		addresses.ranges[startAddress] = currentRange
		manager.addressPools[pool.Network] = addresses
		return manager.saveConfig()
	}
	err = fmt.Errorf("'%s' not found in address pool '%s'", internalCIDR, pool.Network)
	return err
}


func (manager *ResourceManager) selectCell(poolName string, required InstanceResource, mustFulfill bool) (selected string, err error) {
	const (
		HealthCpuUsage float64 = 80
	)
	pool, exists := manager.pools[poolName]
	if !exists {
		return "", fmt.Errorf("invalid pool '%s'", poolName)
	}
	var selectedCapacity float32 = 0.0
	for cellName, _ := range pool.Cells {
		cell, exists := manager.cells[cellName]
		if !exists{
			err = fmt.Errorf("invalid cell '%s' in pool '%s'", cellName, poolName)
			return
		}
		if !cell.Alive{
			log.Printf("<resource_manager> debug: ignore offline cell '%s' when select resource node", cellName)
			continue
		}
		if !cell.Enabled{
			log.Printf("<resource_manager> debug: ignore disabled cell '%s' when select resource node", cellName)
			continue
		}
		var requiredDisk uint64 = 0
		for _, volSize := range required.Disks {
			requiredDisk += volSize
		}
		if mustFulfill{
			//check minimal resource
			if cell.CpuUsage > HealthCpuUsage{
				log.Printf("<resource_manager> debug: ignore cell '%s' due to cpu overload (%.2f%%)", cellName, cell.CpuUsage)
				continue
			}
			if cell.DiskAvailable < requiredDisk{
				log.Printf("<resource_manager> debug: ignore cell '%s' due to insuffient disk (%d GB for %d GB)",
					cellName, cell.DiskAvailable >> 30, requiredDisk >> 30)
				continue
			}
			if cell.MemoryAvailable < uint64(required.Memory){
				log.Printf("<resource_manager> debug: ignore cell '%s' due to insuffient memory (%d GB for %d GB)",
					cellName, cell.MemoryAvailable >> 30, required.Memory >> 30)
				continue
			}
		}
		var realLoad = manager.evaluateRealTimeCapacity(cell, required.Cores, required.Memory, requiredDisk)

		configureLoad, err := manager.evaluateConfigureCapacity(cell, required.Cores, required.Memory, requiredDisk)
		if err != nil{
			return "", err
		}
		var capacity = (realLoad + configureLoad) / 2
		//log.Printf("<resource_manager> debug: '%s' => %.2f", cellName, capacity)
		if capacity > selectedCapacity {
			selected = cellName
			selectedCapacity = capacity
		}
	}
	if 0.0 == selectedCapacity {
		return "", errors.New("no cell fulfill the resource requirement")
	}
	return selected, nil
}

func (manager *ResourceManager) evaluateRealTimeCapacity(cell ManagedComputeCell, requireCore, requireMemory uint, requireDisk uint64) (capacity float32){
	const (
		coreFactor   = 2
		memoryFactor = 1.5
		diskFactor   = 1
		fullCPUUsage = 100
	)

	var availableCores float32
	if cell.CpuUsage >= fullCPUUsage{
		availableCores = 0
	}else{
		availableCores = float32(cell.Cores) * float32(fullCPUUsage - cell.CpuUsage) / 100
	}

	var coreCapacity = availableCores / float32(requireCore)
	var memoryCapacity = float32(cell.MemoryAvailable) / float32(requireMemory)
	var diskCapacity = float32(cell.DiskAvailable / requireDisk)
	capacity = coreFactor * coreCapacity + memoryFactor * memoryCapacity + diskFactor * diskCapacity
	//log.Printf("<resource_manager> debug: real capacity %.2f, core %.2f / %d => %.2f, mem %d / %d => %.2f, disk %d / %d => %.2f",
	//	capacity, availableCores, requireCore, coreCapacity, cell.MemoryAvailable >> 20, requireMemory >> 20, memoryCapacity,
	//	cell.DiskAvailable  >> 30, requireDisk >> 30, diskCapacity)
	return capacity
}

func (manager *ResourceManager) evaluateConfigureCapacity(cell ManagedComputeCell, requireCore, requireMemory uint, requireDisk uint64) (capacity float32, err error){
	const (
		configureScale = 3
		coreFactor     = 2
		memoryFactor   = 1.5
		diskFactor     = 1
	)
	var idList []string
	for instanceID, _ := range cell.Instances{
		idList = append(idList, instanceID)
	}
	for instanceID, _ := range cell.Pending{
		idList = append(idList, instanceID)
	}
	var availableCores = cell.Cores * configureScale
	var availableMemory = cell.Memory * configureScale
	var availableDisk = cell.Disk * configureScale
	for _, instanceID := range idList{
		ins, exists := manager.instances[instanceID]
		if !exists{
			err = fmt.Errorf("invalid instance '%s' in cell '%s'", instanceID, cell.Name)
			return
		}
		availableCores -= ins.Cores
		availableMemory -= uint64(ins.Memory)
		for _, diskSize := range ins.Disks{
			availableDisk -= diskSize
		}
	}
	var coreCapacity = float32(availableCores/requireCore)
	var memoryCapacity = float32(availableMemory/uint64(requireMemory))
	var diskCapacity = float32(availableDisk/requireDisk)
	capacity = coreFactor * coreCapacity + memoryFactor * memoryCapacity + diskFactor * diskCapacity
	//log.Printf("<resource_manager> debug: configure capacity %.2f, core %d / %d => %.2f, mem %d / %d => %.2f, disk %d / %d => %.2f",
	//	capacity, availableCores, requireCore, coreCapacity, availableMemory >> 20, requireMemory >> 20, memoryCapacity,
	//		availableDisk >> 30, requireDisk >> 30, diskCapacity)
	return capacity, nil
}

func (manager *ResourceManager) saveConfig() error {
	var config ResourceData
	var totalPools, totalCells = 0, 0
	config.Zone = manager.zone.Name
	for poolName, poolStatus := range manager.pools {
		var pool = poolDefine{Name: poolName, Enabled: poolStatus.Enabled, Network:poolStatus.Network, Storage:poolStatus.Storage, Failover:poolStatus.Failover}
		pool.Cells = map[string]cellDefine{}
		totalPools++
		for cellName, _ := range poolStatus.Cells {
			if cellStatus, exists := manager.cells[cellName]; exists {
				var cell = cellDefine{Enabled: cellStatus.Enabled, PurgeAppending: cellStatus.PurgeAppending}
				pool.Cells[cellName] = cell
			} else {
				return fmt.Errorf("invalid cell '%s'", cellName)
			}
			totalCells++
		}
		config.Pools = append(config.Pools, pool)
	}
	//storage pools
	for poolName, pool := range manager.storagePools{
		var storage = storageDefine{poolName, pool.Type, pool.Host, pool.Target}
		config.StoragePools = append(config.StoragePools, storage)
	}
	config.AddressPools = make([]addressPoolDefine, 0)
	for poolName, pool := range manager.addressPools{
		var define addressPoolDefine
		define.Name = poolName
		define.Gateway = pool.gateway
		define.DNS = pool.dns
		define.Ranges = make([]AddressRangeStatus, 0)
		for _, startAddress := range pool.rangeStartAddressed{
			currentRange, exists := pool.ranges[startAddress]
			if !exists{
				return fmt.Errorf("invalid start address '%s' in pool '%s'", startAddress, poolName)
			}
			var status AddressRangeStatus
			status.Start = currentRange.startAddress.String()
			status.End = currentRange.endAddress.String()
			status.Netmask = IPv4MaskToString(currentRange.netmask)
			status.Capacity = currentRange.capacity
			status.Allocated = make([]AllocatedAddress, 0)
			for address, instance := range currentRange.allocated{
				status.Allocated = append(status.Allocated, AllocatedAddress{address, instance})
			}
			define.Ranges = append(define.Ranges, status)
		}
		config.AddressPools = append(config.AddressPools, define)
	}
	for _, template := range manager.templates{
		config.SystemTemplates = append(config.SystemTemplates, template)
	}
	data, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(manager.dataFile, data, DefaultConfigPerm); err != nil {
		return err
	}
	log.Printf("<resource_manager> %d pools, %d storages, %d address pool(s), %d cells saved to '%s'",
		totalPools, len(config.StoragePools), len(config.AddressPools),
		totalCells, manager.dataFile)
	return nil
}

func (manager *ResourceManager) generateDefaultTemplates() (templates []SystemTemplate, err error){
	templates = append(templates, CreateSystemTemplate(SystemTemplateConfig{
		Name:            "CentOS 7",
		Admin:           "root",
		OperatingSystem: SystemNameLinux,
		Disk:            DiskBusSCSI,
		Network:         NetworkModelVIRTIO,
		Display:         DisplayDriverVGA,
		Control:         RemoteControlVNC,
		USB:             USBModelXHCI,
		Tablet:          TabletBusUSB,
	}))
	templates = append(templates, CreateSystemTemplate(SystemTemplateConfig{
		Name:            "CentOS 6",
		Admin:           "root",
		OperatingSystem: SystemNameLinux,
		Disk:            DiskBusSATA,
		Network:         NetworkModelVIRTIO,
		Display:         DisplayDriverVGA,
		Control:         RemoteControlVNC,
		USB:             USBModelXHCI,
		Tablet:          TabletBusUSB,
	}))
	templates = append(templates, CreateSystemTemplate(SystemTemplateConfig{
		Name:            "Windows Server 2012",
		Admin:           "Administrator",
		OperatingSystem: SystemNameWindows,
		Disk:            DiskBusSATA,
		Network:         NetworkModelE1000,
		Display:         DisplayDriverVGA,
		Control:         RemoteControlVNC,
		USB:             USBModelXHCI,
		Tablet:          TabletBusUSB,
	}))
	templates = append(templates, CreateSystemTemplate(SystemTemplateConfig{
		Name:            "General",
		Admin:           "root",
		OperatingSystem: SystemNameLinux,
		Disk:            DiskBusSATA,
		Network:         NetworkModelRTL8139,
		Display:         DisplayDriverVGA,
		Control:         RemoteControlVNC,
		USB:             USBModelNone,
		Tablet:          TabletBusUSB,
	}))
	templates = append(templates, CreateSystemTemplate(SystemTemplateConfig{
		Name:            "Legacy",
		Admin:           "root",
		OperatingSystem: SystemNameLinux,
		Disk:            DiskBusIDE,
		Network:         NetworkModelRTL8139,
		Display:         DisplayDriverCirrus,
		Control:         RemoteControlVNC,
		USB:             USBModelNone,
		Tablet:          TabletBusNone,
	}))
	log.Printf("<resource_manager> %d default system template(s) generated", len(templates))
	return 
}

func (manager *ResourceManager) generateDefaultConfig() (err error){
	const (
		DefaultPoolName = "default"
		DefaultZoneName = "default"
	)
	manager.zone = ManagedZone{}
	manager.zone.Name = DefaultZoneName
	manager.cells = map[string]ManagedComputeCell{}
	var defaultPool = ManagedComputePool{}
	defaultPool.Name = DefaultPoolName
	defaultPool.Enabled = true
	defaultPool.Cells = map[string]bool{}
	defaultPool.InstanceNames = map[string]string{}
	manager.pools = map[string]ManagedComputePool{DefaultPoolName: defaultPool}
	var templates []SystemTemplate
	if templates, err = manager.generateDefaultTemplates(); err != nil{
		err = fmt.Errorf("genereate templates fail: %s", err.Error())
		return
	}
	for _, template := range templates{
		manager.templates[template.ID] = template
		manager.allTemplateID = append(manager.allTemplateID, template.ID)
	}
	log.Println("<resource_manager> default configure generated")
	return nil
}

func (manager *ResourceManager) loadConfig() (err error) {
	var configChanged = false
	defer func() {
		if configChanged{
			if err = manager.saveConfig(); err != nil{
				log.Printf("<resource_manager> save config fail after load: %s", err.Error())
				return
			}
		}
	}()
	if _, err = os.Stat(manager.dataFile); os.IsNotExist(err) {
		if err = manager.generateDefaultConfig(); err != nil{
			err = fmt.Errorf("generate default config fail: %s", err.Error())
			return
		}
		configChanged = true
		return
	}
	data, err := ioutil.ReadFile(manager.dataFile)
	if err != nil {
		return err
	}
	var config ResourceData
	if err = json.Unmarshal(data, &config); err != nil {
		return err
	}
	for _, poolDefine := range config.AddressPools{
		var pool ManagedAddressPool
		pool.name = poolDefine.Name
		pool.gateway = poolDefine.Gateway
		pool.dns = poolDefine.DNS
		pool.ranges = map[string]ManagedIPV4AddressRange{}
		pool.rangeStartAddressed = make([]string, 0)
		for _, rangeDefine := range poolDefine.Ranges{
			var status ManagedIPV4AddressRange
			if status.startAddress = net.ParseIP(rangeDefine.Start);nil == status.startAddress{
				return fmt.Errorf("invalid start address '%s' of pool '%s'", rangeDefine.Start, poolDefine.Name)
			}
			if status.endAddress = net.ParseIP(rangeDefine.End);nil == status.endAddress{
				return fmt.Errorf("invalid end address '%s' of pool '%s'", rangeDefine.End, poolDefine.Name)
			}
			status.netmask, err = IPv4ToMask(rangeDefine.Netmask)
			if err != nil{
				return err
			}
			status.capacity = rangeDefine.Capacity
			status.allocated = map[string]string{}
			for _, allocated := range rangeDefine.Allocated{
				status.allocated[allocated.Address] = allocated.Instance
			}
			pool.rangeStartAddressed = append(pool.rangeStartAddressed, rangeDefine.Start)
			pool.ranges[rangeDefine.Start] = status
		}
		manager.addressPools[pool.name] = pool
	}
	var totalCells, totalInstances = 0, 0
	for _, pool := range config.Pools {
		var poolStatus = ManagedComputePool{}
		poolStatus.Enabled = pool.Enabled
		poolStatus.Name = pool.Name
		poolStatus.Cells = map[string]bool{}
		poolStatus.InstanceNames = map[string]string{}
		for cellName, cell := range pool.Cells {
			totalInstances += len(cell.Instances)
			totalCells++
			var cellStatus = ManagedComputeCell{}
			cellStatus.Enabled = cell.Enabled
			cellStatus.PurgeAppending = cell.PurgeAppending
			cellStatus.Name = cellName
			cellStatus.Pool = pool.Name
			cellStatus.Instances = map[string]bool{}
			cellStatus.Pending = map[string]bool{}
			manager.cells[cellName] = cellStatus
			poolStatus.Cells[cellName] = true
		}
		poolStatus.Failover = pool.Failover
		poolStatus.CellCount = uint64(len(pool.Cells))
		poolStatus.Storage = pool.Storage
		poolStatus.Network = pool.Network
		manager.pools[pool.Name] = poolStatus
	}
	//storage pools
	for _, define := range config.StoragePools{
		var pool = StoragePoolInfo{define.Name, define.Type, define.Host, define.Target}
		manager.storagePools[pool.Name] = pool
	}
	var templates []SystemTemplate
	if 0 != len(config.SystemTemplates){
		templates = config.SystemTemplates
	}else{
		if templates, err = manager.generateDefaultTemplates(); err != nil{
			err = fmt.Errorf("generate templates fail: %s", err.Error())
			return
		}
		configChanged = true
	}
	for _, template := range templates{
		manager.templates[template.ID] = template
		manager.allTemplateID = append(manager.allTemplateID, template.ID)
	}

	manager.zone.Name = config.Zone
	log.Printf("<resource_manager> load resource success, %d compute/ %d storage/ %d address pools, %d templates, %d cell. %d instance available",
		len(manager.pools), len(manager.storagePools), len(manager.addressPools),
		len(manager.allTemplateID), len(manager.cells), totalInstances)

	return nil
}

func (s *ResourceUsage) Accumulate(add ResourceUsage) {
	var totalCores = s.Cores + add.Cores
	if 0 == totalCores{
		s.CpuUsage = 0.0
	}else{
		s.CpuUsage = (s.CpuUsage*float64(s.Cores) + add.CpuUsage*float64(add.Cores)) / float64(totalCores)
	}
	s.Cores = totalCores
	s.Memory += add.Memory
	s.MemoryAvailable += add.MemoryAvailable
	s.Disk += add.Disk
	s.DiskAvailable += add.DiskAvailable
	s.BytesWritten += add.BytesWritten
	s.BytesRead += add.BytesRead
	s.BytesSent += add.BytesSent
	s.BytesReceived += add.BytesReceived
	s.WriteSpeed += add.WriteSpeed
	s.ReadSpeed += add.ReadSpeed
	s.SendSpeed += add.SendSpeed
	s.ReceiveSpeed += add.ReceiveSpeed
}

func (s *ResourceUsage) Reset() {
	s.Cores = 0
	s.CpuUsage = 0
	s.Memory = 0
	s.MemoryAvailable = 0
	s.Disk = 0
	s.DiskAvailable = 0
	s.BytesWritten = 0
	s.BytesRead = 0
	s.BytesSent = 0
	s.BytesReceived = 0
	s.WriteSpeed = 0
	s.ReadSpeed = 0
	s.SendSpeed = 0
	s.ReceiveSpeed = 0
}

func (i *InstanceStatistic) Reset() {
	i.RunningInstances = 0
	i.StoppedInstances = 0
	i.LostInstances = 0
	i.MigratingInstances = 0
}

func (i *InstanceStatistic) Accumulate(v InstanceStatistic) {
	i.StoppedInstances += v.StoppedInstances
	i.RunningInstances += v.RunningInstances
	i.LostInstances += v.LostInstances
	i.MigratingInstances += v.MigratingInstances
}

func (c *CellStatistic) Reset() {
	c.OfflineCells = 0
	c.OnlineCells = 0
}

func (c *CellStatistic) Accumulate(v CellStatistic) {
	c.OfflineCells += v.OfflineCells
	c.OnlineCells += v.OnlineCells
}

func (p *PoolStatistic) Reset() {
	p.EnabledPools = 0
	p.DisabledPools = 0
}

func IPv4ToNumber(ip net.IP) (number uint32){
	number = binary.BigEndian.Uint32(ip[12:])
	return number
}

func NumberToIPv4(number uint32) (ip net.IP){
	var bytes = make([]byte, net.IPv4len)
	binary.BigEndian.PutUint32(bytes, number)
	ip = net.IPv4(bytes[0], bytes[1], bytes[2], bytes[3])
	return ip
}

func IPv4ToMask(stringValue string) (mask net.IPMask, err error){
	var ip = net.ParseIP(stringValue)
	if nil == ip {
		err = fmt.Errorf("invalid IP address '%s'", stringValue)
		return
	}
	var v4 = ip.To4()
	if nil == v4 {
		err = fmt.Errorf("invalid IPv4 address '%s'", stringValue)
		return
	}
	mask = net.IPv4Mask(v4[0], v4[1], v4[2], v4[3])
	return mask, nil
}

func IPv4MaskToString(mask net.IPMask) (string){
	return net.IPv4(mask[0], mask[1], mask[2], mask[3]).String()
}