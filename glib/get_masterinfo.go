package glib

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"../common"
)

var (
	AllFrameworks   map[string]map[string]bool //Map of maps for all the frameworks
	CommonFramework map[string]bool            //Common among all the framework
	FrmWrkLck       sync.Mutex                 //Lock for centralized framework
)

func init() {

	AllFrameworks = make(map[string]map[string]bool)
	CommonFramework = make(map[string]bool)
}

func GetListofFrameworks(G *Glib, MasterEP string) {

	resp, err := http.Get(fmt.Sprintf("http://%s/state-summary", MasterEP))

	if err != nil {

		log.Printf("Unable to reach the Master error = %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Printf("Unable to read the body error = %v", err)
		return
	}

	var data map[string]interface{}

	err = json.Unmarshal(body, &data)

	if err != nil {
		log.Printf("Json Unmarshall error = %v", err)
		return
	}

	frameworks_array, found := data["frameworks"].([]interface{})

	if found != true {
		log.Printf("Unable to find the framework array in the nested json object %v", data)
		return
	}

	this_frmwrk, exisit := AllFrameworks[G.Name]
	FrmWrkLck.Lock()
	if !exisit {
		AllFrameworks[G.Name] = make(map[string]bool)
		this_frmwrk = AllFrameworks[G.Name]
	}
	FrmWrkLck.Unlock()

	for _, frwrk_interface := range frameworks_array {

		frwrk, isvalid := frwrk_interface.(map[string]interface{})
		if !isvalid {
			log.Printf("Malformed json object recived from he master %v", frameworks_array)
			continue
		}
		id, isvalid := frwrk["id"].(string)
		if isvalid != true {
			log.Printf("Unabel to get framework id from the json = %v", frwrk)
			continue
		}
		log.Printf("framework ID %v", id)

		this_frmwrk[id] = false

	}
	G.BroadCast(GossipFrameworks(G.Name))
}

func GetMastersResources(G *Glib, MasterEP string) {

	resp, err := http.Get(fmt.Sprintf("http://%s/metrics/snapshot", MasterEP))

	if err != nil {

		log.Printf("Unable to reach the Master error = %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Printf("Unable to read the body error = %v", err)
		return
	}

	var data map[string]interface{}

	err = json.Unmarshal(body, &data)

	if err != nil {
		log.Printf("Json Unmarshall error = %v", err)
		return
	}

	var tCPU, uCPU float64
	var tMem, uMem float64
	var tDisk, uDisk float64

	var found bool

	tCPU, found = data["master/cpus_total"].(float64)
	if !found {
		log.Printf("Data not found from master /state/snapshot")
	}
	uCPU, found = data["master/cpus_used"].(float64)
	if !found {
		log.Printf("Data not found from master /state/snapshot")
	}
	tMem, found = data["master/mem_total"].(float64)
	if !found {
		log.Printf("Data not found from master /state/snapshot")
	}
	uMem, found = data["master/mem_used"].(float64)
	if !found {
		log.Printf("Data not found from master /state/snapshot")
	}
	tDisk, found = data["master/disk_total"].(float64)
	if !found {
		log.Printf("Data not found from master /state/snapshot")
	}
	uDisk, found = data["master/disk_used"].(float64)
	if !found {
		log.Printf("Data not found from master /state/snapshot")
	}

	//G.Name
	common.ALLDCs.Lck.Lock()
	defer common.ALLDCs.Lck.Unlock()
	var mydc *common.DC
	mydc, available := common.ALLDCs.List[G.Name]

	if !available {
		log.Printf("Our datacenter entry is not found yet")
		mydc = &common.DC{}
		mydc.Name = G.Name
		mydc.Endpoint = common.ThisEP
		mydc.Country = common.ThisCountry
		mydc.City = common.ThisCity
		common.ALLDCs.List[G.Name] = mydc
	}

	log.Println("The values are", tCPU, tMem, tDisk, uCPU, uMem, uDisk)

	mydc.CPU = tCPU
	mydc.MEM = tMem
	mydc.DISK = tDisk
	mydc.Ucpu = uCPU
	mydc.Umem = uMem
	mydc.Udisk = uDisk

	GossipDCInfo(G, mydc)

	return

}

func GossipDCInfo(G *Glib, dc *common.DC) {

	var msg Msg

	msg.Name = G.Name
	msg.Type = "DC"
	msg.Body = dc
	msg_bytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Unable to broadcast DC information to other gosipers Marshall error")
	} else {

		G.BroadCast(msg_bytes)
	}
}

func GossipFrameworks(Name string) []byte {

	var msg Msg
	var FW FrameWorkMsG

	msg.Name = Name

	gmap, isvalid := AllFrameworks[Name]
	if isvalid {
		for name := range gmap {
			FW.FrameWorks = append(FW.FrameWorks, name)
		}
	}

	msg.Body = &FW
	data, _ := json.Marshal(msg)

	return data
}

func CollectMasterData(G *Glib, MasterEP string) {

	//First get the channels initialized
	var FrameworkFrequency, ResourceFrequency time.Duration
	FrameworkFrequency = 5
	ResourceFrequency = 3
	getFrameWorkCh := time.After(time.Second * FrameworkFrequency)
	getMasterResourceCh := time.After(time.Second * ResourceFrequency)

	for {
		select {
		case <-getFrameWorkCh:
			getFrameWorkCh = time.After(time.Second * FrameworkFrequency)
			GetListofFrameworks(G, MasterEP)

		case <-getMasterResourceCh:
			getMasterResourceCh = time.After(time.Second * ResourceFrequency)
			GetMastersResources(G, MasterEP)

		}
	}
}
