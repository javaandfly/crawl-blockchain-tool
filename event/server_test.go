package event

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func TestEventServer(t *testing.T) {
	nodes := []string{
		"https://rpc.payload.de",
		"https://rpc.ankr.com/eth",
		"wss://ethereum-rpc.publicnode.com",
		"https://ethereum-rpc.publicnode.com",
		// more nodes...
	}
	clients := make([]*EthClinet, 0)
	for _, node := range nodes {
		client, err := ethclient.Dial(node)
		if err != nil {
			continue
		}
		clients = append(clients, &EthClinet{client: client, sign: Running})
	}

	addresses := []common.Address{common.HexToAddress("0x2EDfFbc62C3dfFD2a8FbAE3cd83A986B5bbB5495")}

	ei := NewEventInfo(100, 20329154, 6, clients, addresses)

	// for i := 0; i < 10; i++ {
	// 	ei.DoEvent(context.Background())
	// }
	err := ei.DoEvent()
	if err != nil {
		t.Error(err)
		return
	}

	for v := range ei.GetEventChan() {
		t.Log(v)
		return
	}
}

func TestCheckNodeStatusServer(t *testing.T) {
	//制造错误节点
	nodes := []string{
		"https://rpc.payloae.de123//1231231",
		"https://rpc.anke.com/eth123/12312312",
		// more nodes...
	}
	clients := make([]*EthClinet, 0)
	for _, node := range nodes {
		//忽略错误
		client, _ := ethclient.Dial(node)

		clients = append(clients, &EthClinet{client: client, sign: Running})
	}

	addresses := []common.Address{common.HexToAddress("0x2EDfFbc62C3dfFD2a8FbAE3cd83A986B5bbB5495")}

	ei := NewEventInfo(1, 20329154, 6, clients, addresses)

	go ei.checkClientPoolStatus(context.Background(), 500, 3)

	for i := 0; i < 10; i++ {
		err := ei.DoEvent()
		if err != nil {
			continue
		}
	}

	time.Sleep(time.Second * 3)

	if len(ei.clients) == 0 {
		t.Log("error node all done")
	} else {
		t.Fail()
	}

}
