package main

import (
	"fmt"
	"hash/crc64"
	"sort"
	"sync"
)

const VirtualNodes = 100

type StorageNode struct {
	name string
	host string
}

type ConsistentHashingRing struct {
	nodes   []uint64                // Simple, contiguous memory
	nodeMap map[uint64]*StorageNode // Fast integer lookups
	mu      sync.RWMutex
}

func NewConsistentHashingRing() *ConsistentHashingRing {
	return &ConsistentHashingRing{
		nodes:   make([]uint64, 0),
		nodeMap: make(map[uint64]*StorageNode),
	}
}

func (c *ConsistentHashingRing) computeHash(key string) uint64 {
	// fast, non-cryptographic hash
	return crc64.Checksum([]byte(key), crc64.MakeTable(crc64.ISO))
}

func (c *ConsistentHashingRing) addNode(node *StorageNode) {
	c.mu.Lock()         // Write Lock
	defer c.mu.Unlock() // Ensure unlock happens even if we panic

	// Add support for virtual nodes, 10 per node
	for i := 0; i < VirtualNodes; i++ {
		virtualNodeKey := fmt.Sprintf("%s#%d", node.host, i)
		slot := c.computeHash(virtualNodeKey)
		c.nodes = append(c.nodes, slot)
		c.nodeMap[slot] = node
	}

	// Keep nodes sorted for efficient clockwise lookup
	sort.Slice(c.nodes, func(i, j int) bool {
		return c.nodes[i] < c.nodes[j]
	})
}

func (c *ConsistentHashingRing) deleteNode(node *StorageNode) {
	c.mu.Lock()         // Write Lock
	defer c.mu.Unlock() // Ensure unlock happens even if we panic

	// Remove all virtual nodes
	for i := 0; i < VirtualNodes; i++ {
		virtualNodeKey := fmt.Sprintf("%s#%d", node.host, i)
		hash := c.computeHash(virtualNodeKey)

		delete(c.nodeMap, hash)
	}

	newNodes := make([]uint64, 0, len(c.nodes)-VirtualNodes)

	for _, hash := range c.nodes {
		if _, exists := c.nodeMap[hash]; exists {
			newNodes = append(newNodes, hash)
		}
	}

	c.nodes = newNodes
}

func (c *ConsistentHashingRing) getNode(key string) *StorageNode {
	c.mu.RLock() // Read Lock
	defer c.mu.RUnlock()

	if len(c.nodes) == 0 {
		return nil
	}

	hash := c.computeHash(key)

	fmt.Printf("Hash for key: %s: %d\n", key, hash)

	// Binary search for first node >= hash (clockwise search)
	idx := sort.Search(len(c.nodes), func(i int) bool {
		return c.nodes[i] >= hash
	})

	fmt.Println("idx: ", idx)

	if idx == len(c.nodes) {
		idx = 0
	}

	return c.nodeMap[c.nodes[idx]]
}

func newStorageNode(name, host string) *StorageNode {
	return &StorageNode{
		name: name,
		host: host,
	}
}

func main() {
	ch := NewConsistentHashingRing()

	ch.addNode(newStorageNode("A", "127.0.0.1"))
	ch.addNode(newStorageNode("B", "192.168.1.2"))
	ch.addNode(newStorageNode("C", "255.255.0.1"))

	fmt.Println("Nodes distribution:")
	for idx, hash := range ch.nodes {
		fmt.Printf("node[%d] - %d\n", idx, hash)
	}

	fmt.Println()

	// Test key distribution
	keys := []string{"user:1", "user:101", "user:1028", "user:17917", "user:2917191", "user:18191910", "user:176181618"}
	for _, key := range keys {
		targetNode := ch.getNode(key)
		fmt.Printf("Key '%s' maps to node: %s\n", key, targetNode.name)
	}

	nodeToBeDeleted := ch.nodeMap[ch.nodes[0]]
	fmt.Printf("Node to be deleted: %s\n", nodeToBeDeleted.name)
	ch.deleteNode(nodeToBeDeleted)

	fmt.Println("Mappings after deletion of node")
	fmt.Println()
	fmt.Println()

	for _, key := range keys {
		targetNode := ch.getNode(key)
		fmt.Printf("Key '%s' maps to node: %s\n", key, targetNode.name)
	}
}
