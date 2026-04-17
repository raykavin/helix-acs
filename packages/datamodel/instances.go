package datamodel

import (
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// InstanceMap holds the discovered instance indices for a CPE's key TR-069 objects.
// Any field left at zero means "not discovered"; the mapper falls back to its
// hardcoded default in that case.
type InstanceMap struct {
	// TR-181: Device.IP.Interface.{WANIPIfaceIdx} carries the public WAN IP.
	WANIPIfaceIdx int
	// TR-181: Device.IP.Interface.{LANIPIfaceIdx} carries the private LAN IP.
	LANIPIfaceIdx int
	// TR-181: Device.PPP.Interface.{PPPIfaceIdx}
	PPPIfaceIdx int

	// TR-181 WiFi instances indexed by band (0=2.4GHz, 1=5GHz, 2=6GHz).
	// A zero entry means the band was not discovered for that slot.
	WiFiSSIDIndices  []int // Device.WiFi.SSID.{i}
	WiFiRadioIndices []int // Device.WiFi.Radio.{i}
	WiFiAPIndices    []int // Device.WiFi.AccessPoint.{i}

	// TR-098: WANDevice.{WANDeviceIdx}.WANConnectionDevice.{WANConnDevIdx}
	WANDeviceIdx  int
	WANConnDevIdx int
	// TR-098: WANIPConnection.{WANIPConnIdx}
	WANIPConnIdx int
	// TR-098: WANPPPConnection.{WANPPPConnIdx}
	WANPPPConnIdx int
	// TR-098: LANDevice.{LANDeviceIdx}
	LANDeviceIdx int
	// TR-098: WLANConfiguration.{i} per band (0=2.4GHz, 1=5GHz, …).
	WLANIndices []int
}

// DiscoverInstances scans a flat parameter map (e.g. from an Inform or a
// GetParameterValues response) and returns the best-known instance indices for
// WAN, LAN, and WiFi objects.
//
// It detects the data model from the parameter key prefixes and delegates to
// the appropriate scanner. Any indices that cannot be determined from the
// available parameters are left at zero so the caller can apply safe defaults.
func DiscoverInstances(params map[string]string) InstanceMap {
	var im InstanceMap
	if isTR181Params(params) {
		discoverTR181(params, &im)
	} else {
		discoverTR098(params, &im)
	}
	return im
}

// isTR181Params returns true when at least one key starts with "Device.".
func isTR181Params(params map[string]string) bool {
	for k := range params {
		if strings.HasPrefix(k, "Device.") {
			return true
		}
	}
	return false
}

// ---- TR-181 discovery -------------------------------------------------------

var (
	reIPIfaceAddr   = regexp.MustCompile(`^Device\.IP\.Interface\.(\d+)\.IPv4Address\.\d+\.IPAddress$`)
	rePPPIface      = regexp.MustCompile(`^Device\.PPP\.Interface\.(\d+)\.`)
	reRadioFreq     = regexp.MustCompile(`^Device\.WiFi\.Radio\.(\d+)\.OperatingFrequencyBand$`)
	reSSIDLower     = regexp.MustCompile(`^Device\.WiFi\.SSID\.(\d+)\.LowerLayers$`)
	reAPRef         = regexp.MustCompile(`^Device\.WiFi\.AccessPoint\.(\d+)\.SSIDReference$`)
	reSSIDAnything  = regexp.MustCompile(`^Device\.WiFi\.SSID\.(\d+)\.`)
)

func discoverTR181(params map[string]string, im *InstanceMap) {
	discoverTR181WAN(params, im)
	discoverTR181PPP(params, im)
	discoverTR181WiFi(params, im)
}

func discoverTR181WAN(params map[string]string, im *InstanceMap) {
	for name, val := range params {
		if val == "" {
			continue
		}
		m := reIPIfaceAddr.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		idx, _ := strconv.Atoi(m[1])
		if isPublicIP(val) && im.WANIPIfaceIdx == 0 {
			im.WANIPIfaceIdx = idx
		} else if isPrivateIP(val) && im.LANIPIfaceIdx == 0 {
			im.LANIPIfaceIdx = idx
		}
	}
}

func discoverTR181PPP(params map[string]string, im *InstanceMap) {
	for name := range params {
		m := rePPPIface.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		idx, _ := strconv.Atoi(m[1])
		if im.PPPIfaceIdx == 0 || idx < im.PPPIfaceIdx {
			im.PPPIfaceIdx = idx
		}
	}
}

func discoverTR181WiFi(params map[string]string, im *InstanceMap) {
	// Step 1: map Radio instance → band via OperatingFrequencyBand.
	radioToBand := map[int]int{}
	for name, val := range params {
		m := reRadioFreq.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		radioIdx, _ := strconv.Atoi(m[1])
		switch val {
		case "2.4GHz":
			radioToBand[radioIdx] = 0
		case "5GHz":
			radioToBand[radioIdx] = 1
		case "6GHz":
			radioToBand[radioIdx] = 2
		}
	}

	// Step 2: map SSID instance → band via LowerLayers → Radio reference.
	ssidToBand := map[int]int{}
	if len(radioToBand) > 0 {
		for name, val := range params {
			m := reSSIDLower.FindStringSubmatch(name)
			if m == nil {
				continue
			}
			ssidIdx, _ := strconv.Atoi(m[1])
			for radioIdx, band := range radioToBand {
				if strings.Contains(val, "Device.WiFi.Radio."+strconv.Itoa(radioIdx)+".") {
					ssidToBand[ssidIdx] = band
				}
			}
		}
	}

	// Fallback: if LowerLayers are absent, assign bands by sorted SSID index
	// (lowest SSID index = 2.4 GHz, next = 5 GHz, etc.).
	if len(ssidToBand) == 0 {
		var indices []int
		seen := map[int]bool{}
		for name := range params {
			m := reSSIDAnything.FindStringSubmatch(name)
			if m == nil {
				continue
			}
			idx, _ := strconv.Atoi(m[1])
			if !seen[idx] {
				seen[idx] = true
				indices = append(indices, idx)
			}
		}
		sort.Ints(indices)
		for band, idx := range indices {
			ssidToBand[idx] = band
		}
	}

	if len(ssidToBand) == 0 {
		return // no WiFi parameters available
	}

	// Step 3: map AccessPoint instance → band via SSIDReference → SSID reference.
	apToBand := map[int]int{}
	for name, val := range params {
		m := reAPRef.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		apIdx, _ := strconv.Atoi(m[1])
		for ssidIdx, band := range ssidToBand {
			if strings.Contains(val, "Device.WiFi.SSID."+strconv.Itoa(ssidIdx)+".") {
				apToBand[apIdx] = band
			}
		}
	}

	// Step 4: build index slices from the band maps.
	maxBand := 0
	for _, b := range ssidToBand {
		if b > maxBand {
			maxBand = b
		}
	}
	for _, b := range radioToBand {
		if b > maxBand {
			maxBand = b
		}
	}

	im.WiFiSSIDIndices = make([]int, maxBand+1)
	im.WiFiRadioIndices = make([]int, maxBand+1)
	im.WiFiAPIndices = make([]int, maxBand+1)

	for ssidIdx, band := range ssidToBand {
		if band < len(im.WiFiSSIDIndices) {
			im.WiFiSSIDIndices[band] = ssidIdx
		}
	}
	for radioIdx, band := range radioToBand {
		if band < len(im.WiFiRadioIndices) {
			im.WiFiRadioIndices[band] = radioIdx
		}
	}
	for apIdx, band := range apToBand {
		if band < len(im.WiFiAPIndices) {
			im.WiFiAPIndices[band] = apIdx
		}
	}
}

// ---- TR-098 discovery -------------------------------------------------------

var (
	reWANIPConn  = regexp.MustCompile(`^InternetGatewayDevice\.WANDevice\.(\d+)\.WANConnectionDevice\.(\d+)\.WANIPConnection\.(\d+)\.`)
	reWANPPPConn = regexp.MustCompile(`^InternetGatewayDevice\.WANDevice\.(\d+)\.WANConnectionDevice\.(\d+)\.WANPPPConnection\.(\d+)\.`)
	reLANDevice  = regexp.MustCompile(`^InternetGatewayDevice\.LANDevice\.(\d+)\.`)
	reWLANCfg    = regexp.MustCompile(`^InternetGatewayDevice\.LANDevice\.\d+\.WLANConfiguration\.(\d+)\.`)
)

func discoverTR098(params map[string]string, im *InstanceMap) {
	discoverTR098WAN(params, im)
	discoverTR098LAN(params, im)
	discoverTR098WLAN(params, im)
}

func discoverTR098WAN(params map[string]string, im *InstanceMap) {
	for name := range params {
		if m := reWANIPConn.FindStringSubmatch(name); m != nil {
			wanDev, _ := strconv.Atoi(m[1])
			wanConn, _ := strconv.Atoi(m[2])
			wanIP, _ := strconv.Atoi(m[3])
			if im.WANDeviceIdx == 0 {
				im.WANDeviceIdx = wanDev
				im.WANConnDevIdx = wanConn
				im.WANIPConnIdx = wanIP
			}
		}
		if m := reWANPPPConn.FindStringSubmatch(name); m != nil {
			wanDev, _ := strconv.Atoi(m[1])
			wanConn, _ := strconv.Atoi(m[2])
			wanPPP, _ := strconv.Atoi(m[3])
			if im.WANDeviceIdx == 0 {
				im.WANDeviceIdx = wanDev
				im.WANConnDevIdx = wanConn
			}
			if im.WANPPPConnIdx == 0 {
				im.WANPPPConnIdx = wanPPP
			}
		}
	}
}

func discoverTR098LAN(params map[string]string, im *InstanceMap) {
	for name := range params {
		m := reLANDevice.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		idx, _ := strconv.Atoi(m[1])
		if im.LANDeviceIdx == 0 || idx < im.LANDeviceIdx {
			im.LANDeviceIdx = idx
		}
	}
}

func discoverTR098WLAN(params map[string]string, im *InstanceMap) {
	seen := map[int]bool{}
	for name := range params {
		m := reWLANCfg.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		idx, _ := strconv.Atoi(m[1])
		seen[idx] = true
	}
	if len(seen) == 0 {
		return
	}
	indices := make([]int, 0, len(seen))
	for idx := range seen {
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	im.WLANIndices = indices
}

// ---- IP classification helpers ----------------------------------------------

var privateRanges []*net.IPNet

func init() {
	for _, cidr := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"100.64.0.0/10", // CGNAT
	} {
		_, network, _ := net.ParseCIDR(cidr)
		if network != nil {
			privateRanges = append(privateRanges, network)
		}
	}
}

// isPrivateIP returns true when s is a valid IPv4 address in a private or
// special-use range (RFC 1918, loopback, link-local, CGNAT).
func isPrivateIP(s string) bool {
	ip := net.ParseIP(s)
	if ip == nil {
		return false
	}
	ip = ip.To4()
	if ip == nil {
		return false
	}
	for _, r := range privateRanges {
		if r.Contains(ip) {
			return true
		}
	}
	return false
}

// isPublicIP returns true when s is a routable, non-private IPv4 address.
func isPublicIP(s string) bool {
	ip := net.ParseIP(s)
	if ip == nil {
		return false
	}
	ip = ip.To4()
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
		return false
	}
	return !isPrivateIP(s)
}
