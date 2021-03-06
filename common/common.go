package common

import (
	"fmt"
	"log"
	"sync"
)

//Declare some structure that will eb common for both Anonymous and Gossiper modulesv
type DC struct {
	OutOfResource bool
	Name          string
	City          string
	Country       string
	Endpoint      string
	CPU           float64
	MEM           float64
	DISK          float64
	Ucpu          float64 //Remaining CPU
	Umem          float64 //Remaining Memory
	Udisk         float64 //Remaining Disk
	LastUpdate    int64   //Time stamp of current DC status
	LastOOR       int64   //Time stamp of when was the last OOR Happpend
	IsActiveDC    bool
}

type alldcs struct {
	Lck  sync.Mutex
	List map[string]*DC
}

type rttbwGossipers struct {
	Lck  sync.Mutex
	List map[string]int64
}

type toanon struct {
	Ch  chan bool
	M   map[string]bool
	Lck sync.Mutex
}

//global consul config
type ConsulConfig struct {
	IsLeader    bool
	DCEndpoint  string
	StorePreFix string
	DCName      string
}

//Declare somecommon types that will be used accorss the goroutines
var (
	ToAnon             toanon    //Structure Sending messages to FedComms module via TCP client
	ALLDCs             alldcs    //The data structure that stores all the Datacenter information
	ThisDCName         string    //This DataCenter's Name
	ThisEP             string    //Thsi Datacenter's Endpoint
	ThisCity           string    //This Datacenters City
	ThisCountry        string    //This Datacentes Country
	ResourceThresold   int       //Threshold value of any resource (CPU, MEM or Disk) after which we need to broadcast OOR
	TriggerPolicyCh    chan bool //Polcy Engine will listen in this Channel
	RttOfPeerGossipers rttbwGossipers
)

func init() {

	ToAnon.M = make(map[string]bool)
	ToAnon.Ch = make(chan bool)
	TriggerPolicyCh = make(chan bool)
	ALLDCs.List = make(map[string]*DC)
	ResourceThresold = 100
	RttOfPeerGossipers.List = make(map[string]int64)
	fmt.Printf("Initalizeing Common")

}

func SupressFrameWorks() {

	log.Println("SupressFrameWorks: called")
	ToAnon.Lck.Lock()
	for k := range ToAnon.M {
		ToAnon.M[k] = true
	}
	ToAnon.Lck.Unlock()

	ToAnon.Ch <- true

	// we set the IsActiveDC flag to TRUE
	_, available := ALLDCs.List[ThisDCName]
	if !available {
		log.Printf("SupressFrameWorks: DC information not available")
		return
	}

	ALLDCs.List[ThisDCName].IsActiveDC = false
	log.Println("SupressFrameWorks: returning")

}

func UnSupressFrameWorks() {
	log.Println("UnSupressFrameWorks: called")
	ToAnon.Lck.Lock()
	for k := range ToAnon.M {
		ToAnon.M[k] = false
	}
	ToAnon.Lck.Unlock()

	ToAnon.Ch <- true

	// we set the IsActiveDC flag to TRUE
	_, available := ALLDCs.List[ThisDCName]
	if !available {
		log.Printf("UnSupressFrameWorks: DC information not available")
		return
	}

	ALLDCs.List[ThisDCName].IsActiveDC = true

	log.Println("UnSupressFrameWorks: returning")
}

func IsCommonMapEmpty() bool {
	return (len(ToAnon.M) == 0)

}
