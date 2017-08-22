package host

import (
	"applatix.io/axdb"
	"applatix.io/axerror"
	"applatix.io/axops/utils"
	"time"
)

type Host struct {
	ID        string   `json:"id,omitempty"`
	Name      string   `json:"name,omitempty"`
	Status    int64    `json:"status,omitempty"`
	PrivateIP []string `json:"private_ip,omitempty"`
	PublicIP  []string `json:"public_ip,omitempty"`
	Mem       float64  `json:"mem,omitempty"`
	CPU       float64  `json:"cpu,omitempty"`
	ECU       float64  `json:"ecu,omitempty"`
	Disk      float64  `json:"disk,omitempty"`
	Model     string   `json:"model,omitempty"`
	Network   float64  `json:"network,omitempty"`
}

func GetAllHosts() ([]Host, *axerror.AXError) {
	hosts := []Host{}
	dbErr := utils.Dbcl.Get(axdb.AXDBAppAXOPS, axdb.AXDBTableHost, nil, &hosts)
	if dbErr != nil {
		return nil, dbErr
	}
	return hosts, nil
}

var cachedHostIdMap map[string]*Host
var hostIdMapMtime int64

func GetHostMap() (map[string]*Host, *axerror.AXError) {
	currentTime := time.Now().UnixNano() / 1e3
	if currentTime < hostIdMapMtime+20*1e6 {
		return cachedHostIdMap, nil
	}

	hostIdMapMtime = currentTime
	hosts, axErr := GetAllHosts()
	if axErr != nil {
		return nil, axErr
	}

	hostIdMap := make(map[string]*Host)
	for i, host := range hosts {
		hostIdMap[host.ID] = &hosts[i]
	}
	cachedHostIdMap = hostIdMap
	return hostIdMap, nil
}

// We may be using a variety types of hosts. However, the admission policy is based just on cpu and memory, regardless of type.
// Using the average price avoids penalizing a certain service because of the scheduling decisions. For instance, two builds,
// one run on on-demand, one run on spot instance, that use the same resources, should show the same cost.
func GetAveragePrice(cached bool) (*utils.EC2Price, *axerror.AXError) {
	if cached {
		if cachedPrice != nil {
			return cachedPrice, nil
		}
	}
	return getAveragePrice()
}

var cachedPrice *utils.EC2Price

func getAveragePrice() (*utils.EC2Price, *axerror.AXError) {
	hostMap, err := GetHostMap()
	if err != nil {
		return nil, err
	}

	var cpu int
	var mem float64
	var cost float64
	var count int

	for _, host := range hostMap {
		hostPrice := utils.ModelPrice[host.Model]
		if hostPrice == nil {
			utils.ErrorLog.Printf("Error, host model %s is not found in pricing table", host.Model)
			continue
		}

		// we don't add the host cost, which is meaningless.
		cpu += hostPrice.Cpu
		mem += hostPrice.Mem
		cost += hostPrice.Cost
		count++
	}
	if cpu == 0 || mem == 0.0 {
		// all the hosts are gone, something is wrong.
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessage("Can't get host info to calculate a valid average price")
	}

	utils.InfoLog.Printf("avg price of %d hosts, host Cost %f CoreCost %f MemCost %v", count, cost/float64(count), cost/float64(cpu), cost/mem/1024)

	recent := &utils.EC2Price{
		Cpu:      cpu / count,
		Mem:      mem / float64(count),
		Cost:     cost / float64(count),
		CoreCost: cost / float64(cpu),
		MemCost:  cost / mem / 1024,
	}

	cachedPrice = recent
	return recent, nil
}

func GetPriceForHostId(hostIdMap map[string]*Host, hostId string) *utils.EC2Price {
	if hostIdMap == nil {
		return nil
	}

	host := hostIdMap[hostId]
	if host == nil {
		utils.InfoLog.Printf("Can't find host id %s in the host table", hostId)
		return nil
	}

	modelPrice := utils.ModelPrice[host.Model]
	if modelPrice == nil {
		utils.InfoLog.Printf("Can't find price info for model %s", host.Model)
		return nil
	}
	return modelPrice
}
