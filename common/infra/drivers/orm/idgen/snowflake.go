package idgen

import (
	"fmt"
	"hash/fnv"
	"log"
	"net"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/sony/sonyflake"

	"github.com/wfusion/gofusion/common/utils"
)

var (
	// NewSnowflakeType FIXME: should not be deleted to avoid compiler optimized
	NewSnowflakeType = reflect.TypeOf(NewSnowflake)

	snowflakeOnce     sync.Once
	snowflakeInstance *snowflake
)

type snowflake struct {
	instance *sonyflake.Sonyflake
}

// NewSnowflake it should be only one snowflake generator per service instance
func NewSnowflake() Generator {
	snowflakeOnce.Do(func() {
		flake := sonyflake.NewSonyflake(sonyflake.Settings{
			StartTime: time.Time{},
			// machine id: hash(host ip + local ip + pid)(8 bit) - local ip(8 bit)
			MachineID: func() (id uint16, err error) {
				pid := os.Getpid()
				hostIP := utils.HostIPInDocker()
				localIP := utils.ClientIP()
				log.Printf("[Common] snowflake get machine id base [host[%s] local[%s] pid[%v]]", hostIP, localIP, pid)
				if hostIP == "" {
					hostIP = utils.ClientIP()
				}
				hash := fnv.New32a()
				_, err = hash.Write([]byte(fmt.Sprintf("%s%s%v", hostIP, localIP, pid)))
				if err != nil {
					return
				}

				high := byte(hash.Sum32() % 255)
				low := net.ParseIP(localIP).To4()[3]
				id = uint16(high)<<8 | uint16(low)
				log.Printf("[Common] snowflake get machine id [%X]", id)
				return
			},
			CheckMachineID: nil,
		})
		if flake == nil {
			panic(ErrNewGenerator)
		}
		snowflakeInstance = &snowflake{instance: flake}
	})

	return snowflakeInstance
}

func (s *snowflake) Next(opts ...utils.OptionExtender) (id uint64, err error) {
	return s.instance.NextID()
}
