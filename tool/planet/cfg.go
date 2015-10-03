package main

import (
	"fmt"
	"net"

	"github.com/gravitational/planet/lib/box"

	"github.com/gravitational/planet/Godeps/_workspace/src/github.com/gravitational/orbit/lib/utils"
	"github.com/gravitational/planet/Godeps/_workspace/src/gopkg.in/alecthomas/kingpin.v2"
)

type Config struct {
	Roles              list
	InsecureRegistries list
	Rootfs             string
	MasterIP           string
	CloudProvider      string
	ClusterID          string
	Env                box.EnvVars
	Mounts             box.Mounts
	Files              []box.File
	IgnoreChecks       bool
	StateDir           string
	ServiceSubnet      CIDR
	PODSubnet          CIDR
}

func (cfg *Config) hasRole(r string) bool {
	for _, rs := range cfg.Roles {
		if rs == r {
			return true
		}
	}
	return false
}

type list []string

func (l *list) Set(val string) error {
	for _, r := range utils.SplitComma(val) {
		*l = append(*l, r)
	}
	return nil
}

func (l *list) String() string {
	return fmt.Sprintf("%v", []string(*l))
}

func CIDRFlag(s kingpin.Settings) *CIDR {
	vars := new(CIDR)
	s.SetValue(vars)
	return vars
}

type CIDR struct {
	val   string
	ip    net.IP
	ipnet net.IPNet
}

func (c *CIDR) Set(v string) error {
	ip, ipnet, err := net.ParseCIDR(v)
	if err != nil {
		return err
	}
	c.val = v
	c.ip = ip
	c.ipnet = *ipnet
	return nil
}

func (c *CIDR) String() string {
	return c.ipnet.String()
}

// FirstIP returns the first IP in this subnet that is not .0
func (c *CIDR) FirstIP() net.IP {
	var ip net.IP
	for ip = IncIP(c.ip.Mask(c.ipnet.Mask)); c.ipnet.Contains(ip); IncIP(ip) {
		break
	}
	return ip
}

func IncIP(ip net.IP) net.IP {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
	return ip
}
