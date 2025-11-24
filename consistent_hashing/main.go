package main

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"sort"
)

type StorageNode struct {
	name string
	host string
}

type ConsistentHashingRing struct {
	ringSize *big.Int
	nodes    []*big.Int
	nodeMap  map[string]*StorageNode
}

func NewConsistentHashingRing() *ConsistentHashingRing {
	return &ConsistentHashingRing{
		ringSize: new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil),
		nodes:    make([]*big.Int, 0),
		nodeMap:  make(map[string]*StorageNode),
	}
}

func (c *ConsistentHashingRing) computeHash(key string) *big.Int {
	hash := sha256.Sum256([]byte(key))
	return new(big.Int).SetBytes(hash[:])
}

func (c *ConsistentHashingRing) addNode(node *StorageNode) {
	slot := c.computeHash(node.host)

	c.nodes = append(c.nodes, slot)
	c.nodeMap[slot.String()] = node

	// Keep nodes sorted for efficient clockwise lookup
	sort.Slice(c.nodes, func(i, j int) bool {
		return c.nodes[i].Cmp(c.nodes[j]) < 0
	})
}

func (c *ConsistentHashingRing) deleteNode(node *StorageNode) {
	hash := c.computeHash(node.host)
	hashStr := hash.String()

	delete(c.nodeMap, hashStr)
	for i, nodeHash := range c.nodes {
		if nodeHash.Cmp(hash) == 0 {
			c.nodes = append(c.nodes[:i], c.nodes[i+1:]...)
			break
		}
	}
}

func (c *ConsistentHashingRing) getNode(key string) *StorageNode {
	hash := c.computeHash(key)

	fmt.Printf("Hash for key: %s: %s, length: %d\n", key, hash.String(), len(hash.String()))

	// Binary search for first node >= hash (clockwise search)
	idx := sort.Search(len(c.nodes), func(i int) bool {
		return c.nodes[i].Cmp(hash) >= 0
	})

	fmt.Println("idx: ", idx)

	if idx == len(c.nodes) {
		idx = 0
	}

	return c.nodeMap[c.nodes[idx].String()]
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
		fmt.Printf("node[%d] - %d, length: %d\n", idx, hash, len(hash.String()))
	}

	fmt.Println()

	// Test key distribution
	keys := []string{"user:1", "user:2", "user:3", "user:4", "user:5", "user:6", "user:7"}
	for _, key := range keys {
		targetNode := ch.getNode(key)
		fmt.Printf("Key '%s' maps to node: %s\n", key, targetNode.name)
	}

	ch.deleteNode(ch.nodeMap[ch.nodes[0].String()])

	fmt.Println()
	fmt.Println("After deleting A")
	fmt.Println()

	for _, key := range keys {
		targetNode := ch.getNode(key)
		fmt.Printf("Key '%s' maps to node: %s\n", key, targetNode.name)
	}
}
