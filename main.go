package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/op/go-logging"
	"github.com/sparrc/go-ping"
)

var log = logging.MustGetLogger("example")
var format = logging.MustStringFormatter(
	`%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{message}`,
)

type Changer struct {
	Ip             map[string]bool
	Script         string
	Max_num_change int
	Num_change     int
	Start          bool
}

type Changer_list struct {
	changer_curent *Changer
	changer_prev   *Changer
}

type Group struct {
	Ip                       []string `json:"ip"`
	Delay                    uint     `json:"delay"`
	Num_allowed_conn_in_time int      `json:"num_allowed_conn_in_time"`
	Count                    int      `json:"count_pac"`
	Count_to_reconnect       int      `json:"count_pac_to_reconect"`
	Script                   string   `json:"script"`
	Time_allowed             int      `json:"time_allowed_conn_for_num"`
}

type Configuration struct {
	Log_path string  `json:"logPath"`
	Groups   []Group `json:"groups"`
}

func change_state(changer_list *Changer_list) {
	changer := changer_list.changer_curent
	log.Info(changer.Max_num_change)
	log.Info(changer.Num_change)
	if changer.Num_change >= changer.Max_num_change {
		log.Warning("The limit of possible changes is exceeded")
	} else {
		var comand_str string
		var comand_str_prev string
		num_elements := len(changer.Ip)
		i := 1
		for _, value := range changer.Ip {
			if i >= num_elements {
				comand_str = comand_str + fmt.Sprintf("%t", value)
			} else {
				comand_str = comand_str + fmt.Sprintf("%t,", value)
			}
			i++
		}
		if changer_list.changer_prev != nil {
			changer_prev := changer_list.changer_prev
			num_elements := len(changer_prev.Ip)
			i := 1
			for _, value := range changer_prev.Ip {
				if i >= num_elements {
					comand_str_prev = comand_str_prev + fmt.Sprintf("%t", value)
				} else {
					comand_str_prev = comand_str_prev + fmt.Sprintf("%t,", value)
				}
				i++
			}
		} else {
			comand_str_prev = ""
		}
		log.Info(changer.Script, comand_str, comand_str_prev)
		out1, _ := exec.Command(changer.Script, comand_str, comand_str_prev).Output()
		if len(out1) != 0 {
			log.Notice(string(out1))
		} else {
			log.Error("Result empty execute : ", changer.Script)
		}
		changer.Num_change++
	}
}

func new_ping(ip string, changer_list *Changer_list, grp Group, ch chan bool, mutex sync.Mutex) {
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		panic(err)
	}
	pinger.Count = grp.Count
	pinger.Run() // blocks until finished
	stats := pinger.Statistics() // get send/receive/rtt stats
	fmt.Printf("\n--- %s ping statistics ---\n", stats.Addr)
	fmt.Printf("%d packets transmitted, %d packets received, %v%% packet loss\n",
		stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)
	fmt.Printf("round-trip min/avg/max/stddev = %v/%v/%v/%v\n",
		stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt)
}

func my_ping(ip string, changer_list *Changer_list, grp Group, ch chan bool, mutex sync.Mutex) {
	var out1 []byte
	change_status := false
	changer :=changer_list.changer_curent
	if changer.Ip[ip] == true {
		out1, _ = exec.Command("ping", ip, fmt.Sprintf("-c %d", grp.Count)).Output()
	} else {
		out1, _ = exec.Command("ping", ip, fmt.Sprintf("-c %d", grp.Count_to_reconnect)).Output()
	}
	if strings.Contains(string(out1), "100% packet loss") ||
		strings.Contains(string(out1), "Destination Host Unreachable") ||
		strings.Contains(string(out1), "Network is unreachable") ||
		len(out1) == 0 {
		if changer.Ip[ip] {
			mutex.Lock()
			changer_list.changer_prev.Ip[ip] = changer.Ip[ip]
			changer.Ip[ip] = false
			mutex.Unlock()
			change_status = true
			// change_state(changer_list)
		}
	} else {
		if !changer.Ip[ip] {
			mutex.Lock()
			changer_list.changer_prev.Ip[ip] = changer.Ip[ip]
			changer.Ip[ip] = true
			mutex.Unlock()
			change_status = true
			// change_state(changer_list)
		}
	}
	//log.Info("PING ", ip, " recv ", len(out1))
	if changer.Start {
		// change_state(changer_list)
		changer.Start = false
		change_status = true
	}
	ch <- change_status
}

func new_start_ping(changer_list *Changer_list, grp Group, c chan int) {
	//changer := changer_list.changer_curent
	for true {
		time.Sleep(time.Duration(grp.Delay) * time.Second)
		ch := make(chan bool)
		var mutex sync.Mutex
		for _, ip := range grp.Ip {
			//go my_ping(ip, changer_list, grp,ch,mutex)
			go new_ping(ip, changer_list, grp,ch,mutex)
		}
		for _, ip := range grp.Ip {
			change_status := <-ch
			if change_status {
				log.Notice("Change ",ip)
				change_state(changer_list)
			}
		}
	}
}

func main() {
	//------------------------------------------------------------------------------
	//open config
	argsWithProg := os.Args
	configuration := Configuration{}
	var config_file_name string
	if len(argsWithProg) == 2 {
		config_file_name = argsWithProg[1]
	}
	if len(config_file_name) == 0 {
		fmt.Println("Please enter the name of the configuration file")
		fmt.Println("Example : pinger *.conf")
	}
	file, err1 := os.Open(config_file_name)
	if err1 != nil {
		log.Error("Error open configuration file!")
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&configuration)
	if err != nil {
		log.Error("Error decode from json configuration file!")
	}
	//------------------------------------------------------------------------------
	//open log
	today := time.Now()
	file_log, err := os.OpenFile(configuration.Log_path+"/"+today.Format("2006.01.02")+".log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer file_log.Close()
	backend1 := logging.NewLogBackend(os.Stderr, "", 0)
	backend2 := logging.NewLogBackend(os.Stderr, "", 0)
	backend2Formatter := logging.NewBackendFormatter(backend2, format)
	backend1Leveled := logging.AddModuleLevel(backend1)
	backend1Leveled.SetLevel(logging.ERROR, "")
	logging.SetBackend(backend1Leveled, backend2Formatter)

	//-----------------------------------------------------------------------------
	//start main logic
	c := make(chan int)
	for _, value := range configuration.Groups {
		change_list := new(Changer_list)
		change := new(Changer)
		change_list.changer_curent = change
		change.Script = value.Script
		change.Max_num_change = value.Num_allowed_conn_in_time
		change.Ip = make(map[string]bool)
		change.Start = true
		for _, ip := range value.Ip {
			change.Ip[ip] = true
		}
		changer_prev := new(Changer)
		changer_prev.Ip = make(map[string]bool)
		for key, value := range change.Ip {
			changer_prev.Ip[key] = value
		}
		change_list.changer_prev = changer_prev
		go func() {
			for true {
				time.Sleep(time.Duration(value.Time_allowed) * time.Second)
				change.Num_change = 0
			}
		}()
		go new_start_ping(change_list, value, c)
	}
	for index, _ := range configuration.Groups {
		err := <-c
		fmt.Println(err, index)
	}
}
