package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
	"net"
	"bufio"
	"github.com/op/go-logging"
	"github.com/sparrc/go-ping"
)

var log = logging.MustGetLogger("example")
var format = logging.MustStringFormatter(
	`%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{message}`,
)

type Changer struct {
	Ip_current_level   map[string]int
	Ip_result          map[string][]int
	pingers            map[string]*ping.Pinger
	Ip_level_statistic map[string][]int
	Script             string
	Max_num_change     int
	Num_change         int
	Start              bool
	num_recv_pac       map[string]time.Duration
	current_statuses   string
	may_i_change       bool
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
	Len_buff_for_analise_pac int      `json:"len_buff_for_analise_pac"`
}

type Configuration struct {
	Log_path string  `json:"logPath"`
	Groups   []Group `json:"groups"`
}

var mutex sync.Mutex

func get_level(ips []int) int {
	hund_perc := len(ips)
	num_connect := 0
	for _, value := range ips {
		if value == 1 {
			num_connect++
		}
	}
	percent := (num_connect * 100) / hund_perc
	level := 0
	switch {
	case percent <= 100 && percent >= 80:
		level = 5
		break
	case percent < 80 && percent >= 60:
		level = 4
		break
	case percent < 60 && percent >= 40:
		level = 3
		break
	case percent < 40 && percent >= 20:
		level = 2
		break
	case percent < 20 && percent > 0:
		level = 1
		break
	case percent == 0:
		level = 0
		break
	}
	return level
}

func change_state(changer_list *Changer_list, grp Group) {
	changer := changer_list.changer_curent
	if changer.Num_change >= changer.Max_num_change {
		if changer.may_i_change {
			log.Warning(fmt.Sprintf("The limit of possible triggers is increased %d !", changer.Max_num_change))
			changer.may_i_change = false
		}
	} else {
		mutex.Lock()
		var statuses_str string
		ind := 1
		sum_ips := len(changer.Ip_result)
		can_run_script := true
		for _, key := range grp.Ip {
			ips := changer.Ip_result[key]
			level := get_level(ips)

			if len(ips) >= grp.Count_to_reconnect {
				can_change := true
				vector_last := ips[len(ips)-grp.Count_to_reconnect : len(ips)]
				for _, ping_res := range vector_last {
					if ping_res == 0 {
						can_change = false
					}
				}

				sum := 0
				for _, elem := range changer_list.changer_curent.Ip_level_statistic[key] {
					sum = sum + elem
				}

				if len(changer.Ip_level_statistic) >= grp.Count {
					for _, ping_res := range changer.Ip_level_statistic[key] {
						if ping_res != 5 {
							can_change = false
						}
					}
				}

				//midle_of_levels := sum/len(changer_list.changer_curent.Ip_level_statistic[key])
				// midle_of_levels - среднее по уровням
				if (can_change) && changer_list.changer_curent.Ip_current_level[key] != 5 {
					log.Info("At the moment the channel has the highest bandwidth ", key)
					level = 5
					changer_list.changer_curent.Ip_result[key] = nil
					changer_list.changer_curent.Ip_result[key] = []int{1}
				}
			}
			if level != 5 && len(ips) >= grp.Count {
				can_change := true
				vector_last := ips[len(ips)-grp.Count : len(ips)]
				for _, ping_res := range vector_last {
					if ping_res == 1 {
						can_change = false
					}
				}

				sum := 0
				for _, elem := range changer_list.changer_curent.Ip_level_statistic[key] {
					sum = sum + elem
				}
				//midle_of_levels := sum/len(changer_list.changer_curent.Ip_level_statistic[key])
				// midle_of_levels - среднее по уровням
				if (can_change) && changer.Ip_current_level[key] != 0 {
					log.Error("The channel is not available ", key)
					level = 0
					changer_list.changer_curent.Ip_result[key] = nil
					changer_list.changer_curent.Ip_result[key] = []int{0}
				}
			}
			if level != 5 && level != 0 {
				last_index := len(changer_list.changer_curent.Ip_level_statistic[key]) - 1
				last_elem := changer_list.changer_curent.Ip_level_statistic[key][last_index]
				if level < last_elem {
					log.Warning("The quality of the channel is falling ", key)
				}
				if level >= last_elem {
					can_run_script = false
				}
			}

			if ind >= sum_ips {
				statuses_str = statuses_str + fmt.Sprintf("%d", level)
			} else {
				statuses_str = statuses_str + fmt.Sprintf("%d,", level)
			}
			changer_list.changer_curent.Ip_current_level[key] = level
			changer_list.changer_curent.Ip_level_statistic[key] = append(changer_list.changer_curent.Ip_level_statistic[key], level)
			// чтобы не занимать излишне место в статистике, массив статистики ограничен числом grp.Count
			// если размер массива больше или равен grp.Count, при добавлении нового значения, значение в начале массива удаляется
			// было [0,5,5,5,5,5,5] при появлении уровня например 4 - стало [5,5,5,5,5,5,4]
			if len(changer_list.changer_curent.Ip_level_statistic[key]) >= grp.Count {
				changer_list.changer_curent.Ip_level_statistic[key] = append(changer_list.changer_curent.Ip_level_statistic[key][:0], changer_list.changer_curent.Ip_level_statistic[key][1:]...)
			}
			ind++
		}
		var out1 []byte
		if changer.current_statuses != statuses_str && can_run_script {
			out1, _ = exec.Command(changer.Script, statuses_str).Output()
			log.Info(changer.Script, statuses_str)
			log.Info(changer.Ip_current_level, fmt.Sprintf("Number of possible triggers %d !", changer.Max_num_change))
			log.Info(changer.Ip_current_level, fmt.Sprintf("Number of triggers %d !", changer.Num_change))
			if len(out1) != 0 {
				log.Info(string(out1))
			} else {
				log.Error("The script did not return anything during execution : ", changer.Script)
			}
			changer.Num_change++
			changer.current_statuses = statuses_str
		}
		mutex.Unlock()
	}
}

func statistic(pinger *ping.Pinger, changer_list *Changer_list, grp Group) {
	stats := pinger.Statistics()
	changer := changer_list.changer_curent
	//changer_list.changer_curent.num_recv_pac[stats.Addr] - хранит число вернувшихся пакетов за предыдущий срез статистики
	//если пакеты приходят то добавляю единицу, если нет, то заношу 0
	if stats.AvgRtt == changer.num_recv_pac[stats.Addr] {
		mutex.Lock()
		changer.Ip_result[stats.Addr] = append(changer.Ip_result[stats.Addr], 0)
		mutex.Unlock()
	} else {
		mutex.Lock()
		changer.Ip_result[stats.Addr] = append(changer.Ip_result[stats.Addr], 1)
		mutex.Unlock()
	}
	if len(changer.Ip_result[stats.Addr]) >= grp.Len_buff_for_analise_pac {
		mutex.Lock()
		changer.Ip_result[stats.Addr] = append(changer.Ip_result[stats.Addr][:0], changer.Ip_result[stats.Addr][1:]...)
		mutex.Unlock()
	}
	//сохраняю число вернувшихся пакетов
	changer.num_recv_pac[stats.Addr] = stats.AvgRtt
}

func new_ping(pinger *ping.Pinger, ip string, changer_list *Changer_list, grp Group) {

	// pinger.OnRecv принимает пакеты
	pinger.OnRecv = func(pkt *ping.Packet) {
		changer := changer_list.changer_curent
		stats := pinger.Statistics()
		mutex.Lock()
		changer.Ip_result[stats.Addr] = append(changer.Ip_result[stats.Addr], 1)
		mutex.Unlock()
		//changer_list.changer_curent.num_recv_pac[stats.Addr] = float64(stats.PacketsRecv)
		//recv_pac := 0
		//for _, pinger_from := range changer_list.changer_curent.pingers {
		//	if pinger_from.Addr() != pinger.Addr() {
		//		if pinger_from.PacketsRecv > pinger.PacketsRecv {
		//			recv_pac = pinger_from.PacketsRecv - pinger.PacketsRecv
		//		}
		//		if pinger_from.PacketsRecv < pinger.PacketsRecv {
		//			recv_pac = pinger.PacketsRecv - pinger_from.PacketsRecv
		//		}
		//		if recv_pac > 3 {
		//			mutex.Lock()
		//			changer_list.changer_curent.Ip_result[pinger_from.Addr()] = append(changer_list.changer_curent.Ip_result[pinger_from.Addr()], 0)
		//			mutex.Unlock()
		//			if len(changer_list.changer_curent.Ip_result[pinger_from.Addr()]) >= grp.Len_buff_for_analise_pac {
		//				mutex.Lock()
		//				changer_list.changer_curent.Ip_result[pinger_from.Addr()] = append(changer_list.changer_curent.Ip_result[pinger_from.Addr()][:0], changer_list.changer_curent.Ip_result[pinger_from.Addr()][1:]...)
		//				mutex.Unlock()
		//			}
		//		}
		//	}
		//}
		// массив введения статистики типа 127.0.0.1[0,1,1,1,1,1,1,1,1,1,1]
		// ограничен числом заданным в конфиге и находится в grp.Len_buff_for_analise_pac
		if len(changer.Ip_result[stats.Addr]) >= grp.Len_buff_for_analise_pac {
			mutex.Lock()
			// при добавлении нового пришедшего или не пришедшего пинга
			// удаляю значение с начала, добавляю значение с конца
			// приняли пакет, было [0,1,1,1,1,1,1,1,1,1,1] стало [1,1,1,1,1,1,1,1,1,1,1]
			changer.Ip_result[stats.Addr] = append(changer.Ip_result[stats.Addr][:0], changer.Ip_result[stats.Addr][1:]...)
			mutex.Unlock()
		}
	}
	// хендлер pinger.OnFinish не должен быть вызван в течении работы программы, если будет вызван то ошибка
	pinger.OnFinish = func(stats *ping.Statistics) {
		log.Warning("One of the pingers prematurely completed the work")
		log.Warning("\n--- %s ping statistics ---\n", stats.Addr)
		log.Warning(fmt.Sprintf("%d packets transmitted, %d packets received, %v%% packet loss\n",
			stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss))
		log.Warning(fmt.Sprintf("round-trip min/avg/max/stddev = %v/%v/%v/%v\n",
			stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt))
	}

	log.Info(fmt.Sprintf("PING %s (%s):\n", pinger.Addr(), pinger.IPAddr()))
	// запускаю бесконечный пинг на ip переданный в функцию
	pinger.Run()
}

func new_start_ping(changer_list *Changer_list, grp Group, c chan int) {
	// создаю пингер
	pingers := []*ping.Pinger{}
	for _, ip := range grp.Ip {
		// передаю пингеру ip из группы из конфигурационного файла
		pinger, err := ping.NewPinger(ip)
		if err != nil {
			panic(err)
		}
		changer_list.changer_curent.pingers[ip] = pinger
		pingers = append(pingers, pinger)
		pinger.SetPrivileged(true)
		// начинаю пинговать ip взятый из грруппы из конфигурационного файла в потоке
		go new_ping(pinger, ip, changer_list, grp)
	}
	// работает в потоке
	for true {
		// каждое время t в секундах, задаётся в конфиге в поле delay
		// подводится статистика
		time.Sleep(time.Duration(grp.Delay) * time.Second)

		// если ping для какого либо ip из группы не возвращается то в статистике задаётся ноль,
		// таким образом в цикле могу определить что канал не доступен
		for _, pinger := range pingers {
			statistic(pinger, changer_list, grp)
		}
		change_state(changer_list, grp)
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
		fmt.Println("Specify the path to the configuration file for the program")
		fmt.Println("The path to the configuration file is passed as an input parameter separated by a space")
		fmt.Println("For example : ./pinger path_to_config_file.conf")
	}
	file, err1 := os.Open(config_file_name)
	if err1 != nil {
		fmt.Println("The configuration file could not be opened ", err1.Error())
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&configuration)
	if err != nil {
		fmt.Println("The configuration file could not be read :", err.Error())
	}
	//------------------------------------------------------------------------------
	//open log
	today := time.Now()
	file_log, err := os.OpenFile(configuration.Log_path+"/"+today.Format("2006.01.02")+".log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Unable to open the log file :", err.Error())
		os.Exit(1)
	}
	defer file_log.Close()
	backend1 := logging.NewLogBackend(file_log, "", 0)
	backend2 := logging.NewLogBackend(file_log, "", 0)
	backend2Formatter := logging.NewBackendFormatter(backend2, format)
	backend1Leveled := logging.AddModuleLevel(backend1)
	backend1Leveled.SetLevel(logging.ERROR, "")
	logging.SetBackend(backend1Leveled, backend2Formatter)

	//-----------------------------------------------------------------------------
	//start main logic
	c := make(chan int)
	groups := [] *Changer{}
	// вытаскиваем по группе
	for _, group := range configuration.Groups {
		change_list := new(Changer_list) //устарело нужно удалить
		change := new(Changer)           // делаю наблюдателя
		groups = append(groups,change)
		change_list.changer_curent = change
		change.Script = group.Script // задаю скрипт для группы
		change.may_i_change = true
		change.Max_num_change = group.Num_allowed_conn_in_time // задаю максимально возможные число срабатываений за время t
		//чтобы определить патери, и принять решение что канал не доступен
		// счётчик принимаемых пакетов, использую в статистике,
		change.num_recv_pac = make(map[string]time.Duration)

		changer_prev := new(Changer) //устарело можно удалить

		change_list.changer_prev = changer_prev // не используется

		// хранит текущее состояние канала по типу [127.0.0.1:[0,1,0,1,1,1,1,1,1,1,1,1]]
		// 0 - пакет не вернулся
		// 1 - пакет вернулся
		// инициализирую карту
		change.Ip_result = make(map[string][]int)

		// хранит массив изменений канала состояние канала по типу [127.0.0.1:[4,3,2,0,2,5]]
		// 0 - 100% патерь на канале за время t
		// 1 - 80% патерь на канале за время t
		// 2 - 60% патерь на канале за время t
		// 3 - 40% патерь на канале за время t
		// 4 - 20% патерь на канале за время t
		// 5 - 0% патерь на канале за время t
		change.Ip_level_statistic = make(map[string][]int)
		change.pingers = make(map[string]*ping.Pinger)
		change.Ip_current_level = make(map[string]int)
		// перебираем ip из конфигурационного файла { ... "ip" : ["12.0.0.1", "server2"] ... }
		for _, ip := range group.Ip {
			// инициализирую массив для карты строкой выше 363стр
			//тип например [127.0.0.1:[0,1,0,1,1,1,1,1,1,1,1,1]]
			change.Ip_result[ip] = []int{}
			// инициализирую массив для карты строкой выше 372стр
			// по типу [127.0.0.1:[4,3,2,0,2,5]]
			change.Ip_level_statistic[ip] = []int{}
			// считаю что канал не доступен [127.0.0.1:[0]]
			change.Ip_level_statistic[ip] = append(change.Ip_level_statistic[ip], 0)
			// заполняю масив нулём [127.0.0.1:[0]]
			change.Ip_result[ip] = append(change.Ip_result[ip], 0)
		}
		// запускаю таймер по истичению которого обнуляю число возможных запусков скрипта
		go func() {
			for true {
				time.Sleep(time.Duration(group.Time_allowed) * time.Second)
				change.Num_change = 0
				change.may_i_change = true
			}
		}()
		go new_start_ping(change_list, group, c)
	}
	// например пингую одновременно ["127.0.0.1", "8.8.8.8"]
	// запускаю в потоке пинги ip которые находятся в конфигурационном файле
	go func() {
		ln, _ := net.Listen("tcp", ":8081")
		conn, _ := ln.Accept()
		for true {
			//time.Sleep(time.Second)
			group_num, _ := bufio.NewReader(conn).ReadString('\n')
			fmt.Print("Enter please num of group : ")
			//if int(group_num) <= len(groups) && int(group_num) !=0 {
			//	fmt.Println(time.Now().Format("Mon Jan _2 15:04:05 2006"),groups[group_num-1].Ip_current_level)
			//}
		}
	}()
	// так как каждая группа анализируется в потоке, цикл перехватывает возможные ошибки работы в потоках
	for index, _ := range configuration.Groups {
		err := <-c
		fmt.Println(err, index)
		log.Fatal(err)
	}
}
