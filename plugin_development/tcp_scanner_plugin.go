package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TCPScannerPlugin represents the TCP scanner plugin
type TCPScannerPlugin struct {
	info PluginInfo
}

// Plugin is the main plugin instance
var Plugin = &TCPScannerPlugin{}

// PluginInfo contains metadata about the plugin
type PluginInfo struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Author      string    `json:"author"`
	Status      string    `json:"status"`
	LoadedAt    time.Time `json:"loaded_at"`
}

func (p *TCPScannerPlugin) GetInfo() PluginInfo {
	// Always return proper info, even if not initialized
	return PluginInfo{
		Name:        "tcp_scanner",
		Type:        "automation",
		Version:     "1.0.0",
		Description: "TCP port scanner for IP addresses, CIDR ranges, and IP-IP ranges",
		Author:      "SecAuto Team",
		Status:      "loaded",
		LoadedAt:    time.Now(),
	}
}

func (p *TCPScannerPlugin) Initialize(config map[string]interface{}) error {
	p.info = PluginInfo{
		Name:        "tcp_scanner",
		Type:        "automation",
		Version:     "1.0.0",
		Description: "TCP port scanner for IP addresses, CIDR ranges, and IP-IP ranges",
		Author:      "SecAuto Team",
		Status:      "loaded",
		LoadedAt:    time.Now(),
	}
	return nil
}

func (p *TCPScannerPlugin) Execute(params map[string]interface{}) (interface{}, error) {
	// Parse parameters
	target, ok := params["target"].(string)
	if !ok {
		return nil, fmt.Errorf("target parameter is required")
	}

	ports, ok := params["ports"].(string)
	if !ok {
		ports = "80,443,22,21,23,25,53,110,143,993,995,8080,8443" // Default ports
	}

	timeout, ok := params["timeout"].(float64)
	if !ok {
		timeout = 5.0 // Default 5 seconds
	}

	maxWorkers, ok := params["max_workers"].(float64)
	if !ok {
		maxWorkers = 100 // Default 100 workers
	}

	// Parse target to get IP addresses
	ips, err := p.parseTarget(target)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target: %v", err)
	}

	// Parse ports
	portList, err := p.parsePorts(ports)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ports: %v", err)
	}

	// Perform scan
	results := p.scanTargets(ips, portList, time.Duration(timeout*float64(time.Second)), int(maxWorkers))

	return map[string]interface{}{
		"success":     true,
		"target":      target,
		"ports":       ports,
		"timeout":     timeout,
		"max_workers": maxWorkers,
		"total_ips":   len(ips),
		"total_ports": len(portList),
		"results":     results,
		"summary": map[string]interface{}{
			"open_ports":     p.countOpenPorts(results),
			"closed_ports":   p.countClosedPorts(results),
			"filtered_ports": p.countFilteredPorts(results),
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (p *TCPScannerPlugin) Cleanup() error {
	return nil
}

// parseTarget parses different target formats
func (p *TCPScannerPlugin) parseTarget(target string) ([]string, error) {
	// Check if it's a single IP
	if net.ParseIP(target) != nil {
		return []string{target}, nil
	}

	// Check if it's a CIDR range
	if strings.Contains(target, "/") {
		return p.parseCIDR(target)
	}

	// Check if it's an IP-IP range
	if strings.Contains(target, "-") {
		return p.parseIPRange(target)
	}

	return nil, fmt.Errorf("invalid target format: %s", target)
}

// parseCIDR parses CIDR notation (e.g., "192.168.1.0/24")
func (p *TCPScannerPlugin) parseCIDR(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); p.incIP(ip) {
		ips = append(ips, ip.String())
	}

	// Remove network and broadcast addresses
	if len(ips) > 2 {
		ips = ips[1 : len(ips)-1]
	}

	return ips, nil
}

// parseIPRange parses IP-IP range (e.g., "192.168.1.1-192.168.1.254")
func (p *TCPScannerPlugin) parseIPRange(ipRange string) ([]string, error) {
	parts := strings.Split(ipRange, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid IP range format: %s", ipRange)
	}

	startIP := net.ParseIP(strings.TrimSpace(parts[0]))
	endIP := net.ParseIP(strings.TrimSpace(parts[1]))

	if startIP == nil || endIP == nil {
		return nil, fmt.Errorf("invalid IP addresses in range: %s", ipRange)
	}

	return p.generateIPRange(startIP, endIP), nil
}

// incIP increments an IP address
func (p *TCPScannerPlugin) incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// generateIPRange generates all IPs between start and end
func (p *TCPScannerPlugin) generateIPRange(start, end net.IP) []string {
	var ips []string
	for ip := make(net.IP, len(start)); !ip.Equal(end); p.incIP(ip) {
		ipCopy := make(net.IP, len(ip))
		copy(ipCopy, ip)
		ips = append(ips, ipCopy.String())
	}
	ips = append(ips, end.String())
	return ips
}

// parsePorts parses port specification
func (p *TCPScannerPlugin) parsePorts(ports string) ([]int, error) {
	var portList []int
	seen := make(map[int]bool)

	parts := strings.Split(ports, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check if it's a port range (e.g., "80-90")
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid port range: %s", part)
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid start port: %s", rangeParts[0])
			}

			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid end port: %s", rangeParts[1])
			}

			if start > end {
				start, end = end, start
			}

			for port := start; port <= end; port++ {
				if port >= 1 && port <= 65535 && !seen[port] {
					portList = append(portList, port)
					seen[port] = true
				}
			}
		} else {
			// Single port
			port, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s", part)
			}

			if port >= 1 && port <= 65535 && !seen[port] {
				portList = append(portList, port)
				seen[port] = true
			}
		}
	}

	return portList, nil
}

// scanTargets performs the actual TCP scan
func (p *TCPScannerPlugin) scanTargets(ips []string, ports []int, timeout time.Duration, maxWorkers int) []map[string]interface{} {
	var results []map[string]interface{}
	var mutex sync.Mutex
	var wg sync.WaitGroup

	// Create work channel
	workChan := make(chan struct {
		ip   string
		port int
	}, len(ips)*len(ports))

	// Fill work channel
	for _, ip := range ips {
		for _, port := range ports {
			workChan <- struct {
				ip   string
				port int
			}{ip, port}
		}
	}
	close(workChan)

	// Start workers
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range workChan {
				result := p.scanPort(work.ip, work.port, timeout)
				mutex.Lock()
				results = append(results, result)
				mutex.Unlock()
			}
		}()
	}

	wg.Wait()
	return results
}

// scanPort scans a single port
func (p *TCPScannerPlugin) scanPort(ip string, port int, timeout time.Duration) map[string]interface{} {
	address := fmt.Sprintf("%s:%d", ip, port)

	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return map[string]interface{}{
			"ip":      ip,
			"port":    port,
			"status":  "closed",
			"error":   err.Error(),
			"address": address,
		}
	}
	defer conn.Close()

	// Try to get service name
	service := p.getServiceName(port)

	return map[string]interface{}{
		"ip":      ip,
		"port":    port,
		"status":  "open",
		"service": service,
		"address": address,
	}
}

// getServiceName returns the service name for common ports
func (p *TCPScannerPlugin) getServiceName(port int) string {
	services := map[int]string{
		21:   "ftp",
		22:   "ssh",
		23:   "telnet",
		25:   "smtp",
		53:   "dns",
		80:   "http",
		110:  "pop3",
		143:  "imap",
		443:  "https",
		993:  "imaps",
		995:  "pop3s",
		8080: "http-proxy",
		8443: "https-alt",
	}

	if service, exists := services[port]; exists {
		return service
	}
	return "unknown"
}

// countOpenPorts counts open ports in results
func (p *TCPScannerPlugin) countOpenPorts(results []map[string]interface{}) int {
	count := 0
	for _, result := range results {
		if status, ok := result["status"].(string); ok && status == "open" {
			count++
		}
	}
	return count
}

// countClosedPorts counts closed ports in results
func (p *TCPScannerPlugin) countClosedPorts(results []map[string]interface{}) int {
	count := 0
	for _, result := range results {
		if status, ok := result["status"].(string); ok && status == "closed" {
			count++
		}
	}
	return count
}

// countFilteredPorts counts filtered ports in results
func (p *TCPScannerPlugin) countFilteredPorts(results []map[string]interface{}) int {
	count := 0
	for _, result := range results {
		if status, ok := result["status"].(string); ok && status == "filtered" {
			count++
		}
	}
	return count
}

// main function for standalone execution
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: tcp_scanner_plugin.exe [info|execute|cleanup]")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "info":
		info := Plugin.GetInfo()
		jsonData, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to marshal plugin info: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonData))

	case "execute":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: tcp_scanner_plugin.exe execute <params_json>")
			os.Exit(1)
		}

		var params map[string]interface{}
		if err := json.Unmarshal([]byte(os.Args[2]), &params); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing parameters: %v\n", err)
			os.Exit(1)
		}

		result, err := Plugin.Execute(params)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing plugin: %v\n", err)
			os.Exit(1)
		}

		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to marshal result: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonData))

	case "cleanup":
		if err := Plugin.Cleanup(); err != nil {
			fmt.Fprintf(os.Stderr, "Error during cleanup: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(`{"success": true}`)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}
}
