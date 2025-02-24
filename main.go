package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/golang/glog"
)

type TargetGroup struct {
	Labels struct {
		NetworkName string `json:"networkname"`
	} `json:"labels"`
	Targets []string `json:"targets"`
}

type TargetGroups []*TargetGroup

func usage() {
	_, _ = fmt.Fprintf(os.Stderr, "usage: example -stderrthreshold=[INFO|WARNING|FATAL] -log_dir=[string] -c config.yml\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {

	configLocation := flag.String("c", "config.yml", "location of config file")
	resultLocation := flag.String("f", "results.json", "location of config file")
	flag.Usage = usage
	flag.Parse()

	sdConfig, _ := newSDConfig(*configLocation)

	tgs := TargetGroups{}
	for _, networks := range sdConfig.Networks {
		hosts, _ := receiveHosts(networks.Network)
		tg := sdConfig.FillTargets(hosts)
		tg.Labels.NetworkName = networks.Labels[0].NetworkName
		tgs = append(tgs, &tg)

	}

	result, err := json.MarshalIndent(tgs, "", "  ")
	if err != nil {
		glog.Fatal(err)
	}
	fmt.Println(string(result))
	_ = os.WriteFile(*resultLocation, []byte(result), 0644)
}

func (s *SDConfig) FillTargets(hostList []string)  TargetGroup {
	tg := TargetGroup{}
	//set concurrency to the core count and multiply it
	concurrency := runtime.NumCPU() * s.Concurrency
	fmt.Println("Concurrency set to", concurrency)
	semaphore := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	count := len(tg.Targets)
	bar := pb.StartNew(count)
	bar.SetWriter(os.Stdout)

	hostChan := make(chan string)
	for _, i := range hostList {
		semaphore <- struct{}{}
		wg.Add(1)
		for _, port := range s.Port {
			go IsOpen(i, strconv.Itoa(port), time.Duration(s.Timeout), hostChan, semaphore, &wg, bar)
		}
	}
	go func() {
		wg.Wait()
		close(semaphore)
		close(hostChan)
	}()
	for i := range hostChan {
		tg.Targets = append(tg.Targets, i)
	}
	bar.Finish()
	return tg
}

func ParseSDConfig(hosts chan string) string {
	tg := TargetGroup{}
	tg.Labels.NetworkName = "node_lab" 
	for i := range hosts {
		tg.Targets = append(tg.Targets, i)
	}

	tgs := []TargetGroup{tg}
	b, err := json.MarshalIndent(tgs, "", "  ")
	if err != nil {
		fmt.Println(err)
	}
	return string(b)
}

func IsOpen(ip string, port string, timeout time.Duration, hostChannel chan string, semaphore chan struct{}, wg *sync.WaitGroup, pb *pb.ProgressBar) {
	defer func() {
		<-semaphore
		wg.Done()
		pb.Add(1)
	}()
	if ip == "" {
		return
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, port), timeout*time.Second)
	if conn != nil {
		_ = conn.Close()
		glog.Info("Connected to: ", net.JoinHostPort(ip, port))
		hostChannel <- net.JoinHostPort(ip, port)
	}
	if err != nil {
		glog.Warning("Can not connect to: ", net.JoinHostPort(ip, port), "- error:", err)
		return
	}
}

func parseHosts(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}
	// remove network address and broadcast address
	return ips[1 : len(ips)-1], nil
}

func receiveHosts(ipNet string) ([]string, error) {
	var hostList []string

	hosts, err := parseHosts(ipNet)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	
	hostList = append(hostList, hosts...)

	glog.Info("Total number of hosts to discover: ", len(hostList))
	return hostList, nil
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
