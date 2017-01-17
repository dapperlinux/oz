package network

import (
	//Builtin
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	//Internal

	//External
	"github.com/j-keck/arping"
	"github.com/milosgajdos83/tenus"
	"github.com/op/go-logging"
)

func BridgeInit(bridgeMAC string, nmIgnoreFile string, log *logging.Logger) (*HostNetwork, error) {
	if os.Getpid() == 1 {
		panic(errors.New("Cannot use BridgeInit from child."))
	}

	htn := &HostNetwork{
		BridgeMAC: bridgeMAC,
		log:       log,
	}

	if nmIgnoreFile != "" {
		if _, err := os.Stat(nmIgnoreFile); os.IsNotExist(err) {
			log.Warning("Warning! Network Manager may not properly configured to ignore the bridge interface! This may result in management conflicts!")
		}
	}

	br, err := tenus.BridgeFromName(ozDefaultInterfaceBridge)
	if err != nil {
		log.Info("Bridge not found, attempting to create a new one")

		br, err = tenus.NewBridgeWithName(ozDefaultInterfaceBridge)
		if err != nil {
			return nil, fmt.Errorf("Unable to create bridge %+v", err)
		}
	}
	htn.Interface = br

	if err := htn.configureBridgeInterface(); err != nil {
		return nil, err
	}

	brL := br.NetInterface()
	addrs, err := brL.Addrs()
	if err != nil {
		return nil, fmt.Errorf("Unable to get bridge interface addresses: %+v", err)
	}

	// Build the ip range which we will use for the network
	if err := htn.buildBridgeNetwork(addrs); err != nil {
		return nil, err
	}

	return htn, nil
}

func NewHostNetwork(name string, log *logging.Logger) *HostNetwork {
	return &HostNetwork{Name: name, log: log}
}

func (hn *HostNetwork) bridgeName() string {
	return ozDefaultInterfaceBridgeBase + hn.Name
}

func (hn *HostNetwork) Initialize() error {
	if os.Getpid() == 1 {
		panic(errors.New("Cannot initialize bridges from child."))
	}
	hn.log.Info("Initializing bridge: %s", hn.bridgeName())

	br, err := tenus.BridgeFromName(hn.bridgeName())
	if err != nil {
		hn.log.Info("Bridge not found, attempting to create a new one")

		br, err = tenus.NewBridgeWithName(hn.bridgeName())
		if err != nil {
			return fmt.Errorf("Unable to create bridge %+v", err)
		}
	}

	hn.Interface = br

	if err := hn.configureBridgeInterface(); err != nil {
		return err
	}

	brL := br.NetInterface()
	addrs, err := brL.Addrs()
	if err != nil {
		return fmt.Errorf("Unable to get bridge interface addresses: %+v", err)
	}

	// Build the ip range which we will use for the network
	if err := hn.buildBridgeNetwork(addrs); err != nil {
		return err
	}

	return nil
}

func (htn *HostNetwork) BridgeReconfigure() (*HostNetwork, error) {
	if os.Getpid() == 1 {
		panic(errors.New("Cannot use BridgeReconfigure from child."))
	}

	newhtn := &HostNetwork{
		BridgeMAC: htn.BridgeMAC,
	}

	if err := htn.Interface.UnsetLinkIp(htn.Gateway, htn.GatewayNet); err != nil {
		htn.log.Warning("Unable to remove old IP address from bridge: %+v", err)
	}

	br, err := tenus.BridgeFromName(htn.bridgeName())
	if err != nil {
		return nil, err
	}
	newhtn.Interface = br

	if err := newhtn.configureBridgeInterface(); err != nil {
		return nil, err
	}

	brL := newhtn.Interface.NetInterface()
	addrs, err := brL.Addrs()
	if err != nil {
		return nil, fmt.Errorf("Unable to get bridge interface addresses: %+v", err)
	}

	// Build the ip range which we will use for the network
	if err := newhtn.buildBridgeNetwork(addrs); err != nil {
		return nil, err
	}

	return newhtn, nil
}

func PrepareSandboxNetwork(stn *SandboxNetwork, htn *HostNetwork, staticByte uint, log *logging.Logger) (*SandboxNetwork, error) {
	if stn == nil {
		stn = new(SandboxNetwork)
		stn.VethHost = tenus.MakeNetInterfaceName(ozDefaultInterfacePrefix)
		stn.VethGuest = stn.VethHost + "1"
	}

	stn.Gateway = htn.Gateway
	stn.Class = htn.Class

	if staticByte > 1 {
		stn.Ip = inet_ntoa(htn.Min - 2 + uint64(staticByte)).String()
	} else {
		// Allocate a new IP address
		stn.Ip = getFreshIP(htn.Min, htn.Max, htn.IpBytes, log)
		if stn.Ip == "" {
			return nil, errors.New("Unable to acquire random IP")
		}
	}

	return stn, nil
}

func NetInit(stn *SandboxNetwork, htn *HostNetwork, log *logging.Logger) error {
	if os.Getpid() == 1 {
		panic(errors.New("Cannot use NetInit from child."))
	}

	// Seed random number generator (poorly but we're not doing crypto)
	rand.Seed(time.Now().Unix() ^ int64(os.Getpid()))

	log.Info("Configuring host veth pair '%s' with: %s", stn.VethHost, stn.Ip+"/"+htn.Class)

	// Create the veth pair
	veth, err := tenus.NewVethPairWithOptions(stn.VethHost, tenus.VethOptions{PeerName: stn.VethGuest})
	if err != nil {
		return fmt.Errorf("Unable to create veth pair %s, %s.", stn.VethHost, err)
	}

	// Fetch the newly created hostside veth
	vethIf, err := net.InterfaceByName(stn.VethHost)
	if err != nil {
		return fmt.Errorf("Unable to fetch veth pair %s, %s.", stn.VethHost, err)
	}

	// Add the host side veth to the bridge
	if err := htn.Interface.AddSlaveIfc(vethIf); err != nil {
		return fmt.Errorf("Unable to add veth pair %s to bridge, %s.", stn.VethHost, err)
	}

	// Bring the host side veth interface up
	if err := veth.SetLinkUp(); err != nil {
		return fmt.Errorf("Unable to bring veth pair %s up, %s.", stn.VethHost, err)
	}

	stn.Veth = veth

	return nil

}

func NetAttach(stn *SandboxNetwork, htn *HostNetwork, childPid int) error {
	if os.Getpid() == 1 {
		panic(errors.New("Cannot use NetAttach from child."))
	}

	// Assign the veth path to the namespace
	if err := stn.Veth.SetPeerLinkNsPid(childPid); err != nil {
		return fmt.Errorf("Unable to add veth pair %s to namespace, %s.", stn.VethHost, err)
	}

	// Parse the ip/class into the the appropriate formats
	vethGuestIp, vethGuestIpNet, err := net.ParseCIDR(stn.Ip + "/" + htn.Class)
	if err != nil {
		return fmt.Errorf("Unable to parse ip %s, %s.", stn.Ip, err)
	}

	// Set interface address in the namespace
	if err := stn.Veth.SetPeerLinkNetInNs(childPid, vethGuestIp, vethGuestIpNet, &htn.Gateway); err != nil {
		return fmt.Errorf("Unable to parse ip link in namespace, %s.", err)
	}

	return nil
}

func NetReconfigure(stn *SandboxNetwork, htn *HostNetwork, childPid int, log *logging.Logger) error {
	if os.Getpid() == 1 {
		panic(errors.New("Cannot use NetReconfigure from child."))
	}
	tenus.DeleteLink(stn.VethHost)
	if err := NetInit(stn, htn, log); err != nil {
		return err
	}
	return NetAttach(stn, htn, childPid)
}

func (stn *SandboxNetwork) Cleanup(log *logging.Logger) {
	if os.Getpid() == 1 {
		panic(errors.New("Cannot use Cleanup from child."))
	}

	if _, err := net.InterfaceByName(stn.VethHost); err != nil {
		log.Info("No veth found to cleanup")
		return
	}

	tenus.DeleteLink(stn.VethHost)
}

func (htn *HostNetwork) configureBridgeInterface() error {
	// Set the bridge mac address so it can be fucking ignored by Network-Manager.
	if htn.BridgeMAC != "" {
		if err := htn.Interface.SetLinkMacAddress(htn.BridgeMAC); err != nil {
			return fmt.Errorf("Unable to set MAC address for gateway", err)
		}
	}

	if htn.Gateway == nil {
		// Lookup an empty ip range
		brIp, brIpNet, err := findEmptyRange()
		if err != nil {
			return fmt.Errorf("Could not find an ip range to assign to the bridge")
		}
		htn.Gateway = brIp
		htn.GatewayNet = brIpNet
		htn.Broadcast = net_getbroadcast(brIp, brIpNet.Mask)
		htn.log.Info("Found available range: %+v", htn.GatewayNet.String())
	}

	if err := htn.Interface.SetLinkIp(htn.Gateway, htn.GatewayNet); err != nil {
		if os.IsExist(err) {
			htn.log.Info("Bridge IP appears to be already assigned")
		} else {
			return fmt.Errorf("Unable to set gateway IP", err)
		}
	}

	// Bridge the interface up
	if err := htn.Interface.SetLinkUp(); err != nil {
		return fmt.Errorf("Unable to bring bridge '%+v' up: %+v", htn.bridgeName(), err)
	}

	return nil
}

func (htn *HostNetwork) buildBridgeNetwork(addrs []net.Addr) error {
	// Try to build the network config from the bridge's address
	addrIndex := -1
	for i, addr := range addrs {
		bIP, brIP, _ := net.ParseCIDR(addr.String())

		// Discard IPv6 (TODO...)
		if bIP.To4() != nil {
			addrIndex = i

			bMask := []byte(brIP.Mask)

			htn.Netmask = net.IPv4(bMask[0], bMask[1], bMask[2], bMask[3])
			htn.Gateway = net.ParseIP(strings.Split(addr.String(), "/")[0])
			htn.Class = strings.Split(addr.String(), "/")[1]
			htn.Broadcast = net_getbroadcast(bIP, brIP.Mask)

			htn.Min = inet_aton(bIP)
			htn.Min++

			htn.Max = inet_aton(htn.Broadcast)
			htn.Max--

			break
		}

	}

	if addrIndex < 0 {
		return errors.New("Could not find IPv4 for bridge interface")
	}

	return nil
}

func FindEmptyRange() (net.IP, *net.IPNet, error) {
	return findEmptyRange()
}

// Look at all the assigned IP address and try to find an available range
// Returns a ip range in the CIDR form if found or an empty string
func findEmptyRange() (net.IP, *net.IPNet, error) {
	type localNet struct {
		min uint64
		max uint64
	}

	var (
		localNets      []localNet
		availableRange string
	)

	// List all the available interfaces and their addresses
	// and calulate their network's min and max values
	ifs, _ := net.Interfaces()
	for _, netif := range ifs {
		// ignore loopback and oz interfaces
		if strings.HasPrefix(netif.Name, "lo") ||
			strings.HasPrefix(netif.Name, ozDefaultInterfaceBridgeBase) ||
			strings.HasPrefix(netif.Name, ozDefaultInterfacePrefix) {
			continue
		}

		// Go through each address on the interface
		addrs, _ := netif.Addrs()
		for _, addr := range addrs {
			bIP, brIP, _ := net.ParseCIDR(addr.String())

			// Discard non IPv4 addresses
			if bIP.To4() != nil {
				min := inet_aton(brIP.IP)
				min++

				max := inet_aton(net_getbroadcast(bIP, brIP.Mask))
				max--

				localNets = append(localNets, localNet{min: min, max: max})
			}
		}
	}

	// Go through the list of private network ranges and
	// look for one in which we cannot find a local network
	for _, ipRange := range privateNetworkRanges {
		bIP, brIP, err := net.ParseCIDR(ipRange)
		if err != nil {
			continue
		}

		bMin := inet_aton(bIP)
		bMax := inet_aton(net_getbroadcast(bIP, brIP.Mask))

		alreadyUsed := false
		for _, add := range localNets {
			if add.min >= bMin && add.min < bMax &&
				add.max > bMin && add.max <= bMax {
				alreadyUsed = true
				break

			}
		}

		// If the range is available, grab a small slice
		if alreadyUsed == false {
			bRange := bMax - bMin
			if bRange > 0xFF {
				bMin = bMax - 0xFE
			}

			// XXX
			availableRange = inet_ntoa(bMin).String() + "/24"

			return net.ParseCIDR(availableRange)
		}

	}

	return nil, nil, errors.New("Could not find an available range")

}

// Try to find an unassigned IP address
// Do this by first trying ozMaxRandTries random IPs or, if that fails, sequentially
func getFreshIP(min, max uint64, reservedBytes []uint, log *logging.Logger) string {
	var newIP string

	for i := 0; i < ozMaxRandTries; i++ {
		newIP = getRandIP(min, max, reservedBytes)
		if newIP != "" {
			break
		}
	}

	if newIP == "" {
		log.Notice("Random IP lookup failed %d times, reverting to sequential select", ozMaxRandTries)

		newIP = getScanIP(min, max, reservedBytes)
	}

	return newIP

}

// Generate a random ip and arping it to see if it is available
// Returns the ip on success or an ip string is the ip is already taken
func getRandIP(min, max uint64, reservedBytes []uint) string {
	if min > max {
		return ""
	}

	randVal := uint64(0)
	for {
		randVal = uint64(rand.Int63n(int64(max - min)))
		reserved := false
		for _, ipbyte := range reservedBytes {
			if ipbyte == uint(byte(randVal)&0xFF) {
				reserved = true
				break
			}
		}
		if !reserved {
			break
		}
	}

	dstIP := inet_ntoa(randVal + min)

	arping.SetTimeout(time.Millisecond * 150)

	_, _, err := arping.PingOverIfaceByName(dstIP, ozDefaultInterfaceBridge)

	if err == arping.ErrTimeout {
		return dstIP.String()
	} else if err != nil {
		return dstIP.String()
	}

	return ""

}

// Go through all possible ips between min and max
// and arping each one until a free one is found
func getScanIP(min, max uint64, reservedBytes []uint) string {
	if min > max {
		return ""
	}

	for i := min; i < max; i++ {
		reserved := false
		for _, ipbyte := range reservedBytes {
			if ipbyte == uint(i&0xFF) {
				reserved = true
				break
			}
		}
		if reserved {
			continue
		}
		dstIP := inet_ntoa(i)

		arping.SetTimeout(time.Millisecond * 150)

		_, _, err := arping.PingOverIfaceByName(dstIP, ozDefaultInterfaceBridge)

		if err == arping.ErrTimeout {
			return dstIP.String()
		} else if err != nil {
			return dstIP.String()
		}
	}

	return ""

}
