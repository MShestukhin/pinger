package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("example")
var format = logging.MustStringFormatter(
	`{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{message}`,
)

type Changer struct {
	Ip map[string] bool
	Script string
	Max_num_change int
	Num_change int
	Start bool
}

type Group struct {
	Ip 	[]string `json:"ip"`
	Delay     uint `json:"delay"`
	Num_allowed_conn_in_time   int `json:"num_allowed_conn_in_time"`
	Count     int `json:"count_pac"`
	Script    string `json:"script"`
	Time_allowed    int `json:"time_allowed_conn_for_num"`
}

type Configuration struct {
	Log_path string `json:"logPath"`
	Groups []Group `json:"groups"`
}

func change_state(changer * Changer)  {
	log.Info(changer.Max_num_change)
	log.Info(changer.Num_change)
	if changer.Num_change >= changer.Max_num_change {
		log.Warning( "The limit of possible changes is exceeded")
	} else {
		var comand_str string
		for _, value := range changer.Ip {
			comand_str = comand_str+fmt.Sprintf("%t ", value)
		}
		log.Info(changer.Script, comand_str)
		out1, _ := exec.Command(changer.Script, comand_str).Output()
		log.Notice(string(out1))
		changer.Num_change++;
	}
}
var start = true
func new_start_ping(changer * Changer, grp Group, c chan int)  {
	for true {
		time.Sleep(time.Duration(grp.Delay) * time.Second)
		for _, ip := range grp.Ip {
			out1, _ := exec.Command("ping", ip, fmt.Sprintf("-c %d", grp.Count)).Output()
			if 	(strings.Contains(string(out1), "100% packet loss") ||
				strings.Contains(string(out1), "Destination Host Unreachable") ||
				strings.Contains(string(out1), "Network is unreachable") ||
				len(out1) == 0){
				if changer.Ip[ip] {
					changer.Ip[ip] = false
					change_state(changer)
				}
			} else {
				if !changer.Ip[ip] {
					changer.Ip[ip] = true
					change_state(changer)
				}
			}
			log.Info("PING ", ip, " recv ", len(out1))
			if changer.Start {
				change_state(changer)
				changer.Start = false
			}

		}
	}
}

//func start_ping(changer * Changer, grp Group, c chan int)  {
//	for true {
//		time.Sleep(time.Duration(grp.Delay) * time.Second)
//		out1, _ := exec.Command("ping", grp.Svc_clr1, fmt.Sprintf("-c %d", grp.Count)).Output()
//		if 	(strings.Contains(string(out1), "100% packet loss") ||
//				strings.Contains(string(out1), "Destination Host Unreachable") ||
//				strings.Contains(string(out1), "Network is unreachable") ||
//				len(out1) == 0){
//			if changer.Clr_prg1 {
//				changer.Clr_prg1 = false
//				change_state(changer)
//			}
//		} else {
//			if !changer.Clr_prg1{
//				changer.Clr_prg1 = true
//				change_state(changer)
//			}
//		}
//		log.Info("PING ", grp.Svc_clr1, " recv ", len(out1))
//		out2, _ := exec.Command("ping", grp.Svc_clr2, fmt.Sprintf("-c %d", grp.Count)).Output()
//		if	(strings.Contains(string(out2), "100% packet loss") ||
//				strings.Contains(string(out2), "Destination Host Unreachable") ||
//				strings.Contains(string(out2), "Network is unreachable") ||
//				len(out2) == 0){
//			if changer.Clr_prg2 {
//				changer.Clr_prg2 = false
//				change_state(changer)
//			}
//		} else {
//			if !changer.Clr_prg2{
//				changer.Clr_prg2 = true
//				change_state(changer)
//			}
//		}
//		log.Info("PING ", grp.Svc_clr2, " recv ", len(out2))
//	}
//	c <- 1
//}

func main() {
	//------------------------------------------------------------------------------
	//open config
	configuration := Configuration{}
	file, err1 := os.Open("conf.json")
	if err1 != nil {log.Error("Error open configuration file!")}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&configuration)
	if err!= nil { log.Error("Error decode from json configuration file!")}
	//------------------------------------------------------------------------------
	//open log
	today := time.Now()
	file_log, err := os.OpenFile(configuration.Log_path + "/"+today.Format("2006.01.02")+".log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil{
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
	for _, value := range configuration.Groups{
		change:=Changer{}
		change.Script = value.Script
		change.Max_num_change = value.Num_allowed_conn_in_time
		change.Ip = make(map[string] bool)
		change.Start = true
		for _, ip := range value.Ip {
			change.Ip[ip] = true
		}
		go func() {
			for true {
				time.Sleep(time.Duration(value.Time_allowed) * time.Second)
				change.Num_change = 0
			}
		}()
		go new_start_ping(&change,value,c)
	}
	for index, _ := range configuration.Groups{
		err := <-c
		fmt.Println(err,index)
	}
}
