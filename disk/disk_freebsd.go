// +build freebsd

package disk

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"syscall"
	"unsafe"

	common "github.com/immerselearning/gopsutil/common"
)

const (
	CTLKern        = 1
	KernDevstat    = 773
	KernDevstatAll = 772
)

func DiskPartitions(all bool) ([]DiskPartitionStat, error) {
	var ret []DiskPartitionStat

	// get length
	count, err := syscall.Getfsstat(nil, MNT_WAIT)
	if err != nil {
		return ret, err
	}

	fs := make([]Statfs, count)
	_, err = Getfsstat(fs, MNT_WAIT)

	for _, stat := range fs {
		opts := "rw"
		if stat.Flags&MNT_RDONLY != 0 {
			opts = "ro"
		}
		if stat.Flags&MNT_SYNCHRONOUS != 0 {
			opts += ",sync"
		}
		if stat.Flags&MNT_NOEXEC != 0 {
			opts += ",noexec"
		}
		if stat.Flags&MNT_NOSUID != 0 {
			opts += ",nosuid"
		}
		if stat.Flags&MNT_UNION != 0 {
			opts += ",union"
		}
		if stat.Flags&MNT_ASYNC != 0 {
			opts += ",async"
		}
		if stat.Flags&MNT_SUIDDIR != 0 {
			opts += ",suiddir"
		}
		if stat.Flags&MNT_SOFTDEP != 0 {
			opts += ",softdep"
		}
		if stat.Flags&MNT_NOSYMFOLLOW != 0 {
			opts += ",nosymfollow"
		}
		if stat.Flags&MNT_GJOURNAL != 0 {
			opts += ",gjounalc"
		}
		if stat.Flags&MNT_MULTILABEL != 0 {
			opts += ",multilabel"
		}
		if stat.Flags&MNT_ACLS != 0 {
			opts += ",acls"
		}
		if stat.Flags&MNT_NOATIME != 0 {
			opts += ",noattime"
		}
		if stat.Flags&MNT_NOCLUSTERR != 0 {
			opts += ",nocluster"
		}
		if stat.Flags&MNT_NOCLUSTERW != 0 {
			opts += ",noclusterw"
		}
		if stat.Flags&MNT_NFS4ACLS != 0 {
			opts += ",nfs4acls"
		}

		d := DiskPartitionStat{
			Device:     common.IntToString(stat.Mntfromname[:]),
			Mountpoint: common.IntToString(stat.Mntonname[:]),
			Fstype:     common.IntToString(stat.Fstypename[:]),
			Opts:       opts,
		}
		ret = append(ret, d)
	}

	return ret, nil
}

func DiskIOCounters() (map[string]DiskIOCountersStat, error) {
	// statinfo->devinfo->devstat
	// /usr/include/devinfo.h

	//	sysctl.sysctl ('kern.devstat.all', 0)
	ret := make(map[string]DiskIOCountersStat)
	mib := []int32{CTLKern, KernDevstat, KernDevstatAll}

	buf, length, err := common.CallSyscall(mib)
	if err != nil {
		return nil, err
	}

	ds := Devstat{}
	devstatLen := int(unsafe.Sizeof(ds))
	count := int(length / uint64(devstatLen))

	buf = buf[8:] // devstat.all has version in the head.
	// parse buf to Devstat
	for i := 0; i < count; i++ {
		b := buf[i*devstatLen : i*devstatLen+devstatLen]
		d, err := parseDevstat(b)
		if err != nil {
			continue
		}
		un := strconv.Itoa(int(d.Unit_number))
		name := common.IntToString(d.Device_name[:]) + un

		ds := DiskIOCountersStat{
			ReadCount:  d.Operations[DEVSTAT_READ],
			WriteCount: d.Operations[DEVSTAT_WRITE],
			ReadBytes:  d.Bytes[DEVSTAT_READ],
			WriteBytes: d.Bytes[DEVSTAT_WRITE],
			ReadTime:   d.Duration[DEVSTAT_READ].Compute(),
			WriteTime:  d.Duration[DEVSTAT_WRITE].Compute(),
			Name:       name,
		}
		ret[name] = ds
	}

	return ret, nil
}

func (b Bintime) Compute() uint64 {
	BINTIME_SCALE := 5.42101086242752217003726400434970855712890625e-20
	return uint64(b.Sec) + b.Frac*uint64(BINTIME_SCALE)
}

// BT2LD(time)     ((long double)(time).sec + (time).frac * BINTIME_SCALE)

// Getfsstat is borrowed from pkg/syscall/syscall_freebsd.go
// change Statfs_t to Statfs in order to get more information
func Getfsstat(buf []Statfs, flags int) (n int, err error) {
	var _p0 unsafe.Pointer
	var bufsize uintptr
	if len(buf) > 0 {
		_p0 = unsafe.Pointer(&buf[0])
		bufsize = unsafe.Sizeof(Statfs{}) * uintptr(len(buf))
	}
	r0, _, e1 := syscall.Syscall(syscall.SYS_GETFSSTAT, uintptr(_p0), bufsize, uintptr(flags))
	n = int(r0)
	if e1 != 0 {
		err = e1
	}
	return
}

func parseDevstat(buf []byte) (Devstat, error) {
	var ds Devstat
	br := bytes.NewReader(buf)
	//	err := binary.Read(br, binary.LittleEndian, &ds)
	err := Read(br, binary.LittleEndian, &ds)
	if err != nil {
		return ds, err
	}

	return ds, nil
}
