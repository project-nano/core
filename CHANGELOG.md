# Change Log

## [1.3.0] - 2020-11-03

### Changed

- Optimize the strategy and error output of computing pool allocation
- Manage security policy of instance
- Manage security policy group

## [1.2.0] - 2020-04-12

### Added

- Query/Change storage path
- Manage system templates
- Create guest using template
- Reset monitor secret
- Signature on stream request for image server

### Changed

- Sync instance statistic on Cell when inconsistent

## [1.1.1] - 2020-01-01

### Added 

- Add DNS to API '/address_pools/'(query address pool list)
- Add CreateTime/MAC address to instance/guest

### Fixed

- Search guest in an empty cell return a proper result
- Properly return a pending error of instance when get status

## [1.1.0] - 2019-11-05

### Added 

- API signature verify
- Add go mod

### Changed

- Call core API via prefix '/api/v1/'
- Change "/media_image_files/:id" to "/media_images/:id/file/"
- Change "/disk_image_files/:id" to "/disk_images/:id/file/"

## [1.0.0] - 2019-07-02

### Added

- Set threshold of CPU/Disk IO/Network

- Batch stop guests

- Automatically synchronize the IP address in the TLS certificate when the IP changes

### Changed

- Generate module name base on listen address

- URL of guest operates change from '/guest/' to '/guests/'.

- Guests/Images match with owner or group

## [0.9.1] - 2019-05-16

### Added

- Modify media image

- Get media image

- Query media/disk image filter by owner and group

- Add new API "GET /media_image_search/" for filtering media images by owner and group

### Changed

- Refactor image server

- The image name is unique in a group

- Results of query disk image sorted by name

- Check image and disk size before clone guest

### Fixed

- Accumulate CPU usage to a null value

- Return empty data when querying zone status

- Use wrong CPU usage when computing real cell load

## [0.8.2] - 2019-04-04

### Fixed

- Media image locked when uploading interrupted

## [0.8.1] - 2019-02-15

### Added

- Modify guest name

- Batch creating/deleting guest

### Changed

- Adapt to new runnable implement

- Locate cert files of image server base on the binary path

## [0.7.1] - 2018-12-26

### Added

- System start time when query zone status

- Reset guest system

## [0.6.1] - 2018-11-30

### Added

- Address pool management: query/create/modify/delete

- Address range management: add/remove

- Instance address allocate/migrate

- Allocated address in instance status

## [0.5.1] - 2018-11-3

### Added

- Enable/Disable cell

- Enable failover in compute pool

- Migrate instance

### Changed

- Optimize load balance algorithm considering both real-time load and instances configured when choosing cell for allocation.

- Sort instance list by lexicographic order 

## [0.4.2] - 2018-10-10

### Fixed

- result.Error output message

## [0.4.1] - 2018-9-30

### Added

- Storage Pool management: Create/Delete/Modify

- NFS storage backend supported

- Allow choosing storage pool when creating/modifying compute pool

- Synchronize storage configure when cell joined or added

- Add storage mount status when getting cell status

- Check duplicate instance name in a pool when creating a guest

- Mark instance status to lost when cell disconnected

- Notify cell to detach storage when removed from pool

### Fixed

- Improper instance count when instance deallocated

- Task put a message to closed proxy channel causing panic

- Task put a message to deallocated proxy session causing panic

## [0.3.1] - 2018-8-25

### Added

- Insert/Eject media in instance

- Add instance create time

- Add create and modify time of images

- Snapshot management: create/restore/delete/query/get

### Fixed

- Wrong instance name sent to cell when create a new guest

## [0.2.3] - 2018-8-17

### Added

- Support initialize guest after created using Cloud-Init in NoCloudMode

- Enable guest system version/modules configure

- Enable change admin password/create new admin/auto resize&mount disk when ci module enabled(cloud-init cloud-utils required in guest)

- Qualify instance/user/group/image name (only '0~9a~Z-' allowed)

## [0.2.2] - 2018-8-6

### Modify

- Stable sorted result of instance/image/cell/pool list

## [0.2.1] - 2018-7-29

### Added

- Modify Cores/Memory/Disk Size

- Shrink guest volume

- Set/Get user password

- Add "system" property in guest

- Fixed: a newly uploaded disk image cannot use in cloning

## [0.1.3] - 2018-7-24

### Modified

- handle instance address changed event

- API redirect disk image requests

- ignore offline cells when compute score

- forward create disk image request when no target guest specified

- Fixed: instance internal and external address

## [0.1.2] - 2018-7-21

### Modified

- gracefully disconnect when module stop

- add version output on the console
