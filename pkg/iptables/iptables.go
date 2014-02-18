package iptables

import (
	"github.com/dotcloud/docker/utils"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"log"
)

type Action string

const (
	Add    Action = "-A"
	Delete Action = "-D"
)

var (
	ErrIp6tablesNAT	    = errors.New("Kernel version does not support IPv6 NAT")
	ErrIptablesNotFound = errors.New("Iptables not found")
	nat                 = []string{"-t", "nat"}
)

type Chain struct {
	Name    string
	Bridge  string
	IPv6	bool
}

func NewChain(name, bridge string) (*Chain, error) {
	if output, err := Raw("-t", "nat", "-N", name); err != nil {
		return nil, err
	} else if len(output) != 0 {
		return nil, fmt.Errorf("Error creating new iptables chain: %s", output)
	}
	chain := &Chain{
		Name:   name,
		Bridge: bridge,
		IPv6:	false,
	}

	if err := chain.Prerouting(Add, "-m", "addrtype", "--dst-type", "LOCAL"); err != nil {
		return nil, fmt.Errorf("Failed to inject docker in PREROUTING chain: %s", err)
	}
	if err := chain.Output(Add, "-m", "addrtype", "--dst-type", "LOCAL", "!", "--dst", "127.0.0.0/8"); err != nil {
		return nil, fmt.Errorf("Failed to inject docker in OUTPUT chain: %s", err)
	}
	return chain, nil
}

func NewChain6(name, bridge string) (*Chain, error) {
	if output, err := Raw6("-t", "nat", "-N", name); err != nil {
		return nil, err
	} else if len(output) != 0 {
		return nil, fmt.Errorf("Error creating new ip6tables chain: %s", output)
	}
	chain := &Chain{
		Name:   name,
		Bridge: bridge,
		IPv6:	true,
	}
	if err := chain.Prerouting(Add, "-m", "addrtype", "--dst-type", "LOCAL"); err != nil {
		return nil, fmt.Errorf("Failed to inject docker in PREROUTING chain: %s", err)
	}
	if err := chain.Output(Add, "-m", "addrtype", "--dst-type", "LOCAL", "!", "--dst", "::1/128"); err != nil {
		return nil, fmt.Errorf("Failed to inject docker in OUTPUT chain: %s", err)
	}
	return chain, nil
}

func RemoveExistingChain(name string) error {
	chain := &Chain{
		Name: name,
		IPv6: false,
	}
	return chain.Remove()
}

func RemoveExistingChain6(name string) error {
	chain := &Chain{
		Name: name,
		IPv6: true,
	}
	return chain.Remove()
}

func (c *Chain) Forward(action Action, ip net.IP, port int, proto, dest_addr string, dest_port int) error {
	daddr := ip.String()
	if ip.IsUnspecified() {
		// iptables interprets "0.0.0.0" as "0.0.0.0/32", whereas we
		// want "0.0.0.0/0". "0/0" is correctly interpreted as "any
		// value" by both iptables and ip6tables.
		daddr = "0/0"
	}
	nat_args := []string{
		"-t",
		"nat",
		fmt.Sprint(action),
		c.Name,
		"-p", proto,
		"-d", daddr,
		"--dport", strconv.Itoa(port),
		"!", "-i", c.Bridge,
		"-j", "DNAT",
		"--to-destination", net.JoinHostPort(dest_addr, strconv.Itoa(dest_port)),
	}
	if !c.IPv6 {
		if output, err := Raw(nat_args...); err != nil {
			return err
		} else if len(output) != 0 {
			return fmt.Errorf("Error iptables forward: %s", output)
		}
	} else {
		if output, err := Raw6(nat_args...); err != nil {
			return err
		} else if len(output) != 0 {
			return fmt.Errorf("Error ip6tables forward: %s", output)
		}
	}

	fAction := action
	if fAction == Add {
		fAction = "-I"
	}
	fwd_args := []string{
		string(fAction),
		"FORWARD",
		"!", "-i", c.Bridge,
		"-o", c.Bridge,
		"-p", proto,
		"-d", dest_addr,
		"--dport", strconv.Itoa(dest_port),
		"-j", "ACCEPT",
	}
	if !c.IPv6 {
		if output, err := Raw(fwd_args...); err != nil {
			return err
		} else if len(output) != 0 {
			return fmt.Errorf("Error iptables forward: %s", output)
		}
	} else {
		if output, err := Raw6(fwd_args...); err != nil {
			return err
		} else if len(output) != 0 {
			return fmt.Errorf("Error ip6tables forward: %s", output)
		}
	}

	return nil
}

func (c *Chain) Prerouting(action Action, args ...string) error {
	a := append(nat, fmt.Sprint(action), "PREROUTING")
	if len(args) > 0 {
		a = append(a, args...)
	}
	if !c.IPv6 {
		if output, err := Raw(append(a, "-j", c.Name)...); err != nil {
			return err
		} else if len(output) != 0 {
			return fmt.Errorf("Error iptables prerouting: %s", output)
		}
	} else {
		if output, err := Raw6(append(a, "-j", c.Name)...); err != nil {
			return err
		} else if len(output) != 0 {
			return fmt.Errorf("Error ip6tables prerouting: %s", output)
		}
	}
	return nil
}

func (c *Chain) Output(action Action, args ...string) error {
	a := append(nat, fmt.Sprint(action), "OUTPUT")
	if len(args) > 0 {
		a = append(a, args...)
	}
	if !c.IPv6 {
		if output, err := Raw(append(a, "-j", c.Name)...); err != nil {
			return err
		} else if len(output) != 0 {
			return fmt.Errorf("Error iptables output: %s", output)
		}
	} else {
		if output, err := Raw6(append(a, "-j", c.Name)...); err != nil {
			return err
		} else if len(output) != 0 {
			return fmt.Errorf("Error ip6tables output: %s", output)
		}
	}
	return nil
}

func (c *Chain) Remove() error {
	// Ignore errors - This could mean the chains were never set up
	c.Prerouting(Delete, "-m", "addrtype", "--dst-type", "LOCAL")
	if !c.IPv6 {
		c.Output(Delete, "-m", "addrtype", "--dst-type", "LOCAL", "!", "--dst", "127.0.0.0/8")
	} else {
		c.Output(Delete, "-m", "addrtype", "--dst-type", "LOCAL", "!", "--dst", "::1/128")
	}
	c.Output(Delete, "-m", "addrtype", "--dst-type", "LOCAL") // Created in versions <= 0.1.6

	c.Prerouting(Delete)
	c.Output(Delete)

	if !c.IPv6 {
		Raw("-t", "nat", "-F", c.Name)
		Raw("-t", "nat", "-X", c.Name)
	} else {
		Raw6("-t", "nat", "-F", c.Name)
		Raw6("-t", "nat", "-X", c.Name)
	}

	return nil
}

// Check if an existing rule exists
func Exists(args ...string) bool {
	if _, err := Raw(append([]string{"-C"}, args...)...); err != nil {
		return false
	}
	return true
}

func Exists6(args ...string) bool {
	if _, err := Raw6(append([]string{"-C"}, args...)...); err != nil {
		return false
	}
	return true
}

func Raw(args ...string) ([]byte, error) {
	path, err := exec.LookPath("iptables")
	if err != nil {
		return nil, ErrIptablesNotFound
	}
	if os.Getenv("DEBUG") != "" {
		fmt.Printf("[DEBUG] [iptables]: %s, %v\n", path, args)
	}
	output, err := exec.Command(path, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("iptables failed: iptables %v: %s (%s)", strings.Join(args, " "), output, err)
	}
	return output, err
}

func Raw6(args ...string) ([]byte, error) {
	path, err := exec.LookPath("ip6tables")
	if err != nil {
		return nil, ErrIptablesNotFound
	}
	if SupportsIPv6NAT() {
		if os.Getenv("DEBUG") != "" {
			fmt.Printf("[DEBUG] [ip6tables]: %s, %v\n", path, args)
		}
		output, err := exec.Command(path, args...).CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("ip6tables failed: ip6tables %v: %s (%s)", strings.Join(args, " "), output, err)
		}
		return output, err
	} else {
		log.Println("WARNING: Linux kernel version is too old to support IPv6 NAT. Please upgrade your kernel to atleast 3.7.0.")
		return nil, nil
	}
}

func SupportsIPv6NAT() bool {
	version, err := utils.GetKernelVersion()
	if err != nil {
		log.Printf("WARNING: %s\n", err)
		return false
	}

	if utils.CompareKernelVersion(version, &utils.KernelVersionInfo{Kernel: 3, Major: 7, Minor: 0}) < 0 {
		return false
	}
	return true
}
