package event

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/javaandfly/crawl-blockchain-tool/utils"
)

type ClientStatus int

const (
	Running ClientStatus = iota + 1
	Fault
	Stop
)

type EventInfo struct {
	maxBlocksNumber  uint64
	beginBlockNumber uint64
	lateBlock        uint64
	mu               sync.RWMutex
	eventChan        chan []types.Log
	clients          []*EthClinet
	addresses        []common.Address
}

type EthClinet struct {
	client    *ethclient.Client
	sign      ClientStatus
	tryNumber int
}

func NewEventInfo(
	maxBlocksNumber,
	beginBlockNumber uint64,
	lateBlock uint64,
	clients []*EthClinet,
	addresses []common.Address,
) *EventInfo {

	return &EventInfo{
		maxBlocksNumber:  maxBlocksNumber,
		beginBlockNumber: beginBlockNumber,
		lateBlock:        lateBlock,
		mu:               sync.RWMutex{},
		eventChan:        make(chan []types.Log, 20),
		clients:          clients,
		addresses:        addresses,
	}
}

func (ei *EventInfo) DoEvent() error {

	var err error

	index, client := ei.getOneClient()
	if client == nil {
		err = fmt.Errorf("ei clinet is nil")
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("DoEvent panic !! %v", r)
		}
		if err != nil {
			ei.refreshErrorStatus(index)
		}
	}()

	blockNum, err := client.client.BlockNumber(context.Background())
	if err != nil {
		return err
	}

	ei.refreshStatus(index)

	//不要跟的太紧，可能发生重组
	blockNum = blockNum - uint64(ei.lateBlock)

	endNum := blockNum
	if blockNum >= uint64(ei.beginBlockNumber)+ei.maxBlocksNumber {
		endNum = uint64(ei.beginBlockNumber) + ei.maxBlocksNumber
	}
	//TODO: 后续优化为多协程模型
	logs, err := client.client.FilterLogs(context.Background(), ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(ei.beginBlockNumber)),
		ToBlock:   big.NewInt(int64(endNum)),
		Addresses: ei.addresses,
	})

	//可自动服务降级

	if err != nil {
		return err
	}

	ei.eventChan <- logs

	ei.beginBlockNumber = endNum

	return err

}

func (ei *EventInfo) GetEventChan() <-chan []types.Log {
	return ei.eventChan
}

func (ei *EventInfo) getOneClient() (int64, *EthClinet) {
	ei.mu.RLock()
	defer ei.mu.RUnlock()

	return utils.RandomOne(ei.clients)
}

func (ei *EventInfo) refreshStatus(index int64) {
	ei.mu.Lock()
	defer ei.mu.Unlock()
	if len(ei.clients) <= int(index) {
		return
	}
	ei.clients[index].sign = Running
	ei.clients[index].tryNumber = 0
}

func (ei *EventInfo) refreshErrorStatus(index int64) {
	ei.mu.Lock()
	defer ei.mu.Unlock()

	if len(ei.clients) <= int(index) {
		return
	}

	ei.clients[index].sign = Fault
	ei.clients[index].tryNumber++
}

/*
*
check 失活节点
*/
func (ei *EventInfo) checkClientPoolStatus(
	ctx context.Context,
	checkTimeGap, maxLimit int,

) {
	timer := time.NewTimer(0)
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C: // wait for timer triggered
		}

		clients := func(ei *EventInfo) []*EthClinet {

			clients := make([]*EthClinet, 0)

			ei.mu.RLock()
			defer ei.mu.RUnlock()
			for _, client := range ei.clients {
				//存活
				if client.sign != Fault && client.tryNumber < maxLimit {
					ei.clients = append(ei.clients, client)
				}

			}

			return clients

		}(ei)

		func(ei *EventInfo) {
			ei.mu.Lock()
			defer ei.mu.Unlock()

			ei.clients = clients
		}(ei)

		timer.Reset(time.Duration(checkTimeGap) * time.Millisecond)
	}
}
