package etcdmgr

import (
	"testing"

	"github.com/rook/rook/pkg/clusterd"
	"github.com/rook/rook/pkg/clusterd/inventory"
	clusterdtest "github.com/rook/rook/pkg/clusterd/test"
	"github.com/rook/rook/pkg/etcdmgr/test"
	"github.com/rook/rook/pkg/util"
	"github.com/stretchr/testify/assert"
)

// TestEtcdMgrLeaders tests Etcd Cluster's grow scenario
func TestEtcdMgrLeaderGrow(t *testing.T) {
	mockContext := test.MockContext{}
	nodes := make(map[string]*inventory.NodeConfig)
	inv := &inventory.Config{Nodes: nodes}

	// adding 1.2.3.4 as the first/existing cluster member
	nodes["a"] = &inventory.NodeConfig{IPAddress: "1.2.3.4"}
	mockContext.AddMembers([]string{"http://1.2.3.4:53379"})
	etcdmgrService := etcdMgrLeader{context: &mockContext}
	etcdmgrService.StartWatchEvents()
	defer etcdmgrService.Close()

	nodes["b"] = &inventory.NodeConfig{IPAddress: "2.3.4.5"}
	nodes["c"] = &inventory.NodeConfig{IPAddress: "3.4.5.6"}

	etcdClient := util.NewMockEtcdClient()
	context := &clusterd.Context{
		EtcdClient: etcdClient,
		Inventory:  inv,
	}

	// mock the agent responses that the deployments were successful to create an instance of embeddedEtcd
	etcdClient.WatcherResponses["/rook/_notify/b/etcd/status"] = "succeeded"
	etcdClient.WatcherResponses["/rook/_notify/c/etcd/status"] = "succeeded"

	addNodeEvt := clusterd.NewAddNodeEvent(context, "b")
	etcdmgrService.Events() <- addNodeEvt

	// wait for the event queue to be empty
	clusterdtest.WaitForEvents(&etcdmgrService)

	// there might be a&c or b&c in the desired states
	assert.True(t, etcdClient.GetChildDirs("/rook/services/etcd/desired").Count() == 2)
	assert.Equal(t, etcdClient.GetValue("/rook/services/etcd/desired/b/ipaddress"), "2.3.4.5")
	assert.Equal(t, etcdClient.GetValue("/rook/services/etcd/desired/c/ipaddress"), "3.4.5.6")
}

func TestEtcdMgrLeaderShrink(t *testing.T) {
	mockContext := test.MockContext{}
	nodes := make(map[string]*inventory.NodeConfig)
	inv := &inventory.Config{Nodes: nodes}

	// adding 1.2.3.4 as the first/existing cluster member
	nodes["a"] = &inventory.NodeConfig{IPAddress: "1.2.3.4"}
	mockContext.AddMembers([]string{"http://1.2.3.4:53379"})
	nodes["b"] = &inventory.NodeConfig{IPAddress: "2.3.4.5"}
	mockContext.AddMembers([]string{"http://2.3.4.5:53379"})
	nodes["c"] = &inventory.NodeConfig{IPAddress: "3.4.5.6"}
	mockContext.AddMembers([]string{"http://3.4.5.6:53379"})
	etcdmgrService := etcdMgrLeader{context: &mockContext}
	etcdmgrService.StartWatchEvents()
	defer etcdmgrService.Close()

	nodes["a"] = &inventory.NodeConfig{IPAddress: "1.2.3.4"}
	nodes["b"] = &inventory.NodeConfig{IPAddress: "2.3.4.5"}
	nodes["c"] = &inventory.NodeConfig{IPAddress: "3.4.5.6"}

	etcdClient := util.NewMockEtcdClient()
	context := &clusterd.Context{
		EtcdClient: etcdClient,
		Inventory:  inv,
	}

	// mock the agent responses that the deployments were successful to create an instance of embeddedEtcd
	etcdClient.WatcherResponses["/rook/_notify/a/etcd/status"] = "succeeded"
	etcdClient.WatcherResponses["/rook/_notify/c/etcd/status"] = "succeeded"
	etcdClient.WatcherResponses["/rook/services/etcd/desired/a/ipaddress"] = "1.2.3.4"
	etcdClient.WatcherResponses["/rook/services/etcd/desired/b/ipaddress"] = "2.3.4.5"
	etcdClient.WatcherResponses["/rook/services/etcd/desired/c/ipaddress"] = "2.3.4.5"

	unhealthyNode := clusterd.UnhealthyNode{AgeSeconds: 10, NodeID: "c"}
	unhealthyNodeEvt := clusterd.NewUnhealthyNodeEvent(context, []*clusterd.UnhealthyNode{&unhealthyNode})
	etcdmgrService.Events() <- unhealthyNodeEvt

	// wait for the event queue to be empty
	clusterdtest.WaitForEvents(&etcdmgrService)

	// two possibilities: there might be a&c or b&c in the desired states
	assert.True(t, etcdClient.GetChildDirs("/rook/services/etcd/desired").Count() == 0)
}
